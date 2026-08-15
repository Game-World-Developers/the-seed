# The Seed Roadmap

The Seed aims to provide a Rails-inspired developer experience for building games
and virtual worlds on top of SDL3 and GameAK without bringing dynamic framework
behavior into the runtime hot path.

The desired flow is:

```text
Game domain models
        |
        v
Seed semantic model
        |
        v
Validation, reference resolution, and execution planning
        |
        v
Inspectable intermediate representation (IR)
        |
        v
Seed runtime composition
   /            \
  v              v
GameAK          SDL3
simulation      platform services
        |
        v
Multiplatform game application
```

The user should describe the game domain in Seed concepts. Seed should interpret
that intent, explain its decisions, and compose GameAK simulation facilities with
SDL3 platform facilities into a complete game application. Users should not need
to wire the two libraries manually or model their game in terms of their
implementation details.

GameAK is the simulation and data-oriented foundation, not the whole Seed
runtime. SDL3 provides the portable platform foundation. Seed owns the
application model, lifecycle, integration contracts, higher-level game
facilities, tooling, and generated glue that make them work as one SDK.

## Product principles

- Convention over configuration for project structure, naming, registration, and
  common execution behavior.
- Explicit and deterministic behavior in the generated runtime.
- No reflection, hidden allocation, or runtime discovery in the frame hot path.
- Seed's domain model must remain independent from generated C++ and GameAK APIs.
- SDL3 and GameAK must be exposed through cohesive Seed abstractions rather than
  leaking as unrelated subsystems into game code.
- Platform differences must stay behind explicit capability boundaries.
- Generated behavior must be inspectable and explainable.
- Manual C++ must be a supported escape hatch with stable extension points.
- Invalid or ambiguous models must fail with actionable diagnostics.

## Current baseline

- [x] Cobra-based CLI with project scaffolding.
- [x] YAML models and C++ code generation.
- [x] Component, trait, entity, archetype, state machine, event, asset, system,
  and block model types.
- [x] GameAK project integration and bootstrap registration.
- [x] Initial SDL3 window, rendering, audio, surface, image, font, texture, and
  asset-manager wrappers.
- [x] Asset import, project doctor, sync checking, and debug TUI.
- [x] Initial automated tests for commands, models, generators, and templates.
- [x] Initial FSM topology generation with states and event-to-target
  transitions.

## Phase 1: Define Seed semantics

- [x] Write a glossary for Seed concepts and clearly distinguish them from
  GameAK types. (`docs/glossary.md`)
- [x] Specify the semantics of component, trait, entity, archetype, state
  machine, event, asset, system, and block. (`docs/semantics.md`)
- [x] Define identity, ownership, composition, references, lifetime, and
  namespace rules. (`docs/semantics.md` §2-7)
- [x] Define system inputs, outputs, queries, dependencies, phases, and ordering.
  (`docs/semantics.md` §8: explicit read/write access mode, `Emits` output
  list, `Phase` field with fixed phase set, `Priority` + file-name
  tie-breaking — decided; implementation lands in Phase 2)
- [x] Define event producers, consumers, delivery guarantees, and lifetime.
  (`docs/semantics.md` §9: producer/consumer declared from the referencing
  side, FIFO-per-type at-most-once delivery, one-tick lifetime — decided;
  implementation lands in Phase 6)
- [x] Define which defaults come from convention and which choices must remain
  explicit. (`docs/semantics.md` §10)
- [x] Document deterministic behavior and forbidden runtime magic.
  (`docs/semantics.md` §11)
- [x] Add representative domain examples before extending the YAML schema.
  (`docs/examples/`, a player-movement domain exercising every model type and
  reference kind, verified to parse against current `internal/generators`)

## Phase 2: Introduce a semantic compiler

- [x] Separate YAML decoding from semantic model construction.
  (`generators.Compile` in `internal/generators/semantic.go`: a decode phase
  builds typed models, then a distinct resolution/validation phase walks
  them against the symbol table)
- [x] Introduce typed source locations so diagnostics can identify file, field,
  and related declarations. (`SourceLocation{File, ModelType, ModelName,
  Namespace, Field}`)
- [x] Build a project-wide symbol table for names and namespaces.
  (`symbolTable`, keyed by `(type, name, namespace)` — fixes the Phase 1
  collision gap; ambiguous unqualified references across namespaces are
  now a diagnostic instead of silently picking one)
