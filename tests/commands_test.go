package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCommandsBinary(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")

	t.Run("build binary", func(t *testing.T) {
		build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
		out, err := build.CombinedOutput()
		if err != nil {
			t.Fatalf("build failed: %v\n%s", err, out)
		}
	})

	t.Run("--help", func(t *testing.T) {
		cmd := exec.Command(binPath, "--help")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--help failed: %v", err)
		}
		if !contains(string(out), "The Seed") {
			t.Fatalf("expected help to contain 'The Seed', got: %s", out)
		}
	})

	t.Run("--version", func(t *testing.T) {
		cmd := exec.Command(binPath, "--version")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--version failed: %v", err)
		}
		if !contains(string(out), "0.1.0") {
			t.Fatalf("expected version 0.1.0, got: %s", out)
		}
	})

	t.Run("new --help", func(t *testing.T) {
		cmd := exec.Command(binPath, "new", "--help")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("new --help failed: %v", err)
		}
		if !contains(string(out), "<project-name>") {
			t.Fatalf("expected new help to contain '<project-name>', got: %s", out)
		}
	})

	t.Run("model generate --help (--all)", func(t *testing.T) {
		cmd := exec.Command(binPath, "model", "generate", "--help")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("model generate --help failed: %v", err)
		}
		if !contains(string(out), "--all") {
			t.Fatalf("expected model generate help to contain '--all', got: %s", out)
		}
	})

	t.Run("sync --help (--check)", func(t *testing.T) {
		cmd := exec.Command(binPath, "sync", "--help")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("sync --help failed: %v", err)
		}
		if !contains(string(out), "--check") {
			t.Fatalf("expected sync help to contain '--check', got: %s", out)
		}
	})

	t.Run("model list --help", func(t *testing.T) {
		cmd := exec.Command(binPath, "model", "list", "--help")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("model list --help failed: %v", err)
		}
		if !contains(string(out), "list") {
			t.Fatalf("expected model list help to contain 'list', got: %s", out)
		}
	})

	t.Run("model delete --help", func(t *testing.T) {
		cmd := exec.Command(binPath, "model", "delete", "--help")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("model delete --help failed: %v", err)
		}
		if !contains(string(out), "<type> <name>") {
			t.Fatalf("expected model delete help to contain '<type> <name>', got: %s", out)
		}
	})

	t.Run("model generate creates model", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(tmpDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binPath, "model", "generate", "component", "TestPos")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("model generate failed: %v\n%s", err, out)
		}
		if !contains(string(out), "create") {
			t.Fatalf("expected 'create', got: %s", out)
		}
	})

	t.Run("fails without .seed_project", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "no-project")
		if err := os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binPath, "model", "generate", "component", "Foo")
		cmd.Dir = emptyDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected error without .seed_project")
		}
		if !contains(string(out), "not a seed project") {
			t.Fatalf("expected 'not a seed project', got: %s", out)
		}
	})

	t.Run("succeeds with .seed_project", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "valid-project")
		if err := os.MkdirAll(projDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binPath, "model", "generate", "component", "Foo")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected success with .seed_project: %v\n%s", err, out)
		}
		if !contains(string(out), "create") {
			t.Fatalf("expected 'create', got: %s", out)
		}
	})
}

