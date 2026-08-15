# Seed Runtime Architecture

This document defines the target runtime architecture per the Phase 4
roadmap checklist. It is a design specification, not a description of
already-built code: the only runtime code that exists today lives in
per-project scaffold templates (`internal/project/templates/*.tmpl`),
generated once into a new project and then hand-edited by the game author.
There is no shared Seed runtime library yet — that is exactly what Phase 4
through Phase 7 build toward.

Where current template code contradicts a decision made here, that is
called out explicitly as a **known gap** — concrete, file-and-line grounded,
not speculative — for Phase 7 to fix when the runtime is actually built.

## 1. Runtime layers and ownership boundaries

| Layer | Owns | Current scaffold equivalent |
|---|---|---|
| **Platform** | SDL3 subsystem init, window, input devices, audio device, wall clock | `seed::Context` (`seed-context.hpp.tmpl`), `seed::Window` (`seed-window.hpp.tmpl`), `seed::Audio` (`seed-audio.hpp.tmpl`), raw `SDL_PollEvent`/`SDL_GetPerformanceCounter` calls inline in `main()` |
| **Application** | Process lifecycle state, the composition root, the frame loop | `main()` in `main.cpp.tmpl` — today an unstructured `int main()`, not a distinct owned object |
| **Simulation** | GameAK `Runtime<>`, Data Blocks, Systems, FSMs, deterministic tick | `gameak::runtime::Runtime<>` constructed in `main()` |
| **Presentation** | Render/audio work derived *from* simulation state, never storage itself | `seed::Renderer2D`/`Renderer3D` (`seed-renderer_*.hpp.tmpl`) |
| **Assets** | Asset identity, loading, caching | `seed::AssetManager` (`seed-asset-manager.hpp.tmpl`), `seed::Image`/`Font`/`Texture` |
| **Game code** | User Systems (`.cpp`), scene-specific glue | `Src/Game/*.cpp`, currently also most of `main()` itself |

**Ownership rule:** each layer owns only the resources listed above; a
layer may *read* another layer's public interface but never reaches into
another layer's internal storage. §4 below is the current, concrete
violation of this rule (Presentation reading Simulation's block storage
directly) that motivates writing the rule down now.

**Known gap:** `main()` today mixes Application-layer responsibilities
(lifecycle, the loop) with Game-code responsibilities (spawning demo
entities, per-frame orbit math) because there is no Application-layer type
to own the loop yet. Phase 7 should extract an `Application`/`Seed::App`
type that owns the loop and delegates all per-frame gameplay logic to
Systems.

## 2. Application lifecycle

```text
process start
  -> Context::create(capabilities)       (Platform init; fatal on failure)
  -> bootstrap                            (register_components, register_controllers)
  -> loading                              (assets, initial scene/archetypes)
  -> main loop                            (§5's fixed-step/variable-rate loop)
  -> suspension                           (platform lifecycle events; not handled today)
  -> shutdown                             (reverse-order RAII teardown)
  -> failure recovery                     (see below)
```

- **Startup → bootstrap → loading → main loop** matches the existing
  `main.cpp.tmpl` structure: `Context::create` → `Window::create` →
  `Audio::create` → `Renderer::create` → `register_components`/
  `register_controllers` → the `while (running)` loop. This ordering is
  the decision: platform services before the GameAK `Runtime<>`, the
  `Runtime<>` before any registration call, all registration before the
  loop starts.
- **Suspension** (OS-level backgrounding, minimize, mobile lifecycle
  events) is not handled anywhere today — there is no `SDL_EVENT_DID_ENTER_
  BACKGROUND`/`WILL_ENTER_FOREGROUND` handling in `main.cpp.tmpl`'s event
  switch. **Decision:** suspension pauses the fixed-step accumulator (§5)
  without ticking simulation, keeps the window/audio device alive, and
  resumes accumulation from zero on foreground — never replays elapsed
  suspended time as simulation steps. Implementation is a Phase 7 item.
- **Shutdown** already works correctly on the happy path: every platform
  wrapper (`Window`, `Audio`, `Context`) is RAII, so falling off the end of
  `main()` unwinds them in reverse construction order with no explicit
  shutdown code needed. This is the pattern going forward — new
  Platform/Presentation/Assets types must stay RAII rather than growing an
  explicit `shutdown()` method.
- **Failure recovery — decision, not yet implemented:** the current
  template treats every `Result` failure identically (log via `SDL_Log`,
  `return 1`), except `Audio::create`, whose failure is logged as a warning
  and does *not* abort. That's the right pattern to generalize: failures
  are classified per capability (§8) as **fatal** (Video/Window — nothing
  useful can happen without it) or **degraded** (Audio, and later
  Gamepad/Haptic — the game continues without that facility). No other
  layer decides this per-call today; Phase 7 should centralize the
  classification instead of leaving it to ad hoc `if` statements in
  generated `main()` code.

