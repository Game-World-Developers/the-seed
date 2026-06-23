# Plan 010 — GameAK Architecture Reference

## Objective

Create a comprehensive reference document capturing the full GameAK architecture (AK + Runtime) as defined by its 26 specs (SPEC-000 through SPEC-025). This document serves as the canonical reference for The Seed's code generation targets.

## Tasks

### Task 1: Document Core Architecture
- SPEC-000 (Vision), SPEC-001 (State), SPEC-005 (Runtime Model)
- Two-layer architecture: AK vs Runtime
- Architectural pillars and core principles

### Task 2: Document AK Layer
- SPEC-013 (FlatVector), SPEC-014 (IntrusiveList), SPEC-015 (AVLTree), SPEC-016 (RBTree)
- SPEC-012 (Error Model — Result, Error, ErrorCode)
- SPEC-008 (Identity)
- SPEC-019 (Bit Representation — BitSet, BitVector, BitFlags, BitPacking)

### Task 3: Document Runtime Core
- SPEC-006 (Data Blocks), SPEC-007 (Runtime Commands), SPEC-011 (Scheduler)
- SPEC-009 (Runtime API), SPEC-010 (Controller API)

### Task 4: Document Controller Types
- SPEC-020 (Finite State Machines), SPEC-021 (Rule Systems)
- SPEC-022 (Runtime Pipelines), SPEC-023 (Event Loops)

### Task 5: Document Advanced Features
- SPEC-017 (Event System), SPEC-024 (Ephemeral Data Blocks)
- SPEC-018 (Data Layout), SPEC-025 (Semantic Storage Inference)

### Task 6: Document Integration Points
- Map The Seed YAML models to GameAK APIs
- Define code generation contracts for each model type
- Bootstrap integration specification

## Status: READY
