package generators

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"Game-Developers-World/seed/internal/baker"
	"Game-Developers-World/seed/internal/generators/templates"
)

func fnv1a(s string) string {
	h := uint32(2166136261)
	for _, c := range []byte(s) {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("0x%08X", h)
}

var validTypes = map[string]bool{
	"component":     true,
	"trait":         true,
	"entity":        true,
	"archetype":     true,
	"state_machine": true,
	"event":         true,
	"system":        true,
	"asset":         true,
	"block":         true,
}

type ModelEntry struct {
	Type      string
	Name      string
	Namespace string
	Path      string
}

func scanAllModels() ([]ModelEntry, error) {
	root := filepath.Join(".", "Models")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading Models: %w", err)
	}

	var models []ModelEntry
	for _, de := range entries {
		if !de.IsDir() {
			continue
		}
		domain := de.Name()
		dir := filepath.Join(root, domain)
		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not read directory %s: %v\n", dir, err)
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			path := filepath.Join(dir, f.Name())
			t, name, ns, err := detectModel(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: could not parse %s: %v\n", path, err)
				continue
			}
			models = append(models, ModelEntry{
				Type: t, Name: name, Namespace: ns, Path: path,
			})
		}
	}
	return models, nil
}

var typeOrder = []string{"component", "trait", "entity", "archetype", "state_machine", "event", "system", "asset", "block"}

func CreateModel(compType, name string) error {
	if !validTypes[compType] {
		return fmt.Errorf("unknown type %q: valid types are component, trait, entity, archetype, state_machine, system", compType)
	}
	modelsDir := filepath.Join(".", "Models", TypeDir[compType])
	yamlPath := filepath.Join(modelsDir, name+".yaml")

	if _, err := os.Stat(yamlPath); err == nil {
		fmt.Printf("  %11s  %s\n", "identical", yamlPath)
		return nil
	}

	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("creating models directory: %w", err)
	}

	data, err := defaultModel(compType, "Core", name)
	if err != nil {
		return fmt.Errorf("generating default model: %w", err)
	}

	if err := os.WriteFile(yamlPath, data, 0644); err != nil {
		return fmt.Errorf("writing model file: %w", err)
	}

	fmt.Printf("  %11s  %s\n", "create", yamlPath)
	return nil
}

func Sync() error {
	models, err := scanAllModels()
	if err != nil {
		return fmt.Errorf("scanning models: %w", err)
	}

	// Build model registry for namespace resolution
	reg := NewModelRegistry()
	for _, m := range models {
		reg.Register(m.Type, m.Name, m.Namespace)
	}

	counts := map[string]int{}
	var allErrs []error

	byType := map[string][]ModelEntry{}
	for _, m := range models {
		byType[m.Type] = append(byType[m.Type], m)
	}

	for _, t := range typeOrder {
		entries := byType[t]
		if len(entries) == 0 {
			continue
		}
		if t == "block" {
			n, err := syncBlocks(entries, reg)
			if err != nil {
				allErrs = append(allErrs, err)
			}
			counts[t] = n
		} else {
			n, err := syncEntries(t, entries, reg)
			if err != nil {
				allErrs = append(allErrs, err)
			}
			counts[t] = n
		}
	}

	fmt.Printf("Synced %d components, %d traits, %d entities, %d archetypes, %d state_machines, %d events, %d assets, %d systems, %d blocks.\n",
		counts["component"], counts["trait"], counts["entity"],
		counts["archetype"], counts["state_machine"], counts["event"], counts["asset"], counts["system"],
		counts["block"])

	if err := injectBootstrap(counts); err != nil {
		fmt.Fprintf(os.Stderr, "  Error updating bootstrap: %v\n", err)
	}

	return errors.Join(allErrs...)
}

