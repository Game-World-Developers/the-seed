package baker

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
)

type TileColors struct {
	Average color.RGBA
}

// BakeAtlas reads a texture atlas PNG and computes the average color
// for each tile. The atlas is assumed to be 256x256 with 16x16 pixel tiles
// arranged in a 16x16 grid (tile index 0..255, row-major).
// Returns a slice of 256 TileColors.
func BakeAtlas(atlasPath string) ([]TileColors, error) {
	f, err := os.Open(atlasPath)
	if err != nil {
		return nil, fmt.Errorf("opening atlas %s: %w", atlasPath, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding atlas %s: %w", atlasPath, err)
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	tileW := w / 16
	tileH := h / 16

	if tileW < 1 || tileH < 1 {
		return nil, fmt.Errorf("atlas too small: %dx%d, need at least 16x16", w, h)
	}

	// Light multipliers for top/bottom/side
	// These match the values used in ChunkMesh.cpp
	_ = 1.0 // top
	_ = 0.5 // bottom
	_ = 0.7 // side

	colors := make([]TileColors, 256)

	for tileIdx := 0; tileIdx < 256; tileIdx++ {
		col := tileIdx % 16
		row := tileIdx / 16

		tx := col * tileW
		ty := row * tileH

		var rSum, gSum, bSum uint64
		var count uint64

		for y := ty; y < ty+tileH && y < h; y++ {
			for x := tx; x < tx+tileW && x < w; x++ {
				r, g, b, a := img.At(x, y).RGBA()
				// RGBA() returns premultiplied 16-bit values
				if a > 0 {
					rSum += uint64(r)
					gSum += uint64(g)
					bSum += uint64(b)
					count++
				}
			}
		}

		if count == 0 {
			colors[tileIdx] = TileColors{color.RGBA{128, 128, 128, 255}}
			continue
		}

		// Convert from 16-bit to 8-bit
		colors[tileIdx] = TileColors{
			color.RGBA{
				R: uint8((rSum / count) >> 8),
				G: uint8((gSum / count) >> 8),
				B: uint8((bSum / count) >> 8),
				A: 255,
			},
		}
	}

	return colors, nil
}

// ApplyLightMultiplier applies a per-face light multiplier to a base color
// and returns the packed 0xRRGGBB value.
func ApplyLightMultiplier(c color.RGBA, light float64) uint32 {
	r := uint8(float64(c.R) * light)
	g := uint8(float64(c.G) * light)
	b := uint8(float64(c.B) * light)
	return (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}

// ResolveAtlasPath searches for a terrain atlas PNG relative to the project root.
// Priority: 1) exact path, 2) Assets/Textures/terrain.png, 3) Assets/terrain.png
func ResolveAtlasPath(rootDir, explicitPath string) (string, error) {
	if explicitPath != "" {
		if fi, err := os.Stat(explicitPath); err == nil && !fi.IsDir() {
			return explicitPath, nil
		}
		if fi, err := os.Stat(filepath.Join(rootDir, explicitPath)); err == nil && !fi.IsDir() {
			return filepath.Join(rootDir, explicitPath), nil
		}
	}

	candidates := []string{
		"Assets/Textures/terrain.png",
		"Assets/terrain.png",
		"terrain.png",
	}

	for _, c := range candidates {
		p := filepath.Join(rootDir, c)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("terrain atlas not found (searched in %s)", rootDir)
}

// BakeAtlasIfExists attempts to bake the texture atlas into tile colors.
// Returns nil if the atlas doesn't exist or if profile is "best-quality".
func BakeAtlasIfExists(rootDir, explicitPath string, profile Profile) ([]TileColors, error) {
	if profile != ProfilePerformance {
		return nil, nil
	}

	atlasPath, err := ResolveAtlasPath(rootDir, explicitPath)
	if err != nil {
		return nil, err
	}

	return BakeAtlas(atlasPath)
}