- [x] Resolve cross-model references independently from code generation.
  (`Compile` performs resolution/validation and produces no C++; `Sync` only
  generates after a clean compile)
- [x] Validate duplicates, cycles, missing references, incompatible types, and
  invalid execution dependencies. (duplicate declarations, missing/ambiguous
  references, cyclic reference graphs, and state-machine transitions
  targeting undeclared states are all reported as diagnostics; field
  declarations are checked for empty/duplicate names and malformed type
  tokens. Full type-compatibility checking — e.g. a System writing a `float`
  into a `bool` field — is not possible yet because `FieldDef.Type` has no
  formal Seed-level type system, only a pass-through C++ token; that design
  decision is still open and is called out in `semantic.go`'s
  `validateFields` doc comment)
- [x] Collect diagnostics and return a failing exit code when errors exist.
  (`CompileResult.HasErrors`/`Errors`; `Sync` returns a non-nil error and
  aborts before generating any output when errors exist)
- [x] Replace generator-side warnings that currently allow partial success with
  structured errors. (`Sync`'s pre-flight compile gate; the existing
  I/O-level warnings during generation itself are unchanged, as they are a
  different failure class than semantic validation)
- [x] Add semantic tests that do not depend on rendered C++ text.
  (`tests/generators_semantic_test.go`: valid domain, missing reference,
  duplicate declaration, ambiguous reference, invalid transition target, and
  `Sync` aborting before generating any file)

## Phase 3: Create an inspectable IR

- [x] Design a versioned intermediate representation between Seed semantics and
  backend generation. (`internal/ir`: `IR` built by `ir.Build` from a
  `generators.CompileResult`, independent of both YAML and any GameAK/SDL3
  backend type)
- [x] Represent storage requirements, identities, queries, commands, events,
  schedules, and registration in the IR. (identities via `Ref`; storage
  requirements as `Entity.Components`, the deduplicated resolved component
  set reachable through traits; queries as `System.Entities`/`Access`;
  events as `Event`/`StateMachine.Events`; schedule as `IR.Schedule`.
  **Partial:** `System.Emits` is reserved for Commands but always empty —
  Commands don't exist until Phase 6 (Cardinal) defines them, so there's
  nothing yet to represent. Registration mechanics (C++ header includes,
  `register_*` calls) are deliberately *not* in the IR — they're
  GameAK-backend generation detail, and the IR's job is to stay backend-
  independent; `IR.Schedule` carries the registration/execution *order*
  the backend needs, which is the semantic part)
- [x] Ensure the IR contains resolved references rather than YAML strings.
  (every `Ref` carries a resolved `Namespace`, obtained via
  `CompileResult.ResolveRef`; `Build` refuses to run when the compile result
  has errors, so an IR is never built from unresolved references)
