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

- [ ] Design a versioned intermediate representation between Seed semantics and
  backend generation.
- [ ] Represent storage requirements, identities, queries, commands, events,
  schedules, and registration in the IR.
- [ ] Ensure the IR contains resolved references rather than YAML strings.
- [ ] Make IR output stable and deterministic for tests and tooling.
- [ ] Add `seed compile` to validate models and produce the IR without generating
  C++.
- [ ] Add an optional machine-readable IR output for editor and CI integration.
- [ ] Define compatibility and migration rules for future IR versions.

## Phase 4: Define the Seed runtime architecture

- [ ] Define the runtime layers and the ownership boundary of each layer:
  platform, application, simulation, presentation, assets, and game code.
- [ ] Define the application lifecycle from process startup through bootstrap,
  loading, main loop, suspension, shutdown, and failure recovery.
- [ ] Define how SDL events enter Seed and become input, window, lifecycle, or
  GameAK events and commands.
- [ ] Define how simulation state becomes render and audio work without coupling
  GameAK storage directly to SDL APIs.
- [ ] Define fixed-step simulation, variable-rate presentation, interpolation,
  frame pacing, and clock ownership.
- [ ] Define thread ownership and synchronization rules for platform, simulation,
  rendering, audio, asset loading, and background work.
- [ ] Define service ownership and dependency injection through `Seed::Context`.
- [ ] Define capability-based APIs so headless and platform-limited targets can
  omit unavailable facilities cleanly.
- [ ] Define error propagation, logging, crash context, and orderly shutdown
  across all runtime layers.
- [ ] Keep the headless/server lifecycle a first-class configuration rather than
  a special case of the graphical client.

## Phase 5: Integrate the GameAK simulation layer

- [ ] Move GameAK-specific decisions out of model parsing and generic analysis.
- [ ] Create an explicit backend interface that consumes the Seed IR.
- [ ] Map Seed storage semantics to GameAK Data Blocks and layout strategies.
- [ ] Map identities, commands, scheduler dependencies, events, and ephemeral
  data to GameAK primitives.
- [ ] Document every mapping rule and its performance implications.
- [ ] Report backend limitations as diagnostics instead of silently degrading.
- [ ] Add golden tests from Seed models through IR to generated GameAK code.
- [ ] Add compile tests against the supported GameAK revision.
- [ ] Pin or record GameAK compatibility instead of depending implicitly on its
  moving `dev` branch.

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

- [ ] Extend the semantic model beyond `event -> target` with typed triggers,
  guards, priorities, exit actions, entry actions, and emitted commands.
- [ ] Keep decision logic separate from effects: FSMs select transitions and
  produce Commands; Systems and the Runtime execute state changes.
- [ ] Define typed event payload matching and reject incompatible transitions at
  compile time.
- [ ] Define how FSM instances bind to entities, worlds, scenes, institutions,
  or application-level state.
- [ ] Define deterministic conflict resolution when multiple transitions are
  eligible.
- [ ] Add transition-level priorities without making YAML ordering an accidental
  semantic rule.
- [ ] Add composable guards with `all`, `any`, and `not`, backed by typed queries.
- [ ] Support named game-defined guards as explicit C++ extension points.
- [ ] Define pure guard constraints: guards may inspect state but must not mutate
  it or produce hidden effects.
- [ ] Add entry, exit, and transition actions that can only express effects by
  producing typed Commands and events.
- [ ] Validate Commands and global invariants before applying changes atomically.
- [ ] Define rejection behavior so failed guards, invalid Commands, and violated
  invariants remain observable without partially changing state.
- [ ] Integrate FSMs with FIFO Event Loops and define delivery, ordering,
  consumption, propagation, and unhandled-event policies.
- [ ] Define how resulting events are scheduled without accidental infinite
  same-tick feedback loops.
- [ ] Record every RNG decision with its seed and position so stochastic agents
  remain replayable and procedurally fair.
- [ ] Produce a causal trace containing tick, event, loop, machine, previous
  state, evaluated guards, selected transition, Commands, and resulting events.
- [ ] Add `seed inspect loops`, `seed inspect fsm <name>`, and
  `seed explain transition <machine> <from> <to>`.
- [ ] Add `seed trace` and deterministic `seed replay` workflows.
- [ ] Visualize live Event Loops, FSM states, transitions, and causal chains in
  `seed debug`.
- [ ] Test transition success, guard rejection, conflicting candidates, Command
  rejection, atomicity, event ordering, RNG replay, and causal trace stability.
- [ ] Implement one complete vertical slice from external event through FSM,
  Command, System, resulting event, and trace.

## Phase 7: Build SDL3 platform and game facilities

- [ ] Turn the initial SDL wrappers into stable Seed interfaces with explicit
  ownership and lifetime rules.
- [ ] Implement application and window management for desktop, mobile, web, and
  headless targets where supported.
- [ ] Implement normalized keyboard, mouse, controller, touch, text, and sensor
  input with configurable action mapping.
- [ ] Define rendering interfaces and lifecycle for surfaces, devices, swapchains,
  cameras, render passes, materials, shaders, and 2D/3D presentation.
- [ ] Keep rendering backend choices modular and avoid coupling game models to a
  single SDL GPU or graphics API path.
- [ ] Implement audio devices, buses, playback, spatial audio hooks, streaming,
  and lifecycle integration.
- [ ] Implement asset identities, discovery, import, metadata, dependency graphs,
  loading, caching, hot reload, packaging, and unloading.
- [ ] Integrate fonts, images, textures, models, materials, shaders, scenes, and
  audio into a single typed asset pipeline.
- [ ] Define scene/world loading and transitions without conflating presentation
  state with GameAK simulation storage.
- [ ] Add save data, preferences, localization, filesystem, clipboard, dialogs,
  and URL/platform-service abstractions as target capabilities.
- [ ] Add development-only hot reload and diagnostics without changing release
  runtime determinism.
- [ ] Provide a small playable vertical slice exercising SDL3, GameAK, assets,
  input, simulation, rendering, and audio together.

## Phase 8: Multiplatform build and distribution

- [ ] Define the supported target matrix and minimum platform/toolchain versions.
- [ ] Make generated XMake projects express target capabilities and platform
  differences declaratively.
- [ ] Support desktop targets first, with reproducible Linux, Windows, and macOS
  builds.
- [ ] Plan and validate Android, iOS, and web targets against SDL3 and GameAK
  constraints.
- [ ] Manage native dependencies, architecture selection, debug symbols, and
  release profiles consistently.
- [ ] Add platform-aware asset cooking, packaging, compression, and manifests.
- [ ] Add application metadata, icons, permissions, signing hooks, and distributable
  bundles without embedding credentials in Seed projects.
- [ ] Add cross-platform CI that builds and runs smoke tests for each supported
  target.
- [ ] Document platform capability differences and graceful fallback behavior.

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
