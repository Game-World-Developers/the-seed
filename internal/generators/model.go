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
	default:
		return nil, fmt.Errorf("unknown type: %s", compType)
	}
}
