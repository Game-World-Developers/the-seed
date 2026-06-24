package project

import (
	"fmt"
	"os"
	"path/filepath"

	"Game-Developers-World/seed/internal/generators"

	"gopkg.in/yaml.v3"
)

type starterModel struct {
	Type string
	Name string
	Data any
}

var starterModels = []starterModel{
	// Components (namespaces are resolved automatically during sync)
	// Core domain
	{Type: "component", Name: "Position", Data: generators.ComponentModel{
		Type: "component", Name: "Position", Namespace: "Core",
		Fields: []generators.FieldDef{
			{Name: "x", Type: "float"},
			{Name: "y", Type: "float"},
			{Name: "z", Type: "float"},
		},
	}},
	{Type: "component", Name: "Velocity", Data: generators.ComponentModel{
		Type: "component", Name: "Velocity", Namespace: "Core",
		Fields: []generators.FieldDef{
			{Name: "x", Type: "float"},
			{Name: "y", Type: "float"},
			{Name: "z", Type: "float"},
		},
	}},
	{Type: "component", Name: "Health", Data: generators.ComponentModel{
		Type: "component", Name: "Health", Namespace: "Core",
		Fields: []generators.FieldDef{
			{Name: "current", Type: "int"},
			{Name: "max", Type: "int"},
		},
	}},
	{Type: "component", Name: "Input", Data: generators.ComponentModel{
		Type: "component", Name: "Input", Namespace: "Player",
		Fields: []generators.FieldDef{
			{Name: "move_x", Type: "float"},
			{Name: "move_y", Type: "float"},
			{Name: "action", Type: "uint8_t"},
		},
	}},
	// Traits
	{Type: "trait", Name: "Physical", Data: generators.TraitModel{
		Type: "trait", Name: "Physical", Namespace: "Core",
		Components: []generators.ModelRef{{Name: "Position"}, {Name: "Velocity"}},
	}},
	{Type: "trait", Name: "Damageable", Data: generators.TraitModel{
		Type: "trait", Name: "Damageable", Namespace: "Core",
		Components: []generators.ModelRef{{Name: "Health"}},
	}},
	// Entities
	{Type: "entity", Name: "PlayerEntity", Data: generators.EntityModel{
		Type: "entity", Name: "PlayerEntity", Namespace: "Player",
		Traits: []generators.ModelRef{{Name: "Physical"}, {Name: "Damageable"}},
	}},
	{Type: "entity", Name: "EnemyEntity", Data: generators.EntityModel{
		Type: "entity", Name: "EnemyEntity", Namespace: "Enemy",
		Traits: []generators.ModelRef{{Name: "Physical"}, {Name: "Damageable"}},
	}},
	// Archetypes
	{Type: "archetype", Name: "PlayerArchetype", Data: generators.ArchetypeModel{
		Type: "archetype", Name: "PlayerArchetype", Namespace: "Player",
		Entity: generators.ModelRef{Name: "PlayerEntity"},
	}},
	{Type: "archetype", Name: "EnemyArchetype", Data: generators.ArchetypeModel{
		Type: "archetype", Name: "EnemyArchetype", Namespace: "Enemy",
		Entity: generators.ModelRef{Name: "EnemyEntity"},
	}},
	// Events
	{Type: "event", Name: "MoveEvent", Data: generators.EventModel{
		Type: "event", Name: "MoveEvent", Namespace: "Game",
		Fields: []generators.FieldDef{
			{Name: "x", Type: "float"},
			{Name: "y", Type: "float"},
		},
	}},
	{Type: "event", Name: "JumpEvent", Data: generators.EventModel{
		Type: "event", Name: "JumpEvent", Namespace: "Game",
		Fields: []generators.FieldDef{
			{Name: "force", Type: "float"},
		},
	}},
	{Type: "event", Name: "AttackEvent", Data: generators.EventModel{
		Type: "event", Name: "AttackEvent", Namespace: "Game",
		Fields: []generators.FieldDef{
			{Name: "target_id", Type: "uint32_t"},
		},
	}},
	// State Machines
	{Type: "state_machine", Name: "MenuScene", Data: generators.StateMachineModel{
		Type: "state_machine", Name: "MenuScene", Namespace: "Game",
		Entity:  generators.ModelRef{Name: "PlayerEntity"},
		Initial: "Menu",
		States: []generators.StateDef{
			{Name: "Menu", Transitions: []generators.TransitionDef{
				{Target: "Options", Event: "OptionsSelected"},
				{Target: "Credits", Event: "CreditsSelected"},
			}},
			{Name: "Options", Transitions: []generators.TransitionDef{
				{Target: "Menu", Event: "BackSelected"},
			}},
			{Name: "Credits", Transitions: []generators.TransitionDef{
				{Target: "Menu", Event: "BackSelected"},
			}},
		},
	}},
	{Type: "state_machine", Name: "GameScene", Data: generators.StateMachineModel{
		Type: "state_machine", Name: "GameScene", Namespace: "Game",
		Entity:  generators.ModelRef{Name: "PlayerEntity"},
		Initial: "Playing",
		States: []generators.StateDef{
			{Name: "Playing", Transitions: []generators.TransitionDef{
				{Target: "Paused", Event: "PauseSelected"},
			}},
			{Name: "Paused", Transitions: []generators.TransitionDef{
				{Target: "Playing", Event: "ResumeSelected"},
				{Target: "GameOver", Event: "QuitSelected"},
			}},
			{Name: "GameOver", Transitions: []generators.TransitionDef{
				{Target: "Playing", Event: "RestartSelected"},
			}},
		},
	}},
	{Type: "state_machine", Name: "PlayerController", Data: generators.StateMachineModel{
		Type: "state_machine", Name: "PlayerController", Namespace: "Game",
		Entity:  generators.ModelRef{Name: "PlayerEntity"},
		Initial: "Idle",
		Events:  []generators.ModelRef{{Name: "MoveEvent"}, {Name: "JumpEvent"}, {Name: "AttackEvent"}},
		States: []generators.StateDef{
			{Name: "Idle", Transitions: []generators.TransitionDef{
				{Target: "Running", Event: "MoveInput"},
				{Target: "Jumping", Event: "JumpInput"},
			}},
			{Name: "Running", Transitions: []generators.TransitionDef{
				{Target: "Idle", Event: "StopInput"},
				{Target: "Jumping", Event: "JumpInput"},
				{Target: "Attacking", Event: "AttackInput"},
			}},
			{Name: "Jumping", Transitions: []generators.TransitionDef{
				{Target: "Idle", Event: "Landed"},
				{Target: "Attacking", Event: "AttackInput"},
			}},
			{Name: "Attacking", Transitions: []generators.TransitionDef{
				{Target: "Idle", Event: "AttackComplete"},
			}},
		},
	}},
	// Systems
	{Type: "system", Name: "MovementSystem", Data: generators.SystemModel{
		Type: "system", Name: "MovementSystem", Namespace: "Game",
		Parts: []generators.SystemPart{
			{
				Name:     "ApplyVelocity",
				Entities: []generators.ModelRef{{Name: "PlayerEntity"}, {Name: "EnemyEntity"}},
				Access:   []generators.ModelRef{{Name: "Velocity"}},
			},
			{
				Name:     "UpdatePosition",
				Entities: []generators.ModelRef{{Name: "PlayerEntity"}, {Name: "EnemyEntity"}},
				Access:   []generators.ModelRef{{Name: "Position"}},
			},
		},
	}},
	{Type: "system", Name: "InputSystem", Data: generators.SystemModel{
		Type: "system", Name: "InputSystem", Namespace: "Game",
		Access: []generators.ModelRef{{Name: "Input", Namespace: "Player"}},
	}}, // No Entities — reads SDL events, fills Input component
	{Type: "system", Name: "RenderSystem", Data: generators.SystemModel{
		Type: "system", Name: "RenderSystem", Namespace: "Game",
		Entities: []generators.ModelRef{{Name: "PlayerEntity"}, {Name: "EnemyEntity"}},
		Access:   []generators.ModelRef{{Name: "Position"}},
	}},
	{Type: "system", Name: "PhysicsSystem", Data: generators.SystemModel{
		Type: "system", Name: "PhysicsSystem", Namespace: "Game",
		Entities: []generators.ModelRef{{Name: "PlayerEntity"}, {Name: "EnemyEntity"}},
		Access:   []generators.ModelRef{{Name: "Position"}, {Name: "Velocity"}},
	}},
	{Type: "system", Name: "AudioSystem", Data: generators.SystemModel{
		Type: "system", Name: "AudioSystem", Namespace: "Game",
	}},
	{Type: "system", Name: "DebugSystem", Data: generators.SystemModel{
		Type: "system", Name: "DebugSystem", Namespace: "Game",
	}},
}

func WriteStarterModels(root string, skipExisting bool) error {
	for _, m := range starterModels {
		dir, ok := generators.TypeDir[m.Type]
		if !ok {
			return fmt.Errorf("unknown type %q", m.Type)
		}
		modelsDir := filepath.Join(root, "Models", dir)
		yamlPath := filepath.Join(modelsDir, m.Name+".yaml")

		if skipExisting {
			if _, err := os.Stat(yamlPath); err == nil {
				continue
			}
		}

		data, err := yaml.Marshal(m.Data)
		if err != nil {
			return fmt.Errorf("marshaling %s: %w", m.Name, err)
		}

		if err := os.WriteFile(yamlPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", yamlPath, err)
		}
		logStatus("create", yamlPath)
	}
	return nil
}
