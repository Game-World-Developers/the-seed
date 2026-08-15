package baker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadProfile(t *testing.T) {
	t.Run("defaults to best-quality when .seed_project doesn't exist", func(t *testing.T) {
		got := ReadProfile(t.TempDir())
		if got != ProfileQuality {
			t.Fatalf("got %q, want %q", got, ProfileQuality)
		}
	})

	t.Run("defaults to best-quality when .seed_project isn't valid YAML", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".seed_project"), []byte("not: [valid yaml"), 0644); err != nil {
			t.Fatal(err)
		}
		got := ReadProfile(dir)
		if got != ProfileQuality {
			t.Fatalf("got %q, want %q", got, ProfileQuality)
		}
	})

	t.Run("reads an explicit best-performance profile", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".seed_project"), []byte("asset_profile: best-performance\n"), 0644); err != nil {
			t.Fatal(err)
		}
		got := ReadProfile(dir)
		if got != ProfilePerformance {
			t.Fatalf("got %q, want %q", got, ProfilePerformance)
		}
	})

	t.Run("reads an explicit best-quality profile", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".seed_project"), []byte("asset_profile: best-quality\n"), 0644); err != nil {
			t.Fatal(err)
		}
		got := ReadProfile(dir)
		if got != ProfileQuality {
			t.Fatalf("got %q, want %q", got, ProfileQuality)
		}
	})

	t.Run("an unrecognized profile value falls back to best-quality", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".seed_project"), []byte("asset_profile: turbo\n"), 0644); err != nil {
			t.Fatal(err)
		}
		got := ReadProfile(dir)
		if got != ProfileQuality {
			t.Fatalf("got %q, want %q", got, ProfileQuality)
		}
	})
}