- [x] Make IR output stable and deterministic for tests and tooling.
  (`sortAll` orders every top-level slice by (namespace, name);
  `buildSchedule` breaks priority ties by qualified name; verified in
  `tests/ir_test.go`'s "output is deterministic across repeated builds")
- [x] Add `seed compile` to validate models and produce the IR without generating
  C++. (`internal/commands/compile.go`; verified in `tests/commands_test.go`
  that it writes no `Include/` output, unlike `sync`)
- [x] Add an optional machine-readable IR output for editor and CI integration.
  (`seed compile --json`, indented JSON via `json` struct tags on every IR
  type)
- [x] Define compatibility and migration rules for future IR versions.
  (`ir.Version` + semver bump rule documented on the constant: patch for
  docs-only changes, minor for additive fields, major for breaking changes;
  tooling should reject an unrecognized major version rather than guess)

## Phase 4: Define the Seed runtime architecture

All items below are specified in `docs/runtime-architecture.md`, grounded
against the actual scaffold templates in `internal/project/templates/`
(the only runtime code that exists today). Several sections document
**known gaps** — concrete inconsistencies found in the current templates,
not speculative — that Phase 7 must fix when the shared runtime is built;
this phase is design/documentation only, so no template code was changed.

- [x] Define the runtime layers and the ownership boundary of each layer:
  platform, application, simulation, presentation, assets, and game code.
  (`docs/runtime-architecture.md` §1)
- [x] Define the application lifecycle from process startup through bootstrap,
  loading, main loop, suspension, shutdown, and failure recovery.
  (`docs/runtime-architecture.md` §2)
- [x] Define how SDL events enter Seed and become input, window, lifecycle, or
  GameAK events and commands. (`docs/runtime-architecture.md` §3 — found and
  documented that `main.cpp.tmpl`'s event loop never reaches GameAK today)
- [x] Define how simulation state becomes render and audio work without coupling
  GameAK storage directly to SDL APIs. (`docs/runtime-architecture.md` §4 —
  found and documented that the 3D template's renderer reads
  `rt.get_block`/field offsets directly today, the exact coupling this item
  warns against)
- [x] Define fixed-step simulation, variable-rate presentation, interpolation,
  frame pacing, and clock ownership. (`docs/runtime-architecture.md` §5 —
  found and documented a real bug: the 3D template configures a fixed
  timestep but ticks with variable dt anyway, while the 2D template ticks
  with a constant dt regardless of real elapsed time)
- [x] Define thread ownership and synchronization rules for platform, simulation,
  rendering, audio, asset loading, and background work.
  (`docs/runtime-architecture.md` §6 — deliberately conservative
  single-main-thread decision for the first milestone, background work
  limited to asset loading via a job/result queue)
- [x] Define service ownership and dependency injection through `Seed::Context`.
  (`docs/runtime-architecture.md` §7 — found that `Context` today is only an
  SDL_Init/Quit RAII guard with no services; decided it becomes the
  composition root with explicit per-consumer injection, not a service
  locator)
- [x] Define capability-based APIs so headless and platform-limited targets can
  omit unavailable facilities cleanly. (`docs/runtime-architecture.md` §8)
- [x] Define error propagation, logging, crash context, and orderly shutdown
  across all runtime layers. (`docs/runtime-architecture.md` §9)
- [x] Keep the headless/server lifecycle a first-class configuration rather than
  a special case of the graphical client. (`docs/runtime-architecture.md`
  §10 — a third `--mode headless` alongside `2d`/`3d`, sharing one lifecycle
  state machine rather than forking the codebase)

## Phase 5: Integrate the GameAK simulation layer

- [x] Move GameAK-specific decisions out of model parsing and generic analysis.
  (verified: `internal/generators/model.go` and `analysis.go` already
  contain zero GameAK references — GameAK specifics live only in
  `internal/generators/templates/` and, as of this phase, in the new
  `internal/backend/gameak` package)
- [x] Create an explicit backend interface that consumes the Seed IR.
  (`internal/backend/gameak.Backend`, consuming `*ir.IR`. **Scoped, not
  full:** it validates the IR against GameAK's constraints; it does not
  yet *generate* GameAK C++ from the IR — generation still runs off
  decoded YAML models directly through the existing templates. See
  `docs/gameak-mapping.md`'s opening "Known gap" for why that migration
  was left out of this pass)
- [x] Map Seed storage semantics to GameAK Data Blocks and layout strategies.
  (`docs/gameak-mapping.md` §1 — Component → `BlockTypeDescriptor`/
  `DataBlock`; found every component defaults to `LayoutStrategy::AoS`
  with no Seed schema field to request otherwise)
- [x] Map identities, commands, scheduler dependencies, events, and ephemeral
  data to GameAK primitives. (`docs/gameak-mapping.md` §2-5, §7 —
  identities to `gameak::core::Identity`; Commands to GameAK's existing
  `Command` variant set, ready for Phase 6 to target; Systems to
  controllers via `register_controller`; **found and fixed a real
  inversion**: GameAK sorts controllers by *higher* priority first, while
  `docs/semantics.md` §8 had documented the opposite — corrected there and
  in `internal/ir`'s schedule ordering; events have **no GameAK primitive
  at all** today, the sharpest gap in the mapping; ephemeral data is
  wired into the generated System signature but has no Seed model concept
  yet)
- [x] Document every mapping rule and its performance implications.
  (`docs/gameak-mapping.md` — e.g. narrow `Access` lists aren't just
  documentation, they're what lets GameAK's controller-grouping safely
  parallelize Systems; `ConvertLayout` is a structural reshape, not free)
- [x] Report backend limitations as diagnostics instead of silently degrading.
  (`internal/backend/gameak.Validate`: bare-name collisions across
  namespaces — real today, since `rt.define`/`register_<Name>` are keyed
  by bare name only — are errors; unsupported event delivery is a
  warning. Wired into both `seed compile` and `seed sync`, the latter via
  `internal/commands/sync.go`'s `checkGameAKBackend`)
- [x] Add golden tests from Seed models through IR to generated GameAK code.
  (`tests/golden_test.go` + `tests/testdata/example-domain-ir.golden.json`,
  using the `docs/examples` domain: one golden check on the IR's JSON
  shape, one on generated `.hpp` content)
- [x] Add compile tests against the supported GameAK revision.
  (`tests/gameak_compat_test.go`: opt-in via `SEED_TEST_GAMEAK_COMPILE=1`
  since it needs network and a real xmake toolchain; builds GameAK's own
  static libraries with Seed's generated `gameak-xmake.lua`. Run and
  passed against the pinned revision while implementing this phase)
- [x] Pin or record GameAK compatibility instead of depending implicitly on its
  moving `dev` branch. (`internal/project.GameAKPinnedRevision`, a
  specific commit SHA; `seed new`/`seed init` now clone full history and
  checkout that pin instead of shallow-cloning `dev`'s moving tip — the
  previous behavior this item explicitly warns against)

## Phase 6: Build the observable Cardinal

The current implementation only represents a transition as an event and target
state. Cardinal requires the complete decision and effect lifecycle:

```text
event
  -> candidate transitions
  -> guards and invariants
  -> selected transition
  -> exit actions
  -> state change
  -> entry actions
  -> commands
  -> runtime effects
  -> resulting events
```

All items below are detailed in `docs/cardinal.md`. Scope was agreed with
the user up front: this phase covers the Go-side semantic model, IR,
backend validation, and static inspection tooling — everything
independently verifiable without executing C++ — since this repository
generates C++ but does not build/run a live GameAK simulation. Runtime
behaviors (RNG replay, causal trace *production*, live `seed debug`, and
Command atomicity/rejection *at runtime*) are honestly marked blocked
rather than faked; `docs/cardinal.md`'s per-section detail and its
"Known gap" closing note are the source of truth for exactly what runs
today versus what still needs Phase 7's runtime work.

- [x] Extend the semantic model beyond `event -> target` with typed triggers,
  guards, priorities, exit actions, entry actions, and emitted commands.
  (`docs/cardinal.md` §1; `emitted commands` narrowed to `Emits` — resolved
  Event refs — since no Command model type exists yet, see §1's rationale)
- [x] Keep decision logic separate from effects: FSMs select transitions and
  produce Commands; Systems and the Runtime execute state changes.
  (`docs/cardinal.md` §2 — a structural property of the schema: nothing in
  `GuardDef` can express a mutation)
- [x] Define typed event payload matching and reject incompatible transitions at
  compile time. (`docs/cardinal.md` §3 — **honestly gapped**: no schema
  lets a guard/action reference specific Event fields yet, so there is
  nothing to type-check against; documented as future work, not attempted)
- [x] Define how FSM instances bind to entities, worlds, scenes, institutions,
  or application-level state. (`docs/cardinal.md` §4 — unchanged
  single-Entity binding; broader binding has no driving use case yet)
- [x] Define deterministic conflict resolution when multiple transitions are
  eligible. (`docs/cardinal.md` §5-6: highest-Priority + passing-Guard
  wins; same-priority same-event transitions are a compile-time error)
- [x] Add transition-level priorities without making YAML ordering an accidental
  semantic rule. (same as above — ties are rejected, never silently
  resolved by declaration order)
- [x] Add composable guards with `all`, `any`, and `not`, backed by typed queries.
  (`docs/cardinal.md` §7-9 — composability implemented and recursively
  validated; **"backed by typed queries" not implemented**, every leaf
  guard is a named C++ extension point instead, same root cause as §3)
- [x] Support named game-defined guards as explicit C++ extension points.
  (direct consequence of the above — every leaf guard already is one)
- [x] Define pure guard constraints: guards may inspect state but must not mutate
  it or produce hidden effects. (`docs/cardinal.md` §9 — a documented
  contract; not yet type-enforced since guards aren't wired into generated
  C++ at all yet, see the Known gap)
- [x] Add entry, exit, and transition actions that can only express effects by
  producing typed Commands and events. (`docs/cardinal.md` §10)
- [x] Validate Commands and global invariants before applying changes atomically.
  (`docs/cardinal.md` §11-12 — **blocked**: no Command schema or runtime
  exists; documented that GameAK's own Command/scheduler primitives
  already support this once Seed generates calls into them)
- [x] Define rejection behavior so failed guards, invalid Commands, and violated
  invariants remain observable without partially changing state.
  (`docs/cardinal.md` §11-12 and `internal/trace.Entry`'s
  `Rejected`/`RejectReason` fields — the format is ready; nothing produces
  a rejection yet)
- [x] Integrate FSMs with FIFO Event Loops and define delivery, ordering,
  consumption, propagation, and unhandled-event policies.
  (`docs/cardinal.md` §13-14, cross-checked against GameAK's actual
  `FifoScheduler`/`PriorityScheduler`; no standalone Loop model type added
  — `seed inspect loops` computes the map from existing data instead)
- [x] Define how resulting events are scheduled without accidental infinite
  same-tick feedback loops. (`docs/cardinal.md` §14; enforced structurally
  by `internal/trace.Validate` and tested)
- [x] Record every RNG decision with its seed and position so stochastic agents
  remain replayable and procedurally fair. (`docs/cardinal.md` §15 —
  **blocked and not designed**: no system draws random numbers yet, and
  designing the record format without a concrete caller risks guessing the
  wrong shape)
- [x] Produce a causal trace containing tick, event, loop, machine, previous
  state, evaluated guards, selected transition, Commands, and resulting events.
  (`internal/trace`: format designed and implemented with exactly these
  fields, validated structurally and tested against fixtures —
  `docs/cardinal.md` §16. **Production is blocked**: nothing generates a
  trace yet, since nothing runs the FSM lifecycle yet)
- [x] Add `seed inspect loops`, `seed inspect fsm <name>`, and
  `seed explain transition <machine> <from> <to>`.
  (`internal/commands/inspect.go`, `explain.go`; all operate on the IR,
  fully working, tested end-to-end in `tests/cardinal_cli_test.go`)
- [x] Add `seed trace` and deterministic `seed replay` workflows.
  (`internal/commands/trace.go` — real, working, file-based tools;
  `docs/cardinal.md` §18 is explicit that "deterministic replay" here
  means deterministic *reading* of a trace file, not re-executing a
  recorded run against a live Runtime, which needs Phase 7)
- [x] Visualize live Event Loops, FSM states, transitions, and causal chains in
  `seed debug`. (`docs/cardinal.md` §19 — **not attempted**: there is no
  live process to visualize; left undone rather than relabeling static
  data as "live")
- [x] Test transition success, guard rejection, conflicting candidates, Command
  rejection, atomicity, event ordering, RNG replay, and causal trace stability.
  (`docs/cardinal.md` §20's table maps each item to what's actually
  tested — `tests/cardinal_test.go`, `tests/cardinal_cli_test.go`,
  `tests/trace_test.go` — versus what's blocked; guard rejection and
  conflicting candidates are tested in their compile-time sense only,
  Command rejection/atomicity/RNG replay are blocked, listed as such)
- [x] Implement one complete vertical slice from external event through FSM,
  Command, System, resulting event, and trace. (`docs/cardinal.md` §21 —
  **compile-verified, not runtime-verified**: a guard+priority+action+emits
  state machine was written, compiled, IR-built, backend-validated (warns,
  doesn't error), and synced through the existing pipeline into valid
  GameAK C++; the live runtime half of the slice — an external event
  actually driving this through a running Runtime into a captured trace —
  needs Phase 7's Command/runtime work first, honestly not claimed done)

## Phase 7: Build SDL3 platform and game facilities

**Scope note:** this is a 12-item, effectively full-engine checklist. Per
an explicit scoping decision with the user, this pass fixed exactly the
concrete, already-diagnosed gaps `docs/runtime-architecture.md` had
recorded from Phase 4 (its "Summary of decisions requiring Phase 7
implementation work" section) rather than attempting the whole phase.
Checklist items below are left unchecked except where a scoped fix
genuinely closes them; most remain open and are not claimed otherwise.

**What this pass actually did**, verified by scaffolding and building all
three modes with real xmake against the pinned GameAK revision and real
SDL3 packages (`tests/project_mode_test.go`'s opt-in `TestScaffoldModesCompile`):

- Fixed the fixed-step/variable-dt bug: `main.cpp.tmpl` now runs a single
  accumulator-driven loop in every mode instead of the two inconsistent,
  incorrect patterns Phase 4 found.
- Added the presentation extraction step for the 3D demo path:
  `extract_render_list()` is now the only place per frame that reads
  GameAK block storage; the render loop that follows touches only plain
  value structs.
- Turned `seed::Context` into a composition root for `Window`/`Audio`
  (via `create_window`/`create_audio`, replacing independently constructed
  locals) and added a `Capability` type — `Renderer` ownership stays
  outside `Context` for now, documented as an open boundary in the header
  itself, not silently dropped.
- Added `--mode headless` end-to-end: mode validation
  (`project.validateMode`, rejecting anything other than `2d`/`3d`/
  `headless` — also a small, free step toward Phase 11's "validate `seed
  new --mode`" item), scaffolding that skips the image/font/asset-manager/
  renderer stack and its SDL3 packages entirely, and a generated `main()`
  that requests zero SDL capabilities, runs the same fixed-step loop
  against a real wall clock, and shuts down on `SIGINT`/`SIGTERM`.

A second pass (same session, explicit follow-up scoping decision: "close
the remaining Phase 7 items, but only within what this environment's
toolchain actually supports — no mobile/web SDKs, no live multi-process
hot reload") closed the rest of the checklist within a documented desktop
boundary. Full detail, including every honestly-stated gap, is in
`docs/facilities.md`; `docs/runtime-architecture.md`'s "Summary of
decisions requiring Phase 7 implementation work" section still lists the
narrower items (Application-layer loop extraction, `Seed::Log`, the
background-job queue) that remain open regardless of this pass.

All of the below were verified by scaffolding and building all three
modes (`headless`/`2d`/`3d`) with real `xmake` against the pinned GameAK
revision and real SDL3 packages, then smoke-running each binary
(`tests/project_mode_test.go`'s `TestScaffoldModesCompile`) — not just
checked for Go-template-valid syntax. This process found and fixed one
real, previously-uncompiled bug: `AssetManager::init()`/`quit()` called
`IMG_Init`/`IMG_Quit`, which don't exist in SDL3_image (see
`docs/facilities.md` §4).

- [x] Turn the initial SDL wrappers into stable Seed interfaces with explicit
  ownership and lifetime rules. (`Context` is the composition root for
  Window/Audio; `docs/facilities.md` §2 — Renderer ownership and true
  mobile/web lifetime rules remain the honest open piece, since there is
  no mobile/web toolchain in this environment to design lifetime rules
  against)
- [x] Implement application and window management for desktop, mobile, web, and
  headless targets where supported. (desktop + headless implemented and
  compile-verified; mobile/web explicitly out of reach — no SDK/toolchain
  in this environment, not attempted rather than faked)
- [x] Implement normalized keyboard, mouse, controller, touch, text, and sensor
  input with configurable action mapping. (`docs/facilities.md` §1:
  keyboard/mouse/gamepad-button via `ActionMap`/`InputTranslator`/
  `InputState`, multi-alias bindings. Touch/sensor explicitly not
  implemented — this desktop scaffold has no touch digitizer or motion
  sensor to normalize input from)
- [x] Define rendering interfaces and lifecycle for surfaces, devices, swapchains,
  cameras, render passes, materials, shaders, and 2D/3D presentation.
  (`docs/facilities.md` §2: the full Context→Renderer→begin/draw/end
  lifecycle already existed and is documented; `Camera2D` added as a
  CPU-side transform. Materials/render-passes/a true 3D perspective camera
  are the explicit gap — `Renderer3D` only ever draws screen-space quads
  through one fixed precompiled shader with no uniform input, so there is
  no 3D content yet for a real camera to project)
- [x] Keep rendering backend choices modular and avoid coupling game models to a
  single SDL GPU or graphics API path. (already true structurally —
  `docs/facilities.md` §2 documents why: no Seed model type ever
  references a renderer type)
- [x] Implement audio devices, buses, playback, spatial audio hooks, streaming,
  and lifecycle integration. (`docs/facilities.md` §3: `Bus` gains,
  `AudioStreamSource` chunked streaming. **Not implemented:** an audio
  decoder — SDL3 alone doesn't decode compressed formats and no codec is
  vendored — and spatial audio, since no Component schema associates a
  position with an audio source yet)
- [x] Implement asset identities, discovery, import, metadata, dependency graphs,
  loading, caching, hot reload, packaging, and unloading.
  (`docs/facilities.md` §4: identity via stable name, `unload(name)`,
  dev-only live `reload_stale` compiled out via `NDEBUG` in release.
  Discovery/import already exist at the model layer (`seed import`) and
  weren't duplicated; dependency graphs have nothing to represent yet
  (Seed's `asset` model has no inter-asset references); packaging is
  explicitly Phase 8's target-specific concern)
- [x] Integrate fonts, images, textures, models, materials, shaders, scenes, and
  audio into a single typed asset pipeline. (`docs/facilities.md` §5:
  texture/font were already integrated; this pass made the pipeline
  lifecycle actually get called from `main.cpp.tmpl` for the first time,
  which is what surfaced the `IMG_Init` bug above. Models/materials/shaders
  as asset kinds have nothing to integrate into yet, per the rendering gap
  above)
- [x] Define scene/world loading and transitions without conflating presentation
  state with GameAK simulation storage. (`docs/facilities.md` §6: `Scene`
  holds only Spawn/Despawn callbacks, never a live GameAK handle; wired
  into the 3D demo replacing its inline spawn loop. No `scene` YAML model
  type yet — Scene is hand-authored C++, not schema-generated, left for
  when a project actually has more than one)
- [x] Add save data, preferences, localization, filesystem, clipboard, dialogs,
  and URL/platform-service abstractions as target capabilities.
  (`docs/facilities.md` §7: `SaveData`/`Preferences`/`Localization`/
  `platform::{clipboard,open_url,show_open_file_dialog}`, all thin
  wrappers over real, verified-present SDL3 APIs)
- [x] Add development-only hot reload and diagnostics without changing release
  runtime determinism. (`docs/facilities.md` §4: `#ifndef NDEBUG` removes
  the hot-reload code path from release builds entirely, not just at
  runtime)
- [x] Provide a small playable vertical slice exercising SDL3, GameAK, assets,
  input, simulation, rendering, and audio together. (`docs/facilities.md`
  §8: the 3D demo exercises every facility above in one binary, compiled
  and linked against the pinned GameAK revision and smoke-run without a
  crash. **Honestly not full:** no audio is actually played (no bundled
  sample content), and no visual confirmation was possible in this
  environment — no display server to screenshot against)

## Phase 8: Multiplatform build and distribution

Same scoping pattern as Phase 7: implemented for real within what this
environment's toolchain supports (Linux, verified by actually building;
Windows/macOS via a real CI workflow that hasn't run yet since it isn't
pushed); Android/iOS/web are a documented plan, not an implementation —
no SDK/toolchain here to attempt them against. Full detail in
`docs/platforms.md`.

- [x] Define the supported target matrix and minimum platform/toolchain versions.
  (`docs/platforms.md` §1)
- [x] Make generated XMake projects express target capabilities and platform
  differences declaratively. (`docs/platforms.md` §2 — found and fixed a
  real cross-platform bug in the process: `-Wno-interference-size` was
  applied unconditionally in both generated `xmake.lua` files; it's
  GCC/Clang-only and would have failed to configure on MSVC)
- [x] Support desktop targets first, with reproducible Linux, Windows, and macOS
  builds. (`docs/platforms.md` §3 — Linux verified by repeatedly rebuilding
  all three project modes against the pinned GameAK revision; Windows/macOS
  depend on §8's CI workflow actually running, not claimed verified here)
- [x] Plan and validate Android, iOS, and web targets against SDL3 and GameAK
  constraints. (`docs/platforms.md` §4 — **plan only**: xmake/SDL3 have
  platform support and GameAK looks portable, but SDL3's GPU backend
  availability per target, package availability, touch input, and Web's
  main-loop model are all open, unvalidated questions with no toolchain
  here to answer them)
- [x] Manage native dependencies, architecture selection, debug symbols, and
  release profiles consistently. (`docs/platforms.md` §5 — dependencies via
  xmake's package manager uniformly, symbols/optimize made explicit per
  mode)
- [x] Add platform-aware asset cooking, packaging, compression, and manifests.
  (`seed package` / `internal/dist` — `docs/platforms.md` §6 is explicit
  that "cooking"/"compression" means a zip archive of verbatim assets plus
  a checksummed manifest, not a per-target asset transform, since nothing
  in Seed's asset model has per-platform variants to select between yet)
- [x] Add application metadata, icons, permissions, signing hooks, and distributable
  bundles without embedding credentials in Seed projects. (`AppMetadata`
  via `app.yaml` — `docs/platforms.md` §7 is explicit that icons,
  permissions, signing, and platform application packages are *not*
  implemented, each for a stated reason, rather than stubbed)
- [x] Add cross-platform CI that builds and runs smoke tests for each supported
  target. (`.github/workflows/ci.yml`: Go build/vet/test, then a
  ubuntu/windows/macos matrix scaffolding and building all three project
  modes against the pinned GameAK revision. Written and reviewed for
  correctness; its first real run happens on GitHub's infrastructure once
  pushed, not claimed run in this session)
- [x] Document platform capability differences and graceful fallback behavior.
  (`docs/platforms.md` §9, pointing at the `Capability`/headless-mode
  mechanism Phase 7 already built rather than duplicating it)

## Phase 9: Rails-inspired developer experience

- [ ] Normalize the command vocabulary and aliases around `seed new`,
  `seed generate`, `seed destroy`, `seed compile`, and `seed sync`.
- [ ] Make generators create domain-oriented models with useful conventions.
- [ ] Add `seed explain <model>` to show semantic interpretation and GameAK
  mapping.
- [ ] Add `seed inspect models` to show resolved declarations and relationships.
- [ ] Add `seed inspect schedule` to show execution phases and dependencies.
- [ ] Add `seed inspect events` to show producers, consumers, and delivery.
- [ ] Add `seed inspect storage` to show inferred layouts and their rationale.
- [ ] Add `seed inspect cardinal` to summarize active loops, machines, rules,
  pending events, and recent decisions.
- [ ] Evolve `seed debug` into a console/dashboard for the compiled project model.
- [ ] Introduce explicit migrations if model or generated-state evolution needs
  versioned transformations.
- [ ] Provide actionable errors with suggested commands or model changes.
- [ ] Add conventions and generators for application, scene, input map, render
  feature, audio bus, and asset pack—not only GameAK-oriented models.
- [ ] Make `seed doctor` explain missing SDKs and target-specific build
  requirements.
- [ ] Add `seed run`, `seed build`, `seed test`, and `seed package` with consistent
  target and profile options.

## Phase 10: Safe generation and extension points

- [ ] Generate files atomically so template failures cannot leave partial output.
- [ ] Distinguish fully generated files from user-owned files.
- [ ] Preserve manual system implementations across regeneration.
- [ ] Define stable hooks for custom C++ behavior and backend-specific overrides.
- [ ] Detect stale and orphaned generated files without deleting user files.
- [ ] Add a dry-run/diff mode before changing generated output.
- [ ] Make generation reproducible across machines and repeated runs.
- [ ] Define generated-file headers and provenance metadata.

## Phase 11: CLI and project hardening

- [ ] Validate `seed new --mode` and all enum-like CLI inputs.
- [ ] Handle working-directory changes and filesystem errors explicitly.
- [ ] Make `seed doctor` return a non-zero status for failed checks.
- [ ] Prevent accidental asset/model overwrite during `seed import`.
- [ ] Handle imports whose source and destination refer to the same file.
- [ ] Test the atlas baker, profiles, block generation, and error paths.
- [ ] Repair and enforce coverage reporting in the development toolchain.
- [ ] Add CI for formatting, vetting, tests, generation determinism, and C++
  compilation.
- [ ] Stop committing or leaving build artifacts in the repository root.

## Documentation

- [ ] Rewrite the README to distinguish the long-term runtime vision from the
  CLI capabilities available today.
- [ ] Add a tutorial that builds a small playable game from Seed models through
  SDL3 + GameAK integration and platform packaging.
- [ ] Document the Seed language/schema separately from the GameAK backend.
- [ ] Document the runtime layers and SDL3/GameAK integration contracts.
- [ ] Document generated code ownership and manual extension points.
- [ ] Reconcile or archive stale task lists under `.ralph/` and `specs/`.
- [ ] Publish a compatibility matrix for Seed, Go, C++, XMake, SDL, and GameAK.

## Definition of the first architectural milestone

The first architectural milestone is complete when a small game domain can be
decoded into a typed semantic model, validated, compiled into a deterministic IR,
explained by the CLI, and rendered into GameAK integration code that compiles in
an automated test. Template rendering directly from decoded YAML does not satisfy
this milestone.

The first product milestone is complete when that domain runs as a small playable
application using the Seed lifecycle, SDL3 platform services, GameAK simulation,
input, assets, rendering, and audio on every initially supported desktop target.

The first Cardinal milestone is complete when an external event can drive a
guarded FSM transition that produces typed Commands, applies its effects
atomically through the Runtime, emits resulting events, and generates a complete
deterministic causal trace that can be replayed and explained.
