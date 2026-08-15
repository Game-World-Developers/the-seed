package tests

import (
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/backend/gameak"
	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/ir"
)

func TestCardinalSemantics(t *testing.T) {
	tmpDir := t.TempDir()

	const entity = "type: entity\nname: PlayerEntity\nnamespace: Core\ntraits: []\n"
	const event = "type: event\nname: GoEvent\nnamespace: Core\n"

	t.Run("ambiguous same-priority transitions for the same event are rejected", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "ambiguous")
		defer inDir(t, dir)()
		writeModel(t, dir, "Entity", "PlayerEntity", entity)
		writeModel(t, dir, "Event", "GoEvent", event)
		writeModel(t, dir, "StateMachine", "Ambiguous", `type: state_machine
name: Ambiguous
namespace: Core
entity: PlayerEntity
initial: Idle
states:
  - name: Idle
    transitions:
      - event: GoEvent
        target: Running
      - event: GoEvent
        target: Walking
  - name: Running
    transitions: []
  - name: Walking
    transitions: []
`)
		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !compiled.HasErrors() {
			t.Fatal("expected an ambiguous-transition error")
		}
		found := false
		for _, e := range compiled.Errors() {
			if contains(e.Message, "assign distinct priorities") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a 'distinct priorities' diagnostic, got: %v", compiled.Errors())
		}
	})

	t.Run("distinct priorities resolve the same ambiguity", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "resolved")
		defer inDir(t, dir)()
		writeModel(t, dir, "Entity", "PlayerEntity", entity)
		writeModel(t, dir, "Event", "GoEvent", event)
		writeModel(t, dir, "StateMachine", "Resolved", `type: state_machine
name: Resolved
namespace: Core
entity: PlayerEntity
initial: Idle
states:
  - name: Idle
    transitions:
      - event: GoEvent
        target: Running
        priority: 10
      - event: GoEvent
        target: Walking
        priority: 5
  - name: Running
    transitions: []
  - name: Walking
    transitions: []
`)
		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if compiled.HasErrors() {
			t.Fatalf("unexpected errors: %v", compiled.Errors())
		}
	})

	t.Run("a guard with zero or multiple kinds set is rejected", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "bad-guard")
		defer inDir(t, dir)()
		writeModel(t, dir, "Entity", "PlayerEntity", entity)
		writeModel(t, dir, "Event", "GoEvent", event)
		writeModel(t, dir, "StateMachine", "BadGuard", `type: state_machine
name: BadGuard
namespace: Core
entity: PlayerEntity
initial: Idle
states:
  - name: Idle
    transitions:
      - event: GoEvent
        target: Running
        guard: {}
  - name: Running
    transitions: []
`)
		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !compiled.HasErrors() {
			t.Fatal("expected a malformed-guard error")
		}
		found := false
		for _, e := range compiled.Errors() {
			if contains(e.Message, "exactly one of name, all, any, or not") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a guard-kind diagnostic, got: %v", compiled.Errors())
		}
	})

	t.Run("a composite guard validates recursively", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "composite-guard")
		defer inDir(t, dir)()
		writeModel(t, dir, "Entity", "PlayerEntity", entity)
		writeModel(t, dir, "Event", "GoEvent", event)
		writeModel(t, dir, "StateMachine", "Composite", `type: state_machine
name: Composite
namespace: Core
entity: PlayerEntity
initial: Idle
states:
  - name: Idle
    transitions:
      - event: GoEvent
        target: Running
        guard:
          all:
            - name: has_stamina
            - not:
                name: is_stunned
  - name: Running
    transitions: []
`)
		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if compiled.HasErrors() {
			t.Fatalf("unexpected errors: %v", compiled.Errors())
		}
		built, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}
		guard := built.StateMachines[0].States[0].Transitions[0].Guard
		if guard == nil || guard.Kind != "all" || len(guard.All) != 2 {
			t.Fatalf("expected a resolved all() guard with 2 children, got: %+v", guard)
		}
		if guard.All[1].Kind != "not" || guard.All[1].Not.Name != "is_stunned" {
			t.Fatalf("expected the second child to be not(is_stunned), got: %+v", guard.All[1])
		}
	})

	t.Run("duplicate entry/exit action names are rejected", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "dup-action")
		defer inDir(t, dir)()
		writeModel(t, dir, "Entity", "PlayerEntity", entity)
		writeModel(t, dir, "StateMachine", "DupAction", `type: state_machine
name: DupAction
namespace: Core
entity: PlayerEntity
initial: Idle
states:
  - name: Idle
    entry:
      - on_enter
      - on_enter
    transitions: []
`)
		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !compiled.HasErrors() {
			t.Fatal("expected a duplicate-action error")
		}
	})

	t.Run("gameak backend warns, but does not error, on Cardinal fields", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "backend-warning")
		defer inDir(t, dir)()
		writeModel(t, dir, "Entity", "PlayerEntity", entity)
		writeModel(t, dir, "Event", "GoEvent", event)
		writeModel(t, dir, "StateMachine", "Guarded", `type: state_machine
name: Guarded
namespace: Core
entity: PlayerEntity
initial: Idle
states:
  - name: Idle
    transitions:
      - event: GoEvent
        target: Running
        guard:
          name: has_stamina
  - name: Running
    transitions: []
`)
		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		built, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}
		diags := gameak.New().Validate(built)
		var errs, warns int
		for _, d := range diags {
			if d.Severity == gameak.SeverityError {
				errs++
			} else {
				warns++
			}
		}
		if errs != 0 {
			t.Fatalf("expected no errors, got: %v", diags)
		}
		if warns == 0 {
			t.Fatal("expected at least one warning about unreflected Cardinal fields")
		}
	})
}