func syncEntries(compType string, entries []ModelEntry, reg *ModelRegistry) (int, error) {
	tplContent, err := templates.Load(compType)
	if err != nil {
		return 0, fmt.Errorf("loading template: %w", err)
	}

	funcMap := template.FuncMap{
		"eventId": fnv1a,
	}
	tpl, err := template.New(compType).Funcs(funcMap).Parse(string(tplContent))
	if err != nil {
		return 0, fmt.Errorf("parsing template: %w", err)
	}

	count := 0
	for _, entry := range entries {
		model, err := readModel(compType, entry.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error reading %s: %v\n", entry.Path, err)
			continue
		}

		if err := model.Resolve(reg); err != nil {
			fmt.Fprintf(os.Stderr, "  Error resolving %s: %v\n", entry.Path, err)
			continue
		}

		if model.HasParts() {
			n, err := syncParts(compType, model, entry.Name, entry.Namespace, tpl)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Error syncing parts for %s: %v\n", entry.Name, err)
			}
			count += n
		} else {
			outDir := filepath.Join(".", "Include", entry.Namespace)
			if compType == "system" && model.HasParts() {
				outDir = filepath.Join(outDir, entry.Name)
			}
			if err := os.MkdirAll(outDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "  Error creating directory %s: %v\n", outDir, err)
				continue
			}

			outputPath := filepath.Join(outDir, entry.Name+".hpp")
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
			fmt.Printf("  %11s  %s\n", "create", outputPath)

			if compType == "system" {
				if err := writeCpp(compType, model, entry.Name, ""); err != nil {
					fmt.Fprintf(os.Stderr, "  Error generating .cpp for %s: %v\n", entry.Name, err)
				} else {
					cppDir := filepath.Join(".", "Src", "Game")
					if model.HasParts() {
						cppDir = filepath.Join(cppDir, entry.Name)
					}
					cppPath := filepath.Join(cppDir, entry.Name+".cpp")
					fmt.Printf("  %11s  %s\n", "create", cppPath)
				}
			}

			count++
		}
	}

	return count, nil
}

