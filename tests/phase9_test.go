package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPhase9DeveloperExperience(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")
	build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	projDir := filepath.Join(tmpDir, "proj")
	for _, dir := range []string{"Entity", "Event", "StateMachine", "Component", "System"} {
		if err := os.MkdirAll(filepath.Join(projDir, "Models", dir), 0755); err != nil {
			t.Fatal(err)
		}
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
	writeFile("Models/Component/Position.yaml", "type: component\nname: Position\nnamespace: Core\nfields:\n  - name: x\n    type: float\n")
	writeFile("Models/Event/StartMoveEvent.yaml", "type: event\nname: StartMoveEvent\nnamespace: Core\n")
	writeFile("Models/System/MovementSystem.yaml", "type: system\nname: MovementSystem\nnamespace: Core\npriority: 5\naccess:\n  - Position\n")
	writeFile("Models/StateMachine/PlayerFSM.yaml", `type: state_machine
name: PlayerFSM
namespace: Core
entity: PlayerEntity
initial: Idle
events:
  - StartMoveEvent
states:
  - name: Idle
    transitions:
      - event: StartMoveEvent
        target: Running
        priority: 1
  - name: Running
    transitions: []
`)

	run := func(args ...string) (string, error) {
		cmd := exec.Command(binPath, args...)
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	t.Run("top-level generate/destroy aliases behave like model generate/delete", func(t *testing.T) {
		out, err := run("generate", "component", "Alias")
		if err != nil {
			t.Fatalf("generate failed: %v\n%s", err, out)
		}
		if _, statErr := os.Stat(filepath.Join(projDir, "Models", "Component", "Alias.yaml")); statErr != nil {
			t.Fatalf("expected Alias.yaml to be created: %v", statErr)
		}

		out, err = run("destroy", "component", "Alias")
		if err != nil {
			t.Fatalf("destroy failed: %v\n%s", err, out)
		}
		if _, statErr := os.Stat(filepath.Join(projDir, "Models", "Component", "Alias.yaml")); !os.IsNotExist(statErr) {
			t.Fatal("expected Alias.yaml to be removed")
		}
	})

	t.Run("inspect models lists every declaration with its relationships", func(t *testing.T) {
		out, err := run("inspect", "models")
		if err != nil {
			t.Fatalf("inspect models failed: %v\n%s", err, out)
		}
		for _, want := range []string{"Core::PlayerEntity", "Core::MovementSystem", "Core::PlayerFSM", "Core::StartMoveEvent"} {
			if !contains(out, want) {
				t.Fatalf("expected output to contain %q, got: %s", want, out)
			}
		}
	})

	t.Run("inspect schedule shows priority order", func(t *testing.T) {
		out, err := run("inspect", "schedule")
		if err != nil {
			t.Fatalf("inspect schedule failed: %v\n%s", err, out)
		}
		if !contains(out, "[  5] Core::MovementSystem") {
			t.Fatalf("expected priority-ordered schedule line, got: %s", out)
		}
	})

	t.Run("inspect events shows consumers", func(t *testing.T) {
		out, err := run("inspect", "events")
		if err != nil {
			t.Fatalf("inspect events failed: %v\n%s", err, out)
		}
		if !contains(out, "Core::PlayerFSM") {
			t.Fatalf("expected StartMoveEvent's consumer listed, got: %s", out)
		}
	})

	t.Run("inspect storage shows AoS layout note", func(t *testing.T) {
		out, err := run("inspect", "storage")
		if err != nil {
			t.Fatalf("inspect storage failed: %v\n%s", err, out)
		}
		if !contains(out, "layout: AoS") {
			t.Fatalf("expected an AoS layout note, got: %s", out)
		}
	})

	t.Run("inspect cardinal summarizes state machine shape", func(t *testing.T) {
		out, err := run("inspect", "cardinal")
		if err != nil {
			t.Fatalf("inspect cardinal failed: %v\n%s", err, out)
		}
		if !contains(out, "1 with explicit priority") {
			t.Fatalf("expected a priority count, got: %s", out)
		}
		if !contains(out, "no live or recorded run") {
			t.Fatalf("expected the static-only disclaimer, got: %s", out)
		}
	})

	t.Run("explain <type> <name> shows references and a gameak mapping note", func(t *testing.T) {
		out, err := run("explain", "system", "MovementSystem")
		if err != nil {
			t.Fatalf("explain failed: %v\n%s", err, out)
		}
		if !contains(out, "Core::Position") || !contains(out, "gameak:") {
			t.Fatalf("expected references and a gameak note, got: %s", out)
		}
	})

	t.Run("explain transition still works alongside the general form", func(t *testing.T) {
		out, err := run("explain", "transition", "PlayerFSM", "Idle", "Running")
		if err != nil {
			t.Fatalf("explain transition failed: %v\n%s", err, out)
		}
		if !contains(out, "priority: 1") {
			t.Fatalf("expected priority in output, got: %s", out)
		}
	})

	t.Run("explain fails cleanly for an unknown model", func(t *testing.T) {
		out, err := run("explain", "component", "DoesNotExist")
		if err == nil {
			t.Fatalf("expected failure, got: %s", out)
		}
	})
}

func TestFacilityGenerators(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")
	build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	projDir := filepath.Join(tmpDir, "proj")
	if err := os.MkdirAll(filepath.Join(projDir, "Models"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("generate scene scaffolds a self-including hpp/cpp pair", func(t *testing.T) {
		cmd := exec.Command(binPath, "generate", "scene", "TitleScreen")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("generate scene failed: %v\n%s", err, out)
		}
		hpp, err := os.ReadFile(filepath.Join(projDir, "Src", "Game", "Scenes", "TitleScreen.hpp"))
		if err != nil {
			t.Fatal(err)
		}
		cpp, err := os.ReadFile(filepath.Join(projDir, "Src", "Game", "Scenes", "TitleScreen.cpp"))
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(hpp), "seed::Scene MakeTitleScreenScene()") {
			t.Fatalf("expected a factory declaration, got: %s", hpp)
		}
		// The .cpp must include its header by the same relative form
		// bootstrap.cpp already uses ("name.hpp", not "Game/Scenes/name.hpp")
		// — this exact mismatch was a real bug this test caught while
		// implementing the generator (verified against a full xmake build).
		if !contains(string(cpp), `#include "TitleScreen.hpp"`) {
			t.Fatalf("expected a same-directory relative include, got: %s", cpp)
		}
	})

	t.Run("model generate scene is the same command nested under model", func(t *testing.T) {
		cmd := exec.Command(binPath, "model", "generate", "scene", "PauseMenu")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("model generate scene failed: %v\n%s", err, out)
		}
		if _, statErr := os.Stat(filepath.Join(projDir, "Src", "Game", "Scenes", "PauseMenu.hpp")); statErr != nil {
			t.Fatalf("expected PauseMenu.hpp: %v", statErr)
		}
	})

	t.Run("generate asset_pack creates the directory and is idempotent", func(t *testing.T) {
		cmd := exec.Command(binPath, "generate", "asset_pack", "PlayerAssets")
		cmd.Dir = projDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("generate asset_pack failed: %v\n%s", err, out)
		}
		if !contains(string(out), "create") {
			t.Fatalf("expected 'create', got: %s", out)
		}
		info, statErr := os.Stat(filepath.Join(projDir, "Assets", "PlayerAssets"))
		if statErr != nil || !info.IsDir() {
			t.Fatalf("expected Assets/PlayerAssets/ directory: %v", statErr)
		}

		cmd2 := exec.Command(binPath, "generate", "asset_pack", "PlayerAssets")
		cmd2.Dir = projDir
		out2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			t.Fatalf("second generate asset_pack failed: %v\n%s", err2, out2)
		}
		if !contains(string(out2), "identical") {
			t.Fatalf("expected 'identical' on re-run, got: %s", out2)
		}
	})
}

