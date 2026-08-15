package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCardinalCommands(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")

	build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	projDir := filepath.Join(tmpDir, "proj")
	if err := os.MkdirAll(filepath.Join(projDir, "Models", "Entity"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projDir, "Models", "Event"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projDir, "Models", "StateMachine"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	writeFile := func(rel, content string) {
		if err := os.WriteFile(filepath.Join(projDir, rel), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("Models/Entity/PlayerEntity.yaml", "type: entity\nname: PlayerEntity\nnamespace: Core\ntraits: []\n")
	writeFile("Models/Event/StartMoveEvent.yaml", "type: event\nname: StartMoveEvent\nnamespace: Core\n")
	writeFile("Models/StateMachine/PlayerFSM.yaml", `type: state_machine
name: PlayerFSM
namespace: Core
entity: PlayerEntity
initial: Idle
events:
  - StartMoveEvent
states:
  - name: Idle
    entry:
      - on_idle
    transitions:
      - event: StartMoveEvent
        target: Running
        priority: 1
        guard:
          name: has_stamina
  - name: Running
    transitions: []
`)

	run := func(args ...string) (string, error) {
		cmd := exec.Command(binPath, args...)
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	t.Run("inspect loops shows the event's consuming state machine", func(t *testing.T) {
		out, err := run("inspect", "loops")
		if err != nil {
			t.Fatalf("inspect loops failed: %v\n%s", err, out)
		}
		if !contains(out, "Core::StartMoveEvent") || !contains(out, "Core::PlayerFSM") {
			t.Fatalf("expected loop mapping in output, got: %s", out)
		}
	})

	t.Run("inspect fsm shows states, guard, priority, and entry action", func(t *testing.T) {
		out, err := run("inspect", "fsm", "PlayerFSM")
		if err != nil {
			t.Fatalf("inspect fsm failed: %v\n%s", err, out)
		}
		for _, want := range []string{"Idle", "Running", "priority=1", "guard=has_stamina", "entry: on_idle"} {
			if !contains(out, want) {
				t.Fatalf("expected output to contain %q, got: %s", want, out)
			}
		}
	})

	t.Run("inspect fsm on an unknown name fails", func(t *testing.T) {
		out, err := run("inspect", "fsm", "DoesNotExist")
		if err == nil {
			t.Fatalf("expected failure, got: %s", out)
		}
	})

	t.Run("explain transition describes the matched transition", func(t *testing.T) {
		out, err := run("explain", "transition", "PlayerFSM", "Idle", "Running")
		if err != nil {
			t.Fatalf("explain transition failed: %v\n%s", err, out)
		}
		for _, want := range []string{"StartMoveEvent", "priority: 1", "guard: has_stamina", "declared"} {
			if !contains(out, want) {
				t.Fatalf("expected output to contain %q, got: %s", want, out)
			}
		}
	})

	t.Run("explain transition on a nonexistent pair reports no transition", func(t *testing.T) {
		out, err := run("explain", "transition", "PlayerFSM", "Running", "Idle")
		if err != nil {
			t.Fatalf("expected a clean exit reporting no transition, got error: %v\n%s", err, out)
		}
		if !contains(out, "no transition") {
			t.Fatalf("expected 'no transition', got: %s", out)
		}
	})

	traceFile := filepath.Join(tmpDir, "trace.json")
	traceContent := `{
  "version": "0.1.0",
  "entries": [
    {"tick": 1, "loop": "PlayerLoop", "event": "StartMoveEvent", "machine": "Core::PlayerFSM", "previous_state": "Idle", "selected_transition": "Running"}
  ]
}`
	if err := os.WriteFile(traceFile, []byte(traceContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("trace validates and summarizes a well-formed trace file", func(t *testing.T) {
		cmd := exec.Command(binPath, "trace", traceFile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("trace failed: %v\n%s", err, out)
		}
		if !contains(string(out), "1 entries") {
			t.Fatalf("expected a summary mentioning 1 entries, got: %s", out)
		}
	})

	t.Run("replay walks the trace deterministically", func(t *testing.T) {
		cmd := exec.Command(binPath, "replay", traceFile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("replay failed: %v\n%s", err, out)
		}
		if !contains(string(out), "Idle -[StartMoveEvent]-> Running") {
			t.Fatalf("expected the transition line, got: %s", out)
		}
	})

	t.Run("trace rejects an inconsistent file", func(t *testing.T) {
		badFile := filepath.Join(tmpDir, "bad-trace.json")
		bad := `{"version":"0.1.0","entries":[{"tick":5,"machine":"M","previous_state":"A","selected_transition":"B"},{"tick":1,"machine":"M","previous_state":"B","selected_transition":"A"}]}`
		if err := os.WriteFile(badFile, []byte(bad), 0644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binPath, "trace", badFile)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected trace to reject a decreasing-tick file, got: %s", out)
		}
	})
}
