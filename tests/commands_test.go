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

	t.Run("compile validates and prints the IR without touching Include/", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "compile-project")
		if err := os.MkdirAll(projDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		gen := exec.Command(binPath, "model", "generate", "component", "Position")
		gen.Dir = projDir
		if out, err := gen.CombinedOutput(); err != nil {
			t.Fatalf("model generate failed: %v\n%s", err, out)
		}

		cmd := exec.Command(binPath, "compile")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("compile failed: %v\n%s", err, out)
		}
		if !contains(string(out), "Compiled OK") {
			t.Fatalf("expected 'Compiled OK', got: %s", out)
		}
		if _, statErr := os.Stat(filepath.Join(projDir, "Include")); !os.IsNotExist(statErr) {
			t.Fatal("expected 'compile' to not create any Include/ output")
		}

		jsonCmd := exec.Command(binPath, "compile", "--json")
		jsonCmd.Dir = projDir
		jsonOut, err := jsonCmd.Output()
		if err != nil {
			t.Fatalf("compile --json failed: %v", err)
		}
		if !contains(string(jsonOut), `"version"`) || !contains(string(jsonOut), `"Position"`) {
			t.Fatalf("expected JSON IR containing version and Position, got: %s", jsonOut)
		}
	})

	t.Run("compile fails on semantic errors without writing output", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "compile-error-project")
		if err := os.MkdirAll(filepath.Join(projDir, "Models", "Trait"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		badTrait := "type: trait\nname: Movable\nnamespace: Core\ncomponents:\n  - DoesNotExist\n"
		if err := os.WriteFile(filepath.Join(projDir, "Models", "Trait", "Movable.yaml"), []byte(badTrait), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(binPath, "compile")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected compile to fail, got: %s", out)
		}
		if !contains(string(out), "missing component") {
			t.Fatalf("expected diagnostic about missing component, got: %s", out)
		}
		if _, statErr := os.Stat(filepath.Join(projDir, "Include")); !os.IsNotExist(statErr) {
			t.Fatal("expected no Include/ output when compile fails")
		}
	})

	t.Run("compile and sync both reject a GameAK backend collision Seed's own symbol table allows", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "backend-collision")
		if err := os.MkdirAll(filepath.Join(projDir, "Models", "Component"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		// Two components named "Position" in different namespaces: valid
		// per Seed's own namespace-aware symbol table (semantic Compile
		// passes), but GameAK registers block types by bare name only —
		// see internal/backend/gameak.
		core := "type: component\nname: Position\nnamespace: Core\nfields: []\n"
		physics := "type: component\nname: Position\nnamespace: Physics\nfields: []\n"
		if err := os.WriteFile(filepath.Join(projDir, "Models", "Component", "Position.yaml"), []byte(core), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, "Models", "Component", "PositionPhysics.yaml"), []byte(physics), 0644); err != nil {
			t.Fatal(err)
		}

		compileCmd := exec.Command(binPath, "compile")
		compileCmd.Dir = projDir
		compileOut, compileErr := compileCmd.CombinedOutput()
		if compileErr == nil {
			t.Fatalf("expected 'compile' to reject the backend collision, got: %s", compileOut)
		}
		if !contains(string(compileOut), "rt.define") {
			t.Fatalf("expected 'compile' diagnostic to mention rt.define, got: %s", compileOut)
		}

		syncCmd := exec.Command(binPath, "sync")
		syncCmd.Dir = projDir
		syncOut, syncErr := syncCmd.CombinedOutput()
		if syncErr == nil {
			t.Fatalf("expected 'sync' to reject the backend collision, got: %s", syncOut)
		}
		if !contains(string(syncOut), "rt.define") {
			t.Fatalf("expected 'sync' diagnostic to mention rt.define, got: %s", syncOut)
		}
		if _, statErr := os.Stat(filepath.Join(projDir, "Include")); !os.IsNotExist(statErr) {
			t.Fatal("expected no Include/ output when sync aborts on a backend collision")
		}
	})

	t.Run("sync --dry-run previews changes without writing anything", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "dry-run-project")
		if err := os.MkdirAll(filepath.Join(projDir, "Models", "Component"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		comp := "type: component\nname: Position\nnamespace: Core\nfields:\n  - name: x\n    type: float\n"
		if err := os.WriteFile(filepath.Join(projDir, "Models", "Component", "Position.yaml"), []byte(comp), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(binPath, "sync", "--dry-run")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("sync --dry-run failed: %v\n%s", err, out)
		}
		if !contains(string(out), "create") || !contains(string(out), "1 to create") {
			t.Fatalf("expected a create summary, got: %s", out)
		}
		if _, statErr := os.Stat(filepath.Join(projDir, "Include")); !os.IsNotExist(statErr) {
			t.Fatal("expected sync --dry-run to write no files")
		}
	})

	t.Run("package builds a manifest+zip bundle from declared assets", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "package-project")
		if err := os.MkdirAll(filepath.Join(projDir, "Models", "Asset"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(projDir, "Assets", "Textures"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, "Assets", "Textures", "Player.bmp"), []byte("fake-bmp"), 0644); err != nil {
			t.Fatal(err)
		}
		asset := "type: asset\nname: PlayerTexture\nnamespace: Core\nkind: texture\npath: Assets/Textures/Player.bmp\n"
		if err := os.WriteFile(filepath.Join(projDir, "Models", "Asset", "PlayerTexture.yaml"), []byte(asset), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(binPath, "package", "--platform", "linux")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("package failed: %v\n%s", err, out)
		}
		if !contains(string(out), "1 asset(s)") {
			t.Fatalf("expected '1 asset(s)' in output, got: %s", out)
		}

		bundlePath := filepath.Join(projDir, "dist")
		entries, err := os.ReadDir(bundlePath)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected exactly 1 file in dist/, got %d", len(entries))
		}
	})

	t.Run("package fails cleanly when an asset file is missing", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "package-missing-asset")
		if err := os.MkdirAll(filepath.Join(projDir, "Models", "Asset"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		asset := "type: asset\nname: GhostTexture\nnamespace: Core\nkind: texture\npath: Assets/Textures/Ghost.bmp\n"
		if err := os.WriteFile(filepath.Join(projDir, "Models", "Asset", "GhostTexture.yaml"), []byte(asset), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(binPath, "package")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected package to fail for a missing asset file, got: %s", out)
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

	t.Run("duplicate names are a warning only, doctor still exits 0", func(t *testing.T) {
		_, err := runCmd("doctor")
		if err != nil {
			t.Fatalf("expected doctor to exit 0 on warnings only, got: %v", err)
		}
	})

	t.Run("doctor exits non-zero when a check fails", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "no-seed-project")
		if err := os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binPath, "doctor")
		cmd.Dir = emptyDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected doctor to exit non-zero without a .seed_project, got: %s", out)
		}
		if !contains(string(out), "✘") {
			t.Fatalf("expected a failing check marker, got: %s", out)
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
