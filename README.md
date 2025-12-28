# The Seed

> The Seed is a high performance, deterministic runtime SDK for building 3D Games and virtual worlds.

It's not a monolithic **game-engine** or **editor**.

The Seed is a distributed as a set of _binaries_, _libraries_, and _tools_ that provides the core runtime, rendering audio, event-execution, script integration, and platform abstraction required to create interactive worlds.

This project is highly inspired on [World-Seed](https://swordartonline.fandom.com/wiki/World_Seed) of [Sword Art Online](https://pt.wikipedia.org/wiki/Sword_Art_Online)

## Core

The core is written on C++ with strong focus on _performance_, _determinism_, and a long-term stability.

Scripting Languages is used to describe rules, events and configurations only at startup - never registering heavy - logic or adopt real-time execution.

## Concept

The Seed is designed to be: - Deterministic and Fast as **Default** - Modular and linkable as **Runtime SDK** - Suitable for both **client** and **headless/server** execution - Fully multiplatform

Keep "The-Seed" as agnostic to gameplay, narrative, and World-Specific logic.
