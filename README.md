# The Seed

The Seed is a **project generator** for C++ applications built with an **ECS-to-DoD** (Entity Component System to Data-oriented Design) architecture.

The generated projects are deterministic, modular runtime SDKs suitable for simulations, games, and virtual worlds.

This project is inspired by [World Seed](https://swordartonline.fandom.com/wiki/World_Seed) from [Sword Art Online](https://pt.wikipedia.org/wiki/Sword_Art_Online).

## Quickstart

```bash
go run main.go
# or
make run
```

Build the binary:

```bash
make build
./build/seed
```

Run tests:

```bash
make tests
```

## Repository Structure

```
├── cmd/          # CLI commands (cobra)
│   └── root.go
├── internal/     # generator core logic
├── templates/    # C++ project templates
├── main.go       # entrypoint
├── Makefile      # build/run/test/clean
└── go.mod
```

## Generated Projects

The Seed generates C++ projects designed with:

- **Determinism and performance** as the default
- **Modular and linkable** as a runtime SDK
- Suitable for both **client** and **headless/server** execution
- Fully multiplatform

The generator keeps the generated project agnostic to gameplay, narrative, and world-specific logic.
