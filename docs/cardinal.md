# Cardinal

Cardinal is the roadmap's name for Seed's complete FSM decision-and-effect
lifecycle:

```text
event
  -> candidate transitions
  -> guards and invariants
  -> selected transition
  -> exit actions
  -> state change
  -> entry actions
  -> commands
  -> runtime effects
  -> resulting events
```

This document specifies that lifecycle and records what this pass actually
built versus what remains **blocked on a live GameAK runtime that this
repository does not build or execute**. Per the scoping decision made
before starting this phase: this pass extends the Go-side semantic model,
IR, backend validation, and inspection tooling — all independently
verifiable without running C++ — and documents runtime-dependent items
(RNG replay, causal trace *production*, live `seed debug` visualization,
Command atomicity/rejection *at runtime*) as real, unimplemented gaps
rather than claiming them done.

## 1. Extended semantic model

`internal/generators/model.go` extends `TransitionDef` and `StateDef`:

```yaml
states:
  - name: Idle
    entry: [on_idle_entered]
    exit: [on_idle_exited]
    transitions:
      - event: StartMoveEvent
        target: Running
        priority: 10
        guard:
          all:
            - name: has_stamina
            - not: { name: is_stunned }
        emits: [MovementStartedEvent]
```

- **Typed triggers:** unchanged from before this pass — a transition's
  `event` is still a bare string matched against the machine's declared
  `events` list (docs/semantics.md §5's known gap). Extending this to a
  fully typed `ModelRef` remains open; see §3.
- **Guards:** `GuardDef` — exactly one of `name` (a leaf, game-defined
  guard), `all`, `any`, or `not` (composite, recursive). §7-9 below.
- **Priorities:** `TransitionDef.Priority`. §5-6.
- **Entry/exit actions:** `StateDef.Entry`/`Exit`, lists of named C++
  action extension points. §10.
- **Emitted events (the representable part of "emitted commands"):**
  `TransitionDef.Emits`, resolved `ModelRef`s to declared Events. Seed has
  no Command model type yet — docs/gameak-mapping.md §3 already
  established that GameAK's own `Command` variants (`CreateBlock`,
  `SetField`, ...) are the right target primitive once Commands are
  modeled, so inventing a parallel Seed Command schema before there's a
  concrete consumer for it would be premature. `Emits` is the part of
  "emitted commands" meaningful to declare and validate *today*: which
  events a transition may cause, which is exactly what feeds a causal
  trace's `resulting_events` field (§16) and `seed inspect loops` (§17).

All of the above are validated by `generators.Compile` (§3, §5-9) and
carried into `internal/ir.IR` with fully resolved references (§2 of
`internal/ir/ir.go`'s existing `Transition`/`State` types, extended with
`Priority`, `Guard`, `Emits`, `Entry`, `Exit`).

## 2. Decision logic separate from effects

