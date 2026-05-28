package Project

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

const seedYamlPath = "Config/project.seed.yml"

type TemplateData struct {
	Name       string
	GameAKPath string
}

func findGameAK() string {
	candidates := []string{
		filepath.Join(os.Getenv("HOME"), "Projects", "GameAK"),
		"/usr/local",
		"/opt/GameAK",
	}

	for _, dir := range candidates {
		header := filepath.Join(dir, "Include", "AK", "Core", "Types.hpp")
		if _, err := os.Stat(header); err == nil {
			return dir
		}
	}

	return ""
}

func Create(name string) error {
	fmt.Printf("Creating project %s...\n", name)

	if err := os.MkdirAll(name, 0o755); err != nil {
		return err
	}
	fmt.Printf("  Created %s/\n", name)

	dirs := []string{
		"Config",
		"Third-Party",
		filepath.Join("Include", name, "Worlds"),
		filepath.Join("Include", name, "Components"),
		filepath.Join("Include", name, "Systems"),
		filepath.Join("Include", name, "Runtime"),
		filepath.Join("Src", "Worlds"),
		filepath.Join("Src", "Components"),
		filepath.Join("Src", "Systems"),
		filepath.Join("Src", "Runtime"),
		filepath.Join("Src", "Generated"),
		"Tests",
	}

	for _, dir := range dirs {
		path := filepath.Join(name, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
		fmt.Printf("  Created %s/\n", filepath.Join(name, dir))
	}

	gameAKPath := findGameAK()
	if gameAKPath != "" {
		fmt.Printf("  Found GameAK at %s\n", gameAKPath)
	}

	data := TemplateData{
		Name:       name,
		GameAKPath: gameAKPath,
	}

	if err := generateYaml(name, data); err != nil {
		return err
	}

	if err := generateFromTemplates(name, data); err != nil {
		return err
	}

	fmt.Printf("Done! Project %s generated successfully.\n", name)
	return nil
}

func generateYaml(name string, data TemplateData) error {
	tmplBytes, err := os.ReadFile(seedYamlPath)
	if err != nil {
		return err
	}

	tmpl, err := template.New("yaml").Parse(string(tmplBytes))
	if err != nil {
		return err
	}

	outputPath := filepath.Join(name, "Config", "project.yaml")
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return err
	}

	fmt.Printf("  Generated Config/project.yaml\n")
	return nil
}

func generateFromTemplates(name string, data TemplateData) error {
	entries := map[string]string{
		"templates/Makefile.tmpl":       "Makefile",
		"templates/gitignore.tmpl":      ".gitignore",
		"templates/gitmodules.tmpl":     ".gitmodules",
		"templates/Main.cpp.tmpl":       filepath.Join("Src", "Main.cpp"),
		"templates/World.hpp.tmpl":      filepath.Join("Include", name, "Worlds", "World.hpp"),
		"templates/World.cpp.tmpl":      filepath.Join("Src", "Worlds", "World.cpp"),
		"templates/Component.hpp.tmpl":  filepath.Join("Include", name, "Components", "Component.hpp"),
		"templates/Component.cpp.tmpl":  filepath.Join("Src", "Components", "Component.cpp"),
		"templates/System.hpp.tmpl":     filepath.Join("Include", name, "Systems", "System.hpp"),
		"templates/System.cpp.tmpl":     filepath.Join("Src", "Systems", "System.cpp"),
		"templates/Runtime.hpp.tmpl":    filepath.Join("Include", name, "Runtime", "Runtime.hpp"),
		"templates/Runtime.cpp.tmpl":    filepath.Join("Src", "Runtime", "Runtime.cpp"),
		"templates/SampleTest.cpp.tmpl": filepath.Join("Tests", "SampleTest.cpp"),
	}

	for tmplPath, outputRel := range entries {
		tmplBytes, err := templateFS.ReadFile(tmplPath)
		if err != nil {
			return err
		}

		tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(tmplBytes))
		if err != nil {
			return err
		}

		outputPath := filepath.Join(name, outputRel)
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}

		if err := tmpl.Execute(file, data); err != nil {
			file.Close()
			return err
		}

		file.Close()
		fmt.Printf("  Generated %s\n", outputRel)
	}

	return nil
}
