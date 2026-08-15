package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"Game-Developers-World/seed/internal/generators"
	projecttemplates "Game-Developers-World/seed/internal/project/templates"
)

var assetDirs = map[string][]string{
	"2d": {"Sprites", "Textures", "Fonts"},
	"3d": {"Models", "Textures", "Materials", "Shaders"},
}

var validModes = map[string]bool{
	"2d":       true,
	"3d":       true,
	"headless": true,
}

func validateMode(mode string) error {
	if !validModes[mode] {
		return fmt.Errorf("invalid mode %q: valid modes are 2d, 3d, headless", mode)
	}
	return nil
}

func projectNameFromDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "GameProject"
	}
	return filepath.Base(cwd)
}

func Scaffold(name, mode string) error {
	if err := validateMode(mode); err != nil {
		return err
	}
	if _, err := exec.LookPath("xmake"); err != nil {
		fmt.Println("Warning: xmake not found. Install it from https://xmake.io")
	}

	root := filepath.Join(".", name)

	dirs := []string{
		filepath.Join(root, "Src", "Game"),
		filepath.Join(root, "Include", "Seed"),

		filepath.Join(root, "Models", "Component"),
		filepath.Join(root, "Models", "Trait"),
		filepath.Join(root, "Models", "Entity"),
		filepath.Join(root, "Models", "Archetype"),
		filepath.Join(root, "Models", "StateMachine"),
		filepath.Join(root, "Models", "Event"),
		filepath.Join(root, "Models", "Asset"),
		filepath.Join(root, "Models", "System"),
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
	if err := CloneGameAK(gameakDir); err != nil {
		return err
	}

	if err := writeProjectFiles(name, mode, root, gameakDir, false); err != nil {
		return err
	}
	if err := WriteStarterModels(root, false); err != nil {
		return err
	}
	// Run sync inside the new project directory
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)
	return generators.Sync()
}

type InitOpts struct {
	Mode      string
	Overwrite bool
}

func ScaffoldInit(opts InitOpts) error {
	if err := validateMode(opts.Mode); err != nil {
		return err
	}
	if _, err := exec.LookPath("xmake"); err != nil {
		fmt.Println("Warning: xmake not found. Install it from https://xmake.io")
	}

	name := projectNameFromDir()
	root := "."
	gameakDir := filepath.Join(root, "Third-Party", "GameAK")

	dirs := []string{
		filepath.Join(root, "Src", "Game"),
		filepath.Join(root, "Include", "Seed"),

		filepath.Join(root, "Models", "Component"),
		filepath.Join(root, "Models", "Trait"),
		filepath.Join(root, "Models", "Entity"),
		filepath.Join(root, "Models", "Archetype"),
		filepath.Join(root, "Models", "StateMachine"),
		filepath.Join(root, "Models", "Event"),
		filepath.Join(root, "Models", "Asset"),
		filepath.Join(root, "Models", "System"),
	}

	assetRoot := filepath.Join(root, "Assets")
	dirs = append(dirs, assetRoot)
	for _, sub := range assetDirs[opts.Mode] {
		dirs = append(dirs, filepath.Join(assetRoot, sub))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
		logStatus("create", dir)
	}

	if _, err := os.Stat(gameakDir); os.IsNotExist(err) {
		logStatus("clone", "GameAK...")
		if err := CloneGameAK(gameakDir); err != nil {
			return err
		}
	}

	if err := writeProjectFiles(name, opts.Mode, root, gameakDir, !opts.Overwrite); err != nil {
		return err
	}
	if err := WriteStarterModels(root, !opts.Overwrite); err != nil {
		return err
	}
	return generators.Sync()
}

