# SDL3 platform and game facilities

This document covers the remaining Phase 7 checklist items closed in this
pass, scoped explicitly to what this project's actual toolchain supports:
desktop Linux/Windows/macOS via SDL3 + xmake, built and verified against
the pinned GameAK revision (`docs/gameak-mapping.md`). Mobile and web
targets need SDKs/toolchains this environment has none of — every gap
below says so plainly rather than describing an untested design as done.

Every facility here was verified the same way as Phase 7's first pass: by
actually scaffolding and building all three project modes (`headless`,
`2d`, `3d`) with `xmake` against the pinned GameAK revision and real SDL3
packages (`tests/project_mode_test.go`'s `TestScaffoldModesCompile`,
opt-in via `SEED_TEST_GAMEAK_COMPILE=1`), then smoke-running each binary.
One real bug was found and fixed this way — see §4.

## 1. Input: normalized keyboard/mouse/controller with action mapping

`seed-input.hpp.tmpl` defines `Action` (a project-extensible enum),
`ActionMap` (up to 3 key/button aliases per action — e.g. WASD *and*
arrow keys bound to the same `MoveLeft`/`MoveRight`/etc.), `InputState`
(a per-frame snapshot with `down`/`pressed`/`released` queries), and
`InputTranslator` (the single place `SDL_Event` becomes `InputState`,
per `docs/runtime-architecture.md` §3's bucketing decision — it never
handles window/lifecycle events itself, only records `quit_requested`).

`main.cpp.tmpl` uses this for both graphical modes: `Action::Pause` (bound
to Escape) quits; the 3D demo's speed/radius controls, previously a raw
`SDL_EVENT_KEY_DOWN` switch mutating floats directly, now read
`input.down(Action::MoveLeft)` etc.

**Gap, stated plainly:** touch and sensor input are not implemented.
This is a desktop scaffold with no touch digitizer or motion sensor to
normalize input from — adding `InputState` fields for capabilities the
target can never satisfy would misrepresent what's implemented rather
than document a real gap. A touch/mobile target would need its own
`InputTranslator` extension when one actually exists to build against.

## 2. Rendering lifecycle, cameras, and backend modularity

The lifecycle already implemented, end to end: `Context` (owns `Window`)
→ `Renderer::create(window)` (device/swapchain setup, `Renderer2D` via
`SDL_Renderer`, `Renderer3D` via `SDL_GPUDevice`) → per-frame
`begin_frame`/`draw_quad`/`end_frame` (3D) or `clear`/`present` (2D).
Backend modularity is structural, not aspirational: `using Renderer =
seed::Renderer2D` or `seed::Renderer3D` is the *only* place a project
commits to a graphics API path — no Seed model (`Component`, `System`,
etc.) ever references a renderer type, matching "avoid coupling game
models to a single SDL GPU or graphics API path" directly.

**Camera**, added this pass (`seed-camera.hpp.tmpl`): `Camera2D` is a
CPU-side pan/zoom transform, applied once per frame during extraction
(`docs/runtime-architecture.md` §4), before quads reach either Renderer.

**Gap, stated plainly — materials, render passes, and a true 3D camera:**
`Renderer3D`'s pipeline is a single fixed, precompiled SPIR-V shader
(`seed-shaders.hpp.tmpl` — no GLSL source is checked in, only compiled
bytecode) with no uniform buffer input, and it only ever draws
screen-space quads, never 3D meshes with real depth. `Camera2D` is
therefore an honest fit for what both renderers actually do today — a
CPU transform on the same value structs `draw_quad` already takes — not
a GPU view/projection matrix, because there is no 3D content or shader
uniform for one to feed. A real materials/render-pass/perspective-camera
system needs GLSL source + a shader recompile step this project doesn't
have yet (verified `glslc`/`glslangValidator`/`spirv-as` are available on
this machine, so it's a feasible follow-up, just a separately-scoped one
given the risk of hand-editing SPIR-V bytecode or rebuilding the pipeline
without a way to visually verify correctness in this environment).

