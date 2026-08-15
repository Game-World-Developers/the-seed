// Package trace defines Seed's causal trace format for Cardinal
// (docs/cardinal.md): the record of every FSM decision — tick, event,
// loop, machine, previous state, evaluated guards, selected transition,
// Commands, and resulting events — needed to explain and deterministically
// replay a run.
//
// No producer exists yet. Cardinal's runtime (guards, actions, the FIFO
// Event Loop integration, and the Runtime's atomic Command application)
// has not been implemented in generated C++ — see docs/cardinal.md's
// "Known gap" note. This package defines the target format and a consumer
// (Validate) that checks a trace's internal consistency, so `seed trace`
// and `seed replay` are ready for when a real producer exists, and so the
// format itself is exercised by tests today rather than only specified in
// prose.
package trace

import "fmt"

// Version identifies the shape of this package's types, following the
// same bump rule as internal/ir.Version: patch for docs-only changes,
// minor for additive fields, major for breaking changes. A `seed replay`
// reading an unrecognized major version should refuse rather than guess.
const Version = "0.1.0"

// GuardEvaluation records one guard's name and whether it passed, in
// evaluation order, for one candidate transition.
type GuardEvaluation struct {
	Guard  string `json:"guard"`
	Passed bool   `json:"passed"`
}

// CommandRecord is a minimal, backend-agnostic record of one Command a
// transition's actions produced — Type matches one of GameAK's Command
// variants (docs/gameak-mapping.md §3: CreateBlock, DestroyBlock,
// SetField, ResizeBlock, ConvertLayout) once Cardinal generates real
// Commands; Detail is a free-form, human-readable summary for `seed
// trace`/`seed explain` output, not a structured payload.
type CommandRecord struct {
	Type   string `json:"type"`
	Detail string `json:"detail,omitempty"`
}

// Entry is one FSM decision at one tick: exactly the fields
// docs/cardinal.md's decision lifecycle calls for. A rejected entry
// (Rejected=true) records why but leaves SelectedTransition empty — the
// state did not change, matching docs/cardinal.md's rejection-observable
// decision.
type Entry struct {
	Tick               uint64            `json:"tick"`
	Loop               string            `json:"loop"`
	Event              string            `json:"event"`
	Machine            string            `json:"machine"`
	PreviousState      string            `json:"previous_state"`
	EvaluatedGuards    []GuardEvaluation `json:"evaluated_guards,omitempty"`
	SelectedTransition string            `json:"selected_transition,omitempty"`
	Rejected           bool              `json:"rejected"`
	RejectReason       string            `json:"reject_reason,omitempty"`
	Commands           []CommandRecord   `json:"commands,omitempty"`
	ResultingEvents    []string          `json:"resulting_events,omitempty"`
}

// Trace is an ordered sequence of Entry records for one run.
type Trace struct {
	Version string  `json:"version"`
	Entries []Entry `json:"entries"`
}

// Diagnostic is a single consistency problem found by Validate.
type Diagnostic struct {
	Index   int
	Message string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("entry[%d]: %s", d.Index, d.Message)
}

// Validate checks a Trace for internal consistency:
//
//   - Tick is non-decreasing across entries (a trace is recorded in tick
//     order; this is what makes replay deterministic).
//   - For entries sharing a Machine, each entry's PreviousState matches
//     that machine's last known state (the SelectedTransition of its most
//     recent non-rejected entry, or its very first PreviousState) — a
//     broken chain means the trace was tampered with, reordered, or
//     produced by a buggy recorder.
//   - A ResultingEvents entry must be consumed (appear as another entry's
//     Event) only at a strictly later tick, never the same tick — matching
//     docs/semantics.md §9's "never same-phase-same-tick delivery" rule,
//     the exact mechanism that prevents accidental infinite same-tick
//     feedback loops.
//
// This is a static, offline check — it says nothing about whether the
// trace matches what a live GameAK Runtime would actually do, since no
// such comparison is possible without a running instrumented Runtime.
func Validate(t *Trace) []Diagnostic {
	var diags []Diagnostic

	lastTick := uint64(0)
	haveTick := false
	lastStateByMachine := map[string]string{}
	haveStateByMachine := map[string]bool{}
	eventTick := map[string][]uint64{} // event name -> ticks it was produced as a resulting event

	for i, e := range t.Entries {
		if haveTick && e.Tick < lastTick {
			diags = append(diags, Diagnostic{i, fmt.Sprintf("tick %d is less than the previous entry's tick %d; trace must be non-decreasing in tick order", e.Tick, lastTick)})
		}
		lastTick = e.Tick
		haveTick = true

		if haveStateByMachine[e.Machine] {
			if e.PreviousState != lastStateByMachine[e.Machine] {
				diags = append(diags, Diagnostic{i, fmt.Sprintf("machine %q previous_state %q does not match its last known state %q", e.Machine, e.PreviousState, lastStateByMachine[e.Machine])})
			}
		}
		haveStateByMachine[e.Machine] = true
		if e.Rejected {
			if e.SelectedTransition != "" {
				diags = append(diags, Diagnostic{i, "rejected entry must not carry a selected_transition"})
			}
			// state unchanged; lastStateByMachine stays as-is (PreviousState, already recorded)
			lastStateByMachine[e.Machine] = e.PreviousState
		} else {
			if e.SelectedTransition == "" {
				diags = append(diags, Diagnostic{i, "non-rejected entry must carry a selected_transition"})
			}
			lastStateByMachine[e.Machine] = e.SelectedTransition
		}

		for _, ev := range e.ResultingEvents {
			eventTick[ev] = append(eventTick[ev], e.Tick)
		}
	}

	for i, e := range t.Entries {
		for _, producedTick := range eventTick[e.Event] {
			if producedTick == e.Tick {
				diags = append(diags, Diagnostic{i, fmt.Sprintf("event %q was both produced and consumed at tick %d; same-tick delivery is disallowed (docs/semantics.md §9)", e.Event, e.Tick)})
			}
		}
	}

	return diags
}
