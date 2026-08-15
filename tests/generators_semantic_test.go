package tests

import (
	"os"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/generators"
)

func containsAny(haystack []string, needle string) bool {
	for _, h := range haystack {
		if contains(h, needle) {
			return true
		}
	}
	return false
}

func writeModel(t *testing.T, root, typeDir, name, content string) {
	t.Helper()
	dir := filepath.Join(root, "Models", typeDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSemanticCompile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid domain produces no error diagnostics", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "valid")
		defer inDir(t, dir)()

		writeModel(t, dir, "Component", "Position", "type: component\nname: Position\nnamespace: Core\nfields:\n  - name: x\n    type: float\n")
		writeModel(t, dir, "Trait", "Movable", "type: trait\nname: Movable\nnamespace: Core\ncomponents:\n  - Position\n")
		writeModel(t, dir, "Entity", "PlayerEntity", "type: entity\nname: PlayerEntity\nnamespace: Core\ntraits:\n  - Movable\n")
		writeModel(t, dir, "Event", "StartMoveEvent", "type: event\nname: StartMoveEvent\nnamespace: Core\n")
		writeModel(t, dir, "StateMachine", "PlayerFSM", "type: state_machine\nname: PlayerFSM\nnamespace: Core\nentity: PlayerEntity\ninitial: Idle\nevents:\n  - StartMoveEvent\nstates:\n  - name: Idle\n    transitions:\n      - event: StartMoveEvent\n        target: Running\n  - name: Running\n    transitions: []\n")

		result, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if result.HasErrors() {
			t.Fatalf("expected no error diagnostics, got: %v", result.Errors())
		}
		if len(result.Models) != 5 {
			t.Fatalf("expected 5 decoded models, got %d", len(result.Models))
		}
	})

	t.Run("missing reference is reported with a source location", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "missing-ref")
		defer inDir(t, dir)()

		writeModel(t, dir, "Trait", "Movable", "type: trait\nname: Movable\nnamespace: Core\ncomponents:\n  - DoesNotExist\n")

		result, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !result.HasErrors() {
			t.Fatal("expected a missing-reference error")
		}
		errs := result.Errors()
		if errs[0].Location.Field != "components" {
			t.Fatalf("expected diagnostic field 'components', got %q", errs[0].Location.Field)
		}
		if errs[0].Location.ModelName != "Movable" {
			t.Fatalf("expected diagnostic to point at 'Movable', got %q", errs[0].Location.ModelName)
		}
	})

	t.Run("duplicate declaration in the same namespace is an error", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "duplicate")
		defer inDir(t, dir)()

		componentDir := filepath.Join(dir, "Models", "Component")
		if err := os.MkdirAll(componentDir, 0755); err != nil {
			t.Fatal(err)
		}
		content := "type: component\nname: Position\nnamespace: Core\nfields:\n  - name: x\n    type: float\n"
		if err := os.WriteFile(filepath.Join(componentDir, "Position.yaml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(componentDir, "PositionAgain.yaml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !result.HasErrors() {
			t.Fatal("expected a duplicate-declaration error")
		}
		found := false
		for _, e := range result.Errors() {
			if contains(e.Message, "duplicate") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a 'duplicate' diagnostic, got: %v", result.Errors())
		}
	})

	t.Run("ambiguous unqualified reference across namespaces is an error", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "ambiguous")
		defer inDir(t, dir)()

		writeModel(t, dir, "Component", "Position", "type: component\nname: Position\nnamespace: Core\nfields: []\n")
		writeModel(t, dir, "Component", "PositionOther", "type: component\nname: Position\nnamespace: Physics\nfields: []\n")
		writeModel(t, dir, "Trait", "Movable", "type: trait\nname: Movable\nnamespace: Core\ncomponents:\n  - Position\n")

		result, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !result.HasErrors() {
			t.Fatal("expected an ambiguous-reference error")
		}
		found := false
		for _, e := range result.Errors() {
			if contains(e.Message, "ambiguous") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected an 'ambiguous' diagnostic, got: %v", result.Errors())
		}
	})

	t.Run("state machine transition to an undeclared state is an invalid execution dependency", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "bad-transition")
		defer inDir(t, dir)()

		writeModel(t, dir, "Entity", "PlayerEntity", "type: entity\nname: PlayerEntity\nnamespace: Core\ntraits: []\n")
		writeModel(t, dir, "StateMachine", "PlayerFSM",
			"type: state_machine\nname: PlayerFSM\nnamespace: Core\nentity: PlayerEntity\ninitial: Idle\nstates:\n  - name: Idle\n    transitions:\n      - event: Go\n        target: Nowhere\n")

		result, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !result.HasErrors() {
			t.Fatal("expected an invalid-transition-target error")
		}
		found := false
		for _, e := range result.Errors() {
			if contains(e.Message, "undeclared state") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected an 'undeclared state' diagnostic, got: %v", result.Errors())
		}
	})

	t.Run("malformed field declarations are reported", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "bad-fields")
		defer inDir(t, dir)()

		writeModel(t, dir, "Component", "Broken",
			"type: component\nname: Broken\nnamespace: Core\nfields:\n  - name: x\n    type: \"not a type!\"\n  - name: x\n    type: float\n")

		result, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !result.HasErrors() {
			t.Fatal("expected malformed field errors")
		}
		var messages []string
		for _, e := range result.Errors() {
			messages = append(messages, e.Message)
		}
		if !containsAny(messages, "invalid type") {
			t.Fatalf("expected an 'invalid type' diagnostic, got: %v", messages)
		}
		if !containsAny(messages, "declared more than once") {
			t.Fatalf("expected a duplicate-field diagnostic, got: %v", messages)
		}
	})

	t.Run("Sync aborts before generating any C++ when errors exist", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "sync-aborts")
		defer inDir(t, dir)()

		writeModel(t, dir, "Trait", "Movable", "type: trait\nname: Movable\nnamespace: Core\ncomponents:\n  - DoesNotExist\n")

		err := generators.Sync()
		if err == nil {
			t.Fatal("expected Sync to fail on semantic errors")
		}

		if _, statErr := os.Stat(filepath.Join(dir, "Include", "Core", "Movable.hpp")); !os.IsNotExist(statErr) {
			t.Fatal("expected no C++ output to be generated when semantic errors exist")
		}
	})
}
