package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	projecttemplates "Game-Developers-World/seed/internal/project/templates"
)

var assetDirs = map[string][]string{
	"2d": {"Sprites", "Textures", "Fonts"},
	"3d": {"Models", "Textures", "Materials", "Shaders"},
}

func Scaffold(name, mode string) error {
	if _, err := exec.LookPath("xmake"); err != nil {
		fmt.Println("Warning: xmake not found. Install it from https://xmake.io")
	}

	root := filepath.Join(".", name)

	dirs := []string{
		filepath.Join(root, "Src", "Game", "Systems"),
		filepath.Join(root, "Include", "Seed"),
		filepath.Join(root, "Include", "Core", "Component"),
		filepath.Join(root, "Include", "Core", "Trait"),
		filepath.Join(root, "Include", "Core", "Entity"),
		filepath.Join(root, "Include", "Core", "Archetype"),
		filepath.Join(root, "Models", "Component"),
		filepath.Join(root, "Models", "Trait"),
		filepath.Join(root, "Models", "Entity"),
		filepath.Join(root, "Models", "Archetype"),
	}

	assetRoot := filepath.Join(root, "Assets")
	dirs = append(dirs, assetRoot)
	for _, sub := range assetDirs[mode] {
		dirs = append(dirs, filepath.Join(assetRoot, sub))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
		logStatus("create", dir)
	}

	logStatus("clone", "GameAK...")
	gameakDir := filepath.Join(root, "Third-Party", "GameAK")
	cmd := exec.Command("git", "clone", "-b", "dev",
		"https://github.com/Game-World-Developers/GameAK.git",
		gameakDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone GameAK: %w", err)
	}

	data := projecttemplates.ProjectData{Name: name, Mode: mode}

	renderWrite := func(tmplName, dest string) error {
		content, err := projecttemplates.Render(tmplName, data)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", tmplName, err)
		}
		if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}
		logStatus("create", dest)
		return nil
	}

	if err := renderWrite("xmake.lua", filepath.Join(root, "xmake.lua")); err != nil {
		return err
	}

	if err := renderWrite("main.cpp", filepath.Join(root, "Src", "main.cpp")); err != nil {
		return err
	}

	if err := renderWrite("bootstrap.hpp", filepath.Join(root, "Src", "Game", "bootstrap.hpp")); err != nil {
		return err
	}

	if err := renderWrite("bootstrap.cpp", filepath.Join(root, "Src", "Game", "bootstrap.cpp")); err != nil {
		return err
	}

	if err := renderWrite("gitignore", filepath.Join(root, ".gitignore")); err != nil {
		return err
	}

	if err := renderWrite("gameak-gitignore", filepath.Join(gameakDir, ".gitignore")); err != nil {
		return err
	}

	if err := renderWrite("gameak-xmake.lua", filepath.Join(gameakDir, "xmake.lua")); err != nil {
		return err
	}

	if err := renderWrite("Seed-Context.hpp", filepath.Join(root, "Include", "Seed", "Context.hpp")); err != nil {
		return err
	}

	if err := renderWrite("Seed-Window.hpp", filepath.Join(root, "Include", "Seed", "Window.hpp")); err != nil {
		return err
	}

	if mode == "2d" {
		if err := renderWrite("Seed-Renderer2D.hpp", filepath.Join(root, "Include", "Seed", "Renderer2D.hpp")); err != nil {
			return err
		}
	} else {
		if err := renderWrite("Seed-Renderer3D.hpp", filepath.Join(root, "Include", "Seed", "Renderer3D.hpp")); err != nil {
			return err
		}
	}

	logDone("Project %s scaffolded at %s/", name, root)
	return nil
}
