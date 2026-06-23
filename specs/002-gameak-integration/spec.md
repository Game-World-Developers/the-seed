# Spec 002 — GameAK + SDL3 Integration

## System Responsibility

Generated C++ game projects SHALL use **GameAK** as the simulation engine and **SDL3** as the rendering frontend.

## Dependencies

### GameAK

- Repository: `https://github.com/Game-World-Developers/GameAK` (branch `dev`)
- Integrated as a subdirectory `GameAK/` in the generated project
- SHALL include a thin `xmake.lua` wrapper that compiles `gameak-core` and `gameak-runtime` static libs
- Original `Makefile` in GameAK SHALL be preserved for standalone development

### SDL3

- Fetched automatically via xrepo: `add_requires("libsdl3")` in `xmake.lua`

## Project Structure

```
<project>/
├── xmake.lua                  # add_subdirs("GameAK") + add_requires("libsdl3")
├── .gitignore
├── GameAK/
│   ├── xmake.lua              # wrapper (thin xmake build rules)
│   ├── Makefile               # original (preserved)
│   ├── Include/GameAk/
│   └── Src/GameAk/
├── include/core/
│   ├── component/             # Position.hpp, Velocity.hpp (generated)
│   ├── trait/                 # Movable.hpp (generated)
│   ├── entity/                # Player.hpp (generated)
│   └── archetype/             # PlayerArchetype.hpp (generated)
├── src/
│   ├── main.cpp               # GameAK Runtime + SDL3 window loop
│   └── game/
│       ├── bootstrap.hpp      # register_components() + register_controllers() decl
│       ├── bootstrap.cpp      # skeleton impl
│       └── systems/           # game systems (render, input, etc.)
└── Models/
    ├── component/
    ├── trait/
    ├── entity/
    └── archetype/
```

## Game Loop (`src/main.cpp`)

```cpp
int main() {
    SDL_Init(SDL_INIT_VIDEO);
    auto* window = SDL_CreateWindow("title", 800, 600, 0);
    auto* renderer = SDL_CreateRenderer(window, nullptr);

    auto rt = gameak::runtime::Runtime<>::configure().build();
    game::register_components(rt);
    game::register_controllers(rt);

    while (running) {
        // SDL events
        rt.tick(1.0f / 60.0f);
        // SDL render
    }
}
```

## Component Registration

Each generated component header SHALL include an inline `register_<Name>()` function:

```cpp
inline void register_Position(gameak::runtime::Runtime<>& rt) {
    rt.define<Position>("Position")
        .has(&Position::x, "x")
        .has(&Position::y, "y")
        .done();
}
```

## `seed new <name>` Flags

- `--clone-gameak` — (default: false) clone GameAK repo into the project
