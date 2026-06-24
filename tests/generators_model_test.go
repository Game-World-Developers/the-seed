package tests

import (
	"os"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/generators"
)

func TestGeneratorsCreateModel(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates a component YAML file", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "component"))()

		err := generators.CreateModel("component", "Health")
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(tmpDir, "component", "Models", "Component", "Health.yaml")
		_, err = os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "name: Health") {
			t.Fatalf("expected YAML to contain 'name: Health', got: %s", data)
		}
	})

	t.Run("creates a trait YAML file", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "trait"))()

		err := generators.CreateModel("trait", "Movable")
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(tmpDir, "trait", "Models", "Trait", "Movable.yaml")
		_, err = os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "name: Movable") {
			t.Fatalf("expected YAML to contain 'name: Movable', got: %s", data)
		}
	})

	t.Run("creates an entity YAML file", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "entity"))()

		err := generators.CreateModel("entity", "Player")
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(tmpDir, "entity", "Models", "Entity", "Player.yaml")
		_, err = os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "name: Player") {
			t.Fatalf("expected YAML to contain 'name: Player', got: %s", data)
		}
	})

	t.Run("creates an archetype YAML file", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "archetype"))()

		err := generators.CreateModel("archetype", "PlayerArchetype")
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(tmpDir, "archetype", "Models", "Archetype", "PlayerArchetype.yaml")
		_, err = os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "name: PlayerArchetype") {
			t.Fatalf("expected YAML to contain 'name: PlayerArchetype', got: %s", data)
		}
	})

	t.Run("creates a state_machine YAML file", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "state_machine"))()

		err := generators.CreateModel("state_machine", "PlayerController")
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(tmpDir, "state_machine", "Models", "StateMachine", "PlayerController.yaml")
		_, err = os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "name: PlayerController") {
			t.Fatalf("expected YAML to contain 'name: PlayerController', got: %s", data)
		}
		if !contains(string(data), "Idle") {
			t.Fatalf("expected YAML to contain 'Idle', got: %s", data)
		}
		if !contains(string(data), "Running") {
			t.Fatalf("expected YAML to contain 'Running', got: %s", data)
		}
	})

	t.Run("creates a system YAML file", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "system"))()

		err := generators.CreateModel("system", "Gravity")
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(tmpDir, "system", "Models", "System", "Gravity.yaml")
		_, err = os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "name: Gravity") {
			t.Fatalf("expected YAML to contain 'name: Gravity', got: %s", data)
		}
	})

	t.Run("returns error for unknown type", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "unknown"))()

		err := generators.CreateModel("invalid", "Test")
		if err == nil {
			t.Fatal("expected error for unknown type")
		}
	})

	t.Run("does not error when file already exists", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "exists"))()

		err := generators.CreateModel("component", "Health")
		if err != nil {
			t.Fatal(err)
		}

		err = generators.CreateModel("component", "Health")
		if err != nil {
			t.Fatal(err)
		}
	})
}
