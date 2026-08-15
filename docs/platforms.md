# Platforms, target matrix, and distribution

This document defines Seed's Phase 8 target matrix and records what was
actually verified versus planned-but-untestable in this environment,
which has a Linux toolchain and no mobile/web SDKs. Consistent with
`docs/facilities.md`'s pattern: every claim below is either something this
pass actually built and ran, or is explicitly marked as a plan, not a
result.

## 1. Target matrix and minimum versions

| Target | Status | Toolchain | Minimum versions |
|---|---|---|---|
| Linux (x86_64) | **Verified.** Scaffolded, built, and smoke-run in this pass and Phase 7's (`tests/project_mode_test.go`). | GCC or Clang, xmake | C++20 compiler (GCC 11+/Clang 14+ for full C++20 support), xmake 2.8+, SDL3 3.x |
| Windows (x86_64) | **Planned, not verified here.** No Windows toolchain in this environment. | MSVC, xmake | Same C++20/xmake/SDL3 floor; MSVC 19.29+ (VS 2019 16.10+) for C++20 |
| macOS (x86_64/arm64) | **Planned, not verified here.** No macOS toolchain in this environment. | Clang (Xcode), xmake | Same floor; Xcode 14+ |
| Android | **Not planned in detail — see §4.** | NDK, xmake's android plat | n/a |
| iOS | **Not planned in detail — see §4.** | Xcode, xmake's iphoneos plat | n/a |
| Web (Emscripten) | **Not planned in detail — see §4.** | Emscripten, xmake's wasm plat | n/a |

This project's own toolchain versions, for reference (what generated this
matrix's Linux row was actually verified against): Go `1.26.4`
(`go.mod`), xmake `3.1.0`, `set_languages("c++20")` in every generated
`xmake.lua`.

## 2. Declarative platform capabilities in generated XMake projects

`xmake.lua.tmpl` and `gameak-xmake.lua.tmpl` now branch on `is_plat(...)`
and scope toolchain-specific flags by `tools = {...}` instead of applying
them unconditionally. This is not a cosmetic change — it fixed a real
cross-platform bug found while writing this section:
`add_cxxflags("-Wno-interference-size")` was applied unconditionally in
both files. That flag silences a GCC/Clang-only warning
(`std::hardware_destructive_interference_size`'s ABI notice) and **does
not exist in MSVC** — a Windows build using the previous, unconditional
form would have failed to configure at all. Both files now scope it with
`{tools = {"clang", "gcc"}}`.

`xmake.lua.tmpl` also now:
- Declares debug/release `symbols`/`optimize`/`strip` explicitly (item 5's
  "debug symbols and release profiles consistently" — previously implicit
  in xmake's `mode.debug`/`mode.release` rule defaults, now stated so a
  reader doesn't have to know those defaults to know what a project ships).
- Branches per `is_plat`: Windows gets `/utf-8` and Unicode CRT defines
  (MSVC doesn't default to UTF-8 source/exec charset or wide entry
  points, unlike GCC/Clang); macOS and Linux branches exist for symmetry
  and as the obvious place for a future platform-specific need, with an
  honest "nothing needed today, verified by our own compile tests" note
  on the Linux branch rather than a speculative flag.

## 3. Desktop targets first: reproducibility

**Linux is verified reproducible** in the sense this project can actually
check: `tests/project_mode_test.go`'s `TestScaffoldModesCompile` scaffolds
fresh `headless`/`2d`/`3d` projects and builds each with `xmake` against
the pinned GameAK revision (`docs/gameak-mapping.md`) and real SDL3
packages, from a clean `xmake.lua` every time — no manual setup step
outside what `seed new` itself does. Run repeatedly while writing this
phase; passed every time.

**Windows and macOS reproducibility is not verified locally** — there is
no toolchain for either here. §8's CI workflow is the verification
mechanism for both: it runs the identical `seed new` → `xmake build`
sequence on GitHub-hosted `windows-latest` and `macos-latest` runners.
This document does not claim those builds succeed before CI has actually
run them; the workflow exists so that claim becomes checkable.

## 4. Android, iOS, and web: plan, not implementation

**What's real:** xmake has built-in platform support for `android`,
`iphoneos`, and `wasm` (via Emscripten) targets, and SDL3 itself supports
all three (it's SDL3's primary reason for existing as a portable
abstraction). GameAK is portable C++20 with no OS-specific code observed
in the pinned revision's `Include`/`Src` (Phase 5's mapping work only
found GameAK depending on standard C++ and its own `Core`/`Runtime`
headers) — nothing found suggests GameAK itself is desktop-only.

**What's not done, and why it's not claimed done:** this environment has
no Android NDK, no Xcode/iOS SDK, and no Emscripten toolchain installed,
so none of the following could be attempted, let alone verified:

- Whether SDL3's GPU backend (`Renderer3D`'s `SDL_GPUDevice` path) has a
  working Vulkan/Metal/WebGPU backend on each target — this is the single
  biggest unknown, since `Renderer3D` is the only renderer using SDL's GPU
  API at all (`Renderer2D`'s `SDL_Renderer` path is more likely to "just
  work" via SDL's own backend abstraction, but that's also unverified
  here).
- Whether xmake's package manager (`add_requires("libsdl3", ...)`) has
  working prebuilt or buildable SDL3/SDL3_image/SDL3_ttf packages for
  each target — Phase 7/8's desktop verification relied on exactly this
  mechanism succeeding, and it's untested cross-platform.
- Touch input (already noted as unimplemented in `docs/facilities.md` §1)
  would need to exist before a touch-primary target like iOS/Android is
  meaningfully playable.
