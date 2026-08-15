package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/project"

	gogit "github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type checkResult struct {
	name   string
	status string
	detail string
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose the project and environment",
	Long: `Check the health of the current seed project and its dependencies.

Verifies:
  - Project structure (.seed_project, directories)
  - External tools (xmake, clang-format)
  - Model consistency (sync status, missing references)
  - System .cpp implementations
  - Orphaned generated files (no corresponding model)
  - Duplicate model names across namespaces
  - Third-party dependencies (GameAK)`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		results := []checkResult{}

		results = append(results, checkProject())
		results = append(results, checkTools()...)
		results = append(results, checkModels()...)
		results = append(results, checkSystemImplementations()...)
		results = append(results, checkOrphanedFiles()...)
		results = append(results, checkDuplicateNames()...)
		results = append(results, checkThirdParty()...)
		results = append(results, checkTargetSDKs()...)

		fmt.Println()
		fmt.Println("═ seed doctor ════════════════════════════════════")
		fmt.Println()

		allOK := true
		for _, r := range results {
			switch r.status {
			case "ok":
				fmt.Printf("  ✓  %s\n", r.name)
			case "warn":
				fmt.Printf("  ⚠  %s\n", r.name)
				allOK = false
			case "fail":
				fmt.Printf("  ✘  %s\n", r.name)
				allOK = false
			}
			if r.detail != "" {
				fmt.Printf("     %s\n", r.detail)
			}
		}

		fmt.Println()
		if allOK {
			fmt.Println("  ✓  All checks passed.")
		} else {
			fmt.Println("  ⚠  Some checks failed. See details above.")
		}

		return nil
	},
}

func checkProject() checkResult {
	if _, err := os.Stat(".seed_project"); os.IsNotExist(err) {
		return checkResult{"Project root (.seed_project)", "fail", "Run 'seed new <name>' first"}
	}

	dirs := []string{"Models", "Include", "Src", "Assets"}
	for _, d := range dirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			return checkResult{"Project structure", "fail", fmt.Sprintf("Missing directory: %s/", d)}
		}
	}

	return checkResult{"Project structure", "ok", ""}
}

func checkTools() []checkResult {
	var results []checkResult

	if _, err := exec.LookPath("xmake"); err != nil {
		results = append(results, checkResult{"xmake", "warn", "Not found. Install from https://xmake.io"})
	} else {
		results = append(results, checkResult{"xmake", "ok", ""})
	}

	if _, err := exec.LookPath("clang-format"); err != nil {
		results = append(results, checkResult{"clang-format", "warn", "Not found. Install clang-format for auto-formatting"})
	} else {
		results = append(results, checkResult{"clang-format", "ok", ""})
	}

	return results
}

func checkModels() []checkResult {
	var results []checkResult

	models, err := generators.ListModels()
	if err != nil {
		return []checkResult{{"Model scan", "fail", err.Error()}}
	}

	if len(models) == 0 {
		results = append(results, checkResult{"Models", "warn", "No models found. Run 'seed model generate <type> <name>'"})
	} else {
		results = append(results, checkResult{fmt.Sprintf("Models (%d total)", len(models)), "ok", ""})
	}

	syncResults, err := generators.CheckSync()
	if err != nil {
		results = append(results, checkResult{"Sync check", "fail", err.Error()})
		return results
	}

	stale := 0
	missing := 0
	for _, r := range syncResults {
		switch r.Status {
		case "stale":
			stale++
		case "missing":
			missing++
		}
	}
	if stale > 0 || missing > 0 {
		msg := ""
		if stale > 0 {
			msg += fmt.Sprintf("%d stale", stale)
		}
		if missing > 0 {
			if msg != "" {
				msg += ", "
			}
			msg += fmt.Sprintf("%d missing", missing)
		}
		results = append(results, checkResult{"Sync status", "warn", msg + ". Run 'seed sync'"})
	} else if len(syncResults) > 0 {
		results = append(results, checkResult{"Sync status", "ok", "All generated files are up to date"})
	}

	problems, err := generators.GetProblems()
	if err != nil {
		results = append(results, checkResult{"Model validation", "fail", err.Error()})
	} else if len(problems) > 0 {
		errCount := 0
		warnCount := 0
		for _, p := range problems {
			switch p.Level {
			case generators.ProblemError:
				errCount++
			case generators.ProblemWarning:
				warnCount++
			}
		}
		msg := ""
		if errCount > 0 {
			msg += fmt.Sprintf("%d errors", errCount)
		}
		if warnCount > 0 {
			if msg != "" {
				msg += ", "
			}
			msg += fmt.Sprintf("%d warnings", warnCount)
		}
		results = append(results, checkResult{"Model validation", "warn", msg})
	} else {
		results = append(results, checkResult{"Model validation", "ok", "No problems"})
	}

	return results
}

