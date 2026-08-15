# CLI and project hardening

Phase 11's 9-item checklist. Three real bugs were found and fixed while
implementing this — a `seed import` file-corruption risk, a silent
`os.Chdir` failure mode, and a broken coverage measurement that had been
under-reporting real coverage by an order of magnitude — none of them
theoretical, all reproduced and verified fixed before being called done.

## 1. Validate `seed new --mode` and all enum-like CLI inputs

`--mode` was already validated (`project.validateMode`, Phase 7) —
rejecting anything other than `2d`/`3d`/`headless` before any
network/filesystem side effect runs. Auditing every other enum-like input
in this pass: `seed generate <type>`/`seed import <kind>` are validated
against `validTypes`/`importKinds` map lookups (pre-existing);
`seed build/run/test --profile` is validated against `debug`/`release`
(Phase 9). `seed package --platform` was deliberately left unvalidated:
it's a free-form label written into the manifest (`internal/dist`), never
branched on internally, and `docs/platforms.md`'s target matrix is still
open-ended (Android/iOS/web are a stated plan, not a closed enum) — a
hard-coded whitelist there would block a legitimate future platform
label for no correctness benefit.

## 2. Handle working-directory changes and filesystem errors explicitly

**Real bug found and fixed:** `project.Scaffold` and `seed new`'s
existing-project path both did `oldDir, _ := os.Getwd()` and
`os.Chdir(root)` with the error silently discarded. If `Chdir` failed
(permission denied, `root` not actually created, a race), the code
proceeded as if it had entered the new project directory while actually
still running from the old one — `writeProjectFiles`/`WriteStarterModels`/
`Sync` would then read and write files relative to the *wrong* directory
with no error and no explanation. Both call sites now check `Getwd`'s and
`Chdir`'s errors explicitly and fail with a clear message instead of
continuing in an unknown location; the deferred `Chdir` back is similarly
checked and reported (not swallowed) if it fails.

## 3. `seed doctor` returns non-zero on failed checks

**Real bug found and fixed:** `doctor`'s `RunE` always `return nil`
regardless of check results — a CI pipeline running `seed doctor` as a
gate could never actually fail on it. Now returns an error when any check
is `"fail"` (a warning alone — e.g. missing `glslc`, non-fatal per
`docs/developer-experience.md` §13 — still exits 0, since warnings are
advisory and shouldn't break a CI step that only wants to know the
project is *usable*). Verified with a real exit-code check both ways:
`tests/commands_test.go`'s `TestDoctorChecks` now asserts exit 0 on
warnings-only and non-zero on an actual failure (no `.seed_project`).

## 4-5. Prevent accidental overwrite in `seed import`; handle same source/destination

**Two real bugs found and fixed together**, both in
`internal/commands/import.go`:

- `copyFile`'s `os.Create(dst)` truncated the destination unconditionally
  — re-running `seed import` on an asset already imported silently
  destroyed whatever was there, and the generated `Models/Asset/<name>.yaml`
  was overwritten the same way. Now both check `os.Stat` first and refuse
  unless `--overwrite` is passed, with a message naming the flag.
- Worse: if `src` and `dst` name the *same file* (importing an asset
  that's already sitting in `Assets/<Kind>/`, or via a symlink), the old
  code would `os.Create(dst)` — truncating the file — *before* finishing
  reading it as `src`, corrupting or emptying it mid-copy. `sameUnderlyingFile`
  resolves symlinks and absolute paths on both sides first; when they
  match, the copy is skipped entirely (reported as `identical ... same
  file`) instead of attempted.

Both verified with real file-content assertions in
`tests/import_test.go` — not just "the command didn't error," but "the
original bytes are still there."

## 6. Test the atlas baker, profiles, block generation, and error paths

**A real gap, not a bug:** `internal/baker` had zero tests before this
pass despite being a real, non-trivial subsystem (image decoding, tile
averaging, profile-gated baking). Added:

- `internal/baker/baker_test.go` — `BakeAtlas` against a fully controlled,
  generated-in-memory PNG (exact expected per-tile averages, not
  approximate), every error path (missing file, undecodable file, atlas
  too small), the transparent-tile gray fallback, and `ApplyLightMultiplier`
  against hand-computed hex values. 100% statement coverage of the package.
- `internal/baker/profile_test.go` — every `ReadProfile` branch (missing
  file, invalid YAML, both valid profile values, an unrecognized value).
- `tests/block_generation_test.go` — the part nothing exercised at all:
  `seed sync` on a `block` model end-to-end, both profiles. Under
  `best-performance`, a real atlas PNG is baked and its exact computed hex
  color (`0x102030` for a flat `(0x10,0x20,0x30)` tile, light multiplier
  1.0 on the top face) is asserted present in the generated
  `BlockRegistry.hpp` — not just "a color," the *correct* one. Also covers
  the fallback path: `best-performance` with no atlas present logs a
  warning and falls back to YAML colors rather than failing the sync.

## 7. Repair and enforce coverage reporting

**Real bug found and fixed:** the Makefile's existing `coverage` target
ran `go test ./... -coverprofile=...` with no `-coverpkg`. Since most of
this repo's coverage comes from `tests/*.go` (black-box tests calling
into `internal/generators`, `internal/commands`, `internal/ir`, etc.),
and Go's default coverage mode only attributes coverage to the package
*under test*, every package exercised solely through `tests/` reported a
false `0.0%` — the total read **~2.7%** instead of the real **~35%**,
an order-of-magnitude under-report that would make anyone reading the
number conclude this codebase is barely tested, which is false. Fixed by
adding `-coverpkg=./...` (which attributes coverage to whichever package
the executed code actually lives in). A new `coverage-check` target fails
the build below `COVERAGE_MIN` (default 30%, real total currently ~35%)
and is wired into `make ci`, so a coverage regression is caught the same
way a test failure already is.

## 8. CI for formatting, vetting, tests, generation determinism, and C++ compilation

`.github/workflows/ci.yml`'s `go` job now runs, in order: `gofmt -l`
(fails on any unformatted file), `go build`, `go vet`, `go test`, the
repaired `make coverage-check` (§7), and a **generation determinism**
check — sync the `docs/examples` domain twice from a clean state and
`sha256sum`-diff every output file, the same check verified manually
while implementing Phase 10's reproducibility guarantee, now enforced on
every push/PR instead of only checked by hand. The pre-existing
`scaffold-build` matrix job (Phase 8) already covers real C++ compilation
against the pinned GameAK revision on Linux/Windows/macOS.

## 9. Stop committing or leaving build artifacts in the repository root

**A real, live risk, not a hypothetical:** there was no `.gitignore` in
this repository at all. The `seed` binary produced by `go build -o seed
./cmd/seed` — the exact command used throughout this project's own
development sessions — sat untracked in the repo root the entire time,
one broad `git add .`/`git add -A` away from being committed by accident.
Added `.gitignore` covering the binary (all its build-output name
variants), `coverage.out`, `dist/`/`build/`, and standard Go/editor noise
— verified with `git check-ignore` that it catches `/seed` at the repo
root while correctly *not* matching `internal/dist/` (a real package
directory, not a build artifact, despite the similar name).
