# Representative domain example

A minimal player-movement domain exercising every model type and reference
kind defined in [../semantics.md](../semantics.md), before the YAML schema is
extended further (Phase 1 checklist item). Every file here parses against the
current `internal/generators/model.go` implementation as-is.

## Composition graph

```text
Component: Position, Velocity, Health
              |
              v  (trait.components -> component)
Trait: Movable(Position, Velocity), Living(Health)
              |
              v  (entity.traits -> trait)
Entity: PlayerEntity(Movable, Living)
              |
   +----------+-----------------------+
   v                                  v
Archetype:                    StateMachine:
DefaultPlayerArchetype        PlayerMovementFSM
(entity -> entity)            (entity -> entity, events -> event)
                                       ^
                                       | (bare transition event/target strings,
                                       |  see semantics.md §5 gap)
                               Event: StartMoveEvent, StopMoveEvent

System: MovementSystem
  entities -> PlayerEntity (entity)
  access   -> Position, Velocity (component)

Block: StoneBlock
  atlas_path -> Assets/Textures/TerrainAtlas.bmp (filesystem, not a model ref)

Asset: PlayerTexture
  path -> Assets/Textures/Player.bmp (filesystem, not a model ref)
```

## What each file demonstrates

- `Component/Position.yaml`, `Velocity.yaml`, `Health.yaml` — plain field
  bundles, no references out (glossary: Component).
- `Trait/Movable.yaml`, `Living.yaml` — reference Components by bare name,
  resolved against the registry (semantics.md §5).
- `Entity/PlayerEntity.yaml` — references both Traits, showing composition by
  reference rather than embedding (semantics.md §4).
- `Archetype/DefaultPlayerArchetype.yaml` — a concrete configuration of the
  Entity, distinct from a GameAK archetype signature (glossary: Archetype).
- `Event/StartMoveEvent.yaml`, `StopMoveEvent.yaml` — one event carrying
  fields, one carrying none, showing Events are optional-payload messages.
- `StateMachine/PlayerMovementFSM.yaml` — binds to `PlayerEntity`, declares
  its consumed `Events` via `ModelRef`, and separately references the same
  events by bare string inside `states[].transitions[].event`/`target` — this
  duplication is the exact gap called out in semantics.md §5 (transition
  fields aren't typed `ModelRef`s yet, so the two references aren't checked
  for consistency by the tool today).
- `System/MovementSystem.yaml` — references `PlayerEntity` via `entities` and
  `Position`/`Velocity` via `access`, both currently read *and* write with no
  mode marker (semantics.md §8 decision: default to read, mark write
  explicitly — not yet implemented, so this file uses the schema as it exists
  today).
- `Asset/PlayerTexture.yaml`, `Block/StoneBlock.yaml` — filesystem paths
  (`path`, `atlas_path`) are deliberately *not* `ModelRef`s, showing the
  boundary between model references and asset references (glossary: Asset
  vs. Block).

## Known-gap callouts visible in this example

- `PlayerMovementFSM`'s `states[].transitions[]` reference `StartMoveEvent`/
  `StopMoveEvent`/`Idle`/`Running` as bare strings — a typo here (e.g.
  `StartMovEvent`) would not be caught by `Resolve` today, only surfaced (if
  at all) once generated C++ fails to compile. This is the concrete
  motivation for semantics.md §5's decision to make these typed `ModelRef`s
  in Phase 2.
- `MovementSystem`'s `access` list mixes what is actually a write
  (`Position`, updated by the system) with what could be read-only
  (`Velocity`, if the system only reads it) — nothing in the current schema
  lets this example say which is which; that's semantics.md §8's read/write
  decision, also pending Phase 2 implementation.
