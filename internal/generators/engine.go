package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"Game-Developers-World/seed/internal/generators/templates"
)

var validTypes = map[string]bool{
	"component": true,
	"trait":     true,
	"entity":    true,
	"archetype": true,
}

var typeDir = map[string]string{
	"component": "Component",
	"trait":     "Trait",
	"entity":    "Entity",
	"archetype": "Archetype",
}

func CreateModel(compType, name string) error {
	if !validTypes[compType] {
		return fmt.Errorf("unknown type %q: valid types are component, trait, entity, archetype", compType)
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

	fmt.Printf("Synced %d components, %d traits, %d entities, %d archetypes.\n",
		counts["component"], counts["trait"], counts["entity"], counts["archetype"])

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

		fmt.Printf("  Generated  %s\n", outputPath)
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
	}
	return ""
}

// Generate creates a YAML model and renders its C++ header (legacy behavior).
func Generate(compType, name string) error {
	if err := CreateModel(compType, name); err != nil {
		return err
	}
	return renderOne(compType, name)
}

func renderOne(compType, name string) error {
	modelsDir := filepath.Join(".", "Models", compType)
	yamlPath := filepath.Join(modelsDir, name+".yaml")

	model, err := readModel(compType, yamlPath)
	if err != nil {
		return fmt.Errorf("reading model: %w", err)
	}

	tplContent, err := templates.Load(compType)
	if err != nil {
		return fmt.Errorf("loading template: %w", err)
	}

	tpl, err := template.New(compType).Parse(string(tplContent))
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	outDir := outputDir(model)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outputPath := filepath.Join(outDir, name+".hpp")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if err := tpl.Execute(f, model); err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}

	fmt.Printf("  Generated  %s\n", outputPath)
	return nil
}