## 3. Audio: buses, playback, streaming

`seed-audio.hpp.tmpl`'s `Audio` now has named `Bus` gains (`Master`,
`Music`, `SFX` — composed multiplicatively, `enqueue(bus, ...)` applies
`bus_gain * master_gain` before submission). `AudioStreamSource` submits a
large, already-decoded sample buffer a fixed-size chunk per frame instead
of one large call — bounded, predictable per-frame work, which is what
"streaming" means for the frame hot path.

**Gap, stated plainly:** there is no audio *decoder* — SDL3 alone doesn't
decode compressed formats (mp3/ogg/etc.), and this project vendors no
codec library. `AudioStreamSource` streams already-decoded float PCM;
decoding a file into that buffer in the first place needs a codec
dependency this pass didn't add. Spatial audio (positional attenuation,
panning) is likewise not implemented — nothing in the current Component
schema associates a listener/emitter position with an audio source to
attenuate against.

## 4. Asset pipeline: identity, loading, caching, unloading, dev-only hot reload

`AssetEntry` now carries its source `Path` alongside `Kind`/name, which is
what makes `AssetManager::reload_stale(renderer)` possible: every loaded
asset's file mtime is checked (`SDL_GetPathInfo`) and reloaded if it
changed on disk — a real, live-in-the-running-process hot reload, not a
restart-and-recheck approximation. `unload(name)` removes and destroys a
single named asset (previously only `clear_all()` existed). Identity is
the stable string `name` every entry is addressed by — never a raw
pointer or load order — so code holding a name can always re-resolve the
current handle across a reload.

**A real, pre-existing bug found and fixed by this pass's compile
verification:** `AssetManager::init()`/`quit()` called `IMG_Init`/
`IMG_INIT_PNG`/`IMG_INIT_JPG`/`IMG_Quit` — none of which exist in SDL3_image
(unlike SDL2_image, each `IMG_Load*` call now initializes only the codec
it needs). This had never actually been compiled before this pass, since
nothing in `main.cpp.tmpl` constructed an `AssetManager` prior to this
work. Fixed to only call `TTF_Init`/`TTF_Quit` (the SDL3_ttf API, which
does still need explicit init).

**Dev-only, not release-affecting:** `reload_stale` and its mtime bookkeeping
are compiled out entirely under `#ifndef NDEBUG` (in both the class itself
and every call site in `main.cpp.tmpl`) — a release binary doesn't merely
skip the check at runtime, it doesn't contain the code at all, so it
cannot affect release timing or behavior. This is what "hot reload ...
without changing release runtime determinism" requires: a runtime flag
that disables hot reload would still leave the `stat()`-per-frame cost's
*code path* in the release binary; `NDEBUG` removes it entirely.

**Gap, stated plainly — discovery, dependency graphs, packaging:**
Discovery/import already exist at the model layer (`seed import`,
`internal/commands/import.go`) before generation; re-implementing
filesystem scanning inside `AssetManager` would duplicate that layer, not
complement it. A dependency graph (asset A requires asset B) has nothing
to represent yet — Seed's `asset` model is `Kind` + `Path` only, no
inter-asset references. Packaging (cooking assets into a release bundle)
is explicitly a Phase 8, target-specific concern (platform-aware asset
cooking is already listed there) and needs target-matrix/compression
decisions this class has no basis to make alone.

Auto-generating `register_<Name>(AssetManager&, ...)` calls into
`bootstrap.cpp` (the way components/systems/state machines already are)
was investigated and deliberately not done: `asset.hpp.tmpl`'s generated
signature differs by `Kind` (`texture` needs an `SDL_Renderer*`, `font`
doesn't), and `Renderer3D` has no `SDL_Renderer*` to give it at all (it's
an `SDL_GPUDevice` wrapper) — the existing `collectRegistrations`/
`injectBootstrap` machinery in `internal/generators/engine.go` was built
for uniform `register_X(rt)` calls and would need real design work to
support heterogeneous signatures correctly, not a quick extension. Left
as a documented, concrete follow-up rather than forced through.

