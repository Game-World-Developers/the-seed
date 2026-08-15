package tests

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/ir"
)

func TestIRBuild(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("resolves entity storage requirements through traits", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "storage")
		defer inDir(t, dir)()

		writeModel(t, dir, "Component", "Position", "type: component\nname: Position\nnamespace: Core\nfields:\n  - name: x\n    type: float\n")
		writeModel(t, dir, "Component", "Health", "type: component\nname: Health\nnamespace: Core\nfields:\n  - name: hp\n    type: int\n")
		writeModel(t, dir, "Trait", "Movable", "type: trait\nname: Movable\nnamespace: Core\ncomponents:\n  - Position\n")
		writeModel(t, dir, "Trait", "Living", "type: trait\nname: Living\nnamespace: Core\ncomponents:\n  - Health\n")
		writeModel(t, dir, "Entity", "PlayerEntity", "type: entity\nname: PlayerEntity\nnamespace: Core\ntraits:\n  - Movable\n  - Living\n")

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
		if len(built.Entities) != 1 {
			t.Fatalf("expected 1 entity, got %d", len(built.Entities))
		}
		entity := built.Entities[0]
		if len(entity.Components) != 2 {
			t.Fatalf("expected entity storage to resolve 2 components via traits, got %d: %v", len(entity.Components), entity.Components)
		}
		if entity.Components[0].Qualified() != "Core::Health" || entity.Components[1].Qualified() != "Core::Position" {
			t.Fatalf("expected deterministic sorted storage [Core::Health, Core::Position], got %v", entity.Components)
		}
	})

	t.Run("Build refuses to run over a compile result with errors", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "refuses")
		defer inDir(t, dir)()

		writeModel(t, dir, "Trait", "Movable", "type: trait\nname: Movable\nnamespace: Core\ncomponents:\n  - Missing\n")

		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if !compiled.HasErrors() {
			t.Fatal("expected a semantic error to set up this test")
		}
		if _, err := ir.Build(compiled); err == nil {
			t.Fatal("expected ir.Build to refuse a CompileResult with errors")
		}
	})

	t.Run("output is deterministic across repeated builds", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "deterministic")
		defer inDir(t, dir)()

		writeModel(t, dir, "Component", "Zebra", "type: component\nname: Zebra\nnamespace: Core\nfields: []\n")
		writeModel(t, dir, "Component", "Apple", "type: component\nname: Apple\nnamespace: Core\nfields: []\n")
		writeModel(t, dir, "System", "ZSystem", "type: system\nname: ZSystem\nnamespace: Core\npriority: 0\naccess:\n  - Zebra\n")
		writeModel(t, dir, "System", "ASystem", "type: system\nname: ASystem\nnamespace: Core\npriority: 0\naccess:\n  - Apple\n")

		var outputs [][]byte
		for i := 0; i < 2; i++ {
			compiled, err := generators.Compile()
			if err != nil {
				t.Fatal(err)
			}
			built, err := ir.Build(compiled)
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(built)
			if err != nil {
				t.Fatal(err)
			}
			outputs = append(outputs, data)
		}
		if string(outputs[0]) != string(outputs[1]) {
			t.Fatalf("expected identical IR JSON across repeated builds, got:\n%s\nvs\n%s", outputs[0], outputs[1])
		}

		// Schedule tie-break: same priority, alphabetical by qualified name.
		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		built, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}
		if len(built.Schedule) != 2 || built.Schedule[0].Name != "ASystem" || built.Schedule[1].Name != "ZSystem" {
			t.Fatalf("expected schedule tie-break to sort ASystem before ZSystem, got %v", built.Schedule)
		}
	})
}
