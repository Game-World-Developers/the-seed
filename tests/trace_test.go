package tests

import (
	"encoding/json"
	"testing"

	"Game-Developers-World/seed/internal/trace"
)

func syntheticTrace() *trace.Trace {
	return &trace.Trace{
		Version: trace.Version,
		Entries: []trace.Entry{
			{
				Tick: 1, Loop: "PlayerLoop", Event: "StartMoveEvent", Machine: "Core::PlayerMovementFSM",
				PreviousState:      "Idle",
				EvaluatedGuards:    []trace.GuardEvaluation{{Guard: "has_stamina", Passed: true}},
				SelectedTransition: "Running",
				Commands:           []trace.CommandRecord{{Type: "SetField", Detail: "Position.x = 10"}},
			},
			{
				Tick: 2, Loop: "PlayerLoop", Event: "StopMoveEvent", Machine: "Core::PlayerMovementFSM",
				PreviousState:      "Running",
				SelectedTransition: "Idle",
				ResultingEvents:    []string{"StoppedEvent"},
			},
			{
				Tick: 3, Loop: "PlayerLoop", Event: "StoppedEvent", Machine: "Core::AudioFSM",
				PreviousState:      "Playing",
				SelectedTransition: "Silent",
			},
		},
	}
}

func TestTraceValidate(t *testing.T) {
	t.Run("accepts a causally consistent trace", func(t *testing.T) {
		diags := trace.Validate(syntheticTrace())
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics, got: %v", diags)
		}
	})

	t.Run("round-trips through JSON", func(t *testing.T) {
		tr := syntheticTrace()
		data, err := json.Marshal(tr)
		if err != nil {
			t.Fatal(err)
		}
		var decoded trace.Trace
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if len(trace.Validate(&decoded)) != 0 {
			t.Fatalf("round-tripped trace should still validate cleanly: %v", trace.Validate(&decoded))
		}
	})

	t.Run("rejects a decreasing tick", func(t *testing.T) {
		tr := syntheticTrace()
		tr.Entries[1].Tick = 0
		diags := trace.Validate(tr)
		if len(diags) == 0 {
			t.Fatal("expected a decreasing-tick diagnostic")
		}
	})

	t.Run("rejects a broken previous_state chain", func(t *testing.T) {
		tr := syntheticTrace()
		tr.Entries[1].PreviousState = "SomethingElse"
		diags := trace.Validate(tr)
		if len(diags) == 0 {
			t.Fatal("expected a broken-chain diagnostic")
		}
	})

	t.Run("rejects a rejected entry carrying a selected_transition", func(t *testing.T) {
		tr := syntheticTrace()
		tr.Entries[0].Rejected = true
		tr.Entries[0].RejectReason = "guard failed"
		diags := trace.Validate(tr)
		if len(diags) == 0 {
			t.Fatal("expected a rejected-entry diagnostic")
		}
	})

	t.Run("rejects same-tick event production and consumption", func(t *testing.T) {
		tr := syntheticTrace()
		tr.Entries[1].ResultingEvents = []string{"StoppedEvent"}
		tr.Entries[2].Tick = 2 // consumed same tick it was produced
		diags := trace.Validate(tr)
		found := false
		for _, d := range diags {
			if contains(d.Message, "same-tick delivery") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a same-tick delivery diagnostic, got: %v", diags)
		}
	})
}
