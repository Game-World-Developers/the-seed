package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type importAssetKind struct {
	dir  string
	exts []string
}

var importKinds = map[string]importAssetKind{
	"sprite":   {"Sprites", []string{".png", ".jpg", ".jpeg", ".bmp", ".gif"}},
	"texture":  {"Textures", []string{".png", ".jpg", ".jpeg", ".bmp", ".tga"}},
	"font":     {"Fonts", []string{".ttf", ".otf", ".woff", ".woff2"}},
	"audio":    {"Audio", []string{".wav", ".ogg", ".mp3", ".flac"}},
	"model":    {"Models", []string{".fbx", ".gltf", ".glb", ".obj"}},
	"material": {"Materials", []string{".mat", ".json"}},
	"shader":   {"Shaders", []string{".glsl", ".hlsl", ".spv"}},
}

type assetYAML struct {
	Type      string `yaml:"type"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Kind      string `yaml:"kind"`
	Path      string `yaml:"path"`
}

func capitalize(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

var importCmd = &cobra.Command{
	Use:   "import <kind> <path> [name]",
	Short: "Import an asset into the project",
	Long: `Copy a third-party asset into the project's Assets/ directory and
create a corresponding Asset YAML model for code generation.

Kinds: sprite, texture, font, audio, model, material, shader

Examples:
  seed import sprite hero.png                  → Assets/Sprites/hero.png + Models/Asset/Hero.yaml
  seed import font pixel.ttf PixelFont  → Assets/Fonts/pixel.ttf + Models/Asset/PixelFont.yaml
  seed import audio boom.wav                   → Assets/Audio/boom.wav + Models/Asset/Boom.yaml`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}

		kind := args[0]
		srcPath := args[1]

		ak, ok := importKinds[kind]
		if !ok {
			return fmt.Errorf("unknown asset kind %q. Valid: sprite, texture, font, audio, model, material, shader", kind)
		}

		ext := strings.ToLower(filepath.Ext(srcPath))
		valid := false
		for _, e := range ak.exts {
			if ext == e {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("file %q not recognized as %s (expected: %s)",
				srcPath, kind, strings.Join(ak.exts, ", "))
		}

		baseName := filepath.Base(srcPath)
		name := strings.TrimSuffix(baseName, ext)
		if len(args) >= 3 {
			name = args[2]
		}
		name = capitalize(name)

		destDir := filepath.Join(".", "Assets", ak.dir)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("creating asset directory: %w", err)
		}

		destPath := filepath.Join(destDir, baseName)
		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("copying asset: %w", err)
		}
		fmt.Printf("  %11s  %s\n", "create", destPath)

		relPath, _ := filepath.Rel(".", destPath)
		asset := assetYAML{
			Type:      "asset",
			Name:      name,
			Namespace: "Game",
			Kind:      kind,
			Path:      relPath,
		}

		assetDir := filepath.Join(".", "Models", "Asset")
		if err := os.MkdirAll(assetDir, 0755); err != nil {
			return fmt.Errorf("creating model directory: %w", err)
		}

		yamlPath := filepath.Join(assetDir, name+".yaml")
		out, err := yaml.Marshal(&asset)
		if err != nil {
			return fmt.Errorf("marshaling asset YAML: %w", err)
		}
		if err := os.WriteFile(yamlPath, out, 0644); err != nil {
			return fmt.Errorf("writing asset model: %w", err)
		}
		fmt.Printf("  %11s  %s\n", "create", yamlPath)

		fmt.Println()
		fmt.Println(`  Run "seed sync" to generate C++ code.`)
		return nil
	},
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func init() {
	rootCmd.AddCommand(importCmd)
}
