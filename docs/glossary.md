# Seed Glossary

This glossary defines Seed's own vocabulary and distinguishes each term from
the GameAK or SDL3 concept it eventually compiles into. Seed concepts are the
only vocabulary game authors should need; GameAK/SDL3 terms are implementation
detail behind the compiler.

Each entry: **Seed meaning** — what the author writes and thinks in.
**Compiles to** — what it becomes at the GameAK/SDL3 layer today.

## Component

**Seed meaning:** A named, typed bundle of plain data fields (`FieldDef{Name,
Type}`) with no behavior. Declared as a top-level model (`type: component`) or
inline as a `parts` entry inside another model.
**Compiles to:** A GameAK Data Block field layout. Components are storage, not
objects — they carry no methods and no identity of their own.
**Distinguish from:** *Trait* (a named group of components), *Field* (a single
typed slot inside a component).

## Trait

**Seed meaning:** A named, reusable set of Component references
(`TraitModel.Components []ModelRef`). Traits let an Entity declare "I have
these capabilities" without repeating the component list.
**Compiles to:** A composition unit expanded into the entity's archetype
signature at generation time — traits do not exist as a separate GameAK
primitive.
**Distinguish from:** *Component* (the data itself), *Entity* (what accumulates
traits).

## Entity

**Seed meaning:** A named kind of game object defined purely by the Traits it
has (`EntityModel.Traits []ModelRef`). An Entity model is a template/schema,
not a single instance.
**Compiles to:** A GameAK archetype signature (set of component types).
**Distinguish from:** *Archetype* (a concrete Seed model that binds an Entity
to a specific configuration), a GameAK "entity" (a runtime instance/handle —
Seed's Entity is the *definition*, not the instance).

## Archetype

**Seed meaning:** A named, concrete configuration of one Entity
(`ArchetypeModel.Entity ModelRef`) — e.g. "ZombieArchetype" configuring the
"MonsterEntity". Distinct from the GameAK term of the same name.
**Compiles to:** A specific spawn/initialization preset over the Entity's
GameAK archetype signature.
**Distinguish from:** GameAK archetype (a component-type signature — closer to
Seed's *Entity*), Entity (the schema an Archetype configures).

## State machine

**Seed meaning:** A named FSM bound to an Entity (`StateMachineModel.Entity`),
with an `Initial` state, a set of `States`, and `Transitions` from each state
triggered by `Event` references. Can carry a `Priority` for conflict
resolution and declare the `Events` it consumes.
**Compiles to:** GameAK state/transition storage plus generated dispatch code.
Today only `event -> target` is modeled; Phase 6 (Cardinal) extends this with
guards, actions, and Commands.
**Distinguish from:** *Event* (the trigger), *System* (behavior that runs
every tick regardless of state).

## Event

**Seed meaning:** A named, typed message (`EventModel.Fields`) that can drive
state machine transitions or be consumed by systems.
**Compiles to:** A GameAK event/message type plus queue plumbing.
**Distinguish from:** *State machine* (the consumer that reacts to an event),
*Command* (Phase 6 concept: an effect a transition produces, not yet
implemented).

## System

**Seed meaning:** A named unit of per-tick behavior that operates over a set
of Entities (`SystemModel.Entities`) with declared data `Access`
(`[]ModelRef` to Components), and an optional scheduling `Priority`.
**Compiles to:** A GameAK system registered into the scheduler, with its
`Access` list becoming the query/read-write declaration.
**Distinguish from:** *State machine* (event-driven, discrete states) —
Systems run continuously; state machines transition on events.

## Asset

**Seed meaning:** A named reference to an external resource (`Kind`, `Path`)
such as a texture, model, or sound.
**Compiles to:** An entry in the generated asset manifest/loader wired to
SDL3's asset-manager wrappers.
**Distinguish from:** *Block* (a domain-specific voxel/tile definition that
happens to reference an asset via `AtlasPath`).

## Block

**Seed meaning:** A named voxel/tile definition: atlas `TileIndex`, per-face
`Colors`, physical flags (`Solid`, `Transparent`, `Fluid`), `Hardness`, and
`Drop` item name. A domain-specific model, not a generic Seed primitive.
**Compiles to:** Baked tile-color/atlas data (see `internal/baker`) plus
generated block-registry code.
**Distinguish from:** *Asset* (the underlying texture/atlas resource a Block's
`AtlasPath` points at).

## Field

**Seed meaning:** A single named, typed data slot (`FieldDef{Name, Type}`)
inside a Component or Event.
**Compiles to:** A struct member in generated C++.

## Part

**Seed meaning:** A named sub-declaration inside a model's `parts` list,
letting one YAML file define several closely related variants (e.g. several
component variants, or several state-machine configurations) that share a
`Type`/`Name`/`Namespace` header. Every model type implements
`HasParts`/`ExtractParts`, though not all currently use it meaningfully.
**Compiles to:** Additional generated types/entries alongside the parent
model's own declaration.

## Namespace

**Seed meaning:** A string grouping label (`Namespace` field, e.g. "Core") on
every model, used to qualify references (`ModelRef.Qualify()` →
`Namespace::Name`) and avoid name collisions across domains.
**Compiles to:** A C++ namespace or qualified identifier prefix.

## Reference (`ModelRef`)

**Seed meaning:** A named pointer from one model to another (e.g. a Trait's
component list, a System's entity list), written in YAML as a bare name or an
explicit `{name, namespace}` map. Unqualified references are resolved against
the project-wide `ModelRegistry` during `Resolve`.
**Compiles to:** A qualified C++ type reference, only after successful
resolution — see [semantics.md](semantics.md) for resolution rules.

## Model / Model type

**Seed meaning:** Any top-level YAML declaration identified by its `type`
field. The nine registered types today are `component`, `trait`, `entity`,
`archetype`, `state_machine`, `event`, `system`, `asset`, `block`
(`modelRegistry` in `internal/generators/model.go`).
**Compiles to:** One or more generated C++ files under the type's directory
(`TypeDir` mapping).

## Registry (`ModelRegistry`)

**Seed meaning:** The project-wide table of `(type, name) -> namespace`,
populated by scanning all models before resolution begins, used to resolve
unqualified references.
**Compiles to:** Not a runtime concept — a compile-time-only bookkeeping
structure with no GameAK/SDL3 counterpart.
