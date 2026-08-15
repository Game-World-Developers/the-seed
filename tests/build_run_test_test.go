package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestBuildRunTestCommands verifies `seed build`/`seed run`/`seed test`
// wrap xmake with a consistent --profile translation. --profile requires
// running xmake's config step first (xmake build/run/test don't accept
// -m directly, only `xmake f` does — see internal/commands/build.go), so
// this is worth a real, if slow, xmake invocation rather than a
// help-text-only check.
//
// Opt-in like tests/gameak_compat_test.go: needs network (GameAK clone +
// SDL3 package fetch) and a real xmake toolchain.
func TestBuildRunTestCommands(t *testing.T) {
	if os.Getenv("SEED_TEST_GAMEAK_COMPILE") != "1" {
		t.Skip("skipping build/run/test command test; set SEED_TEST_GAMEAK_COMPILE=1 to run it (requires network and xmake)")
	}
	if _, err := exec.LookPath("xmake"); err != nil {
		t.Skip("xmake not found in PATH")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")
	build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	workDir := filepath.Join(tmpDir, "proj")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}
	newCmd := exec.Command(binPath, "new", "demo", "--mode", "headless")
	newCmd.Dir = workDir
	if out, err := newCmd.CombinedOutput(); err != nil {
		t.Fatalf("seed new failed: %v\n%s", err, out)
	}
	projDir := filepath.Join(workDir, "demo")

	t.Run("build --profile debug compiles cleanly", func(t *testing.T) {
		cmd := exec.Command(binPath, "build", "--profile", "debug")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("build failed: %v\n%s", err, out)
		}
	})

	t.Run("build rejects an invalid --profile before touching xmake", func(t *testing.T) {
		cmd := exec.Command(binPath, "build", "--profile", "turbo")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected an error for an invalid profile, got: %s", out)
		}
		if !contains(string(out), "invalid --profile") {
			t.Fatalf("expected an 'invalid --profile' message, got: %s", out)
		}
	})

	t.Run("test reports no test targets rather than failing", func(t *testing.T) {
		cmd := exec.Command(binPath, "test")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected 'test' to succeed with no test targets defined, got: %v\n%s", err, out)
		}
	})
}