func TestActionableErrorSuggestions(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")
	build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	t.Run("a missing reference suggests the exact 'seed generate' command", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "missing-ref")
		if err := os.MkdirAll(filepath.Join(projDir, "Models", "Trait"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, ".seed_project"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		bad := "type: trait\nname: Movable\nnamespace: Core\ncomponents:\n  - DoesNotExist\n"
		if err := os.WriteFile(filepath.Join(projDir, "Models", "Trait", "Movable.yaml"), []byte(bad), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(binPath, "compile")
		cmd.Dir = projDir
		out, _ := cmd.CombinedOutput()
		if !contains(string(out), "try: seed generate component DoesNotExist") {
			t.Fatalf("expected an actionable suggestion, got: %s", out)
		}
	})

	t.Run("an ambiguous-priority conflict suggests adding a priority field", func(t *testing.T) {
		projDir := filepath.Join(tmpDir, "ambiguous-priority")
		for _, dir := range []string{"Entity", "Event", "StateMachine"} {
			if err := os.MkdirAll(filepath.Join(projDir, "Models", dir), 0755); err != nil {
				t.Fatal(err)
			}
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
		writeFile("Models/Event/GoEvent.yaml", "type: event\nname: GoEvent\nnamespace: Core\n")
		writeFile("Models/StateMachine/Ambiguous.yaml", `type: state_machine
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

		cmd := exec.Command(binPath, "compile")
		cmd.Dir = projDir
		out, _ := cmd.CombinedOutput()
		if !contains(string(out), "try: add a 'priority: N' field") {
			t.Fatalf("expected a priority-field suggestion, got: %s", out)
		}
	})
}
