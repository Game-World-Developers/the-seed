# Seed Semantics

This document specifies the semantics of each Seed model type as they exist
today (`internal/generators/model.go`), and states the identity, ownership,
composition, reference, lifetime, and namespace rules that follow from the
current implementation. Terms are defined in [glossary.md](glossary.md).

This is a semantics *specification*, written against current behavior. Where
the current code is silent or under-specified, that gap is called out
explicitly rather than invented — those gaps are the open questions Phase 2
(the semantic compiler) must resolve before they can be enforced.

## 1. Model types and their semantics

| Type | Identity key | Structural content | References out |
|---|---|---|---|
| `component` | `(type, name)` in `namespace` | `Fields []FieldDef` | none |
| `trait` | `(type, name)` in `namespace` | — | `Components []ModelRef` → component |
| `entity` | `(type, name)` in `namespace` | — | `Traits []ModelRef` → trait |
| `archetype` | `(type, name)` in `namespace` | — | `Entity ModelRef` → entity |
| `state_machine` | `(type, name)` in `namespace` | `Initial string`, `States []StateDef`, `Priority int` | `Entity ModelRef` → entity, `Events []ModelRef` → event, and each `StateDef.Transitions[].Event` (a bare string name, **not** a `ModelRef** — see §5 gap) |
| `event` | `(type, name)` in `namespace` | `Fields []FieldDef` | none |
| `system` | `(type, name)` in `namespace` | `Priority int` | `Entities []ModelRef` → entity, `Access []ModelRef` → component |
| `asset` | `(type, name)` in `namespace` | `Kind string`, `Path string` | none (paths are filesystem, not model refs) |
| `block` | `(type, name)` in `namespace` | tile/physics fields | none (`AtlasPath` is filesystem, not a model ref) |

## 2. Identity

- A model's identity is the triple `(Type, Name, Namespace)`. `ModelTypeName`
  (used as the `ModelRegistry` map key) is only `(Type, Name)` — namespace is
  stored as the map *value*, which means **the registry cannot hold two models
  of the same type and name in different namespaces** without one silently
  overwriting the other on `Register`. This is a real ambiguity in current
  behavior, not a documented rule; Phase 2 must decide whether namespaces are
  part of the identity key or truly global per type.
- Identity is assigned by the YAML author (`name`, `namespace` fields), not
  generated or inferred. There is no UUID or synthetic ID at the model layer.
- Two models of *different* types may share a `Name` with no conflict —
  identity is always scoped by `Type`.

## 3. Ownership

- A model's `.yaml` file is the sole owner of its declaration. `detectModel`
  reads the `type`/`name`/`namespace` header directly from the file; there is
  no ownership indirection (e.g. no model can "claim" another file).
- Composition is by reference (`ModelRef`), never by embedding one model's
  full declaration inside another's file — with the sole exception of `parts`,
  which are owned sub-declarations local to the parent file (see §4).
- Generated C++ output is owned by the generator, not by the author, once
  produced — Phase 10 is where "which generated files are user-editable" gets
  formally defined; today ownership of *generated* artifacts is undefined.

## 4. Composition

Composition happens in two ways:

1. **By reference** — the normal case. A Trait composes Components, an Entity
   composes Traits, a System composes Entities/Components, a StateMachine
   composes an Entity/Events. This is the primary, type-checked composition
   mechanism (`Resolve` walks these references — see §6).
2. **By parts** — a single YAML file can declare a `parts` list of named
   sibling declarations under the same `Type`/`Name`/`Namespace` header
   (`HasParts`/`ExtractParts`). This exists on every model type but is only
   semantically meaningful where the type has structure to vary (component
   field sets, trait component sets, state-machine state sets, system
   entity/access sets) — `entity`, `archetype`, `event`, `asset`, and `block`
   implement the interface methods but always return `false`/`nil`, i.e. parts
   are a no-op for those five types today.

Composition is **acyclic by convention only** — nothing in `Resolve` currently
detects reference cycles (e.g. Trait A → Component that... nothing loops back
today, but System/Entity graphs are not checked). Cycle detection is a Phase 2
item, not yet implemented.

## 5. References and resolution

- A reference is written in YAML either as a bare string (just a name) or as
  an explicit map (`{name, namespace}` — `ModelRef.UnmarshalYAML`).
- Bare references are resolved by `(type, name)` lookup against the
  project-wide `ModelRegistry`, which is populated by scanning all model files
  before any `Resolve` call runs. This means **reference resolution is a
  distinct pass after parsing**, already separated from decoding — this part
  of Phase 2's "separate YAML decoding from semantic model construction" goal
  is effectively already true for references, though not yet for validation
  (see §8).
- Resolution mutates the reference in place, replacing a name-only `ModelRef`
  with a fully qualified one carrying the resolved `Namespace`.
- **Gap:** `StateDef.Transitions[].Event` and `.Target` are plain `string`
  fields, not `ModelRef`. Event names in transitions are never resolved
  against the registry and never validated to exist — this is the most
  concrete near-term Phase 2 item (typed source locations + validated
  references would catch a typo'd event/target state today only at C++
  compile time, if at all).
- **Gap:** Missing-reference errors (`reg.Resolve` failing) are returned as a
  bare `error`, not a structured diagnostic with file/line/field context —
  this is exactly what Phase 2's "typed source locations" work item is for.

## 6. Lifetime

- Model *definitions* (the YAML declarations) have compile-time lifetime only
  — they exist for the duration of a `seed sync`/generate run and produce
  static C++ output. There is currently no runtime representation of "a
  Component model" or "a System model" as such; only their generated code
  artifacts persist.
- Runtime lifetime (of entities, components, events as GameAK objects at
  play-time) is **not defined by Seed's semantic model today** — this is
  explicitly a Phase 4 (runtime architecture) and Phase 6 (Cardinal event/FSM
  lifetime) gap, not something this document can specify yet because no
  runtime lifecycle exists to describe.

## 7. Namespace rules

- Every model carries a `Namespace string` field, defaulting to `"Core"` when
  omitted in `defaultModel` (the generator default), but the field is not
  required or defaulted during `readModel`/parsing of hand-written YAML — an
  empty namespace is a valid, distinct namespace from `"Core"` unless the
  author sets it explicitly. This is a convention-vs-explicit gap: "empty
  namespace defaults to Core" is true only for scaffolded files, not for
  hand-authored ones.
- Namespace qualification renders as `Namespace::Name` (`ModelRef.Qualify`),
  matching generated C++ namespacing.
- As noted in §2, namespaces are **not** part of the `ModelRegistry`'s
  uniqueness key today, so two same-named models of the same type in
  different namespaces collide silently. Treat this as a known defect to fix
  in Phase 2, not a documented rule to design around.

## 8. System inputs, outputs, and access

Given current fields (`SystemModel.Entities`, `.Access`, `.Priority`):

- **Inputs:** the `Entities` a System declares are the archetypes/entity kinds
  it operates over; `Access` is the list of Components it reads or writes.
  **Decision:** `Access` entries are read-only by default; write access must
  be marked explicitly. Since `ModelRef` already accepts either a bare string
  or an explicit map (`{name, namespace}`), the same mechanism extends to
  carry access mode without a breaking YAML change:
  `access: [Position, {name: Velocity, mode: write}]`. A bare string or a map
  omitting `mode` means read-only. This keeps the common case (read) terse
  and makes mutation an explicit opt-in, matching the "explicit and
  deterministic" product principle. Implementing this mode field in
  `ModelRef`/`SystemModel` is a Phase 2 (semantic compiler) task; this section
  fixes the target semantics so Phase 2 has a spec to implement against.
- **Outputs:** a System's outputs are exactly its `write`-mode `Access`
  entries (§ above) plus any `Event`s it is declared to emit. **Decision:**
  Systems gain an explicit `Emits []ModelRef` field (→ event) alongside
  `Access`, so downstream tooling (`seed inspect events`, Phase 9) can compute
  producer/consumer graphs without inferring intent. A System with no
  `Emits` produces no events by definition — this is required, not inferred.
- **Ordering:** `Priority int` is the primary scheduling hint, lower runs
  first. **Correction (Phase 5):** GameAK's actual controller execution
  order is the opposite — `Runtime::execute_single_tick` stable-sorts
  registered controllers with `a.priority > b.priority`, i.e. **higher
  priority runs first**. Seed's templates pass `Priority` through to
  `register_controller` unmodified, so today's generated behavior is
  GameAK's (higher first), not this section's original "lower first"
  intent. Since no project has depended on either direction yet, the fix
  is to adopt GameAK's direction as Seed's own semantics rather than
  translate it at generation time — one fewer mapping rule to maintain.
  Read every "lower runs first" statement below as superseded by "higher
  priority runs first"; see `docs/gameak-mapping.md`'s scheduler mapping
  section for the full backend rationale. **Decision:** ties are broken by
  declaration order within a single
  phase — the order Systems are discovered while walking `Models/System/` in
  lexicographic filename order (already the natural, deterministic order a
  directory walk produces, and consistent with §11's determinism
  requirement). This makes tie-breaking fully deterministic without adding a
  new field: same `Priority` + same phase = alphabetical by model file name.
- **Dependencies/phases:** **Decision:** Systems execute in a fixed sequence
  of named phases (`PreUpdate`, `Update`, `PostUpdate` as the initial set,
  matching the fixed-step/variable-rate split Phase 4 will formalize). A
  System declares its phase with a `Phase string` field defaulting to
  `"Update"` by convention when omitted. Within a phase, ordering follows
  `Priority` then declaration order as above. Cross-system data dependencies
  (System B must see System A's writes) are expressed indirectly: a System
  that reads a Component another System writes in an earlier phase sees the
  update; same-phase read-after-write ordering is undefined and must be
  avoided by moving the writer to an earlier phase — Seed does not do
  automatic dependency inference. This is deliberately conservative: explicit
  phases over implicit dependency graphs, consistent with "no hidden runtime
  discovery."

## 9. Event semantics

- An Event is pure data (`Fields`) with no declared producer or consumer list
  of its own today. **Decision:** producers and consumers are declared from
  the referencing side, not the Event itself, and are two independent lists:
  a System declares events it produces via `Emits` (§8), and a StateMachine
  declares events it consumes via the existing `Events []ModelRef` field
  and/or `StateDef.Transitions[].Event`. An Event with no declared producer
  is a valid external/platform-injected event (e.g. input events raised by
  the SDL3 layer, not by any Seed System) — Seed does not require every event
  to have an in-model producer.
- **Delivery guarantees (decision, to be implemented in Phase 6):** events are
  delivered at-most-once per tick to each declared consumer, in FIFO order
  per event type, within the consumer's scheduled phase (§8). Events raised
  during phase N are visible to consumers no earlier than phase N+1 — same-
  phase same-tick delivery is disallowed to prevent accidental infinite
  same-tick feedback loops (the exact hazard the roadmap's Phase 6 calls out).
  An event with zero consumers is dropped, not an error — Seed does not
  require exhaustive handling.
- **Lifetime (decision):** an event instance lives for exactly one tick. It is
  visible to consumers during the delivery phase described above and is
  discarded afterward; nothing may hold a reference to an event across ticks.
  This bounds event-related memory to a single frame's worth of instances,
  consistent with "no hidden allocation... in the frame hot path."
- This section fixes target semantics; the FIFO Event Loop implementation,
  causal trace, and replay tooling remain Phase 6 deliverables.

## 10. Convention vs. explicit — current state

Per the roadmap's principle ("Convention over configuration... Explicit and
deterministic behavior"), here is what is currently convention (inferred by
the tool) vs. explicit (author-declared) vs. undefined:

- **Explicit today:** model `type`, `name`, `namespace` (when set), all
  references, state machine states/transitions, system entities/access,
  block physical properties.
- **Convention today:** default `namespace: Core` (scaffold-only, see §7),
  default file path (`Models/<TypeDir>/<Name>.yaml`, `modelPath`), default
  field values in scaffolded models (`defaultModel`), directory-per-type
  layout (`TypeDir`).
- **Resolved by decision in this document (Phase 2 will implement):**
  read access is the convention when `Access` mode is omitted, write must be
  explicit (§8); System `Phase` defaults to `"Update"` by convention when
  omitted, but the phase set itself is fixed and explicit (§8); tie-breaking
  for equal `Priority` is the convention "lexicographic by file name" (§8),
  never an explicit field; event lifetime is fixed at one tick by convention,
  not configurable (§9).
- **Still open (deferred to Phase 2 implementation, not semantics):**
  part-merge semantics for the five model types where `HasParts` is currently
  a no-op (`entity`, `archetype`, `event`, `asset`, `block`) — deferred
  because none of these types have structural variation that plausibly needs
  `parts` yet; revisit if a concrete use case appears.

## 11. Deterministic behavior and forbidden runtime magic

Restated from the roadmap's product principles, grounded in what's checkable
today:

- No reflection, hidden allocation, or runtime discovery in the frame hot
  path — not yet enforceable by tooling since no frame hot path exists yet
  (Phase 4/7). Today's determinism surface is generation-time only: given the
  same set of `.yaml` files, `seed sync` must produce byte-identical C++
  output. This is not currently tested (see roadmap Phase 3: "Make IR output
  stable and deterministic for tests and tooling").
- Ordering of models within a directory scan, map iteration in
  `ModelRegistry.data`, and slice iteration during `Resolve` must not affect
  generated output — Go map iteration order is randomized, so any generator
  logic that ranges over `ModelRegistry.data` directly (rather than over an
  ordered slice) is a latent nondeterminism bug to audit for in Phase 2/3.

## Decisions carried into Phase 2 (implementation, not further design)

These were open questions during Phase 1 drafting; each now has a fixed
answer in the sections above and only needs the semantic compiler to enforce
it:

1. `ModelRegistry` key must include `Namespace` (§2) — fix the collision.
2. `StateDef.Transitions[].Event`/`.Target` must become typed `ModelRef`s
   validated at resolve time (§5).
3. `System.Access` gains a read/write `mode` (default read), and `System`
   gains an explicit `Emits []ModelRef` output list (§8).
4. Equal-`Priority` Systems run in lexicographic file-name order within their
   phase (§8).
5. Empty `Namespace` must be normalized to `"Core"` uniformly at parse time,
   not only in the scaffold generator (§7).
6. Systems gain an explicit `Phase string` field (`PreUpdate`/`Update`/
   `PostUpdate`), defaulting to `"Update"` (§8).
7. Event delivery is FIFO-per-type, at-most-once per consumer per tick, never
   same-phase-same-tick, and an event instance's lifetime is exactly one tick
   (§9).

See [examples/](examples/) for representative domain models exercising every
type and reference kind defined above, per the Phase 1 checklist item
requiring examples before the YAML schema is extended further.
