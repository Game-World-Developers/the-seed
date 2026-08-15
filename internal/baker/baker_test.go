package baker

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeTestAtlas creates a tileSize*16 x tileSize*16 PNG where tile i's
// pixels are all set to fill(i) — a fully controlled atlas so baked
// averages are exact, not approximate.
func writeTestAtlas(t *testing.T, path string, tileSize int, fill func(tileIdx int) color.RGBA) {
	t.Helper()
	size := tileSize * 16
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for tileIdx := 0; tileIdx < 256; tileIdx++ {
		col := tileIdx % 16
		row := tileIdx / 16
		c := fill(tileIdx)
		for y := row * tileSize; y < (row+1)*tileSize; y++ {
			for x := col * tileSize; x < (col+1)*tileSize; x++ {
				img.SetRGBA(x, y, c)
			}
		}
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

func TestBakeAtlas(t *testing.T) {
	t.Run("errors when the file doesn't exist", func(t *testing.T) {
		_, err := BakeAtlas(filepath.Join(t.TempDir(), "missing.png"))
		if err == nil {
			t.Fatal("expected an error for a missing atlas file")
		}
	})

	t.Run("errors when the file isn't a decodable image", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "not-an-image.png")
		if err := os.WriteFile(path, []byte("not a png"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := BakeAtlas(path)
		if err == nil {
			t.Fatal("expected a decode error")
		}
	})

	t.Run("errors when the atlas is smaller than a 16x16 grid", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tiny.png")
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()

		_, err = BakeAtlas(path)
		if err == nil {
			t.Fatal("expected an 'atlas too small' error")
		}
	})

	t.Run("computes the exact average color per tile on a controlled atlas", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "atlas.png")
		// Every tile is a distinct flat color: tile i -> (i, 255-i, 128, 255).
		writeTestAtlas(t, path, 4, func(i int) color.RGBA {
			return color.RGBA{R: uint8(i), G: uint8(255 - i), B: 128, A: 255}
		})

		colors, err := BakeAtlas(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(colors) != 256 {
			t.Fatalf("expected 256 tile colors, got %d", len(colors))
		}
		for _, tileIdx := range []int{0, 1, 42, 255} {
			got := colors[tileIdx].Average
			want := color.RGBA{R: uint8(tileIdx), G: uint8(255 - tileIdx), B: 128, A: 255}
			if got != want {
				t.Fatalf("tile %d: got %+v, want %+v", tileIdx, got, want)
			}
		}
	})

	t.Run("fully transparent tiles fall back to a neutral gray", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "atlas.png")
		writeTestAtlas(t, path, 2, func(i int) color.RGBA {
			return color.RGBA{} // fully transparent (alpha 0) for every tile
		})

		colors, err := BakeAtlas(path)
		if err != nil {
			t.Fatal(err)
		}
		want := color.RGBA{128, 128, 128, 255}
		if colors[0].Average != want {
			t.Fatalf("expected the neutral gray fallback %+v, got %+v", want, colors[0].Average)
		}
	})
}

func TestApplyLightMultiplier(t *testing.T) {
	cases := []struct {
		name  string
		color color.RGBA
		light float64
		want  uint32
	}{
		{"full intensity is a no-op", color.RGBA{R: 200, G: 100, B: 50, A: 255}, 1.0, 0xC86432},
		{"half intensity halves each channel", color.RGBA{R: 200, G: 100, B: 50, A: 255}, 0.5, 0x643219},
		{"zero intensity is black", color.RGBA{R: 200, G: 100, B: 50, A: 255}, 0.0, 0x000000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ApplyLightMultiplier(c.color, c.light)
			if got != c.want {
				t.Fatalf("got 0x%06X, want 0x%06X", got, c.want)
			}
		})
	}
}

func TestResolveAtlasPath(t *testing.T) {
	t.Run("prefers an exact explicit path", func(t *testing.T) {
		dir := t.TempDir()
		explicit := filepath.Join(dir, "custom.png")
		if err := os.WriteFile(explicit, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := ResolveAtlasPath(dir, explicit)
		if err != nil {
			t.Fatal(err)
		}
		if got != explicit {
			t.Fatalf("got %q, want %q", got, explicit)
		}
	})

	t.Run("resolves an explicit path relative to the project root", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "Custom"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Custom", "atlas.png"), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := ResolveAtlasPath(dir, "Custom/atlas.png")
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(dir, "Custom", "atlas.png")
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to the conventional candidate paths", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "Assets", "Textures"), 0755); err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(dir, "Assets", "Textures", "terrain.png")
		if err := os.WriteFile(want, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := ResolveAtlasPath(dir, "")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("errors when nothing matches", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ResolveAtlasPath(dir, "")
		if err == nil {
			t.Fatal("expected a not-found error")
		}
	})
}

func TestBakeAtlasIfExists(t *testing.T) {
	t.Run("best-quality profile skips baking entirely, even with no atlas present", func(t *testing.T) {
		dir := t.TempDir()
		colors, err := BakeAtlasIfExists(dir, "", ProfileQuality)
		if err != nil {
			t.Fatal(err)
		}
		if colors != nil {
			t.Fatalf("expected nil colors for best-quality profile, got %d entries", len(colors))
		}
	})

	t.Run("best-performance profile bakes a resolved atlas", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "Assets", "Textures"), 0755); err != nil {
			t.Fatal(err)
		}
		writeTestAtlas(t, filepath.Join(dir, "Assets", "Textures", "terrain.png"), 2, func(i int) color.RGBA {
			return color.RGBA{R: 10, G: 20, B: 30, A: 255}
		})

		colors, err := BakeAtlasIfExists(dir, "", ProfilePerformance)
		if err != nil {
			t.Fatal(err)
		}
		if len(colors) != 256 {
			t.Fatalf("expected 256 baked tile colors, got %d", len(colors))
		}
	})

	t.Run("best-performance profile with no atlas present is a real error", func(t *testing.T) {
		dir := t.TempDir()
		_, err := BakeAtlasIfExists(dir, "", ProfilePerformance)
		if err == nil {
			t.Fatal("expected an error when no atlas can be found under best-performance")
		}
	})
}