- Web's `main()`-blocking-loop model doesn't work under Emscripten without
  restructuring to `emscripten_set_main_loop` — `main.cpp.tmpl`'s
  fixed-step accumulator loop (Phase 7) would need a real rewrite, not a
  new `is_plat` branch, to run under wasm at all.

Claiming any of these validated without the toolchain to check them would
be exactly the kind of unverified claim this project has avoided making
in every prior phase; they're recorded here as the concrete first
questions a future pass with the right toolchain needs to answer, not as
already-answered.

## 5. Native dependencies, architecture selection, debug symbols, release profiles

- **Native dependencies:** managed by xmake's package manager
  (`add_requires`), already consistent across targets in the generated
  `xmake.lua` — the same `libsdl3`/`libsdl3_image`/`libsdl3_ttf` package
  names resolve per-platform through xmake's `xmake-repo`, GameAK is
  vendored source built the same way everywhere via
  `includes("Third-Party/GameAK")`.
- **Architecture selection:** xmake's `-a`/`--arch` flag already covers
  this (e.g. `xmake f -a arm64`); nothing project-specific was needed or
  added, since neither `xmake.lua.tmpl` nor `gameak-xmake.lua.tmpl`
  hardcodes an architecture anywhere.
- **Debug symbols / release profiles:** §2's explicit `is_mode("debug")`
  branch.

## 6. Platform-aware asset cooking, packaging, compression, manifests

`seed package` (`internal/commands/package.go`, backed by `internal/dist`)
builds a `manifest.json` (every declared Asset's name, kind, path, size,
and SHA-256 checksum — computed from the IR, so it only ever sees
resolved, validated data, never raw YAML) and a zip bundle containing it
plus every referenced asset file, labeled with a `--platform` string.

**What "cooking"/"compression" means here, precisely, and what it
doesn't:** the bundle is a zip archive (real compression of the archive
as a whole) containing assets verbatim — there is no per-target asset
*transform* (texture format conversion, mip generation, audio
transcoding). Nothing in Seed's `asset` model (`Kind` + `Path`) carries
per-platform variants or transform hints yet, so there's nothing for a
cooking step to select between. `dist.BuildManifest`/`WriteBundle` are
structured so a future cooking step has an obvious place to plug in
(transform each asset before adding it to the archive, record the
transformed size/checksum) without changing the manifest shape.

Verified: `tests/dist_test.go` and `tests/commands_test.go`'s `package`
subtests build a real manifest, hash real file content, and open the
resulting zip back up to confirm both `manifest.json` and the asset file
are present and correct — not just "the command exited 0."

## 7. Application metadata, icons, permissions, signing, distributable bundles

`dist.AppMetadata` (name/version/identifier) loads from an optional
`app.yaml` at the project root, defaulting sensibly when absent — this is
what a packaged bundle's manifest and any future platform bundle
descriptor (an Android manifest, an iOS Info.plist) would read from.

**Deliberately not implemented, stated plainly:**

- **Icons:** need actual image content this tool has no way to generate;
  `AppMetadata` has no icon field so nothing implies one is wired up.
- **Permissions:** no permission model exists in Seed's schema (nothing
  a Component/System declares maps to "needs camera access" or similar);
  inventing one without a concrete platform target to validate against
  would be speculative.
- **Signing:** explicitly the thing this checklist item warns against
  automating carelessly ("without embedding credentials in Seed
  projects") — a real signing hook needs a platform-specific credential
  (an Apple Developer certificate, a Windows Authenticode cert) that must
  live outside the repository entirely (a CI secret, a local keychain),
  which this pass has no mechanism to provide or verify. Left undone
  rather than stubbed with a hook nothing actually calls.
- **Distributable bundles** (a `.app`, an `.msix`, an APK): `seed package`
  produces an asset bundle, not a platform application package — building
  one of those needs the actual per-platform build (§4's unresolved
  questions) to exist first.

## 8. Cross-platform CI

`.github/workflows/ci.yml` (added this pass): on every push/PR, runs
`go build`/`go vet`/`go test ./...` (the fast, always-on Go suite this
project already relies on), then a matrix job on `ubuntu-latest`,
`windows-latest`, and `macos-latest` that installs xmake, runs `seed new`
for all three modes, and builds each with `xmake build -y` against the
pinned GameAK revision and real SDL3 packages — the exact same sequence
`tests/project_mode_test.go`'s opt-in local test runs, now running for
real on all three OSes instead of only Linux (which is all this
environment could execute directly). This workflow is the actual
verification mechanism for §1/§3's Windows/macOS rows — it was written
and reviewed for correctness but, being a GitHub Actions workflow, its
first real run happens on GitHub's infrastructure once pushed, not in
this session.

## 9. Platform capability differences and graceful fallback

This is already specified and partly implemented, not new to this
document: `seed::Capability` (`docs/runtime-architecture.md` §8,
implemented in `seed-context.hpp.tmpl`) is the mechanism a target's
missing facility is expressed through — `--mode headless` (Phase 7) is
the concrete example already built: it requests zero SDL capabilities and
`writeProjectFiles` skips scaffolding the image/font/asset-manager/
renderer stack entirely for it, rather than attempting and catching a
failure. `docs/facilities.md` documents each facility's own fallback
behavior (Audio unavailability is a logged warning, not fatal; SDL's own
clipboard/URL/dialog wrappers already degrade to a `false`/empty result
on a target lacking the facility). This section exists to point at that
existing material rather than duplicate it — there is no
platform-capability behavior introduced by Phase 8 itself beyond what
Phase 7 already specified and built.
