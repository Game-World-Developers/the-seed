package generators

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"Game-Developers-World/seed/internal/generators/templates"
)

var validTypes = map[string]bool{
	"component":     true,
	"trait":         true,
	"entity":        true,
	"archetype":     true,
	"state_machine": true,
	"system":        true,
}

var typeDir = map[string]string{
	"component":     "Component",
	"trait":         "Trait",
	"entity":        "Entity",
	"archetype":     "Archetype",
	"state_machine": "StateMachine",
	"system":        "System",
}

func CreateModel(compType, name string) error {
	if !validTypes[compType] {
		return fmt.Errorf("unknown type %q: valid types are component, trait, entity, archetype, state_machine, system", compType)
	}

	modelsDir := filepath.Join(".", "Models", typeDir[compType])
	yamlPath := filepath.Join(modelsDir, name+".yaml")

	if _, err := os.Stat(yamlPath); err == nil {
		fmt.Printf("  Already exists  %s\n", yamlPath)
		return nil
	}

	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("creating models directory: %w", err)
	}

	data, err := defaultModel(compType, name)
	if err != nil {
		return fmt.Errorf("generating default model: %w", err)
	}

	if err := os.WriteFile(yamlPath, data, 0644); err != nil {
		return fmt.Errorf("writing model file: %w", err)
	}

	fmt.Printf("  Created  %s\n", yamlPath)
	return nil
}

func Sync() error {
	counts := map[string]int{}
	var syncErr error

	for compType := range validTypes {
		n, err := syncDir(compType)
		if err != nil {
			syncErr = err
		}
		counts[compType] = n
	}

	fmt.Printf("Synced %d components, %d traits, %d entities, %d archetypes, %d state_machines, %d systems.\n",
		counts["component"], counts["trait"], counts["entity"],
		counts["archetype"], counts["state_machine"], counts["system"])

	if err := injectBootstrap(counts); err != nil {
		fmt.Fprintf(os.Stderr, "  Error updating bootstrap: %v\n", err)
	}

	return syncErr
}

func syncDir(compType string) (int, error) {
	modelsDir := filepath.Join(".", "Models", typeDir[compType])
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading %s: %w", modelsDir, err)
	}

	tplContent, err := templates.Load(compType)
	if err != nil {
		return 0, fmt.Errorf("loading template: %w", err)
	}

	tpl, err := template.New(compType).Parse(string(tplContent))
	if err != nil {
		return 0, fmt.Errorf("parsing template: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		yamlPath := filepath.Join(modelsDir, entry.Name())

		model, err := readModel(compType, yamlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error reading %s: %v\n", yamlPath, err)
			continue
		}

		outDir := outputDir(model)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "  Error creating directory %s: %v\n", outDir, err)
			continue
		}

		outputPath := filepath.Join(outDir, name+".hpp")
		f, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error creating %s: %v\n", outputPath, err)
			continue
		}

		if err := tpl.Execute(f, model); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "  Error rendering %s: %v\n", outputPath, err)
			continue
		}
		f.Close()

		if err := formatCode(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not format %s: %v\n", outputPath, err)
		}
		fmt.Printf("  Generated  %s\n", outputPath)

		if compType == "system" {
			if err := writeCpp(compType, model, name); err != nil {
				fmt.Fprintf(os.Stderr, "  Error generating .cpp for %s: %v\n", name, err)
			} else {
				cppPath := filepath.Join(".", "Src", "Game", name+".cpp")
				fmt.Printf("  Generated  %s\n", cppPath)
			}
		}

		count++
	}

	return count, nil
}

func outputDir(model any) string {
	switch m := model.(type) {
	case ComponentModel:
		return filepath.Join(".", "Include", m.Namespace, "Component")
	case TraitModel:
		return filepath.Join(".", "Include", m.Namespace, "Trait")
	case EntityModel:
		return filepath.Join(".", "Include", m.Namespace, "Entity")
	case ArchetypeModel:
		return filepath.Join(".", "Include", m.Namespace, "Archetype")
	case StateMachineModel:
		return filepath.Join(".", "Include", m.Namespace)
	case SystemModel:
		return filepath.Join(".", "Include", m.Namespace)
	}
	return ""
}