func syncBlocks(entries []ModelEntry, reg *ModelRegistry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	// Determine project root (where .seed_project lives)
	wd, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("getting work dir: %w", err)
	}

	profile := baker.ReadProfile(wd)

	// Load the block template
	tplContent, err := templates.Load("block")
	if err != nil {
		return 0, fmt.Errorf("loading block template: %w", err)
	}

	funcMap := template.FuncMap{}
	tpl, err := template.New("block").Funcs(funcMap).Parse(tplContent)
	if err != nil {
		return 0, fmt.Errorf("parsing block template: %w", err)
	}

	// Read all block models
	var blockModels []*BlockModel
	var atlasPath string
	for _, entry := range entries {
		model, err := readModel("block", entry.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error reading block %s: %v\n", entry.Path, err)
			continue
		}
		bm := model.(*BlockModel)
		blockModels = append(blockModels, bm)
		if bm.AtlasPath != "" {
			atlasPath = bm.AtlasPath
		}
	}

	if len(blockModels) == 0 {
		return 0, nil
	}

	// Bake the texture atlas to get tile colors
	tileColors, err := baker.BakeAtlasIfExists(wd, atlasPath, profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: could not bake atlas: %v (using YAML colors)\n", err)
		// Fall back to colors defined in YAML
	}

	// Build template data
	type BlockEntry struct {
		Name         string
		EnumName     string
		ColorTop     uint32
		ColorBottom  uint32
		ColorSide    uint32
		Solid        string
		Transparent  string
		Fluid        string
		Hardness     float32
	}
	type BlockData struct {
		Namespace string
		Profile   string
		Blocks    []BlockEntry
	}

	entryName := "BlockRegistry"
	ns := blockModels[0].Namespace

	var blocks []BlockEntry
	for _, bm := range blockModels {
		enumName := bm.Name
		if len(enumName) > 0 && enumName[0] >= 'a' && enumName[0] <= 'z' {
			enumName = strings.ToUpper(enumName[:1]) + enumName[1:]
		}

		// Determine colors: prefer atlas-baked, fall back to YAML-defined
		colorTop := bm.Colors.Top
		colorBottom := bm.Colors.Bottom
		colorSide := bm.Colors.Side

		if tileColors != nil && int(bm.TileIndex) < len(tileColors) {
			// Atlas-baked colors override YAML colors
			colorTop = baker.ApplyLightMultiplier(tileColors[bm.TileIndex].Average, 1.0)
			colorBottom = baker.ApplyLightMultiplier(tileColors[bm.TileIndex].Average, 0.5)
			colorSide = baker.ApplyLightMultiplier(tileColors[bm.TileIndex].Average, 0.7)
		}

		solidStr := "true"
		if !bm.Solid {
			solidStr = "false"
		}
		transparentStr := "true"
		if !bm.Transparent {
			transparentStr = "false"
		}
		fluidStr := "true"
		if !bm.Fluid {
			fluidStr = "false"
		}

		blocks = append(blocks, BlockEntry{
			Name:         bm.Name,
			EnumName:     enumName,
			ColorTop:     colorTop,
			ColorBottom:  colorBottom,
			ColorSide:    colorSide,
			Solid:        solidStr,
			Transparent:  transparentStr,
			Fluid:        fluidStr,
			Hardness:     float32(bm.Hardness),
		})
	}

	data := BlockData{
		Namespace: ns,
		Profile:   string(profile),
		Blocks:    blocks,
	}

	// Write output header
	outDir := filepath.Join(".", "Include", ns)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return 0, fmt.Errorf("creating output directory: %w", err)
	}

	outputPath := filepath.Join(outDir, entryName+".hpp")
	f, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("creating %s: %w", outputPath, err)
	}
	defer f.Close()

	if err := tpl.Execute(f, data); err != nil {
		return 0, fmt.Errorf("rendering %s: %w", outputPath, err)
	}

	if err := formatCode(outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: could not format %s: %v\n", outputPath, err)
	}
	fmt.Printf("  %11s  %s\n", "create", outputPath)

	return len(blockModels), nil
}

func syncParts(compType string, model Model, parentName, parentNS string, tpl *template.Template) (int, error) {
	baseDir := filepath.Join(".", "Include", parentNS, parentName)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return 0, fmt.Errorf("creating parent dir: %w", err)
	}

	parts, extras := model.ExtractParts()
	count := 0
	for i, partName := range parts {
		outputPath := filepath.Join(baseDir, partName+".hpp")
		f, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error creating %s: %v\n", outputPath, err)
			continue
		}
		data := partData(model, partName, extras[i])
		if err := tpl.Execute(f, data); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "  Error rendering %s: %v\n", outputPath, err)
			continue
		}
		f.Close()

		if err := formatCode(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not format %s: %v\n", outputPath, err)
		}
		fmt.Printf("  %11s  %s\n", "create", outputPath)

		if compType == "system" {
			if err := writeCpp(compType, model, partName, parentName); err != nil {
				fmt.Fprintf(os.Stderr, "  Error generating .cpp for %s: %v\n", partName, err)
			} else {
				cppPath := filepath.Join(".", "Src", "Game", parentName, partName+".cpp")
				fmt.Printf("  %11s  %s\n", "create", cppPath)
			}
		}

		count++
	}
	return count, nil
}

func writeCpp(compType string, model Model, name string, parent string) error {
	var tplContent string
	var err error
	if parent != "" {
		tplContent, err = templates.LoadPartCpp(compType)
	} else {
		tplContent, err = templates.LoadCpp(compType)
	}
	if err != nil {
		return err
	}
	tpl, err := template.New(compType + "_cpp").Parse(tplContent)
	if err != nil {
		return err
	}
	srcDir := filepath.Join(".", "Src", "Game")
	if parent != "" {
		srcDir = filepath.Join(srcDir, parent)
	} else if compType == "system" && model.HasParts() {
		srcDir = filepath.Join(srcDir, name)
	}
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return err
	}
	outputPath := filepath.Join(srcDir, name+".cpp")
	if _, err := os.Stat(outputPath); err == nil {
		return nil
	}

	data := any(model)
	if parent != "" {
		data = partData(model, name, extractPartExtras(model, name))
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := tpl.Execute(f, data); err != nil {
		return err
	}
	return formatCode(outputPath)
}

