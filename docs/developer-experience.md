# Rails-inspired developer experience

This document covers Phase 9's 14-item checklist. Every command below was
actually run against a real project in this pass (most against the
`docs/examples` domain or a freshly scaffolded project) — output shown in
this document's companion PRs/commits is real CLI output, not illustrative
prose.

## 1. Command vocabulary and aliases

`seed generate <type> <name>` and `seed destroy <type> <name>` are now
top-level commands (`internal/commands/generate.go`,
`internal/commands/delete.go`), matching Rails' `rails generate`/`rails
destroy` convention directly. They're **additive aliases**: `seed model
generate`/`seed model delete` keep working unchanged — both sets of
commands call the exact same `runGenerate`/`runDelete` functions, so
there's no behavioral drift between the two spellings to keep in sync.
`seed compile` and `seed sync` were already top-level and unchanged.

## 2. Domain-oriented generator conventions

Already true before this pass and unchanged here:
`generators.defaultModel` (`internal/generators/model.go`) scaffolds
sensible starting content per type — a component gets a `value: float`
field, a state machine gets a two-state Idle/Running skeleton with a
real transition, a system gets empty `Entities`/`Access` ready to fill
in. `generators.CreateAll` (`seed generate --all` / `seed new`'s starter
models) goes further: it creates a small *cohesive* domain (Player/Enemy
entities, Movement/Physics/Input/Render/Audio/Debug systems, a
MenuScene/GameScene/PlayerController state machine set) rather than nine
disconnected, arbitrarily-named stubs — reviewed in this pass and found
already domain-oriented, not generic; no changes made.

## 3. `seed explain <model>`

`seed explain <type> <name>` (`internal/commands/explain.go`) shows a
model's resolved references (its semantic interpretation — what it
actually points at, post-compile) and a GameAK mapping note per type,
pointing at the relevant `docs/gameak-mapping.md` section rather than
duplicating it inline. `seed explain transition <machine> <from> <to>`
(pre-existing, Phase 6) is a separate, more specific subcommand for one
transition's full detail (guard, priority, emits) — both coexist under
the same `explain` command tree since cobra dispatches by matching the
first argument against a subcommand name before falling back to the
parent's own handler.

## 4-8. `seed inspect models/schedule/events/storage/cardinal`

All five (`internal/commands/inspect.go`, `inspect_extra.go`) read the
IR — never raw YAML — so every relationship, ordering, and storage
requirement shown is already resolved and validated:

- **`models`** — every declaration with its direct references.
- **`schedule`** — `IR.Schedule`, highest-Priority-first (the corrected
  GameAK ordering from `docs/semantics.md` §8/`docs/gameak-mapping.md` §4).
- **`events`** — producer/consumer map from transition `Emits` and event
  names, explicit that this is a *declared-relationship* view, not a
  live-delivery report (`docs/cardinal.md` §5's gap).
- **`storage`** — each entity's deduplicated component set plus the AoS
  layout note (`docs/gameak-mapping.md` §1's gap: no layout-selection
  schema field exists).
- **`cardinal`** — state machine shape (states/transitions/guards/
  priorities) plus the loop map, explicitly labeled "static declarations
  only" with a pointer to `seed trace`/`seed replay` for anything about
  an actual run — matching `docs/cardinal.md`'s "blocked on a live
  runtime" gap rather than pretending to summarize one.

## 9. Evolve `seed debug`

`internal/tui/tui.go` gained a **Cardinal** screen (toggled with `c`,
mirroring the existing Problems screen's `p` toggle): it runs the same
`generators.Compile` → `ir.Build` pipeline as `seed compile`/`seed
inspect cardinal` and shows the resulting schedule and state-machine
summary — this is the concrete "console/dashboard for the **compiled**
project model" the roadmap asks for, as opposed to the TUI's other tabs
(Components/Traits/.../Problems), which still read pre-compile data via
`generators.ListModels`/`GetProblems`. Evolving those to be IR-based too
is a reasonable follow-up, not done in this pass — a full TUI rewrite
carries more risk than the CLI-command work in this phase, and the
highest-value gap (no compiled-model view existed at all) is closed.

## 10. Explicit migrations

**Decision: not introduced, deliberately, because the trigger condition
in the roadmap's own wording ("if model or generated-state evolution
needs versioned transformations") hasn't occurred.** `internal/ir.Version`
and `internal/trace.Version` are both still `0.1.0` — no field has been
removed or had its meaning changed since either was introduced (only
additive fields, e.g. Cardinal's `Priority`/`Guard`/`Emits` added to
`Transition` in Phase 6). A migration mechanism designed now would be
speculative: it's unknown whether a future breaking IR change needs
field-renaming, structural reshaping, or something else, and guessing
wrong produces a migration tool that doesn't fit the actual first breaking
change when one happens. The versioning discipline that makes migrations
*possible* later is already in place (the patch/minor/major version-bump
rule documented on the `Version` constant in `internal/ir`,
`internal/trace`, and `internal/dist`); introducing the migration
*runner* is deferred to the first time it's actually needed.

## 11. Actionable errors with suggested commands

`internal/commands/suggest.go`'s `suggestFor` matches diagnostic message
patterns from `generators.Diagnostic` and `gameak.Diagnostic` (both
already-established types, Phase 2 and Phase 5) and appends a concrete
next step: a missing reference suggests the exact `seed generate <type>
<name>` to fill the gap; an ambiguous reference suggests qualifying with
an explicit namespace; a same-priority transition conflict suggests
adding a `priority:` field; a malformed guard points at
`docs/cardinal.md`. Wired into both `seed compile` and `seed sync`'s
diagnostic printing. This matches diagnostic message *text*, not a
structured error code — documented in the file's own doc comment as a
deliberate, contained trade-off: matching this codebase's own diagnostic
message constants, not arbitrary user-facing text, so a wording change
breaks the match loudly (missing suggestion) rather than silently wrong.

## 12. Generators for non-GameAK-oriented facilities

`seed generate scene <name>` and `seed generate asset_pack <name>`
(`internal/commands/generate_facility.go`) scaffold the two Phase 7
facilities that have real per-instance boilerplate worth generating: a
`seed::Scene` factory function pair (`.hpp`/`.cpp`, wired with
`on_enter`/`on_exit` stubs) and an `Assets/<name>/` directory for a
related asset group. Verified to actually compile by dropping a generated
scene into a real scaffolded project and running `seed build` — this
caught and fixed a real bug: the generated `.cpp`'s include path
(`"Game/Scenes/<name>.hpp"`) didn't match how `Src/Game/*.cpp` files
actually resolve same-directory headers (a bare `"<name>.hpp"`, matching
`bootstrap.cpp`'s own pattern) — fixed and re-verified before this was
considered done.

**Input map, audio bus, and render feature intentionally have no
generator** — a reasoned scope decision, not an oversight, documented in
`generate_facility.go`'s own doc comment: `seed::ActionMap`,
`seed::Bus`, and the `Renderer2D`/`Renderer3D` selection are each a
single, already-directly-usable type or enum an author writes inline
(`seed::ActionMap map; map.bind_key(...)`); there's no per-instance
boilerplate a generator would remove, so scaffolding an empty file for
each would exist only to check a box, not to help anyone.

## 13. `seed doctor` explains missing SDKs and target-specific requirements

`checkTargetSDKs` (`internal/commands/doctor.go`) detects the current
project's mode from which renderer header was scaffolded and reports
mode-specific requirements: for `3d`, whether `glslc` is available
(explicitly **not** a failure if missing — `Renderer3D` ships precompiled
SPIR-V, `glslc` is only needed to modify shader source yourself); for
every graphical mode, which SDL3 packages xmake will fetch on first
build. `checkThirdParty` also gained a real check: it opens
`Third-Party/GameAK`'s git history (via go-git, already a dependency) and
compares its checked-out commit against `project.GameAKPinnedRevision`
(Phase 5's pin), warning if they've diverged — since
`docs/gameak-mapping.md`'s entire mapping was only verified against that
exact pin.

## 14. `seed run`, `seed build`, `seed test` with consistent options

All three (`internal/commands/build.go`) wrap `xmake build`/`run`/`test`
with a single shared `--profile debug|release` flag. This isn't a
mechanical pass-through: `xmake build`/`run`/`test` don't accept a
build-mode flag directly (only `xmake f`/`xmake config` does) — found
while implementing this, so `--profile` runs that config step first, only
when given, leaving xmake's own persisted configuration untouched
otherwise. `seed run` additionally runs `seed sync` first by default
(`--no-sync` to skip) — "sync, build, and run" as one command, matching
why Rails-style tooling exists at all: an author shouldn't have to
remember a multi-step sequence for the common case. `seed test` succeeds
with "no tests to run" on a freshly scaffolded project (no test targets
exist until an author defines their own in `xmake.lua` — this command
doesn't impose a Seed testing convention on top of a game author's own
choice of C++ test framework).

All three were verified against a real scaffolded project and a real
`xmake` invocation (`tests/build_run_test_test.go`), not just help-text
checks — including that an invalid `--profile` is rejected before xmake
is even invoked.
