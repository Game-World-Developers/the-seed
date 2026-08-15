package tests

import (
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/backend/gameak"
	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/ir"
)

func TestGameAKBackendValidate(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("flags components with the same bare name across namespaces", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "component-collision")
		defer inDir(t, dir)()

		writeModel(t, dir, "Component", "Position", "type: component\nname: Position\nnamespace: Core\nfields: []\n")
		writeModel(t, dir, "Component", "PositionPhysics", "type: component\nname: Position\nnamespace: Physics\nfields: []\n")

		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if compiled.HasErrors() {
			t.Fatalf("unexpected semantic errors: %v", compiled.Errors())
		}
		built, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}

		diags := gameak.New().Validate(built)
		found := false
		for _, d := range diags {
			if d.Severity == gameak.SeverityError && contains(d.Message, "rt.define") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a bare-name collision error, got: %v", diags)
		}
	})

	t.Run("flags systems with the same bare name across namespaces", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "system-collision")
		defer inDir(t, dir)()

		writeModel(t, dir, "System", "Gravity", "type: system\nname: Gravity\nnamespace: Core\npriority: 0\n")
		writeModel(t, dir, "System", "GravityPhysics", "type: system\nname: Gravity\nnamespace: Physics\npriority: 0\n")

		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		built, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}

		diags := gameak.New().Validate(built)
		found := false
		for _, d := range diags {
			if d.Severity == gameak.SeverityError && contains(d.Message, "register_Gravity") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a controller-name collision error, got: %v", diags)
		}
	})

	t.Run("warns that events have no GameAK delivery primitive", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "event-warning")
		defer inDir(t, dir)()

		writeModel(t, dir, "Event", "StartMoveEvent", "type: event\nname: StartMoveEvent\nnamespace: Core\n")

		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		built, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}

		diags := gameak.New().Validate(built)
		if len(diags) != 1 || diags[0].Severity != gameak.SeverityWarning {
			t.Fatalf("expected exactly one warning diagnostic, got: %v", diags)
		}
	})

	t.Run("no diagnostics for a namespace-clean domain", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "clean")
		defer inDir(t, dir)()

		writeModel(t, dir, "Component", "Position", "type: component\nname: Position\nnamespace: Core\nfields: []\n")
		writeModel(t, dir, "System", "Gravity", "type: system\nname: Gravity\nnamespace: Core\npriority: 0\naccess:\n  - Position\n")

		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		built, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}

		diags := gameak.New().Validate(built)
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics, got: %v", diags)
		}
	})
}
