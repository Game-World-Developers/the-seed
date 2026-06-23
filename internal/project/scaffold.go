package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	projecttemplates "Game-Developers-World/seed/internal/project/templates"
)

var assetDirs = map[string][]string{
	"2d": {"Sprites", "Textures", "Fonts"},
	"3d": {"Models", "Textures", "Materials", "Shaders"},
}

func projectNameFromDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "GameProject"
	}
	return filepath.Base(cwd)
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
	if _, err := gogit.PlainClone(gameakDir, false, &gogit.CloneOptions{
		URL:           "https://github.com/Game-World-Developers/GameAK.git",
		ReferenceName: plumbing.NewBranchReferenceName("dev"),
		SingleBranch:  true,
		Depth:         1,
	}); err != nil {
		return fmt.Errorf("failed to clone GameAK: %w", err)
	}

	return writeProjectFiles(name, mode, root, gameakDir, false)
}

type InitOpts struct {
	Mode      string
	Overwrite bool
}

func ScaffoldInit(opts InitOpts) error {
	if _, err := exec.LookPath("xmake"); err != nil {
		fmt.Println("Warning: xmake not found. Install it from https://xmake.io")
	}

	name := projectNameFromDir()
	root := "."
	gameakDir := filepath.Join(root, "Third-Party", "GameAK")

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
		if _, err := gogit.PlainClone(gameakDir, false, &gogit.CloneOptions{
			URL:           "https://github.com/Game-World-Developers/GameAK.git",
			ReferenceName: plumbing.NewBranchReferenceName("dev"),
			SingleBranch:  true,
			Depth:         1,
		}); err != nil {
			return fmt.Errorf("failed to clone GameAK: %w", err)
		}
	}

	return writeProjectFiles(name, opts.Mode, root, gameakDir, !opts.Overwrite)
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

	if mode == "2d" {
		if err := renderStatic("seed-renderer_2d.hpp", filepath.Join(root, "Include", "Seed", "renderer_2d.hpp")); err != nil {
			return err
		}
	} else {
		if err := renderStatic("seed-renderer_3d.hpp", filepath.Join(root, "Include", "Seed", "renderer_3d.hpp")); err != nil {
			return err
		}
	}

	logDone("Project %s scaffolded at %s/", name, root)
	return nil
}