func checkThirdParty() []checkResult {
	gameakPath := filepath.Join(".", "Third-Party", "GameAK")
	gameakHpp := filepath.Join(gameakPath, "Include", "GameAk", "Runtime", "runtime.h")

	if _, err := os.Stat(gameakHpp); os.IsNotExist(err) {
		return []checkResult{
			{"GameAK", "fail", "Not found. Run 'seed new' or clone to Third-Party/GameAK"},
		}
	}

	results := []checkResult{{"GameAK", "ok", ""}}

	repo, err := gogit.PlainOpen(gameakPath)
	if err != nil {
		// Not a git checkout (e.g. vendored by hand) — nothing to compare
		// the pin against, and that's not itself a problem this check
		// should report on.
		return results
	}
	head, err := repo.Head()
	if err != nil {
		return results
	}
	actual := head.Hash().String()
	if actual == project.GameAKPinnedRevision {
		results = append(results, checkResult{"GameAK revision matches pin", "ok", ""})
	} else {
		results = append(results, checkResult{
			"GameAK revision matches pin", "warn",
			fmt.Sprintf("Third-Party/GameAK is at %s, pinned revision is %s (docs/gameak-mapping.md) — mappings and generated code were only verified against the pin",
				shortSHA(actual), shortSHA(project.GameAKPinnedRevision)),
		})
	}
	return results
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// checkTargetSDKs reports build requirements specific to the current
// project's mode (2d/3d/headless), detected from which renderer header
// was scaffolded — "Make seed doctor explain missing SDKs and
// target-specific build requirements" from the roadmap. Nothing here is a
// hard failure: xmake resolves the SDL3/SDL3_image/SDL3_ttf packages
// itself on first build (already exercised by this project's own compile
// tests), so this is explanatory, not a preflight gate duplicating what
// xmake already does.
func checkTargetSDKs() []checkResult {
	var results []checkResult

	mode := "headless"
	if _, err := os.Stat(filepath.Join(".", "Include", "Seed", "renderer_3d.hpp")); err == nil {
		mode = "3d"
	} else if _, err := os.Stat(filepath.Join(".", "Include", "Seed", "renderer_2d.hpp")); err == nil {
		mode = "2d"
	}

	switch mode {
	case "headless":
		results = append(results, checkResult{
			"Target: headless", "ok",
			"No SDL_image/SDL_ttf required; xmake fetches libsdl3 only on first build",
		})
	case "2d":
		results = append(results, checkResult{
			"Target: 2d", "ok",
			"xmake fetches libsdl3, libsdl3_image, libsdl3_ttf on first build (needs network the first time)",
		})
	case "3d":
		results = append(results, checkResult{
			"Target: 3d", "ok",
			"xmake fetches libsdl3, libsdl3_image, libsdl3_ttf on first build (needs network the first time)",
		})
		if _, err := exec.LookPath("glslc"); err != nil {
			results = append(results, checkResult{
				"glslc (GLSL -> SPIR-V compiler)", "warn",
				"Not found — not required to build (Renderer3D ships precompiled SPIR-V, see seed-shaders.hpp.tmpl), only if you modify shader source yourself",
			})
		} else {
			results = append(results, checkResult{"glslc (GLSL -> SPIR-V compiler)", "ok", ""})
		}
	}

	return results
}

func systemPartNames(name string) []string {
	yamlPath := filepath.Join(".", "Models", generators.TypeDir["system"], name+".yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil
	}
	var header struct {
		Parts []struct {
			Name string `yaml:"name"`
		} `yaml:"parts,omitempty"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return nil
	}
	var names []string
	for _, p := range header.Parts {
		names = append(names, p.Name)
	}
	return names
}

func checkSystemImplementations() []checkResult {
	models, err := generators.ListModels()
	if err != nil {
		return []checkResult{{"System implementations", "fail", err.Error()}}
	}

	var results []checkResult
	hasSystems := false
	for _, m := range models {
		if m.Type != "system" {
			continue
		}
		hasSystems = true

		parts := systemPartNames(m.Name)
		if len(parts) > 0 {
			for _, pn := range parts {
				cppPath := filepath.Join(".", "Src", "Game", m.Name, pn+".cpp")
				if _, err := os.Stat(cppPath); os.IsNotExist(err) {
					results = append(results, checkResult{
						name:   fmt.Sprintf("System %s/%s (part: %s)", m.Namespace, m.Name, pn),
						status: "warn",
						detail: "Missing .cpp — run 'seed sync' to generate",
					})
				}
			}
		} else {
			cppPath := filepath.Join(".", "Src", "Game", m.Name+".cpp")
			if _, err := os.Stat(cppPath); os.IsNotExist(err) {
				results = append(results, checkResult{
					name:   fmt.Sprintf("System %s/%s", m.Namespace, m.Name),
					status: "warn",
					detail: "Missing .cpp — run 'seed sync' to generate",
				})
			}
		}
	}

	if hasSystems && len(results) == 0 {
		results = append(results, checkResult{"System implementations", "ok", "All systems have .cpp files"})
	}
	return results
}

func expectedGeneratedFiles() (map[string]bool, error) {
	models, err := generators.ListModels()
	if err != nil {
		return nil, err
	}

	expected := map[string]bool{
		filepath.Join(".", "Src", "Game", "bootstrap.cpp"): true,
		filepath.Join(".", "Src", "Game", "bootstrap.hpp"): true,
	}
	for _, m := range models {
		parts := systemPartNames(m.Name)
		if m.Type == "system" {
			if len(parts) > 0 {
				for _, pn := range parts {
					expected[filepath.Join(".", "Include", m.Namespace, m.Name, pn+".hpp")] = true
					expected[filepath.Join(".", "Src", "Game", m.Name, pn+".cpp")] = true
				}
			} else {
				expected[filepath.Join(".", "Include", m.Namespace, m.Name+".hpp")] = true
				expected[filepath.Join(".", "Src", "Game", m.Name+".cpp")] = true
			}
		} else if len(parts) > 0 {
			for _, pn := range parts {
				expected[filepath.Join(".", "Include", m.Namespace, m.Name, pn+".hpp")] = true
			}
		} else {
			expected[filepath.Join(".", "Include", m.Namespace, m.Name+".hpp")] = true
		}
	}
	return expected, nil
}

func checkOrphanedFiles() []checkResult {
	expected, err := expectedGeneratedFiles()
	if err != nil {
		return []checkResult{{"Orphaned files", "fail", err.Error()}}
	}

	if len(expected) == 0 {
		return nil
	}

	var orphans []string

	walkFn := func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			if strings.HasPrefix(fi.Name(), ".") || fi.Name() == "Third-Party" || fi.Name() == "Assets" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".hpp") && !strings.HasSuffix(path, ".cpp") {
			return nil
		}
		if strings.HasPrefix(path, filepath.Join(".", "Include", "Seed")+string(filepath.Separator)) {
			return nil
		}
		if !expected[path] {
			orphans = append(orphans, path)
		}
		return nil
	}

	filepath.Walk(filepath.Join(".", "Include"), walkFn)
	filepath.Walk(filepath.Join(".", "Src", "Game"), walkFn)

	if len(orphans) > 0 {
		sort.Strings(orphans)
		msg := fmt.Sprintf("%d files with no corresponding model", len(orphans))
		var examples []string
		for i := 0; i < len(orphans) && i < 5; i++ {
			examples = append(examples, orphans[i])
		}
		msg += ": " + strings.Join(examples, ", ")
		if len(orphans) > 5 {
			msg += fmt.Sprintf(" (and %d more)", len(orphans)-5)
		}
		return []checkResult{{
			name:   "Orphaned files",
			status: "warn",
			detail: msg,
		}}
	}
	return []checkResult{{"Orphaned files", "ok", "All generated files have matching models"}}
}

func checkDuplicateNames() []checkResult {
	models, err := generators.ListModels()
	if err != nil {
		return []checkResult{{"Duplicate model names", "fail", err.Error()}}
	}

	seen := map[string]string{}
	var duplicates []string

	for _, m := range models {
		key := m.Type + "." + m.Name
		if prevNS, ok := seen[key]; ok && prevNS != m.Namespace {
			duplicates = append(duplicates, fmt.Sprintf("%s %q appears in namespace %q and %q",
				m.Type, m.Name, prevNS, m.Namespace))
		}
		seen[key] = m.Namespace
	}

	if len(duplicates) > 0 {
		return []checkResult{{
			name:   "Duplicate model names",
			status: "warn",
			detail: strings.Join(duplicates, "; "),
		}}
	}
	return []checkResult{{"Duplicate model names", "ok", "No duplicate names across namespaces"}}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
