package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDoctorTargetSDKChecks verifies "seed doctor" explains target-specific
// build requirements (glslc for 3D, expected SDL3 packages per mode) and
// checks the vendored GameAK revision against the pin — real filesystem/
// git checks, not just a static message. Opt-in like the other
// GameAK-clone-dependent tests: needs network to actually clone GameAK.
func TestDoctorTargetSDKChecks(t *testing.T) {
	if os.Getenv("SEED_TEST_GAMEAK_COMPILE") != "1" {
		t.Skip("skipping doctor target check test; set SEED_TEST_GAMEAK_COMPILE=1 to run it (requires network)")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")
	build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	t.Run("3d project mentions glslc and the SDL3 packages it needs", func(t *testing.T) {
		workDir := filepath.Join(tmpDir, "3d-work")
		if err := os.MkdirAll(workDir, 0755); err != nil {
			t.Fatal(err)
		}
		newCmd := exec.Command(binPath, "new", "demo", "--mode", "3d")
		newCmd.Dir = workDir
		if out, err := newCmd.CombinedOutput(); err != nil {
			t.Fatalf("seed new failed: %v\n%s", err, out)
		}

		cmd := exec.Command(binPath, "doctor")
		cmd.Dir = filepath.Join(workDir, "demo")
		out, _ := cmd.CombinedOutput()
		for _, want := range []string{"Target: 3d", "glslc", "libsdl3_image", "GameAK revision matches pin"} {
			if !contains(string(out), want) {
				t.Fatalf("expected doctor output to mention %q, got: %s", want, out)
			}
		}
	})

	t.Run("headless project reports no image/ttf packages needed", func(t *testing.T) {
		workDir := filepath.Join(tmpDir, "headless-work")
		if err := os.MkdirAll(workDir, 0755); err != nil {
			t.Fatal(err)
		}
		newCmd := exec.Command(binPath, "new", "demo", "--mode", "headless")
		newCmd.Dir = workDir
		if out, err := newCmd.CombinedOutput(); err != nil {
			t.Fatalf("seed new failed: %v\n%s", err, out)
		}

		cmd := exec.Command(binPath, "doctor")
		cmd.Dir = filepath.Join(workDir, "demo")
		out, _ := cmd.CombinedOutput()
		if !contains(string(out), "Target: headless") {
			t.Fatalf("expected 'Target: headless', got: %s", out)
		}
		if contains(string(out), "glslc") {
			t.Fatalf("expected no glslc mention for a headless project, got: %s", out)
		}
	})
}