func partData(m Model, partName string, extras map[string]any) map[string]any {
	data := map[string]any{
		"Name":      partName,
		"Namespace": m.ModelNamespace(),
		"Parent":    m.ModelName(),
	}
	switch v := m.(type) {
	case *SystemModel:
		data["Priority"] = v.Priority
	case *StateMachineModel:
		data["Priority"] = v.Priority
	}
	for k, v := range extras {
		data[k] = v
	}
	return data
}

func extractPartExtras(m Model, partName string) map[string]any {
	names, extras := m.ExtractParts()
	for i, n := range names {
		if n == partName {
			return extras[i]
		}
	}
	return nil
}

func collectRegistrations(compType string) (headers []string, calls []string, err error) {
	models, err := scanAllModels()
	if err != nil {
		return nil, nil, err
	}
	for _, m := range models {
		if m.Type != compType {
			continue
		}
		model, err := readModel(compType, m.Path)
		if err != nil {
			continue
		}
		ns := model.ModelNamespace()

		if model.HasParts() {
			parts, _ := model.ExtractParts()
			for _, pn := range parts {
				headers = append(headers, partRegistrationHeader(compType, ns, m.Name, pn))
				calls = append(calls, fmt.Sprintf(`    if (auto res = register_%s(rt); !res) return res;`, pn))
			}
		} else {
			headers = append(headers, registrationHeader(compType, ns, m.Name, ""))
			calls = append(calls, fmt.Sprintf(`    if (auto res = register_%s(rt); !res) return res;`, m.Name))
		}
	}
	return headers, calls, nil
}

func registrationHeader(compType, ns, parentName, partName string) string {
	prefix := ns
	if partName != "" {
		return fmt.Sprintf(`#include <%s/%s/%s.hpp>`, prefix, parentName, partName)
	}
	if compType == "system" {
		return fmt.Sprintf(`#include <%s/%s.hpp>`, ns, parentName)
	}
	return fmt.Sprintf(`#include <%s/%s.hpp>`, prefix, parentName)
}