## 3. SDL events → Seed input/window/lifecycle/GameAK events

**Known gap:** `main.cpp.tmpl`'s event loop is a single `switch` inline in
`main()` that either flips local booleans (`running = false`) or mutates
demo-specific state directly (`speed`, `radius` in the 3D template) — SDL
events never reach GameAK at all. There is no path from `SDL_Event` to a
System, an FSM transition, or a Command.

**Decision:** one `SDL_PollEvent` loop per frame, run once by the
Application layer before `Runtime::tick`, classifying every event into
exactly one of four buckets before anything else happens:

1. **Lifecycle events** (`SDL_EVENT_QUIT`, background/foreground, low
   memory) — handled directly by the Application layer's state machine
   (§2); never forwarded to GameAK.
2. **Window events** (resize, focus, display change) — handled by the
   Platform/Presentation layers directly (e.g. swapchain resize); never
   forwarded to GameAK as gameplay data.
3. **Input events** (keyboard, mouse, gamepad, touch) — accumulated into a
   normalized per-frame `InputState` snapshot (not delivered as individual
   raw SDL events to game code), which Systems read during their phase
   (§8 of `docs/semantics.md`). Action mapping (raw key → named action) is
   a Phase 7 facility, not defined here.
4. **Everything else Seed doesn't special-case** is dropped, not silently
   forwarded — an unhandled `SDL_Event` type is not automatically turned
   into a GameAK Event; only the input snapshot above and explicit Command
   emission (Phase 6) enter simulation state.

This bucketing happens once, before `Runtime::tick`, so simulation for a
given frame sees a consistent, already-classified view of "what happened
since last tick" rather than raw platform events interleaved with tick
execution.

## 4. Simulation state → render/audio work

