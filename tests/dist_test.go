package tests

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/dist"
	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/ir"
)

func TestDistManifestAndBundle(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("builds a manifest and zip bundle from declared assets", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "with-assets")
		defer inDir(t, dir)()

		if err := os.MkdirAll(filepath.Join(dir, "Assets", "Textures"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Assets", "Textures", "Player.bmp"), []byte("fake-bmp-content"), 0644); err != nil {
			t.Fatal(err)
		}
		writeModel(t, dir, "Asset", "PlayerTexture",
			"type: asset\nname: PlayerTexture\nnamespace: Core\nkind: texture\npath: Assets/Textures/Player.bmp\n")

		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if compiled.HasErrors() {
			t.Fatalf("unexpected errors: %v", compiled.Errors())
		}
		model, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}

		app := dist.AppMetadata{Name: "TestGame", Version: "1.2.3", Identifier: "com.example.testgame"}
		manifest, err := dist.BuildManifest(model, app, dir, "linux")
		if err != nil {
			t.Fatal(err)
		}
		if len(manifest.Assets) != 1 {
			t.Fatalf("expected 1 asset in manifest, got %d", len(manifest.Assets))
		}
		if manifest.Assets[0].Size != int64(len("fake-bmp-content")) {
			t.Fatalf("expected size %d, got %d", len("fake-bmp-content"), manifest.Assets[0].Size)
		}
		if manifest.Assets[0].SHA256 == "" {
			t.Fatal("expected a non-empty checksum")
		}

		outPath := filepath.Join(dir, "dist", "bundle.zip")
		if err := dist.WriteBundle(manifest, dir, outPath); err != nil {
			t.Fatal(err)
		}

		zr, err := zip.OpenReader(outPath)
		if err != nil {
			t.Fatal(err)
		}
		defer zr.Close()

		var foundManifest, foundAsset bool
		for _, f := range zr.File {
			switch f.Name {
			case "manifest.json":
				foundManifest = true
				rc, err := f.Open()
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					t.Fatal(err)
				}
				var decoded dist.Manifest
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Fatal(err)
				}
				if decoded.App.Name != "TestGame" || decoded.Platform != "linux" {
					t.Fatalf("unexpected manifest content: %+v", decoded)
				}
			case "Assets/Textures/Player.bmp":
				foundAsset = true
			}
		}
		if !foundManifest {
			t.Fatal("expected manifest.json in the bundle")
		}
		if !foundAsset {
			t.Fatal("expected the asset file in the bundle")
		}
	})

	t.Run("fails when a declared asset's file is missing", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "missing-asset")
		defer inDir(t, dir)()

		writeModel(t, dir, "Asset", "GhostTexture",
			"type: asset\nname: GhostTexture\nnamespace: Core\nkind: texture\npath: Assets/Textures/DoesNotExist.bmp\n")

		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		model, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}

		_, err = dist.BuildManifest(model, dist.AppMetadata{Name: "X"}, dir, "linux")
		if err == nil {
			t.Fatal("expected an error for a missing asset file")
		}
	})

	t.Run("LoadAppMetadata defaults sensibly and reads app.yaml when present", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "app-meta")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		defaulted, err := dist.LoadAppMetadata(dir)
		if err != nil {
			t.Fatal(err)
		}
		if defaulted.Version != "0.1.0" || defaulted.Identifier == "" {
			t.Fatalf("expected sensible defaults, got: %+v", defaulted)
		}

		appYAML := "name: MyGame\nversion: 2.0.0\nidentifier: com.example.mygame\n"
		if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(appYAML), 0644); err != nil {
			t.Fatal(err)
		}
		loaded, err := dist.LoadAppMetadata(dir)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Name != "MyGame" || loaded.Version != "2.0.0" || loaded.Identifier != "com.example.mygame" {
			t.Fatalf("expected app.yaml to be read, got: %+v", loaded)
		}
	})
}
