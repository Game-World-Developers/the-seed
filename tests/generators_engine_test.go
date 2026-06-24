package tests

import (
	"os"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/generators"
)

func TestGeneratorsSync(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("generates C++ headers from YAML models", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "sync-full"))()

		err := generators.CreateModel("component", "Position")
		if err != nil {
			t.Fatal(err)
		}

		err = generators.CreateModel("trait", "Movable")
		if err != nil {
			t.Fatal(err)
		}

		err = generators.CreateModel("entity", "PlayerEntity")
		if err != nil {
			t.Fatal(err)
		}

		err = generators.CreateModel("state_machine", "PlayerController")
		if err != nil {
			t.Fatal(err)
		}

		err = generators.CreateModel("system", "Gravity")
		if err != nil {
			t.Fatal(err)
		}

		err = generators.Sync()
		if err != nil {
			t.Fatal(err)
		}

		compHpp := filepath.Join(tmpDir, "sync-full", "Include", "Core", "Position.hpp")
		traitHpp := filepath.Join(tmpDir, "sync-full", "Include", "Core", "Movable.hpp")
		smHpp := filepath.Join(tmpDir, "sync-full", "Include", "Core", "PlayerController.hpp")
		sysHpp := filepath.Join(tmpDir, "sync-full", "Include", "Core", "Gravity.hpp")
		sysCpp := filepath.Join(tmpDir, "sync-full", "Src", "Game", "Gravity.cpp")

		_, err = os.Stat(compHpp)
		if err != nil {
			t.Fatal(err)
		}

		_, err = os.Stat(traitHpp)
		if err != nil {
			t.Fatal(err)
		}

		_, err = os.Stat(smHpp)
		if err != nil {
			t.Fatal(err)
		}

		_, err = os.Stat(sysHpp)
		if err != nil {
			t.Fatal(err)
		}

		_, err = os.Stat(sysCpp)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(compHpp)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "struct Position") {
			t.Fatalf("expected header to contain 'struct Position', got: %s", data)
		}
		if !contains(string(data), "rt.define") {
			t.Fatalf("expected header to contain 'rt.define', got: %s", data)
		}

		smData, err := os.ReadFile(smHpp)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(smData), "register_controller") {
			t.Fatalf("expected state machine header to contain 'register_controller', got: %s", smData)
		}
		if !contains(string(smData), "add_transition") {
			t.Fatalf("expected state machine header to contain 'add_transition', got: %s", smData)
		}

		sysData, err := os.ReadFile(sysHpp)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(sysData), "CommandProducer") {
			t.Fatalf("expected system header to contain 'CommandProducer', got: %s", sysData)
		}
		if !contains(string(sysData), "register_controller") {
			t.Fatalf("expected system header to contain 'register_controller', got: %s", sysData)
		}
	})

	t.Run("generates system .cpp only once (does not overwrite)", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "sync-cpp-once"))()

		err := generators.CreateModel("system", "MySystem")
		if err != nil {
			t.Fatal(err)
		}

		err = generators.Sync()
		if err != nil {
			t.Fatal(err)
		}

		cppPath := filepath.Join(tmpDir, "sync-cpp-once", "Src", "Game", "MySystem.cpp")
		data, err := os.ReadFile(cppPath)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "field_span") {
			t.Fatalf("expected .cpp to contain the system skeleton with field_span, got: %s", data)
		}

		// Second sync should not overwrite user code
		err = generators.Sync()
		if err != nil {
			t.Fatal(err)
		}

		data2, err := os.ReadFile(cppPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data2) != string(data) {
			t.Fatal("expected second sync to not overwrite .cpp content")
		}
	})

	t.Run("handles empty model directories", func(t *testing.T) {
		defer inDir(t, filepath.Join(tmpDir, "sync-empty"))()

		err := generators.Sync()
		if err != nil {
			t.Fatal(err)
		}
	})
}