**Known gap, concretely:** the 3D template's render loop calls
`rt.get_block(o.id)` and `block->field<float>(offsetof(Core::Position, x))`
directly inside the drawing code (`main.cpp.tmpl`, render section). The
Renderer reads GameAK block storage and field offsets by hand — exactly the
coupling the roadmap says to avoid ("without coupling GameAK storage
directly to SDL APIs").

**Decision:** introduce an explicit **extraction step** between simulation
and presentation, owned by the Application layer, that runs once per
render frame (not once per fixed simulation step — see §5's interpolation
note):

```text
Runtime::tick (simulation, fixed-step)
        |
        v
extraction (Application layer): query renderable/audible components,
        write into Seed-owned presentation buffers (e.g. a RenderList of
        {transform, material, mesh} value structs — never raw block
        pointers or field offsets)
        |
        v
Renderer/AudioBus consume the presentation buffers only
```

The Renderer and AudioBus types never call into `gameak::runtime::Runtime`
themselves. This also gives interpolation (§5) a natural home: extraction
can read both the previous and current simulation state and blend them
before handing the Renderer a single interpolated buffer, instead of the
Renderer needing any awareness of simulation stepping at all. Implementing
the extraction step and presentation buffer types is a Phase 7 item; this
section fixes the target shape.

## 5. Fixed-step simulation, variable-rate presentation, interpolation, pacing, clock ownership

**Known gap, concretely:** the two scaffold modes are inconsistent with
each other and with the fixed-step principle:

- The 3D template calls `rt.set_fixed_timestep(fixed_dt)` (implying a
  fixed step) but then calls `rt.tick(dt)` with the *real, variable* frame
  delta every frame — the fixed timestep is configured but never actually
  used to drive ticking.
- The 2D template calls `rt.tick(fixed_dt)` every frame with the
  *constant* configured value, ignoring the real elapsed wall time
  entirely — simulation speed is tied to however fast the render loop
  happens to spin, not to real time.

Neither matches a real fixed-step/variable-rate-presentation architecture.

**Decision:** a single accumulator-driven loop, owned by the Application
layer, replaces both patterns:

```text
frame_dt = clamp(now - last, 0, kMaxFrameDt)   // clamp avoids the
                                                 // spiral-of-death after a
                                                 // stall (breakpoint, OS
                                                 // hitch, etc.)
accumulator += frame_dt
while (accumulator >= fixed_dt) {
    previous_state = current_state   // cheap snapshot for interpolation
    rt.tick(fixed_dt)
    accumulator -= fixed_dt
}
alpha = accumulator / fixed_dt
extraction(previous_state, current_state, alpha)   // §4's interpolated read
render(); present();                                // every frame, uncapped
                                                      // relative to simulation
```

- **Fixed-step simulation:** `Runtime::tick` is always called with the
  constant `fixed_dt`, never a measured frame delta. `fixed_dt` itself is
  derived once at startup from the detected display refresh rate (already
  done in `main.cpp.tmpl`'s `detect_refresh_rate()`), clamped to `kMinDt`
  (already present as a constant, just not consistently used).
- **Variable-rate presentation:** rendering happens once per real frame
  regardless of how many (zero, one, or several) fixed steps ran that
  frame — matching what the 2D template's `renderer.clear/present` already
  does unconditionally, just now decoupled from tick count.
- **Interpolation:** presentation always reads a blend of the previous and
  current simulation states at `alpha`, never the raw current state alone
  — this removes the visible stutter a naive fixed-step loop produces when
  the fixed step and frame rate aren't perfectly aligned.
- **Frame pacing:** presentation is paced by the platform (vsync, already
  requested via `SDL_HINT_RENDER_VSYNC` in `main.cpp.tmpl`), not by a
  manual sleep in the Application loop.
- **Clock ownership:** the Application layer owns exactly one wall clock
  (`SDL_GetPerformanceCounter`/`Frequency`, as already used). Neither
  GameAK's `Runtime` nor the Renderer ever reads the clock themselves —
  they only ever receive a `dt` or `alpha` value handed to them. This is
  what makes deterministic replay (Phase 6) possible later: simulation
  never depends on wall-clock reads it doesn't control.

## 6. Thread ownership and synchronization

There is no threading in the current templates — `main()` runs everything
serially on the thread SDL was initialized on, which is also a hard SDL3
constraint (most SDL calls must happen on the thread that called
`SDL_Init`).

**Decision (first milestone, deliberately conservative):** Platform,
Simulation (`Runtime::tick`), Presentation (`Renderer`/`AudioBus` command
submission), and Game code (Systems) all run on the single main thread for
the first playable milestone (Phase 7's target). The only work allowed off
the main thread is:

- **Asset loading** (disk I/O, decoding, atlas baking — `internal/baker`
  already runs synchronously today and is the natural first candidate for
  a background job).
- **Background/CPU-bound work with no simulation-state access** (e.g.
  procedural generation that produces data, not mutations).

Both integrate back into the main thread via a simple job/result queue
drained at a fixed point in the loop (before extraction, §4) — the
`Runtime<>` and any GameAK storage are never accessed from a worker thread.
This is intentionally not a full job-system design: multi-threaded
rendering, simulation sharding, or a general task graph are explicitly out
of scope until a working single-threaded baseline exists and profiling
shows where threading actually pays off.

## 7. Service ownership and dependency injection through `Seed::Context`

**Known gap:** today's `seed::Context` (`seed-context.hpp.tmpl`) is only an
RAII guard around `SDL_Init`/`SDL_Quit` — it holds no services and is
never passed to anything. `Window`, `Audio`, `Renderer`, and the GameAK
`Runtime<>` are all separate local variables in `main()`, wired together
by argument-passing (e.g. `Renderer::create(win_res.value())`), not through
`Context`.

**Decision:** `Context` becomes the composition root: it owns `Window`,
`Audio`, `AssetManager`, the active `Renderer`, and the GameAK `Runtime<>`,
constructed in the fixed order already established in `main.cpp.tmpl`
(Context → Window → Audio → Renderer → Runtime → registration). Systems
and other game code never receive `Context` itself (no ambient global
service locator, matching "no hidden runtime discovery" from the product
principles) — they receive only the specific service references they
declare a need for, passed explicitly at registration time. This keeps
dependency injection **explicit per-consumer** rather than a global
lookup, and is a Phase 7 implementation item against `seed-context.hpp.tmpl`
and the generated `bootstrap.cpp`.

## 8. Capability-based APIs

**Known gap:** `main.cpp.tmpl` unconditionally requests
`SDL_INIT_VIDEO | SDL_INIT_AUDIO | SDL_INIT_GAMEPAD | SDL_INIT_HAPTIC |
SDL_INIT_SENSOR` and unconditionally constructs a `Window` and `Renderer`
(fatal if either fails); only `Audio` failure is tolerated. There is no way
to build a project that doesn't want a window at all (§10), and capability
absence is discovered by *attempting* construction and checking the
`Result`, not by querying availability up front.

**Decision:** define a `Capability` enum (`Video`, `Audio`, `Gamepad`,
`Haptic`, `Sensor`, ... extensible) that:

- `Context::create` accepts as a set, translating directly to the
  `SDL_InitFlags` bitmask already used today (the mapping is mechanical —
  no behavior change to how `SDL_Init` is called).
- Every Platform/Presentation service (`Window`, `Renderer`, `Audio`,
  `AssetManager`) declares the capabilities it requires and is skipped
  during construction — cleanly, not attempted-then-logged — when the
  project's target configuration (§10) doesn't request that capability.
  This replaces "attempt and warn on failure" (today's Audio pattern) with
  "don't attempt what wasn't requested" as the primary mechanism;
  attempt-and-degrade remains the fallback for a *requested* capability
  that turns out to be unavailable on the running machine (e.g. no audio
  device physically present).
- Systems/game code can query which capabilities are active (`Context`
  exposes a read-only capability set) to skip capability-specific logic
  without `#ifdef`s scattered through game code.

## 9. Error propagation, logging, crash context, orderly shutdown

- **Error propagation:** the existing `gameak::core::Result<T>`/`Error`
  pattern, checked explicitly at each call site, is the established
  convention (every `*_res` check in `main.cpp.tmpl`) and is kept as-is —
  no exceptions for expected failure paths (init failures, missing
  assets), consistent with "no hidden runtime magic."
- **Logging — decision, not yet implemented:** today every failure path
  calls raw `SDL_Log` with an ad hoc message. There is no shared logging
  facility carrying structured context. **Decision:** introduce a thin
  `Seed::Log` wrapper over `SDL_Log` that tags every message with the
  originating layer (Platform/Application/Simulation/Presentation/Assets)
  and, once Cardinal (Phase 6) exists, the current tick/system name — so a
  log line is traceable to *what was executing* without a debugger
  attached. Message content and log levels otherwise follow SDL3's
  existing categories.
- **Crash context:** unhandled exceptions from user System code are not
  caught anywhere today (none of the templates wrap `rt.tick()` in
  `try`/`catch`). **Decision:** the Application layer's tick call is the
  single place allowed to catch an exception escaping game code; on catch,
  it logs the current system/tick (once that context exists, per above)
  and performs the same orderly shutdown path as a fatal init failure —
  Seed does not attempt to resume simulation after an uncaught exception,
  since Runtime state at that point cannot be trusted as consistent with
  the "atomic Command application" invariant Phase 6 introduces.
- **Orderly shutdown:** already correct on the happy path via RAII (§2).
  The decision here is to keep it that way — new Platform/Presentation
  types added for capabilities (§8) must be RAII-wrapped the same way
  `Window`/`Audio`/`Context` already are, never given an explicit
  `shutdown()`/`close()` method that callers must remember to invoke.

## 10. Headless/server lifecycle as first-class

**Known gap:** there is no headless mode. `main.cpp.tmpl` always
constructs a `Window` and a `Renderer`; there is no project mode that
skips them. `seed new` currently only distinguishes `--mode 2d`/`3d`
(both graphical).

**Decision:** headless is a third project mode (`--mode headless`,
alongside `2d`/`3d`), not a degraded special case of the graphical client:

- `Context::create` is called with a capability set (§8) that omits
  `Video` (and `Audio`, unless explicitly requested) — no `Window`,
  `Renderer`, or `Audio` construction is attempted at all, rather than
  attempted and discarded.
- The Application lifecycle (§2) and the fixed-step accumulator loop (§5)
  are **identical** to the graphical client — headless mode drives
  `Runtime::tick` off the same wall clock (for a live server) or a
  deterministic external tick source (for tests/CI/replay, once Phase 6's
  `seed replay` exists), and simply has no extraction/render/audio stage
  to run each frame.
- No SDL window/input events exist to poll in this mode; only lifecycle
  events relevant to a server process (signals for shutdown, e.g.
  `SIGINT`/`SIGTERM`) are handled, mapped into the same Application
  lifecycle state machine used by §2, not a separate one.

This keeps headless a configuration of one lifecycle rather than a fork of
the codebase — the concrete Phase 7 deliverable is a `main.cpp.tmpl`
variant (or a shared loop body parameterized by capability set) generated
from `--mode headless`.

## Summary of decisions requiring Phase 7 implementation work

Every section above states a decision; the following are the concrete,
file-grounded fixes Phase 7 needs to make against the current scaffold
templates, in rough priority order (later items depend on earlier ones):

1. Fix the fixed-step/variable-dt inconsistency in `main.cpp.tmpl` (§5) —
   the most immediately visible bug, affecting both existing modes.
2. Extract an Application-layer loop type instead of inline `main()` logic
   (§1, §2).
3. Add the extraction step so Presentation stops reading GameAK block
   storage directly (§4).
4. Route SDL input events through a per-frame `InputState` snapshot
   instead of ad hoc local-variable mutation (§3).
5. Turn `Context` into the composition root (§7) and add the `Capability`
   set (§8), including the `--mode headless` project variant (§10).
6. Add `Seed::Log` and the tick-boundary exception boundary (§9).
7. Add the background-job queue for asset loading (§6).
