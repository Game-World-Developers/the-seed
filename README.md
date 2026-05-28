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

## Usage

```bash
seed new [project name]
```

Creates a new project with:
- ECS-to-DoD directory layout (Config, Include, Src, Tests, Third-Party)
- C++ source files generated from embedded templates
- Git repository initialization with an initial commit
- Auto-detection of [GameAK](https://github.com/gameworlddevelopers/GameAK) when available

```
cd myproject && make
```

## Repository Structure

```
├── cmd/
│   ├── root.go           # CLI root command (cobra)
│   ├── root_test.go
│   ├── new.go            # `seed new [project name]`
│   └── new_test.go
├── config/
│   └── project.seed.yml  # YAML template for generated config
├── internal/
│   └── project/
│       ├── create.go          # core generation logic
│       ├── create_test.go
│       └── templates/         # embedded C++ templates (*.tmpl)
├── main.go               # entrypoint
├── Makefile              # build / run / test / clean
├── go.mod / go.sum
└── README.md
```

## Generated Projects

The Seed generates C++ projects designed with:

- **Determinism and performance** as the default
- **Modular and linkable** as a runtime SDK
- Suitable for both **client** and **headless/server** execution
- Fully multiplatform

The generator keeps the generated project agnostic to gameplay, narrative, and world-specific logic.
