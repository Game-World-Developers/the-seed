package generators

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type FieldDef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type ComponentModel struct {
	Name      string     `yaml:"name"`
	Namespace string     `yaml:"namespace"`
	Fields    []FieldDef `yaml:"fields"`
}

type TraitModel struct {
	Name       string   `yaml:"name"`
	Namespace  string   `yaml:"namespace"`
	Components []string `yaml:"components"`
}

type EntityModel struct {
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
	Traits    []string `yaml:"traits"`
}

type ArchetypeModel struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Entity    string `yaml:"entity"`
}

type TransitionDef struct {
	Target string `yaml:"target"`
	Event  string `yaml:"event"`
}

type StateDef struct {
	Name        string          `yaml:"name"`
	Transitions []TransitionDef `yaml:"transitions"`
}

type StateMachineModel struct {
	Name      string     `yaml:"name"`
	Namespace string     `yaml:"namespace"`
	Entity    string     `yaml:"entity"`
	Initial   string     `yaml:"initial"`
	States    []StateDef `yaml:"states"`
}

type SystemModel struct {
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
	Entities  []string `yaml:"entities"`
	Access    []string `yaml:"access"`
}

func readModel(compType, path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	switch compType {
	case "component":
		var m ComponentModel
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return m, nil
	case "trait":
		var m TraitModel
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return m, nil
	case "entity":
		var m EntityModel
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return m, nil
	case "archetype":
		var m ArchetypeModel
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return m, nil
	case "state_machine":
		var m StateMachineModel
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return m, nil
	case "system":
		var m SystemModel
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unknown type: %s", compType)
	}
}

func defaultModel(compType, name string) ([]byte, error) {
	switch compType {
	case "component":
		m := ComponentModel{
			Name:      name,
			Namespace: "Core",
			Fields:    []FieldDef{{Name: "value", Type: "float"}},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "trait":
		m := TraitModel{
			Name:       name,
			Namespace:  "Core",
			Components: []string{},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "entity":
		m := EntityModel{
			Name:      name,
			Namespace: "Core",
			Traits:    []string{},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "archetype":
		m := ArchetypeModel{
			Name:      name,
			Namespace: "Core",
			Entity:    name,
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "state_machine":
		m := StateMachineModel{
			Name:      name,
			Namespace: "Game",
			Entity:    "PlayerEntity",
			Initial:   "Idle",
			States: []StateDef{
				{
					Name: "Idle",
					Transitions: []TransitionDef{
						{Target: "Running", Event: "StartEvent"},
					},
				},
				{
					Name: "Running",
					Transitions: []TransitionDef{
						{Target: "Idle", Event: "StopEvent"},
					},
				},
			},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "system":
		m := SystemModel{
			Name:      name,
			Namespace: "Game",
			Entities:  []string{},
			Access:    []string{},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unknown type: %s", compType)
	}
}
