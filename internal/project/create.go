package project

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const seedYamlPath = "config/project.seed.yml"

type TemplateData struct {
	Name       string
	GameAKPath string
}

func FindGameAK() string {
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

func initGitRepo(projectDir string, projectName string) error {
	if _, err := os.Stat(filepath.Join(projectDir, ".git")); err == nil {
		return nil
	}

	name := "The Seed"
	email := "seed@gameworlddevelopers.org"

	cfg, err := config.LoadConfig(config.GlobalScope)
	if err == nil {
		if cfg.User.Name != "" {
			name = cfg.User.Name
		}
		if cfg.User.Email != "" {
			email = cfg.User.Email
		}
	}

	repo, err := git.PlainInit(projectDir, false)
	if err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	if _, err := wt.Add("."); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	if _, err := wt.Commit(fmt.Sprintf("Starting a new game project called %s", projectName), &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
	}); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	fmt.Printf("  Initialized git repository\n")
	return nil
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
		filepath.Join("Assets", "Audio"),
		filepath.Join("Assets", "Fonts"),
		filepath.Join("Build", "Objects"),
		filepath.Join("Build", "IR"),
		filepath.Join("Build", "Generated"),
		filepath.Join("Build", "Cache"),
		filepath.Join("Models", "Traits"),
		filepath.Join("Models", "Archetypes"),
		filepath.Join("Models", "Entities"),
		filepath.Join("Models", "Components"),
		filepath.Join("Include", "Components"),
		filepath.Join("Include", "Systems"),
		filepath.Join("Include", "Runtime"),
		filepath.Join("Include", "Seed"),
		filepath.Join("Src", "Components"),
		filepath.Join("Src", "Systems"),
		filepath.Join("Src", "Runtime"),
		filepath.Join("Src", "Seed"),
		filepath.Join("Tests", "Components"),
		filepath.Join("Tests", "Systems"),
		filepath.Join("Tests", "Runtime"),
		filepath.Join("Tests", "Seed"),
	}

	for _, dir := range dirs {
		path := filepath.Join(name, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
		fmt.Printf("  Created %s/\n", filepath.Join(name, dir))
	}

	gameAKPath := FindGameAK()
	if gameAKPath != "" {
		fmt.Printf("  Found GameAK at %s\n", gameAKPath)
	}

	if err := GenerateYaml(name, TemplateData{Name: name, GameAKPath: gameAKPath}); err != nil {
		return err
	}

	if err := initGitRepo(name, name); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: could not init git repo: %v\n", err)
	}

	fmt.Printf("Done! Project %s generated successfully.\n", name)
	return nil
}

func GenerateYaml(name string, data TemplateData) error {
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
