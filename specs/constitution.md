# The Seed — Constitution

## Principles

1. **Determinism First** — All generated code MUST be deterministic. Same input → same output.
2. **Performance as Default** — C++ generated code MUST favour zero-cost abstractions, no hidden allocations.
3. **Modular by Design** — Every generated component MUST be linkable as a standalone module.
4. **Specs Drive Code** — All features MUST start as a spec in `specs/`. No spec, no implementation.
5. **SDD + Ralph** — Use Spec-Driven Development for architecture, Ralph-Wiggum loop for iterative implementation.
6. **Idiomatic Go CLI** — The `seed` CLI MUST follow Cobra conventions and Go best practices.
7. **C++17 Minimum** — Generated C++ MUST target at least C++17, prefer C++20 where applicable.

## Spec Format

Each spec in `specs/<NNN>-<name>/` MUST contain:
- `spec.md` — System responsibility, data structures, API contracts, state machine
- `plan.md` — Implementation plan broken into tasks
- `tasks.md` — Actionable task list (optional, can reference `plan.md`)

## Quality Gates

Before any Ralph iteration is considered complete:
- `go build ./...` MUST pass
- `go vet ./...` MUST pass
- Spec MUST be validated against constitution
