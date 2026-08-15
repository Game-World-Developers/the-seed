package tests

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/generators"
)

func writeAtlasPNG(t *testing.T, path string, fill func(tileIdx int) color.RGBA) {
	t.Helper()
	const tileSize = 2
	img := image.NewRGBA(image.Rect(0, 0, tileSize*16, tileSize*16))
	for tileIdx := 0; tileIdx < 256; tileIdx++ {
		col, row := tileIdx%16, tileIdx/16
		c := fill(tileIdx)
		for y := row * tileSize; y < (row+1)*tileSize; y++ {
			for x := col * tileSize; x < (col+1)*tileSize; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// TestBlockGeneration exercises Block model sync end to end — a real gap
// before this pass, since no test invoked syncBlocks at all: with the
// YAML-declared colors (best-quality profile, the default) and with the
// atlas-baking path (best-performance profile), which pulls in
// internal/baker.
func TestBlockGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("best-quality profile: BlockRegistry uses the YAML-declared colors", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "quality")
		defer inDir(t, dir)()

		writeModel(t, dir, "Block", "StoneBlock", `type: block
name: StoneBlock
namespace: Core
tile_index: 1
colors:
  top: 0x112233
  bottom: 0x445566
  side: 0x778899
solid: true
transparent: false
hardness: 1.5
drop: StoneBlock
`)

		if err := generators.Sync(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "Include", "Core", "BlockRegistry.hpp"))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		for _, want := range []string{"StoneBlock", "0x112233", "0x445566", "0x778899", "true"} {
			if !contains(content, want) {
				t.Fatalf("expected BlockRegistry.hpp to contain %q, got:\n%s", want, content)
			}
		}
	})

	t.Run("best-performance profile: colors come from the baked atlas, not the YAML", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "performance")
		defer inDir(t, dir)()

		if err := os.WriteFile(filepath.Join(dir, ".seed_project"), []byte("asset_profile: best-performance\n"), 0644); err != nil {
			t.Fatal(err)
		}
		// Tile 1 is a flat, distinctive color so the baked hex is
		// predictable and clearly not the YAML-declared placeholder.
		writeAtlasPNG(t, filepath.Join(dir, "Assets", "Textures", "terrain.png"), func(i int) color.RGBA {
			return color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 255}
		})
		writeModel(t, dir, "Block", "StoneBlock", `type: block
name: StoneBlock
namespace: Core
tile_index: 1
colors:
  top: 0xAAAAAA
  bottom: 0xAAAAAA
  side: 0xAAAAAA
solid: true
transparent: false
hardness: 1.5
drop: StoneBlock
atlas_path: Assets/Textures/terrain.png
`)

		if err := generators.Sync(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "Include", "Core", "BlockRegistry.hpp"))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if contains(content, "0xAAAAAA") {
			t.Fatalf("expected the YAML placeholder color to be overridden by the baked atlas, got:\n%s", content)
		}
		// Top light multiplier is 1.0, so the top color should be exactly
		// the baked tile color (0x102030) — see baker.ApplyLightMultiplier
		// and internal/generators/engine.go's syncBlocks.
		if !contains(content, "0x102030") {
			t.Fatalf("expected the baked top color 0x102030, got:\n%s", content)
		}
	})

	t.Run("best-performance profile with no atlas falls back to YAML colors with a warning, not a failure", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "performance-no-atlas")
		defer inDir(t, dir)()

		if err := os.WriteFile(filepath.Join(dir, ".seed_project"), []byte("asset_profile: best-performance\n"), 0644); err != nil {
			t.Fatal(err)
		}
		writeModel(t, dir, "Block", "StoneBlock", `type: block
name: StoneBlock
namespace: Core
tile_index: 1
colors:
  top: 0xBBBBBB
  bottom: 0xBBBBBB
  side: 0xBBBBBB
solid: true
transparent: false
hardness: 1.5
drop: StoneBlock
`)

		if err := generators.Sync(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "Include", "Core", "BlockRegistry.hpp"))
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "0xBBBBBB") {
			t.Fatalf("expected the YAML color to be used as a fallback, got:\n%s", data)
		}
	})
}