func writeProjectFiles(name, mode, root, gameakDir string, skipExisting bool) error {
	data := projecttemplates.ProjectData{Name: name, Mode: mode}

	renderDynamic := func(tmplName, dest string) error {
		if skipExisting {
			if _, err := os.Stat(dest); err == nil {
				return nil
			}
		}
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

	renderStatic := func(tmplName, dest string) error {
		if skipExisting {
			if _, err := os.Stat(dest); err == nil {
				return nil
			}
		}
		content, err := projecttemplates.RenderStatic(tmplName)
		if err != nil {
			return fmt.Errorf("reading %s: %w", tmplName, err)
		}
		if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}
		logStatus("create", dest)
		return nil
	}

	if err := renderDynamic("xmake.lua", filepath.Join(root, "xmake.lua")); err != nil {
		return err
	}

	if err := renderDynamic("main.cpp", filepath.Join(root, "Src", "main.cpp")); err != nil {
		return err
	}

	if err := renderStatic("bootstrap.hpp", filepath.Join(root, "Src", "Game", "bootstrap.hpp")); err != nil {
		return err
	}

	if err := renderStatic("bootstrap.cpp", filepath.Join(root, "Src", "Game", "bootstrap.cpp")); err != nil {
		return err
	}

	if err := renderStatic("gitignore", filepath.Join(root, ".gitignore")); err != nil {
		return err
	}

	if err := renderStatic("clang-format", filepath.Join(root, ".clang-format")); err != nil {
		return err
	}

	if err := renderStatic("clang-tidy", filepath.Join(root, ".clang-tidy")); err != nil {
		return err
	}

	if err := renderStatic("cppcheck-suppressions", filepath.Join(root, ".cppcheck-suppressions")); err != nil {
		return err
	}

	if err := renderStatic("gameak-gitignore", filepath.Join(gameakDir, ".gitignore")); err != nil {
		return err
	}

	if err := renderStatic("gameak-xmake.lua", filepath.Join(gameakDir, "xmake.lua")); err != nil {
		return err
	}

	if err := renderStatic("seed-context.hpp", filepath.Join(root, "Include", "Seed", "context.hpp")); err != nil {
		return err
	}

	if err := renderStatic("seed-window.hpp", filepath.Join(root, "Include", "Seed", "window.hpp")); err != nil {
		return err
	}

	if err := renderStatic("seed-audio.hpp", filepath.Join(root, "Include", "Seed", "audio.hpp")); err != nil {
		return err
	}

	// Headless targets have no window/renderer to feed assets to, so the
	// image/font/asset-manager stack (and the SDL_image/SDL_ttf packages
	// it needs) is skipped entirely rather than scaffolded unused — see
	// docs/runtime-architecture.md §8/§10.
	if mode != "headless" {
		if err := renderStatic("seed-surface.hpp", filepath.Join(root, "Include", "Seed", "surface.hpp")); err != nil {
			return err
		}

		if err := renderStatic("seed-font.hpp", filepath.Join(root, "Include", "Seed", "font.hpp")); err != nil {
			return err
		}

		if err := renderStatic("seed-image.hpp", filepath.Join(root, "Include", "Seed", "image.hpp")); err != nil {
			return err
		}

		if err := renderStatic("seed-asset-manager.hpp", filepath.Join(root, "Include", "Seed", "asset_manager.hpp")); err != nil {
			return err
		}
	}

	switch mode {
	case "2d":
		if err := renderStatic("seed-texture.hpp", filepath.Join(root, "Include", "Seed", "texture.hpp")); err != nil {
			return err
		}
		if err := renderStatic("seed-renderer_2d.hpp", filepath.Join(root, "Include", "Seed", "renderer_2d.hpp")); err != nil {
			return err
		}
	case "3d":
		if err := renderStatic("seed-renderer_3d.hpp", filepath.Join(root, "Include", "Seed", "renderer_3d.hpp")); err != nil {
			return err
		}
		if err := renderStatic("seed-shaders.hpp", filepath.Join(root, "Include", "Seed", "shaders.hpp")); err != nil {
			return err
		}
	case "headless":
		// no renderer to scaffold
	}

	if err := os.WriteFile(filepath.Join(root, ".seed_project"), []byte{}, 0644); err != nil {
		return fmt.Errorf("creating .seed_project: %w", err)
	}
	logStatus("create", filepath.Join(root, ".seed_project"))

	logDone("Project %s scaffolded at %s/", name, root)
	return nil
}