## 5. Fonts, images, textures integrated into one typed pipeline

Already true structurally before this pass (`AssetManager::load_texture`/
`load_font`, both cached by name) and unchanged in shape here — this
pass's contribution is making the pipeline's lifecycle (init → load →
hot-reload → unload → quit) actually get *called* from `main.cpp.tmpl`
for the first time (see §4's bug it surfaced) and adding the missing
unload/identity-tracking pieces. "Models" (3D meshes/materials/shaders as
first-class asset kinds) are not integrated — there is no mesh loader or
material system at all (§2's gap), so there is nothing yet for a `model`
asset kind to feed.

## 6. Scene/world loading, decoupled from GameAK storage

`seed-scene.hpp.tmpl`'s `Scene` holds `Spawn`/`Despawn` callbacks — never
a `gameak::core::Identity`, `DataBlock`, or other live simulation handle —
so a Scene can be described entirely without a live `Runtime`. The 3D
demo's orbiter-spawning loop, previously inline in `main()`, is now
`MainScene`'s `on_enter` callback, entered once via
`SceneManager::transition_to`. This mirrors §4's extraction-step pattern:
declarative intent kept separate from the storage it eventually populates.

**Gap, stated plainly:** there is no `scene` YAML model type — Scene is
hand-authored C++ today, not something `seed sync` generates. Adding one
that declares which Archetypes/Systems/StateMachines belong to a scene is
reasonable future work once a project has more than one Scene and the
duplication of hand-written spawn callbacks becomes a real cost; not
designed speculatively ahead of that need (consistent with how earlier
phases treated schema additions).

## 7. Save data, preferences, localization, clipboard, dialogs, URLs

`seed-platform-services.hpp.tmpl`, all thin, direct wrappers over real
SDL3 APIs (verified present in the installed SDL3 headers before writing
against them): `SaveData` (via `SDL_GetPrefPath` + `SDL_LoadFile`/
`SDL_SaveFile`), `Preferences` (a flat key=value file under `SaveData`'s
directory — deliberately not a general serialization format, since
`SaveData`'s raw byte API already covers structured needs), `Localization`
(`SDL_GetPreferredLocales` picks which flat string-table file to load —
real OS-level locale detection, not guessed from environment variables),
and `platform::{set_clipboard_text, get_clipboard_text, open_url,
show_open_file_dialog}` over `SDL_SetClipboardText`/`SDL_GetClipboardText`/
`SDL_OpenURL`/`SDL_ShowOpenFileDialog`. Each is capability-gated by SDL
itself (a target lacking the facility gets a `false`/empty result, not a
crash), so Seed doesn't reinvent capability detection for them.

**Gap, stated plainly:** no localization pluralization/formatting (a flat
string table only); dialogs are async by SDL3's own design and this
wrapper doesn't add a synchronous facade over that.

## 8. Vertical slice

The 3D mode's scaffolded demo is the actual vertical slice, and it is
compile- and smoke-run-verified end to end: `Context` (SDL3) owns
`Window`/`Audio` and requests `Bus` gains; `AssetManager` runs its full
init/hot-reload/quit lifecycle; `ActionMap`/`InputTranslator` drive
gameplay parameters from real input; a `Scene` spawns entities into a real
`gameak::runtime::Runtime<>` via `Command`s; the fixed-step loop ticks
that Runtime deterministically; `extract_render_list` reads the resulting
block state through a `Camera2D` transform into plain `QuadDesc` values;
`Renderer3D` draws them. Every piece from this document is present and
exercised in the same binary — verified compiling and linking against the
pinned GameAK revision, and running for several seconds without a crash
(`tests/project_mode_test.go`).

**What "vertical slice" does not mean here:** no audio is actually
*played* (bus gains are configured but the demo has no bundled sample
data to enqueue — §3's decoder gap), and there is no visual confirmation
this environment could produce (no display server to screenshot against —
smoke-running under a timeout with no crash and no error output is the
verification available here, consistent with how Phase 7's first pass
validated the 3D binary).
