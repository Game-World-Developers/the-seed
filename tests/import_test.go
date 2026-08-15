package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestImportSafety(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")
	build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	newProject := func(t *testing.T, name string) string {
		t.Helper()
		dir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	run := func(t *testing.T, dir string, args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command(binPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	t.Run("re-importing without --overwrite is rejected, content is untouched", func(t *testing.T) {
		dir := newProject(t, "overwrite-protection")
		if err := os.WriteFile(filepath.Join(dir, "hero.png"), []byte("original-bytes"), 0644); err != nil {
			t.Fatal(err)
		}

		if out, err := run(t, dir, "import", "sprite", "hero.png"); err != nil {
			t.Fatalf("first import failed: %v\n%s", err, out)
		}

		// Change the source so a silent overwrite would be detectable.
		if err := os.WriteFile(filepath.Join(dir, "hero.png"), []byte("different-bytes"), 0644); err != nil {
			t.Fatal(err)
		}

		out, err := run(t, dir, "import", "sprite", "hero.png")
		if err == nil {
			t.Fatalf("expected re-import to be rejected, got: %s", out)
		}
		if !contains(out, "already exists") || !contains(out, "--overwrite") {
			t.Fatalf("expected an 'already exists ... --overwrite' message, got: %s", out)
		}

		data, err := os.ReadFile(filepath.Join(dir, "Assets", "Sprites", "hero.png"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "original-bytes" {
			t.Fatalf("expected the original asset to survive the rejected re-import, got: %q", data)
		}
	})

	t.Run("--overwrite allows replacing an existing asset", func(t *testing.T) {
		dir := newProject(t, "overwrite-allowed")
		if err := os.WriteFile(filepath.Join(dir, "hero.png"), []byte("v1"), 0644); err != nil {
			t.Fatal(err)
		}
		if out, err := run(t, dir, "import", "sprite", "hero.png"); err != nil {
			t.Fatalf("first import failed: %v\n%s", err, out)
		}

		if err := os.WriteFile(filepath.Join(dir, "hero.png"), []byte("v2"), 0644); err != nil {
			t.Fatal(err)
		}
		if out, err := run(t, dir, "import", "sprite", "hero.png", "--overwrite"); err != nil {
			t.Fatalf("overwrite import failed: %v\n%s", err, out)
		}

		data, err := os.ReadFile(filepath.Join(dir, "Assets", "Sprites", "hero.png"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "v2" {
			t.Fatalf("expected the asset to be updated to v2, got: %q", data)
		}
	})

	t.Run("importing a file already at its destination does not truncate it", func(t *testing.T) {
		dir := newProject(t, "same-file")
		spritesDir := filepath.Join(dir, "Assets", "Sprites")
		if err := os.MkdirAll(spritesDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(spritesDir, "hero.png"), []byte("already-here"), 0644); err != nil {
			t.Fatal(err)
		}

		out, err := run(t, dir, "import", "sprite", "Assets/Sprites/hero.png")
		if err != nil {
			t.Fatalf("import of an in-place file failed: %v\n%s", err, out)
		}
		if !contains(out, "identical") || !contains(out, "same file") {
			t.Fatalf("expected an 'identical ... same file' message, got: %s", out)
		}

		data, err := os.ReadFile(filepath.Join(spritesDir, "hero.png"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "already-here" {
			t.Fatalf("expected the file to survive untouched, got: %q", data)
		}
	})
}
