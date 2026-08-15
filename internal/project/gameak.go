package project

import (
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GameAKRepo is the upstream GameAK repository Seed integrates against.
const GameAKRepo = "https://github.com/Game-World-Developers/GameAK.git"

// GameAKPinnedRevision is the exact GameAK commit `seed new`/`seed init`
// clones, on the `dev` branch (GameAK has no tagged releases at the time
// of writing — verified via `git ls-remote --tags`). GameAK's `dev` branch
// is a moving target with no compatibility guarantees; cloning its
// current tip on every `seed new` (the previous behavior) meant a
// project's build could silently start failing the moment upstream
// pushed a breaking change, with no way to reproduce an older, working
// project on a fresh machine.
//
// This pin is Seed's compatibility record for Phase 5's "pin or record
// GameAK compatibility instead of depending implicitly on its moving dev
// branch" — see docs/gameak-mapping.md for what was verified against it.
//
// To move the pin: verify the new commit against the compile tests in
// tests/gameak_compat_test.go (skipped unless SEED_TEST_GAMEAK_COMPILE=1,
// since it requires xmake and a full toolchain), then update this
// constant in the same commit as any generator/template change the new
// revision requires.
const GameAKPinnedRevision = "5049747541ef4e247a9b2d5ce67f896590c4d0d9"

// CloneGameAK clones GameAKRepo into dir and checks out the pinned
// revision. Unlike a shallow single-branch clone of `dev`, this fetches
// full history so the pinned commit is always reachable regardless of how
// far `dev` has moved upstream by the time this runs. Exported so
// tests/gameak_compat_test.go can reuse the exact same clone/pin logic
// `seed new`/`seed init` use, instead of duplicating it.
func CloneGameAK(dir string) error {
	repo, err := gogit.PlainClone(dir, false, &gogit.CloneOptions{
		URL: GameAKRepo,
	})
	if err != nil {
		return fmt.Errorf("failed to clone GameAK: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("opening GameAK worktree: %w", err)
	}
	if err := wt.Checkout(&gogit.CheckoutOptions{
		Hash: plumbing.NewHash(GameAKPinnedRevision),
	}); err != nil {
		return fmt.Errorf("checking out pinned GameAK revision %s: %w", GameAKPinnedRevision, err)
	}
	return nil
}