func TestDoctorChecks(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")

	t.Run("build binary", func(t *testing.T) {
		build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
		out, err := build.CombinedOutput()
		if err != nil {
			t.Fatalf("build failed: %v\n%s", err, out)
		}
	})

	projDir := filepath.Join(tmpDir, "testproj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize project skeleton
	if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"Models/System", "Models/Component", "Include", "Src/Game", "Assets"} {
		if err := os.MkdirAll(filepath.Join(projDir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Stub GameAK so the third-party check passes
	gameakDir := filepath.Join(projDir, "Third-Party", "GameAK", "Include", "GameAk", "Runtime")
	if err := os.MkdirAll(gameakDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameakDir, "runtime.h"), []byte("// stub"), 0644); err != nil {
		t.Fatal(err)
	}

	runCmd := func(args ...string) (string, error) {
		cmd := exec.Command(binPath, args...)
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	t.Run("doctor passes on empty project", func(t *testing.T) {
		out, err := runCmd("doctor")
		if err != nil {
			t.Fatalf("doctor failed: %v\n%s", err, out)
		}
		if !contains(out, "seed doctor") {
			t.Fatalf("expected doctor header, got: %s", out)
		}
		if !contains(out, "Project structure") {
			t.Fatalf("expected project structure check, got: %s", out)
		}
	})

	// Create a system model
	systemYAML := `type: system
name: Gravity
namespace: Core
entities: []
access: []
`
	if err := os.WriteFile(filepath.Join(projDir, "Models", "System", "Gravity.yaml"), []byte(systemYAML), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("sync creates system", func(t *testing.T) {
		out, err := runCmd("sync")
		if err != nil {
			t.Fatalf("sync failed: %v\n%s", err, out)
		}
		// .hpp should exist
		hpp := filepath.Join(projDir, "Include", "Core", "Gravity.hpp")
		if _, err := os.Stat(hpp); os.IsNotExist(err) {
			t.Fatalf("expected .hpp after sync: %s", hpp)
		}
		// .cpp should exist
		cpp := filepath.Join(projDir, "Src", "Game", "Gravity.cpp")
		if _, err := os.Stat(cpp); os.IsNotExist(err) {
			t.Fatalf("expected .cpp after sync: %s", cpp)
		}
	})

	t.Run("detects missing .cpp", func(t *testing.T) {
		// Delete the .cpp to simulate missing implementation
		cpp := filepath.Join(projDir, "Src", "Game", "Gravity.cpp")
		if err := os.Remove(cpp); err != nil {
			t.Fatal(err)
		}

		out, err := runCmd("doctor")
		if err != nil {
			t.Fatalf("doctor failed: %v\n%s", err, out)
		}
		if !contains(out, "System Core/Gravity") {
			t.Fatalf("expected 'System Core/Gravity' warning, got: %s", out)
		}
		if !contains(out, "Missing .cpp") {
			t.Fatalf("expected 'Missing .cpp' warning, got: %s", out)
		}
	})

	// Restore .cpp for subsequent tests
	os.WriteFile(filepath.Join(projDir, "Src", "Game", "Gravity.cpp"), []byte("// placeholder"), 0644)

	t.Run("detects orphaned files", func(t *testing.T) {
		// Create an orphan .hpp in Include with no model
		orphanHpp := filepath.Join(projDir, "Include", "Core", "Orphan.hpp")
		if err := os.WriteFile(orphanHpp, []byte("// orphan"), 0644); err != nil {
			t.Fatal(err)
		}

		out, err := runCmd("doctor")
		if err != nil {
			t.Fatalf("doctor failed: %v\n%s", err, out)
		}
		if !contains(out, "Orphaned files") {
			t.Fatalf("expected 'Orphaned files' warning, got: %s", out)
		}
		if !contains(out, "Orphan.hpp") {
			t.Fatalf("expected 'Orphan.hpp' in warning, got: %s", out)
		}

		// Clean up orphan
		if err := os.Remove(orphanHpp); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("detects duplicate model names", func(t *testing.T) {
		// Create a duplicate system with same name but different namespace
		dupYAML := `type: system
name: Gravity
namespace: Game
entities: []
access: []
`
		if err := os.WriteFile(filepath.Join(projDir, "Models", "System", "GravityDup.yaml"), []byte(dupYAML), 0644); err != nil {
			t.Fatal(err)
		}

		out, err := runCmd("doctor")
		if err != nil {
			t.Fatalf("doctor failed: %v\n%s", err, out)
		}
		if !contains(out, "Duplicate model names") {
			t.Fatalf("expected 'Duplicate model names' warning, got: %s", out)
		}
		if !contains(out, `"Gravity"`) {
			t.Fatalf("expected model name 'Gravity' in warning, got: %s", out)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
