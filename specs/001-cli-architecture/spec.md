# Spec 001 — CLI Architecture

## System Responsibility

The `seed` CLI is a scaffolding and code generation tool for creating C++ game projects.
It SHALL provide commands to create new projects and generate game components from YAML model definitions.

## NOT in scope

- Building/compiling the generated C++ project
- Running the generated project
- Package management for C++ dependencies

## Build System

Generated projects MUST use **xmake** as the build system (`xmake.lua`).

## Commands

### `seed new <project-name>`

Scaffold a new C++ game project in `<project-name>/` directory.

**Verifies:** `xmake` is installed (warns if not).

**Generated structure:**
```
<project>/
├── xmake.lua
├── .gitignore
├── src/main.cpp
├── include/core/
│   ├── component/
│   ├── trait/
│   ├── entity/
│   └── archetype/
└── Models/
    ├── component/
    ├── trait/
    ├── entity/
    └── archetype/
```

### `seed generate <type> <name>`

Generate a game component from a YAML model.

**Types:**
- `component` — POD struct with fields (most basic item)
- `trait` — Composed of components via `std::tuple`
- `entity` — Composed of traits
- `archetype` — Model/specialization that defines entity structure

**Flow:**
1. If `Models/<type>/<name>.yaml` does not exist, create default YAML
2. Read YAML model
3. Render C++ header from Go text/template
4. Write to `include/<namespace>/<type>/<name>.hpp`

### YAML Model Format

**component:**
```yaml
name: Position
namespace: core
fields:
  - { name: x, type: float }
```

**trait:**
```yaml
name: Movable
namespace: core
components: [Position, Velocity]
```

**entity:**
```yaml
name: Player
namespace: core
traits: [Movable]
```

**archetype:**
```yaml
name: Player
namespace: core
entity: Player
```

## State Machine

```
Idle → new <name> → Scaffolding → Done
Idle → generate <type> <name> → (YAML exists? → Yes → Render) / (No → Create YAML → Render) → Done
```
