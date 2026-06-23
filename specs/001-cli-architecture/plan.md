# Plan 001 — CLI Architecture Implementation

## Tasks

### Task 1: Bootstrap Go CLI ✅
- Create `cmd/seed/main.go` calling root command
- Create `internal/commands/root.go` with root cobra.Command
- Create `internal/commands/new.go` with `seed new <name>`
- Create `internal/commands/generate.go` with `seed generate <type> <name>`

### Task 2: Project Scaffolding ✅
- Create `internal/project/scaffold.go` — xmake.lua, Models/, include/ structure
- Create `internal/project/config.go` — project config types

### Task 3: Generator Engine ✅
- Create `internal/generators/engine.go` — read YAML, render template, write file
- Create `internal/generators/model.go` — YAML structs + default model generation
- Create `internal/generators/templates/` — C++ templates (component, trait, entity, archetype)
- Templates use Go `text/template` for dynamic rendering

### Task 4: Wire Commands ✅
- `seed new` calls `project.Scaffold()`
- `seed generate` calls `generators.Generate()`
- `seed generate` creates default YAML if missing

## Pending
- [ ] `seed generate` with `--dir` flag for custom project path
- [ ] `seed generate --all` to regenerate all models
- [ ] `seed model` subcommand for explicit YAML creation