func writeCpp(compType string, model any, name string) error {
	tplContent, err := templates.LoadCpp(compType)
	if err != nil {
		return err
	}
	tpl, err := template.New(compType + "_cpp").Parse(tplContent)
	if err != nil {
		return err
	}
	srcDir := filepath.Join(".", "Src", "Game")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return err
	}
	outputPath := filepath.Join(srcDir, name+".cpp")
	if _, err := os.Stat(outputPath); err == nil {
		return nil // already exists, don't overwrite user code
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := tpl.Execute(f, model); err != nil {
		return err
	}
	return formatCode(outputPath)
}

func collectRegistrations(compType string) (headers []string, calls []string, err error) {
	dir, ok := typeDir[compType]
	if !ok {
		return nil, nil, nil
	}
	modelsDir := filepath.Join(".", "Models", dir)
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		yamlPath := filepath.Join(modelsDir, entry.Name())
		model, err := readModel(compType, yamlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error reading %s: %v\n", yamlPath, err)
			continue
		}
		ns := namespaceOf(model)
		if compType == "component" {
			headers = append(headers, fmt.Sprintf(`#include <%s/Component/%s.hpp>`, ns, name))
			calls = append(calls, fmt.Sprintf(`    register_%s(rt);`, name))
		} else if compType == "state_machine" {
			headers = append(headers, fmt.Sprintf(`#include <%s/%s.hpp>`, ns, name))
			calls = append(calls, fmt.Sprintf(`    register_%s(rt);`, name))
		} else if compType == "system" {
			headers = append(headers, fmt.Sprintf(`#include <%s/%s.hpp>`, ns, name))
			calls = append(calls, fmt.Sprintf(`    register_%s(rt);`, name))
		}
	}
	return headers, calls, nil
}

func namespaceOf(model any) string {
	switch m := model.(type) {
	case ComponentModel:
		return m.Namespace
	case StateMachineModel:
		return m.Namespace
	case SystemModel:
		return m.Namespace
	}
	return "Core"
}

func injectBootstrap(counts map[string]int) error {
	bootstrapPath := filepath.Join(".", "Src", "Game", "bootstrap.cpp")
	data, err := os.ReadFile(bootstrapPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := string(data)

	// Inject component registrations
	if counts["component"] > 0 {
		compHeaders, compCalls, err := collectRegistrations("component")
		if err != nil {
			return err
		}
		content = replaceSeedBlock(content, "components", compHeaders, compCalls)
	}

	// Inject controller registrations (state_machines + systems)
	controllerCount := counts["state_machine"] + counts["system"]
	if controllerCount > 0 {
		var allHeaders []string
		var allCalls []string
		for _, ct := range []string{"state_machine", "system"} {
			h, c, err := collectRegistrations(ct)
			if err != nil {
				return err
			}
			allHeaders = append(allHeaders, h...)
			allCalls = append(allCalls, c...)
		}
		content = replaceSeedBlock(content, "controllers", allHeaders, allCalls)
	}

	return os.WriteFile(bootstrapPath, []byte(content), 0644)
}

func replaceSeedBlock(content, blockName string, headers, calls []string) string {
	beginMarker := fmt.Sprintf("// @seed:begin(%s)", blockName)
	endMarker := fmt.Sprintf("// @seed:end(%s)", blockName)

	beginLine := strings.Index(content, beginMarker)
	if beginLine == -1 {
		return content
	}
	// Detect indentation from the begin marker line
	indent := ""
	for i := beginLine - 1; i >= 0; i-- {
		if content[i] == '\n' {
			indent = content[i+1 : beginLine]
			break
		}
	}

	beginBlock := beginLine + len(beginMarker)
	// Skip past the newline after the begin marker
	if beginBlock < len(content) && content[beginBlock] == '\n' {
		beginBlock++
	}

	endBlock := strings.Index(content[beginBlock:], endMarker)
	if endBlock == -1 {
		return content
	}

	var sb strings.Builder
	sb.WriteString(content[:beginBlock])
	for _, h := range headers {
		sb.WriteString(indent)
		sb.WriteString(h)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	for _, c := range calls {
		sb.WriteString(indent)
		sb.WriteString(c)
		sb.WriteString("\n")
	}
	sb.WriteString(content[beginBlock+endBlock:])
	return sb.String()
}

func formatCode(path string) error {
	if _, err := exec.LookPath("clang-format"); err != nil {
		return nil
	}
	cmd := exec.Command("clang-format", "-i", path)
	return cmd.Run()
}