This is a structural property of the schema, not a runtime mechanism (none
exists yet): a `GuardDef` can only be a boolean expression over named,
external conditions — there is no field anywhere in `GuardDef` through
which a guard could specify a mutation. Entry/exit actions and a
transition's `Emits` are the *only* places effects are declared, and both
are declarative (a name, or a list of event refs) rather than executable
Seed-level code. The actual state change (moving from the previous to the
target state) is not something YAML or the IR performs — it is exactly the
runtime step (§4's "state change" in the lifecycle diagram) that stays
GameAK's responsibility once Cardinal's runtime exists. Seed's semantic
layer selects; it does not execute.

## 3. Typed event payload matching — not implemented, honestly gapped

Rejecting an incompatible transition "at compile time" requires knowing
each Event's field *types* and matching them against what a guard/action
expects — but nothing in the schema lets a guard or action declare which
Event fields it consumes (guards are opaque named C++ functions; see §7-8).
There is therefore nothing yet for the semantic compiler to type-check
against. What Compile *does* validate today: that a transition's `event`
string, if it matches one of the machine's declared `Events`, resolves to
a real Event declaration (existing behavior, docs/semantics.md §5) and
that `Emits` entries reference real, declared Events (§1). Full typed
payload matching depends on a Command/action schema expressive enough to
reference specific fields, which does not exist yet — recorded here as
future work, not attempted.

## 4. FSM instance binding

Unchanged from `docs/semantics.md` §1's table: a `state_machine` model
binds to exactly one `Entity` via its `Entity` field — no world/scene/
institution/application-level binding concept exists in the schema. Extending
binding beyond a single Entity is a schema question with no concrete use
case driving it yet in this codebase (no scene or world model type exists
at all — that's Phase 7's "scene/world loading" item), so it's left
unaddressed rather than speculatively designed.

## 5-6. Deterministic conflict resolution and transition priorities

**Decision, implemented:** among a state's transitions sharing the same
`Event`, the transition with the highest `Priority` (matching the
GameAK-priority-direction correction from `docs/semantics.md` §8/
`docs/gameak-mapping.md` §4 — higher runs/wins first, consistently) is the
candidate; a `Guard`, if present, must also pass. **Two transitions for
the same event at the same priority is a compile-time error** —
`internal/generators/semantic.go`'s `validateStates` groups each state's
transitions by `(Event, Priority)` and rejects any group with more than
one member, with a diagnostic naming the state, event, and priority. This
is exactly the roadmap's "without making YAML ordering an accidental
semantic rule": Seed refuses to guess a winner from declaration order,
full stop — the author must break the tie explicitly.

What remains for the (not-yet-existing) runtime: guard evaluation order
among transitions at *different* priorities for the same event (highest
priority evaluated first; if its guard fails, fall through to the next)
is specified by this ordering but not implemented anywhere, since no
runtime evaluates guards yet.

## 7-9. Guards

- **Composability (`all`/`any`/`not`):** implemented in `GuardDef`/
  `ir.Guard`, validated recursively (`compiler.validateGuard`) — a
  malformed guard (zero or multiple kinds set on one node, per `GuardDef.
  Kind`) is a compile-time error at any nesting depth.
- **"Backed by typed queries":** not implemented. A leaf guard is a bare
  name (`GuardDef.Name`), not a query over specific Components/fields —
  there is no schema for expressing "true if Position.x > 10" declaratively.
  Every leaf guard is necessarily a **named, game-defined C++ extension
  point** (§8) rather than a Seed-expressible query; this satisfies "named
  game-defined guards as explicit C++ extension points" directly while
  leaving typed-query guards as future work, same root cause as §3.
- **Purity:** a *documented contract*, not something Seed's tooling
  enforces today. A guard is a hand-written C++ function; nothing stops
  its author from mutating state inside it once the runtime exists to call
  it — Seed's semantic model can't see into hand-written C++ bodies (this
  is inherent to the "manual C++ escape hatch" pattern established in
  Phase 4, not specific to guards). The contract: **guards receive only a
  read-only/const view of state and must not produce Commands or other
  side effects.** Enforcing this at the type level (e.g. generating a
  guard signature that only compiles against a `const StateView&`) is a
  Phase 7 codegen decision once guards are actually wired into generated
  C++ — see §21's Known gap.

## 10. Entry, exit, and transition actions

Same shape as guards: `StateDef.Entry`/`Exit` are named C++ extension
points, validated for non-empty, non-duplicate names
(`compiler.validateActionNames`). The **contract** (not yet
type-enforced, same reasoning as §9): an action may only express effects
by producing typed Commands/events, never by mutating state directly. In
today's schema, `Emits` (§1) is the only structured way a transition
declares an effect; entry/exit action *bodies* are still opaque
hand-written C++ once the runtime exists to call them.

## 11-12. Command/invariant validation, atomicity, and rejection

**Not implemented — genuinely blocked on both a Command schema (§1's
"not yet" note) and a live runtime.** What can be said today, grounded in
`docs/gameak-mapping.md` §3: GameAK's `Runtime::submit_command` already
enqueues Commands into a scheduler that validates and applies them — the
atomicity and rejection primitives Cardinal needs already exist on
GameAK's side (`RejectedCommand`, `core::Result<void>` returns).
**Nothing needs to be built in GameAK for this** — the work, once Seed has
a Command schema, is generating calls into that existing API and
propagating `RejectedCommand`/`Error` back out as an observable diagnostic
or trace entry (§16's `Entry.Rejected`/`RejectReason` fields already exist
in `internal/trace` for exactly this, ready for a producer). Declaring
this "validated" today would be false — no Seed-generated code calls
`submit_command` from FSM logic yet.

## 13-14. FIFO Event Loop integration, delivery, and same-tick feedback

Restating and lightly extending `docs/semantics.md` §9's existing
decisions, now cross-checked against GameAK's real primitives
(`docs/gameak-mapping.md` §4's `FifoScheduler`/`PriorityScheduler`):
delivery is at-most-once per tick per consumer, FIFO per event type, never
same-phase-same-tick — a resulting event produced at tick N is visible no
earlier than tick N+1. `internal/trace.Validate` (§16) enforces exactly
this rule structurally: an event appearing in one entry's
`resulting_events` must not be consumed (appear as another entry's
`Event`) at the *same* tick, which is precisely the mechanism that
prevents an accidental infinite same-tick feedback loop. This is validated
in `tests/trace_test.go`'s "rejects same-tick event production and
consumption" — a real, if synthetic-fixture-based, test of the rule.
Seed has no standalone "Loop" model type; `seed inspect loops` (§17)
computes the event→state-machine consumer map from existing data instead
of requiring a new schema concept.

## 15. RNG replay — blocked, format not yet designed

**Not implemented, and not designed in this pass.** Recording "every RNG
decision with its seed and position" requires a runtime that makes RNG
decisions in the first place — nothing in Seed's generated code or GameAK
integration draws random numbers today. Designing the record format ahead
of a concrete stochastic-agent use case risks guessing wrong about what
"position" needs to mean (per-entity? per-system? per-tick counter?)
without a real caller to validate against. Left explicitly for whenever
Phase 7 introduces the first system that actually needs randomness.

## 16. Causal trace

**Format designed and implemented; production is blocked.**
`internal/trace` defines `Trace`/`Entry` with exactly the fields the
roadmap's lifecycle diagram calls for: `Tick`, `Event`, `Loop`, `Machine`,
`PreviousState`, `EvaluatedGuards`, `SelectedTransition` (or
`Rejected`+`RejectReason`), `Commands`, `ResultingEvents`. `trace.Validate`
checks real structural/causal properties without needing a live runtime to
compare against: non-decreasing ticks, an unbroken `PreviousState` chain
per machine, and the §14 same-tick rule. This is genuinely tested
(`tests/trace_test.go`) against hand-written fixtures. **What's missing:**
nothing in Seed-generated C++ produces a `Trace` — that requires the
runtime work in §11-15 to exist first. `seed trace`/`seed replay` (§18)
are real, working commands *over a trace file*; they have no live producer
to point at yet.

## 17. `seed inspect loops`/`fsm`/`seed explain transition`

Implemented, all operating on the IR (no runtime needed):

- `seed inspect loops` — the event→consuming-state-machines map (§13).
- `seed inspect fsm <name>` — full state/transition/guard/priority/action
  dump for one machine.
- `seed explain transition <machine> <from> <to>` — the specific
  transition(s) between two states, including guard, priority, whether the
  event is declared, and emitted events.

Tested end-to-end as CLI commands in `tests/cardinal_cli_test.go`.

## 18. `seed trace` and `seed replay`

Implemented as **offline, file-based tools**: `seed trace <file>` validates
a trace file's internal consistency and prints a summary; `seed replay
<file>` validates then walks the sequence in tick order, printing each
transition/rejection/Command/resulting-event. Neither drives a live
process — "deterministic `seed replay`" in the sense of *re-executing* a
recorded run against a running GameAK Runtime so the game reproduces it is
explicitly a Phase 7 runtime feature (needs the RNG/Command runtime work
from §11-15 first). What exists today is deterministic in the sense that
matters for a file-based tool: given the same trace file, `replay` always
produces the same output.

## 19. `seed debug` visualization — not attempted

**Blocked and not attempted in this pass.** "Visualize *live* Event Loops,
FSM states, transitions, and causal chains" requires a running,
instrumented process to observe — `seed debug` (`internal/tui`) today
shows only static model/sync-status data, and extending it to show static
Cardinal structure (guards/priorities/actions from the IR) would be a
reasonable, bounded follow-up, but "live" visualization has no live
process to visualize yet. Left undone rather than faked with static data
relabeled as "live."

## 20. Tests

What's covered, and by what — being specific about which checklist words
map to a static check versus a genuinely blocked runtime behavior:

| Checklist item | Status |
|---|---|
| Transition success | `tests/cardinal_test.go`: distinct priorities compile cleanly; `tests/cardinal_cli_test.go`: `explain transition` describes a real match |
| Guard rejection | Only the *structural* sense is testable today (malformed guard → compile error); *runtime* guard evaluation rejecting a transition is blocked (§9, §11) |
| Conflicting candidates | `tests/cardinal_test.go`: same-priority same-event transitions are rejected at compile time (§5-6) |
| Command rejection | Blocked (§11) — no Command schema or runtime exists |
| Atomicity | Blocked (§11) — a property of GameAK's `Runtime::tick`, not exercised by any Seed-generated call yet |
| Event ordering | `tests/trace_test.go`: same-tick production/consumption is rejected; monotonic-tick and causal-chain checks |
| RNG replay | Blocked (§15) — no format, no producer |
| Causal trace stability | `tests/trace_test.go`: round-trips through JSON and re-validates cleanly |

## 21. Vertical slice — compile-verified, not runtime-verified

**What was actually run, end to end, while implementing this phase:** a
guard+priority+entry/exit-action+emits-equipped state machine (see
`tests/cardinal_cli_test.go`'s fixture) was written as YAML, passed
`generators.Compile` (all Cardinal validations green), built into an
`ir.IR`, validated by `internal/backend/gameak` (correctly *warning*, not
erroring, that the Cardinal fields aren't reflected in generated C++ yet —
§1's `checkCardinalSupport`), and synced through the existing pipeline —
`seed sync` still succeeds and produces valid, compilable GameAK C++ for
the rest of the model (state/transition skeleton unchanged, since
`state_machine.hpp.tmpl` wasn't modified in this pass). What this is
**not**: an external event driving a live FSM through a running GameAK
Runtime, producing real Commands, being observed by a System, emitting a
resulting event, and appearing in a captured causal trace — that full
runtime slice needs §11-15's Command/runtime work and §16's trace producer,
none of which exist yet. Calling the compile-verified slice "complete" per
the roadmap's literal wording would be inaccurate; it is accurately
described as the semantic/tooling half of the slice, done, with the
runtime half remaining.

## Known gap: generation does not consume any new Cardinal field

Consistent with `docs/gameak-mapping.md`'s opening note (generation still
runs off decoded YAML models through existing templates, not the IR):
`internal/generators/templates/state_machine.hpp.tmpl` was **not**
modified in this pass. It still generates only `fsm.add_state`/
`fsm.add_transition` calls from `Name`/`Initial`/`States[].Transitions[].
Event`/`.Target` — `Priority`, `Guard`, `Entry`, `Exit`, and `Emits` are
fully modeled, validated, and IR-resolved, but have zero effect on
generated C++ today. This is exactly what `internal/backend/gameak.
checkCardinalSupport` warns about on every state machine that uses them,
so the gap is visible at `seed compile`/`seed sync` time rather than
discovered by an author wondering why their guard never fires. Wiring
these fields into real GameAK FSM/Command generation is Phase 7 work,
gated on first checking whether GameAK's `Fsm<K, V>` type
(`Include/GameAk/Runtime/fsm.h`, not inspected in this pass beyond what
Phase 5 already reviewed) has any guard/action hook to generate against,
or whether Cardinal's runtime needs to bypass `Fsm<K,V>` and drive
transitions through Systems + Commands directly instead.
