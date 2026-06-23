# Spec 010 — GameAK Architecture Reference

Status: READY

Last updated: 2026-06-23

---

## Table of Contents

1. [Overview](#1-overview)
2. [AK — Abstraction Kit](#2-ak--abstraction-kit)
3. [Runtime — Core Concepts](#3-runtime--core-concepts)
4. [Data Blocks](#4-data-blocks)
5. [Runtime Identity](#5-runtime-identity)
6. [Runtime Commands](#6-runtime-commands)
7. [Scheduler](#7-scheduler)
8. [Controller API](#8-controller-api)
9. [Controller Types](#9-controller-types)
10. [Runtime API](#10-runtime-api)
11. [Error Model](#11-error-model)
12. [Data Layout](#12-data-layout)
13. [Bit Representation](#13-bit-representation)
14. [Semantic Storage Inference](#14-semantic-storage-inference)
15. [Event System](#15-event-system)
16. [Ephemeral Data Blocks](#16-ephemeral-data-blocks)
17. [Integration Points with The Seed](#17-integration-points-with-the-seed)

---

## 1. Overview

GameAK is an **Abstraction Kit (AK) and simulation runtime** focused on transforming structured data into efficient execution models.

### Two-Layer Architecture

```
GameAK
├── AK (Abstraction Kit)         ← usable standalone
│   ├── Generic Containers       │   flat_vector, IntrusiveList, AVLTree, RBTree
│   ├── Bit Primitives           │   BitSet, BitVector, BitFlags, BitPacking
│   ├── Value Types              │   Result<T>, Identity, Error, SemanticConstraint
│   └── Platform Layer           │   SIMD, compiler abstractions, capability detection
│
└── Runtime                      ← depends on AK
    ├── State Model              │   Data Blocks, Commands, Scheduler, Identities
    ├── Controllers              │   FSM, Rule Systems, Pipelines, Event Loops
    ├── Data Layout              │   AoS, SoA, AoSoA, layout conversion
    ├── Event System             │   lifecycle notifications (TickBegin, BlockCreated, etc.)
    └── Ephemeral State          │   single-tick Data Blocks for intra-tick communication
```

### Architectural Pillars

1. **Generic Abstractions** — reusable building blocks (containers, value types, bit primitives)
2. **Data Layout** — multiple memory layout strategies (AoS, SoA, AoSoA)
3. **Bit Representation** — bit-oriented structures for storage, query, filtering
4. **State Transformation** — deterministic transformation via Controllers

### Core Principles

- **State is the source of truth.** Rendering, audio, networking are projections of state.
- **Layout independent from behavior.** Storage strategy is a Runtime concern.
- **Representation independent from layout.** Encoding is separate from organization.
- **Controllers operate on state.** Controllers contain behavior, not state.
- **Rendering, audio, networking, tooling are external to the Runtime.**

---

## 2. AK — Abstraction Kit

The AK is a general-purpose library of reusable abstractions with no dependency on the Runtime. It can be used standalone.

### Value Types

**Result\<T\>** (SPEC-012) — A discriminated union containing either a value or an error. `[[nodiscard]]`. Used throughout the Runtime Core.

```cpp
template<typename T>
class Result {
    bool has_value() const;
    T& value();
    const Error& error() const;
};
```

**Error** (SPEC-012) — Contains an ErrorCode and optional diagnostic message. No exceptions in Runtime Core.

```cpp
enum class ErrorCode : uint32_t {
    None, BlockNotFound, BlockTypeMismatch, InvalidIdentity,
    CommandRejected, CommandInvalid, TypeNotRegistered,
    AllocationFailed, InvalidOperation, InternalError,
    ControllerFailed, DuplicateRegistration, CapacityExceeded,
    LayoutMismatch,
};

class Error {
    ErrorCode code() const;
    const char* message() const; // optional, diagnostics only
};
```

**Identity** (SPEC-008) — Runtime Identity is a class type wrapping a 64-bit unsigned integer. Wraps a `uint64_t` with type safety, preventing implicit conversions.

```cpp
class Identity {
    // Default-constructed identity is invalid (zero)
    Identity();
    bool is_valid() const;
    // Comparison operators: ==, !=, <, <=, >, >=
    // Hashable
};
```

**SemanticConstraint** (SPEC-025) — Describes the semantic range of a data field. The Runtime consumes this at block type registration to infer minimum storage size.

```cpp
struct SemanticConstraint {
    int64_t min;           // Minimum inclusive value
    int64_t max;           // Maximum inclusive value
    bool is_bool;          // If true, field is boolean (1 bit)
    uint32_t enum_count;   // Number of distinct enum values
};

constexpr uint32_t bits_for(const SemanticConstraint& sc);
```

### Containers

**flat_vector\<T, InlineN\>** (SPEC-013) — Small-buffer-optimized vector. Stores up to `InlineN` elements inline without heap allocation. Default `InlineN = 8`. Methods: `push_back`, `emplace_back`, `pop_back`, `clear`, `size`, `capacity`, `empty`, `operator[]`, `at`, `data`, `begin/end`, `reserve`, `resize`, `back`, `front`, `erase`.

**IntrusiveList\<T\>** (SPEC-014) — Doubly-linked intrusive list. Elements inherit from `intrusive_node`. Circular sentinel for O(1) begin/end. No per-element allocation. Methods: `push_front`, `push_back`, `pop_front`, `pop_back`, `insert`, `erase`, `clear`, `size`, `empty`, `front`, `back`, `begin/end`.

**avl_tree\<Key, Value, Compare\>** (SPEC-015) — Self-balancing BST with balance factors. Rotations: Left-Left, Right-Right, Left-Right, Right-Left. Methods: `insert`, `find`, `contains`, `size`, `empty`, `begin/end`.

**rb_tree\<Key, Value, Compare\>** (SPEC-016) — Self-balancing BST with red-black coloring invariants. Methods: `insert`, `find`, `contains`, `size`, `empty`, `begin/end`.

### Bit Primitives (SPEC-019)

- **BitSet<N>** — Fixed-size compile-time bitset. AND, OR, XOR, NOT, shift, count.
- **BitVector** — Dynamic-size bitset. Uses `flat_vector` internally. Grows on demand.
- **BitFlags** — Named interface over fixed-size bit set (default 64 bits). Dynamic variant available.
- **BitPacking** — Compile-time bit field definitions. Pack/unpack multiple values into a word.

---

## 3. Runtime — Core Concepts

### Execution Model (SPEC-005)

The Runtime executes this cycle:

1. Controllers read State (read-only StateView)
2. Controllers produce Commands
3. Commands are submitted to the Scheduler
4. Scheduler validates Commands
5. Scheduler applies accepted Commands atomically
6. State is updated

### Tick Ordering (SPEC-009)

Within a single `tick()`, execution proceeds in:

1. **Controller Execution Phase** — All registered Controllers execute. Controllers read current state and produce Commands.
2. **Command Processing Phase** — All pending Commands are validated and executed by the Scheduler.
3. **Result Reporting** — A TickResult is produced summarizing the tick execution.

Commands submitted between ticks (via `submit_command()`) are queued and processed during the next `tick()`.

### Key Constraints

- State is the source of truth.
- Data Blocks contain no execution logic.
- Controllers contain behavior, not state.
- Controllers must not hold mutable state between ticks.
- Commands are immutable after creation.
- Commands are applied atomically (all or nothing).
- No exceptions in Runtime Core — all failures use Result/Error.
- Runtime is single-threaded by default.

---

## 4. Data Blocks (SPEC-006)

A Data Block is the **smallest independently addressable unit of simulation state.**

### Properties

- Contains data only — no execution logic.
- Owned and managed by the Runtime.
- Possesses a Runtime Identity.
- Typed — each block has a type defining its structure and semantics.
- May contain one or more fields.
- May be fixed-size or variable-size.
- Must not directly contain other Data Blocks (relationships use Runtime Identities).
- Storage layout and representation are independent from block semantics.

### Type Registration

Data Block types must be registered with the Runtime before use:

```cpp
struct BlockTypeDescriptor {
    uint32_t type_id;              // Unique type identifier
    size_t size;                   // Size in bytes
    size_t alignment;              // Alignment requirement
    const char* name;              // Human-readable name for diagnostics
    LayoutStrategy layout;         // AoS, SoA, AoSoA (default: AoS)
    AoSoAConfig aosoa_config;      // Chunk size for AoSoA
    std::vector<FieldDescriptor> fields;
    bool ephemeral;                // true = single-tick lifetime
    SemanticConstraint* semantic;  // Optional: infer size from semantics
};

Result<void> register_block_type(const BlockTypeDescriptor& desc);
```

### Block Lifecycle

- **Creation:** Controllers or consumers request creation through Runtime APIs. Runtime handles allocation, identity generation, initialization.
- **Destruction:** Controllers or consumers request destruction through Runtime APIs. Runtime handles lifecycle and cleanup.
- **Storage:** Runtime may change layout or representation during execution. Changes must be transparent to Controllers.

---

## 5. Runtime Identity (SPEC-008)

A Runtime Identity is a **stable 64-bit identifier** used to reference Data Blocks.

### Properties

- 64-bit unsigned integer, wrapped in a class type for type safety.
- Generated exclusively by the Runtime (monotonically increasing counter starting at 1).
- Zero (0) is reserved as the invalid identity sentinel.
- Globally unique within a Runtime instance.
- Destroyed identities are never reused.
- Stable for the entire lifetime of the associated Data Block.
- Stable across memory location, layout, and representation changes.
- Comparable (equality, relational) and hashable.

### Usage

```cpp
Identity id = runtime.create_block(TYPE_ID);
if (id.is_valid()) { /* use id */ }
```

---

## 6. Runtime Commands (SPEC-007)

A Runtime Command is a **request to modify simulation state.** Commands are produced by Controllers and executed by the Runtime.

### Properties

- **Immutable** — cannot be modified after creation.
- **Typed** — each Command type defines intent, validation, and execution semantics.
- **Identity-based** — reference Data Blocks through Runtime Identities, not memory addresses.
- **Atomic** — succeed completely or fail completely. No partial application.
- **Deterministic** — identical commands in identical order produce identical state.
- Commands are not Runtime Entities (no persistent identity).
- Must not contain arbitrary user data (payloads are explicitly defined by Command type).
- May target one or more Data Blocks.

### Constraints

- Controllers produce Commands; Runtime executes Commands.
- Controllers must not execute Commands directly.
- Commands do not directly mutate State.
- The Runtime may reject Commands that fail validation.
- Rejected Commands must not modify State.
- Commands may be cancelled before execution (cancelled == no state modification).

---

## 7. Scheduler (SPEC-011)

The Scheduler is responsible for **validating, ordering, and executing Commands.**

### Default Behavior (FIFO)

- Commands execute in First-In-First-Out order.
- Ordering matches submission order.
- Deterministic: same commands + same order → same resulting state.

### Validation

- Validates every Command before execution.
- Verifies referenced Runtime Identities exist.
- Verifies Command types match target Data Block types.
- Invalid Commands are rejected with explicit error information.
- Validation occurs during `tick()`, immediately before execution (not at submission time).

### Atomicity

- Every Command is applied atomically.
- A Command either succeeds completely or fails completely.

### Replaceability

The Scheduler may be replaced with a custom implementation during Runtime creation. Custom schedulers must preserve deterministic semantics.

---

## 8. Controller API (SPEC-010)

A Controller is a **callable runtime component that transforms simulation state by producing Runtime Commands.**

### Signature

```cpp
using Controller = std::function<Result<void>(
    StateView& state_view,
    CommandProducer& commands,
    EphemeralProducer& ephemeral
)>;
```

### Requirements

- Controllers are callables (function objects, lambdas, etc.).
- Controllers receive a **read-only StateView** — must not mutate state directly.
- Controllers produce Commands through a **CommandProducer.**
- Controllers may create ephemeral Data Blocks through an **EphemeralProducer.**
- Controllers must return a **Result** type.
- Controllers must not hold mutable state between ticks.
- State required by a Controller must be stored as simulation state (Data Blocks).

### Registration

```cpp
Result<void> register_controller(Controller controller, int priority = 0);
```

- Higher priority values execute before lower values.
- Default priority is 0.
- Controllers with equal priority preserve registration order (stable sort).

### Priority

Priorities are signed integers. Higher values execute first. Default is 0.

---

## 9. Controller Types

### 9.1 Finite State Machines (SPEC-020)

An FSM is a Controller that transitions between states based on events, producing Commands on transitions.

#### Support

- Both declarative (builder/DSL) and imperative (`add_state`, `add_transition`) definition.
- Entry actions — executed when entering a state, may produce Commands.
- Exit actions — executed when leaving a state, may produce Commands.
- Timed/automatic transitions.
- Events are a dedicated type (not generic Data Blocks), identified by integer/enum values.
- Events are submitted to the FSM via Command.

#### Constraints

- FSM must be a Controller (callable signature).
- FSM must not hold mutable state between ticks.
- All transitions must be explicitly defined (no implicit transitions).
- Only one active state at a time.
- Events are passed through a dedicated FSM event channel.

#### Out of Scope (initial)

- Hierarchical/nested state machines.
- Orthogonal regions / concurrent states.
- Guards (conditional transitions) — deferred to future spec.
- FSM composition.

#### Builder API (conceptual)

```cpp
// Declarative FSM definition
runtime.fsm("PlayerController")
    .state("Idle")
        .on<MoveEvent>()->go_to("Running")
        .on<JumpEvent>()->go_to("Jumping")
    .state("Running")
        .on<StopEvent>()->go_to("Idle")
        .on<JumpEvent>()->go_to("Jumping")
    .state("Jumping")
        .on<LandEvent>()->go_to("Idle")
    .build();
```

### 9.2 Rule Systems (SPEC-021)

A Rule System is a Controller that evaluates a set of rules (condition + action) and produces Commands for all matching rules.

- Conditions are query-based expressions (support AND/OR/NOT composition).
- Each rule has a unique priority for evaluation ordering.
- Multiple rules may fire in a single tick.
- Rules are evaluated in priority order (higher first).
- Rule actions produce Commands.
- Supports dynamic add/remove of rules between ticks.
- Rule System is a Controller.

### 9.3 Pipelines (SPEC-022)

A Pipeline is a Controller composed of an ordered sequence of stages. Each stage is a Controller.

- Stages execute sequentially in deterministic order.
- Stage output (Commands) feeds input of subsequent stages.
- Stage failure does not skip subsequent stages (stage isolation).
- Pipeline is nestable (Pipelines may contain other Pipelines).
- Supports add/remove stage APIs between ticks.
- Supports stage-local state.
- Pipeline is itself a Controller.

### 9.4 Event Loops (SPEC-023)

An Event Loop is a Controller that maintains a FIFO event queue and dispatches events to registered handlers.

- Event types are identified by integer/enum values.
- Event data is a typed template parameter.
- Handlers receive event data and a CommandProducer.
- Events are processed in FIFO order within a single tick.
- Multiple handlers for the same event — all execute in registration order.
- Unhandled events are silently consumed (not an error).
- Events may be enqueued from Controllers during Controller Execution Phase or Command Processing Phase.
- Event Loop is a Controller.

---

## 10. Runtime API (SPEC-009)

### Creation

```cpp
struct RuntimeConfig {
    LogLevel log_level;       // optional, defaults to warn
};

Runtime create(const RuntimeConfig& config);
Runtime create();             // default config
```

### Tick

```cpp
TickResult tick(float time_delta);
```

Returns a `TickResult` containing:
- Number of commands executed
- Number of commands rejected
- Number of controllers executed
- Execution status (Success, PartialFailure, CriticalFailure)

### Pause / Resume

```cpp
void pause();
void resume();
```

When paused, `tick()` returns immediately with an empty TickResult. Commands submitted while paused are queued and processed after resume.

### Fixed Timestep

```cpp
void set_fixed_timestep(float dt);
void clear_fixed_timestep();
float fixed_timestep() const;
```

When set, `tick()` accumulates delta and executes sub-ticks at the fixed timestep.

### Queries

```cpp
bool has_block(Identity id) const;
uint32_t count_blocks_by_type(uint32_t type_id) const;
std::vector<Identity> find_blocks_by_type(uint32_t type_id) const;
std::vector<Identity> find_blocks(std::function<bool(const DataBlock&)> pred) const;
```

### Block Relationships

```cpp
Result<void> relate(Identity parent, Identity child);
Result<void> unrelate(Identity parent, Identity child);
std::vector<Identity> children_of(Identity parent) const;
std::vector<Identity> parents_of(Identity child) const;
```

### Snapshot (Serialization)

```cpp
struct Snapshot {
    std::unordered_map<Identity, DataBlock> blocks;
    std::unordered_map<uint32_t, BlockTypeDescriptor> types;
    uint64_t next_identity;
};

Snapshot save() const;
void load(const Snapshot& snapshot);
```

### Command Submission

```cpp
Result<void> submit_command(const Command& cmd);
```

---

## 11. Error Model (SPEC-012)

The Runtime Core must not use exceptions. All failures are communicated through explicit value-based mechanisms.

### Error Codes

```
None                  No error
BlockNotFound         Data Block not found
BlockTypeMismatch     Operation type does not match block type
InvalidIdentity       Runtime Identity is invalid or zero
CommandRejected       Command was rejected by validation
CommandInvalid        Command is malformed or incomplete
TypeNotRegistered     Data Block type has not been registered
AllocationFailed      Memory allocation failed
InvalidOperation      Operation is not valid in current state
InternalError         Internal Runtime error
ControllerFailed      Controller execution failed
DuplicateRegistration Type or Controller already registered
CapacityExceeded      Runtime capacity limit reached
LayoutMismatch        Layout conversion failed
```

### Result Type

```cpp
template<typename T>
class [[nodiscard]] Result {
    bool has_value() const;
    T& value();
    const Error& error() const;
    // Implicit conversion from T and Error
};
```

---

## 12. Data Layout (SPEC-018)

Layout determines the physical arrangement of data fields within and across Data Blocks of the same type.

### Strategies

| Strategy | Description | Best For |
|----------|-------------|---------|
| **AoS** (Array of Structs) | Fields for one entity stored contiguously | Touching multiple fields of same entity |
| **SoA** (Struct of Arrays) | Each field in its own contiguous array | Touching same field across many entities |
| **AoSoA** (Array of Struct of Arrays) | Entities in chunks, fields as SoA per chunk | Compromise between AoS and SoA |

### Properties

- Layout is a property of `BlockTypeDescriptor` (type-level), not individual Data Blocks.
- Multiple layout strategies may coexist across different block types.
- Layout must be transparent to Controllers (same read API regardless of layout).
- Runtime may convert layout during execution (preserves identity and values).
- Layout conversion is requested via Command.
- AoSoA chunk size is configurable per type.

---

## 13. Bit Representation (SPEC-019)

Bit representations operate at the bit level for storage, query, filtering, synchronization, or execution.

### Primitives

| Type | Description |
|------|-------------|
| **BitSet\<N\>** | Fixed-size compile-time bitset. Heap-free. Supports AND, OR, XOR, NOT, shift, count. |
| **BitVector** | Dynamic-size bitset. Uses flat_vector internally. Grows on demand. |
| **BitFlags** | Named interface over fixed-size bit set (default 64 bits). Static and dynamic variants. |
| **BitPacking** | Compile-time field definitions. Pack/unpack multiple values into a single word. |

### Constraints

- BitSet size must be known at compile time.
- BitFlags backed by fixed-size integer (default 64 bits).
- All bit operations must be deterministic.
- BitSet and BitFlags must not allocate on the heap.

---

## 14. Semantic Storage Inference (SPEC-025)

The physical representation of data should not be the developer's responsibility. The developer describes meaning, constraints, and domain. The Runtime infers bit width, physical layout, compaction, alignment.

### SemanticConstraint

```cpp
struct SemanticConstraint {
    int64_t min;            // Minimum inclusive value
    int64_t max;            // Maximum inclusive value
    bool is_bool;           // If true, field is boolean (1 bit)
    uint32_t enum_count;    // Number of distinct enum values
};
```

### bits_for

```cpp
constexpr uint32_t bits_for(const SemanticConstraint& sc);
```

| Input | Result |
|-------|--------|
| Bool | 1 bit |
| Enum (4 values) | 2 bits |
| Range 0..100 | 7 bits |
| Range -100..100 | 8 bits |
| Range 0..1000000 | 20 bits |
| Single value (42) | 1 bit |
| Zero range (0..0) | 0 (invalid) |

### Integration with BlockTypeDescriptor

When `BlockTypeDescriptor::size == 0` and `semantic` points to a valid constraint, the Runtime infers `size` from `bits_for` (rounded up to bytes). Explicit size takes precedence over inference.

---

## 15. Event System (SPEC-017)

Separate from Event Loops (SPEC-023). This is a **lifecycle notification** system for runtime events.

### Event Types

```cpp
enum class EventType : uint32_t {
    TickBegin,       // Start of each tick/sub-tick
    TickEnd,         // End of each tick/sub-tick
    BlockCreated,    // After block creation via command
    BlockDestroyed,  // After block destruction via command
};
```

### API

```cpp
using EventHandler = std::function<void(const Event&)>;
using EventId = uint64_t;

EventId listen(EventType type, EventHandler handler);
void unlisten(EventId id);
```

### Event Data

```cpp
struct Event {
    EventType type;
    Identity identity;       // Populated for BlockCreated/BlockDestroyed
    uint32_t block_type_id;  // Populated for BlockCreated
};
```

### Constraints

- Handlers are called synchronously during the tick.
- Handlers receive a const Event reference — must not modify Runtime.
- Block events are detected by diffing block map (only when handlers exist).
- Each sub-tick in fixed timestep fires independent events.

---

## 16. Ephemeral Data Blocks (SPEC-024)

A second class of Data Block with a **single-tick lifetime.**

### Properties

- `BlockTypeDescriptor::ephemeral = true`
- Created directly via `EphemeralProducer::create(type_id)` — no Command required.
- Visible to all Controllers within the same tick via standard queries.
- Automatically destroyed by the Runtime at TickEnd.
- NOT included in snapshots (save/load).
- May participate in relationships during their lifetime.
- May be read by Commands to produce persistent blocks ("spill to persistent" pattern).

### Controller Signature Update

```cpp
using Controller = std::function<Result<void>(
    StateView&, CommandProducer&, EphemeralProducer&
)>;
```

### Purpose

Replaces ad-hoc intra-tick communication (MessageBox, EventLoop timing tricks) with a single primitive: a Data Block that dies at end of frame.

---

## 17. Integration Points with The Seed

This section maps The Seed's YAML model types to the GameAK APIs, defining exactly what C++ code each model must generate.

### Component → BlockTypeDescriptor + register_block_type

Each YAML component definition generates:

1. A POD struct with the declared fields
2. A `register_<Name>()` function that:
   - Creates a `BlockTypeDescriptor` with the component's fields
   - Calls `runtime.register_block_type(descriptor)`

```cpp
// Generated from component.yaml
struct Position {
    float x;
    float y;
};

inline void register_Position(gameak::runtime::Runtime<>& rt) {
    auto desc = gameak::runtime::BlockTypeDescriptor()
        .with_type_id(/* auto-assigned or hash-based */)
        .with_name("Position")
        .with_size(sizeof(Position))
        .with_layout(gameak::runtime::LayoutStrategy::AoS);
    rt.register_block_type(desc);
}
```

### Trait → Component Grouping

A trait is a semantic grouping of components. It does not generate a separate block type — it generates a type trait or tuple alias that Controllers can use to query sets of components.

### Entity → Component Composition

An entity is a composition of traits (and therefore components). It generates a struct that aggregates the components, serving as a convenience type for Controller authors.

### StateMachine → FSM Builder API

Each YAML state machine generates:

1. An event enum (if events are defined inline)
2. A `register_<Name>()` function that builds and registers an FSM using the GameAK FSM API

```cpp
// Generated from state_machine.yaml
inline void register_PlayerController(gameak::runtime::Runtime<>& rt) {
    rt.fsm("PlayerController")
        .state("Idle")
            .on<MoveEvent>()->go_to("Running")
        .state("Running")
            .on<StopEvent>()->go_to("Idle")
        .build();
}
```

### System → Controller Callable

Each YAML system generates:

1. A header with a Controller callable declaration
2. A `.cpp` file with the Controller implementation (user-editable)
3. A `register_<Name>()` function

```cpp
// Generated header
#include <GameAk/Runtime/runtime.h>

namespace Game {

struct Gravity {
    gameak::core::Result<void> operator()(
        gameak::runtime::StateView& view,
        gameak::runtime::CommandProducer& cmds,
        gameak::runtime::EphemeralProducer& ephemeral
    );
};

inline void register_Gravity(gameak::runtime::Runtime<>& rt) {
    rt.register_controller(Gravity{});
}

} // namespace Game
```

### Bootstrap Integration

The `seed sync` command manages `bootstrap.cpp` by inserting generated registrations between seed markers:

```cpp
// @seed:begin(components)
#include <Game/Component/Position.hpp>
register_Position(rt);
// @seed:end(components)

// @seed:begin(controllers)
#include <Game/PlayerController.hpp>
register_PlayerController(rt);
#include <Game/Gravity.hpp>
register_Gravity(rt);
// @seed:end(controllers)
```

---

## Spec References

| Spec | Title | Status |
|------|-------|--------|
| SPEC-000 | GameAK Vision | READY |
| SPEC-001 | Runtime State | READY |
| SPEC-002 | Tech Stack | READY |
| SPEC-003 | Platform Architecture | READY |
| SPEC-004 | Coding and Design Standards | READY |
| SPEC-005 | Runtime Model | READY |
| SPEC-006 | Data Blocks | IMPLEMENTED |
| SPEC-007 | Runtime Commands | IMPLEMENTED |
| SPEC-008 | Runtime Identity | IMPLEMENTED |
| SPEC-009 | Runtime API | IMPLEMENTED |
| SPEC-010 | Controller API | IMPLEMENTED |
| SPEC-011 | Scheduler | IMPLEMENTED |
| SPEC-012 | Error Model | IMPLEMENTED |
| SPEC-013 | Flat Vector | IMPLEMENTED |
| SPEC-014 | Intrusive List | IMPLEMENTED |
| SPEC-015 | AVL Tree | IMPLEMENTED |
| SPEC-016 | Red-Black Tree | IMPLEMENTED |
| SPEC-017 | Event System | IMPLEMENTED |
| SPEC-018 | Data Layout | IMPLEMENTED |
| SPEC-019 | Bit Representation | IMPLEMENTED |
| SPEC-020 | Finite State Machines | IMPLEMENTED |
| SPEC-021 | Rule Systems | IMPLEMENTED |
| SPEC-022 | Runtime Pipelines | IMPLEMENTED |
| SPEC-023 | Event Loops | IMPLEMENTED |
| SPEC-024 | Ephemeral Data Blocks | IMPLEMENTED |
| SPEC-025 | Semantic Storage Inference | IMPLEMENTED |