func partRegistrationHeader(compType, ns, parentName, partName string) string {
	return fmt.Sprintf(`#include <%s/%s/%s.hpp>`, ns, parentName, partName)
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

	if counts["component"] > 0 {
		compHeaders, compCalls, err := collectRegistrations("component")
		if err != nil {
			return err
		}
		content = replaceSeedBlock(content, "components_include", compHeaders, nil)
		content = replaceSeedBlock(content, "components_reg", nil, compCalls)
	}

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
		content = replaceSeedBlock(content, "controllers_include", allHeaders, nil)
		content = replaceSeedBlock(content, "controllers_reg", nil, allCalls)
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
	indent := ""
	for i := beginLine - 1; i >= 0; i-- {
		if content[i] == '\n' {
			indent = content[i+1 : beginLine]
			break
		}
	}

	beginBlock := beginLine + len(beginMarker)
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

func DeleteModel(compType, name string) error {
	if !validTypes[compType] {
		return fmt.Errorf("unknown type %q", compType)
	}
	yamlPath := modelPath(compType, name)

	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return fmt.Errorf("model %s/%s not found", compType, name)
	}

	model, err := readModel(compType, yamlPath)
	if err != nil {
		return fmt.Errorf("reading model: %w", err)
	}

	if err := os.Remove(yamlPath); err != nil {
		return fmt.Errorf("removing %s: %w", yamlPath, err)
	}
	fmt.Printf("  Removed  %s\n", yamlPath)

	ns := model.ModelNamespace()
	outDir := filepath.Join(".", "Include", ns)

	if model.HasParts() {
		parentDir := filepath.Join(outDir, name)
		parts, _ := model.ExtractParts()
		for _, pn := range parts {
			hppPath := filepath.Join(parentDir, pn+".hpp")
			if err := os.Remove(hppPath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "  Warning: could not remove %s: %v\n", hppPath, err)
			} else if err == nil {
				fmt.Printf("  Removed  %s\n", hppPath)
			}
			if compType == "system" {
				cppPath := filepath.Join(".", "Src", "Game", name, pn+".cpp")
				if err := os.Remove(cppPath); err != nil && !os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "  Warning: could not remove %s: %v\n", cppPath, err)
				} else if err == nil {
					fmt.Printf("  Removed  %s\n", cppPath)
				}
			}
		}
		os.Remove(parentDir)
		if compType == "system" {
			os.Remove(filepath.Join(".", "Src", "Game", name))
		}
	} else {
		hppPath := filepath.Join(outDir, name+".hpp")
		if err := os.Remove(hppPath); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "  Warning: could not remove %s: %v\n", hppPath, err)
		} else if err == nil {
			fmt.Printf("  Removed  %s\n", hppPath)
		}

		if compType == "system" {
			cppPath := filepath.Join(".", "Src", "Game", name+".cpp")
			if err := os.Remove(cppPath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "  Warning: could not remove %s: %v\n", cppPath, err)
			} else if err == nil {
				fmt.Printf("  Removed  %s\n", cppPath)
			}
		}
	}

	if err := Sync(); err != nil {
		return fmt.Errorf("re-syncing after delete: %w", err)
	}
	return nil
}

type ModelInfo struct {
	Type      string
	Name      string
	Namespace string
}

func ListModels() ([]ModelInfo, error) {
	entries, err := scanAllModels()
	if err != nil {
		return nil, err
	}
	var models []ModelInfo
	for _, e := range entries {
		models = append(models, ModelInfo{
			Type: e.Type, Name: e.Name, Namespace: e.Namespace,
		})
	}
	return models, nil
}

type SyncStatus struct {
	Type   string
	Name   string
	Status string
}

func CheckSync() ([]SyncStatus, error) {
	models, err := scanAllModels()
	if err != nil {
		return nil, err
	}

	var results []SyncStatus
	for _, m := range models {
		model, err := readModel(m.Type, m.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not read %s: %v\n", m.Path, err)
			continue
		}

		outDir := filepath.Join(".", "Include", m.Namespace)
		hppPath := filepath.Join(outDir, m.Name+".hpp")
		if model.HasParts() {
			parts, _ := model.ExtractParts()
			if len(parts) > 0 {
				hppPath = filepath.Join(outDir, m.Name, parts[0]+".hpp")
			}
		}

		yamlInfo, err := os.Stat(m.Path)
		if err != nil {
			continue
		}
		hppInfo, err := os.Stat(hppPath)
		if os.IsNotExist(err) {
			results = append(results, SyncStatus{Type: m.Type, Name: m.Name, Status: "missing"})
			continue
		}
		if hppInfo.ModTime().Before(yamlInfo.ModTime()) {
			results = append(results, SyncStatus{Type: m.Type, Name: m.Name, Status: "stale"})
		} else {
			results = append(results, SyncStatus{Type: m.Type, Name: m.Name, Status: "ok"})
		}
	}
	return results, nil
}

func CreateAll(projectName string) error {
	defaultNames := map[string]string{
		"component":     projectName + "Position",
		"trait":         "Movable",
		"entity":        "PlayerEntity",
		"archetype":     "PlayerArchetype",
		"state_machine": "PlayerController",
		"event":         "PlayerJoined",
		"system":        "GravitySystem",
	}
	for t, n := range defaultNames {
		if err := CreateModel(t, n); err != nil {
			return err
		}
	}
	return nil
}
