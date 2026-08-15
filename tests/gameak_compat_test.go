package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/project"
	projecttemplates "Game-Developers-World/seed/internal/project/templates"
)

// TestGameAKCompiles verifies that Seed's pinned GameAK revision
// (project.GameAKPinnedRevision) actually builds with xmake, using the
// exact gameak-xmake.lua Seed generates for every new project
// (internal/project/templates/gameak-xmake.lua.tmpl). It only builds
// GameAK's own static libraries (gameak-core, gameak-runtime) — no SDL3
// packages are required, so this doesn't need network access to a
// package repository, only to GitHub to clone GameAK once.
//
// This is Phase 5's "compile tests against the supported GameAK
// revision." It's opt-in (SEED_TEST_GAMEAK_COMPILE=1) rather than run by
// default: it clones a real (if small) repository and invokes a real
// toolchain, both of which are appropriate for a deliberate compatibility
// check but not for every `go test ./...` run in an offline or
// toolchain-less environment.
func TestGameAKCompiles(t *testing.T) {
	if os.Getenv("SEED_TEST_GAMEAK_COMPILE") != "1" {
		t.Skip("skipping GameAK compile test; set SEED_TEST_GAMEAK_COMPILE=1 to run it (requires network and xmake)")
	}
	if _, err := exec.LookPath("xmake"); err != nil {
		t.Skip("xmake not found in PATH")
	}

	dir := t.TempDir()
	gameakDir := filepath.Join(dir, "GameAK")

	if err := project.CloneGameAK(gameakDir); err != nil {
		t.Fatalf("cloning pinned GameAK revision %s: %v", project.GameAKPinnedRevision, err)
	}

	xmakeLua, err := projecttemplates.RenderStatic("gameak-xmake.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameakDir, "xmake.lua"), []byte(xmakeLua), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("xmake", "build", "gameak-runtime")
	cmd.Dir = gameakDir
	cmd.Env = append(os.Environ(), "XMAKE_ROOT=y")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("GameAK pinned revision %s failed to build against Seed's gameak-xmake.lua: %v\n%s",
			project.GameAKPinnedRevision, err, out)
	}
}
