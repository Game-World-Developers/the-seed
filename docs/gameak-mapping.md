# Seed → GameAK Mapping

This document records how Seed's semantic model (`docs/semantics.md`) and
IR (`internal/ir`) map onto GameAK's actual runtime primitives, and the
performance implications of each mapping. It is grounded against GameAK
commit `5049747541ef4e247a9b2d5ce67f896590c4d0d9` — the revision Seed is
pinned to (`internal/project.GameAKPinnedRevision`; see that file's doc
comment for how the pin is maintained) — not against GameAK's `dev`
branch generally, which moves without any compatibility guarantee.

**Known gap — scope of this pass:** the mapping described here is largely
*already implemented* by the hand-written templates in
`internal/generators/templates/`, which generate GameAK C++ directly from
decoded YAML `Model`s (Phase 2's `generators.Model`), not from the
resolved `internal/ir.IR`. Phase 5's "explicit backend interface that
consumes the Seed IR" is implemented in `internal/backend/gameak` as a
**validation** interface — it consumes `*ir.IR` and reports GameAK-specific
limitations (§6 below) — but it does not yet *generate* GameAK C++ itself.
Rerouting the existing template-based generation to run off the IR instead
of the raw decoded models is future work: the templates and `Model` types
carry structure (e.g. `Parts`) the IR does not currently represent, and
migrating them is a larger, separately-scoped refactor than validation.
This document describes the mapping as it exists in the templates today;
`internal/backend/gameak` is where that mapping's *rules* are enforced as
diagnostics rather than left to fail at C++ compile time or silently
misbehave at runtime.

## 1. Storage: Components → Data Blocks

| Seed | GameAK |
|---|---|
| `ComponentModel` | `BlockTypeDescriptor` (`Include/GameAk/Runtime/block_type.h`) |
| `ComponentModel.Fields[]` | `FieldDescriptor{name, offset, size, alignment}` |
| an instance's data | `DataBlock` (`Include/GameAk/Runtime/data_block.h`) — a byte buffer addressed by field offset |

`component.hpp.tmpl` generates a `register_<Name>` function that calls
`rt.define("<Name>")`, then `builder.has("<field>", &Namespace::Name::field)`
per field — GameAK computes each field's offset/size/alignment from the
pointer-to-member itself, so Seed never has to compute or emit an offset by
hand. **Performance implication:** GameAK defaults every block type to
`LayoutStrategy::AoS` (`layout_strategy.h`) unless told otherwise; `SoA`,
`AoSoA`, and `Archetype` layouts exist and are reachable through
`CommandConvertLayout` (see §3), but nothing in Seed's schema lets an
author request one — every Seed component is AoS today. This is a real,
current limitation (not a bug): choosing a layout strategy per Component
is a plausible future schema field, not yet added.

## 2. Identities

| Seed | GameAK |
|---|---|
| An `EntityModel` declaration (a *kind*, docs/semantics.md's Entity vs. instance distinction) | not represented — Entity is compile-time only |
| A live instance | `gameak::core::Identity` — an opaque `uint64_t` handle (`Include/GameAk/Core/identity.h`) |

GameAK's `Identity` carries no type information of its own; the block's
`type_id` (an internal `uint32_t` GameAK assigns at `rt.define` time, looked
up by name — see §6's collision risk) is what associates an `Identity` back
to a component's field layout. Seed's `Entity`/`Archetype` distinction
(glossary.md) has no GameAK counterpart at all yet — an "Archetype" in
GameAK's own vocabulary (`ArchetypeConfig`, a *layout strategy*, see §1) is
unrelated to Seed's `archetype` model type, which is closer to a spawn
preset. This naming collision between the two systems' vocabularies is
exactly why `docs/glossary.md` calls out Archetype as "distinct from the
GameAK term of the same name."

## 3. Commands

| Seed (target, Phase 6) | GameAK (`Include/GameAk/Runtime/command.h`) |
|---|---|
| not yet modeled — `System.Emits`/State-machine transition Commands are Phase 6 (Cardinal) concepts | `CommandCreateBlock`, `CommandDestroyBlock`, `CommandSetField`, `CommandResizeBlock`, `CommandConvertLayout` |

GameAK already has a complete, typed Command system — `Runtime::submit_command`
enqueues a `Command` (one of the five payload variants above) into a
scheduler (`FifoScheduler` by default, `PriorityScheduler` available via
`RuntimeBuilder::scheduler<PriorityScheduler>()`) that processes them
during `tick`. This is precisely the primitive Phase 6's Cardinal
Commands need to compile down to once they're defined — `CommandSetField`
already matches "an FSM transition's effect sets a field," and
`CommandCreateBlock`/`DestroyBlock` already match entity spawn/despawn.
**Nothing needs to be built in GameAK for Phase 6 Commands** — the mapping
work when Phase 6 arrives is entirely on Seed's side (defining the typed
Command model and generating calls into this existing API), not GameAK's.
**Performance implication:** `CommandConvertLayout` (switching a block
type's layout strategy at runtime) is presumably not free — it's a
structural reshape of every existing instance of that type — so it should
map to an explicit, rare Seed operation, not something triggered
implicitly by ordinary gameplay Commands.

## 4. Scheduler dependencies: Systems → Controllers

| Seed | GameAK |
|---|---|
| `SystemModel.Priority` | `register_controller(controller, priority)`'s `priority` arg |
| `SystemModel.Access` | `register_controller(controller, priority, type_access)`'s `type_access` arg |

`system.hpp.tmpl` looks up each `Access` component's registered `type_id`
by name (see §6 for the collision this bare-name lookup risks) and passes
the resulting set as `type_access`. **Corrected mapping rule (this pass):**
`Runtime::execute_single_tick` stable-sorts registered controllers with
`a.priority > b.priority` — **higher priority runs first**, ties broken by
registration order. `docs/semantics.md` §8 originally documented the
opposite ("lower runs first") before this mapping was checked against
GameAK's actual source; it has been corrected to match GameAK's direction
rather than have Seed silently invert `Priority` at generation time — see
that document's §8 correction note.

**Performance implication, and a capability Seed doesn't use yet:**
`execute_single_tick` also groups registered controllers by *pairwise-disjoint
`type_access`* and runs controllers within a group potentially concurrently
(`internal/GameAk` groups controllers, then executes groups sequentially —
see the "Parallel Group Formation" comment in `runtime.h`). A System that
declares an accurate, narrow `Access` list therefore isn't just
documentation — it's what lets GameAK safely parallelize that System
against unrelated ones. Declaring `Access` too broadly (or omitting it,
which skips the `type_access` overload entirely) forfeits this for free.
This is a concrete reason, beyond correctness, that Phase 2's
read/write-qualified `Access` decision (docs/semantics.md §8) matters:
today's flat `Access` list can only ever be treated as "might write," so
GameAK's grouping is more conservative than it needs to be until that
lands.

## 5. Events

| Seed | GameAK |
|---|---|
| `EventModel` (arbitrary author-defined events with fields) | **no equivalent** |
| — | `EventBus` (`Include/GameAk/Runtime/event_bus.h`): a *fixed* enum — `TickBegin`, `TickEnd`, `BlockCreated`, `BlockDestroyed` — with no user-extensible event type |

This is the sharpest mapping gap in the whole document. GameAK's
`EventBus` is a low-level lifecycle notification mechanism, not a
general typed-event system — there is no GameAK primitive a Seed `Event`
model could compile down to today. Concretely, `event.hpp.tmpl` generates
only a struct and an FNV-1a `constexpr uint32_t k<Name>Id`; nothing
consumes that ID. Meanwhile `state_machine.hpp.tmpl` builds a
`gameak::runtime::Fsm<std::string, std::string>` and calls
`fsm.add_transition(state, "<Event>", target)` — the transition matches
on the **raw event-name string**, entirely bypassing both the generated
`kEventId` constant and GameAK's `EventBus`. An Event model is therefore
only "live" today as a string an FSM transition happens to match against;
declaring an `Event` with fields, or one no transition references, has no
runtime effect at all. `internal/backend/gameak` reports this as a
warning (§6) on every declared Event so it's visible before an author
wonders why nothing happened. Closing this gap for real — giving Seed's
`Event` model actual delivery semantics — is what Phase 6 (Cardinal) is
for; nothing here should be read as "already solved."

## 6. Backend limitations reported as diagnostics

`internal/backend/gameak.Backend.Validate` checks the IR against exactly
the constraints documented above that Seed's own namespace-aware symbol
table (docs/semantics.md §2) cannot see, because they come from GameAK's
C++ API surface rather than Seed's model rules:

- **Bare-name collisions (error):** `rt.define("<Name>")` (component block
  types) and the generated `register_<Name>` C++ function (systems and
  state machines) are both keyed by bare `Name`, not a namespace-qualified
  one. Two same-named declarations in different namespaces are legal in
  Seed (§2 of `docs/semantics.md`) but collide at this layer — silently
  (block type registration just overwrites) or as a real C++ symbol
  collision (controllers). Reported before generation runs, in both `seed
  compile` and `seed sync` (see `internal/commands/sync.go`'s
  `checkGameAKBackend`), rather than discovered as a linker error far from
  the offending YAML.
- **Unsupported event delivery (warning):** every declared `Event` gets a
  warning per §5, since none are backend-supported yet. Warning, not
  error, because declaring an Event that nothing currently consumes is not
  itself invalid — Phase 6 will give it meaning without requiring the YAML
  to change.

## 7. What is explicitly not mapped in this pass

- **Ephemeral data:** `system.hpp.tmpl`'s generated `operator()` signature
  already takes a `gameak::runtime::EphemeralProducer&` parameter, but no
  Seed model concept produces or declares ephemeral data — it's available
  in the generated C++ skeleton for hand-written System bodies to use
  directly, with no Seed-level semantics around it yet.
- **`ConvertLayout`/layout strategy selection** (§1): no Seed schema field
  exists to request `SoA`/`AoSoA`/`Archetype` layout for a Component.
- **Generating GameAK C++ from the IR** rather than from decoded YAML
  models directly (this document's opening "Known gap").

These are recorded here as scope boundaries, not silently dropped — each
is a legitimate follow-up but was not required to satisfy Phase 5's
checklist, which asks for the mapping to be *documented* and its
limitations *reported as diagnostics where they're currently reachable*,
not for every gap to be closed in this pass.
