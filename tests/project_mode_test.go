package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"Game-Developers-World/seed/internal/project"
)

// TestInvalidModeRejected checks that project.Scaffold validates --mode
// before doing any network or filesystem work (the pinned-GameAK clone in
// particular is expensive) — see project.validateMode, added alongside
// the new "headless" mode in this phase.
func TestInvalidModeRejected(t *testing.T) {
	dir := t.TempDir()
	defer inDir(t, dir)()

	err := project.Scaffold("bogus-project", "not-a-real-mode")
	if err == nil {
		t.Fatal("expected an error for an invalid --mode")
	}
	if !contains(err.Error(), "invalid mode") {
		t.Fatalf("expected an 'invalid mode' error, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "bogus-project")); !os.IsNotExist(statErr) {
		t.Fatal("expected no project directory to be created for an invalid mode")
	}
}

// TestScaffoldModesCompile scaffolds a fresh project for every supported
// --mode (2d, 3d, headless) and builds it with xmake against the pinned
// GameAK revision and real SDL3 packages — the strongest available check
// that internal/project/templates/main.cpp.tmpl and seed-context.hpp.tmpl
// (rewritten in this phase for the fixed-step accumulator loop, the
// extraction step, and Context-as-composition-root) actually produce
// compilable C++, not just Go-template-valid text.
//
// Opt-in like tests/gameak_compat_test.go: needs network (GameAK clone +
// SDL3 package fetch) and a real xmake toolchain.
func TestScaffoldModesCompile(t *testing.T) {
	if os.Getenv("SEED_TEST_GAMEAK_COMPILE") != "1" {
		t.Skip("skipping scaffold compile test; set SEED_TEST_GAMEAK_COMPILE=1 to run it (requires network and xmake)")
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

	for _, mode := range []string{"headless", "2d", "3d"} {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			workDir := filepath.Join(tmpDir, mode)
			if err := os.MkdirAll(workDir, 0755); err != nil {
				t.Fatal(err)
			}

			newCmd := exec.Command(binPath, "new", "demo", "--mode", mode)
			newCmd.Dir = workDir
			if out, err := newCmd.CombinedOutput(); err != nil {
				t.Fatalf("seed new --mode %s failed: %v\n%s", mode, err, out)
			}

			projDir := filepath.Join(workDir, "demo")
			mainCpp, err := os.ReadFile(filepath.Join(projDir, "Src", "main.cpp"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(mainCpp), "{{") {
				t.Fatalf("main.cpp for mode %s contains an unrendered template directive:\n%s", mode, mainCpp)
			}
			if mode != "headless" {
				for _, want := range []string{"InputTranslator", "seed::AssetManager", "seed::Camera2D"} {
					if !strings.Contains(string(mainCpp), want) {
						t.Fatalf("main.cpp for mode %s missing expected facility %q", mode, want)
					}
				}
			}
			if mode == "3d" {
				for _, want := range []string{"seed::Scene", "seed::SceneManager"} {
					if !strings.Contains(string(mainCpp), want) {
						t.Fatalf("main.cpp for mode 3d missing expected facility %q", want)
					}
				}
			}

			buildCmd := exec.Command("xmake", "build", "-y")
			buildCmd.Dir = projDir
			out, err := buildCmd.CombinedOutput()
			if err != nil {
				t.Fatalf("xmake build failed for mode %s: %v\n%s", mode, err, out)
			}
		})
	}
}
