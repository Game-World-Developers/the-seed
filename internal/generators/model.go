package generators

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type FieldDef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type ModelRef struct {
	Name      string
	Namespace string
}

func (r *ModelRef) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err == nil {
		r.Name = s
		return nil
	}
	type raw ModelRef
	return value.Decode((*raw)(r))
}

func (r ModelRef) MarshalYAML() (any, error) {
	if r.Namespace == "" {
		return r.Name, nil
	}
	return map[string]string{"name": r.Name, "namespace": r.Namespace}, nil
}

func (r ModelRef) Qualify() string {
	if r.Namespace == "" {
		return r.Name
	}
	return r.Namespace + "::" + r.Name
}

type ModelTypeName struct {
	Type string
	Name string
}

type ModelRegistry struct {
	data map[ModelTypeName]string
}

func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{data: make(map[ModelTypeName]string)}
}

func (r *ModelRegistry) Register(mType, name, namespace string) {
	r.data[ModelTypeName{mType, name}] = namespace
}

func (r *ModelRegistry) Lookup(mType, name string) (string, bool) {
	ns, ok := r.data[ModelTypeName{mType, name}]
	return ns, ok
}

func (r *ModelRegistry) Resolve(mType, name string) (ModelRef, error) {
	ns, ok := r.Lookup(mType, name)
	if !ok {
		return ModelRef{}, fmt.Errorf("%s %q not found", mType, name)
	}
	return ModelRef{Name: name, Namespace: ns}, nil
}

func (r *ModelRegistry) MustResolve(mType, name string) ModelRef {
	ref, err := r.Resolve(mType, name)
	if err != nil {
		panic(err)
	}
	return ref
}

type (
	ComponentPart struct {
		Name   string     `yaml:"name"`
		Fields []FieldDef `yaml:"fields"`
	}

	TraitPart struct {
		Name       string     `yaml:"name"`
		Components []ModelRef `yaml:"components"`
	}

	StateMachinePart struct {
		Name    string     `yaml:"name"`
		Entity  ModelRef   `yaml:"entity"`
		Initial string     `yaml:"initial"`
		States  []StateDef `yaml:"states"`
	}

	SystemPart struct {
		Name     string     `yaml:"name"`
		Entities []ModelRef `yaml:"entities"`
		Access   []ModelRef `yaml:"access"`
	}
)

type ComponentModel struct {
	Type      string          `yaml:"type"`
	Name      string          `yaml:"name"`
	Namespace string          `yaml:"namespace"`
	Fields    []FieldDef      `yaml:"fields"`
	Parts     []ComponentPart `yaml:"parts,omitempty"`
}

type TraitModel struct {
	Type       string      `yaml:"type"`
	Name       string      `yaml:"name"`
	Namespace  string      `yaml:"namespace"`
	Components []ModelRef  `yaml:"components"`
	Parts      []TraitPart `yaml:"parts,omitempty"`
}

type EntityModel struct {
	Type      string     `yaml:"type"`
	Name      string     `yaml:"name"`
	Namespace string     `yaml:"namespace"`
	Traits    []ModelRef `yaml:"traits"`
}

type ArchetypeModel struct {
	Type      string   `yaml:"type"`
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
	Entity    ModelRef `yaml:"entity"`
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
	Type      string             `yaml:"type"`
	Name      string             `yaml:"name"`
	Namespace string             `yaml:"namespace"`
	Entity    ModelRef           `yaml:"entity"`
	Initial   string             `yaml:"initial"`
	States    []StateDef         `yaml:"states"`
	Parts     []StateMachinePart `yaml:"parts,omitempty"`
	Priority  int                `yaml:"priority,omitempty"`
	Events    []ModelRef         `yaml:"events,omitempty"`
}

type EventModel struct {
	Type      string     `yaml:"type"`
	Name      string     `yaml:"name"`
	Namespace string     `yaml:"namespace"`
	Fields    []FieldDef `yaml:"fields,omitempty"`
}

type AssetModel struct {
	Type      string `yaml:"type"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Kind      string `yaml:"kind"`
	Path      string `yaml:"path"`
}

type SystemModel struct {
	Type      string       `yaml:"type"`
	Name      string       `yaml:"name"`
	Namespace string       `yaml:"namespace"`
	Entities  []ModelRef   `yaml:"entities"`
	Access    []ModelRef   `yaml:"access"`
	Parts     []SystemPart `yaml:"parts,omitempty"`
	Priority  int          `yaml:"priority,omitempty"`
}

type Model interface {
	ModelName() string
	ModelNamespace() string
	ModelType() string
	HasParts() bool
	ExtractParts() ([]string, []map[string]any)
	Resolve(reg *ModelRegistry) error
}

func (m *ComponentModel) ModelName() string      { return m.Name }
func (m *ComponentModel) ModelNamespace() string { return m.Namespace }
func (m *ComponentModel) ModelType() string      { return m.Type }
func (m *ComponentModel) HasParts() bool         { return len(m.Parts) > 0 }
func (m *ComponentModel) ExtractParts() ([]string, []map[string]any) {
	var names []string
	var extras []map[string]any
	for _, p := range m.Parts {
		names = append(names, p.Name)
		extras = append(extras, map[string]any{"Fields": p.Fields})
	}
	return names, extras
}
func (m *ComponentModel) Resolve(reg *ModelRegistry) error { return nil }

func (m *TraitModel) ModelName() string      { return m.Name }
func (m *TraitModel) ModelNamespace() string { return m.Namespace }
func (m *TraitModel) ModelType() string      { return m.Type }
func (m *TraitModel) HasParts() bool         { return len(m.Parts) > 0 }
func (m *TraitModel) ExtractParts() ([]string, []map[string]any) {
	var names []string
	var extras []map[string]any
	for _, p := range m.Parts {
		names = append(names, p.Name)
		extras = append(extras, map[string]any{"Components": p.Components})
	}
	return names, extras
}
func (m *TraitModel) Resolve(reg *ModelRegistry) error {
	for i, c := range m.Components {
		ref, err := reg.Resolve("component", c.Name)
		if err != nil {
			return fmt.Errorf("trait %q: %w", m.Name, err)
		}
		m.Components[i] = ref
	}
	for pi, p := range m.Parts {
		for j, c := range p.Components {
			ref, err := reg.Resolve("component", c.Name)
			if err != nil {
				return fmt.Errorf("trait %q part %q: %w", m.Name, p.Name, err)
			}
			m.Parts[pi].Components[j] = ref
		}
	}
	return nil
}

func (m *EntityModel) ModelName() string                          { return m.Name }
func (m *EntityModel) ModelNamespace() string                     { return m.Namespace }
func (m *EntityModel) ModelType() string                          { return m.Type }
func (m *EntityModel) HasParts() bool                             { return false }
func (m *EntityModel) ExtractParts() ([]string, []map[string]any) { return nil, nil }
func (m *EntityModel) Resolve(reg *ModelRegistry) error {
	for i, t := range m.Traits {
		ref, err := reg.Resolve("trait", t.Name)
		if err != nil {
			return fmt.Errorf("entity %q: %w", m.Name, err)
		}
		m.Traits[i] = ref
	}
	return nil
}

func (m *ArchetypeModel) ModelName() string                          { return m.Name }
func (m *ArchetypeModel) ModelNamespace() string                     { return m.Namespace }
func (m *ArchetypeModel) ModelType() string                          { return m.Type }
func (m *ArchetypeModel) HasParts() bool                             { return false }
func (m *ArchetypeModel) ExtractParts() ([]string, []map[string]any) { return nil, nil }
func (m *ArchetypeModel) Resolve(reg *ModelRegistry) error {
	ref, err := reg.Resolve("entity", m.Entity.Name)
	if err != nil {
		return fmt.Errorf("archetype %q: %w", m.Name, err)
	}
	m.Entity = ref
	return nil
}

func (m *StateMachineModel) ModelName() string      { return m.Name }
func (m *StateMachineModel) ModelNamespace() string { return m.Namespace }
func (m *StateMachineModel) ModelType() string      { return m.Type }
func (m *StateMachineModel) HasParts() bool         { return len(m.Parts) > 0 }
func (m *StateMachineModel) ExtractParts() ([]string, []map[string]any) {
	var names []string
	var extras []map[string]any
	for _, p := range m.Parts {
		names = append(names, p.Name)
		em := map[string]any{
			"Initial": p.Initial,
			"States":  p.States,
		}
		if p.Entity.Name != "" {
			em["Entity"] = p.Entity
		}
		extras = append(extras, em)
	}
	return names, extras
}
func (m *StateMachineModel) Resolve(reg *ModelRegistry) error {
	if m.Entity.Name != "" {
		ref, err := reg.Resolve("entity", m.Entity.Name)
		if err != nil {
			return fmt.Errorf("state_machine %q: %w", m.Name, err)
		}
		m.Entity = ref
	}
	for i, ev := range m.Events {
		ref, err := reg.Resolve("event", ev.Name)
		if err != nil {
			return fmt.Errorf("state_machine %q event: %w", m.Name, err)
		}
		m.Events[i] = ref
	}
	for pi, p := range m.Parts {
		if p.Entity.Name != "" {
			ref, err := reg.Resolve("entity", p.Entity.Name)
			if err != nil {
				return fmt.Errorf("state_machine %q part %q: %w", m.Name, p.Name, err)
			}
			m.Parts[pi].Entity = ref
		}
	}
	return nil
}

func (m *EventModel) ModelName() string                          { return m.Name }
func (m *EventModel) ModelNamespace() string                     { return m.Namespace }
func (m *EventModel) ModelType() string                          { return m.Type }
func (m *EventModel) HasParts() bool                             { return false }
func (m *EventModel) ExtractParts() ([]string, []map[string]any) { return nil, nil }
func (m *EventModel) Resolve(reg *ModelRegistry) error           { return nil }

func (m *AssetModel) ModelName() string                          { return m.Name }
func (m *AssetModel) ModelNamespace() string                     { return m.Namespace }
func (m *AssetModel) ModelType() string                          { return m.Type }
func (m *AssetModel) HasParts() bool                             { return false }
func (m *AssetModel) ExtractParts() ([]string, []map[string]any) { return nil, nil }
func (m *AssetModel) Resolve(reg *ModelRegistry) error           { return nil }

func (m *SystemModel) ModelName() string      { return m.Name }
func (m *SystemModel) ModelNamespace() string { return m.Namespace }
func (m *SystemModel) ModelType() string      { return m.Type }
func (m *SystemModel) HasParts() bool         { return len(m.Parts) > 0 }
func (m *SystemModel) ExtractParts() ([]string, []map[string]any) {
	var names []string
	var extras []map[string]any
	for _, p := range m.Parts {
		names = append(names, p.Name)
		extras = append(extras, map[string]any{
			"Entities": p.Entities,
			"Access":   p.Access,
		})
	}
	return names, extras
}
func (m *SystemModel) Resolve(reg *ModelRegistry) error {
	for i, e := range m.Entities {
		ref, err := reg.Resolve("entity", e.Name)
		if err != nil {
			return fmt.Errorf("system %q entity: %w", m.Name, err)
		}
		m.Entities[i] = ref
	}
	for i, a := range m.Access {
		ref, err := reg.Resolve("component", a.Name)
		if err != nil {
			return fmt.Errorf("system %q access: %w", m.Name, err)
		}
		m.Access[i] = ref
	}
	for pi, p := range m.Parts {
		for j, e := range p.Entities {
			ref, err := reg.Resolve("entity", e.Name)
			if err != nil {
				return fmt.Errorf("system %q part %q entity: %w", m.Name, p.Name, err)
			}
			m.Parts[pi].Entities[j] = ref
		}
		for j, a := range p.Access {
			ref, err := reg.Resolve("component", a.Name)
			if err != nil {
				return fmt.Errorf("system %q part %q access: %w", m.Name, p.Name, err)
			}
			m.Parts[pi].Access[j] = ref
		}
	}
	return nil
}

var modelRegistry = map[string]func() Model{
	"component":     func() Model { return &ComponentModel{} },
	"trait":         func() Model { return &TraitModel{} },
	"entity":        func() Model { return &EntityModel{} },
	"archetype":     func() Model { return &ArchetypeModel{} },
	"state_machine": func() Model { return &StateMachineModel{} },
	"event":         func() Model { return &EventModel{} },
	"system":        func() Model { return &SystemModel{} },
	"asset":         func() Model { return &AssetModel{} },
}

var TypeDir = map[string]string{
	"component":     "Component",
	"trait":         "Trait",
	"entity":        "Entity",
	"archetype":     "Archetype",
	"state_machine": "StateMachine",
	"event":         "Event",
	"system":        "System",
	"asset":         "Asset",
}

type modelHeader struct {
	Type      string `yaml:"type"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

func detectModel(path string) (string, string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", err
	}
	var h modelHeader
	if err := yaml.Unmarshal(data, &h); err != nil {
		return "", "", "", err
	}
	if h.Type == "" {
		return "", "", "", fmt.Errorf("missing 'type' field in %s", path)
	}
	if !validTypes[h.Type] {
		return "", "", "", fmt.Errorf("unknown type %q in %s", h.Type, path)
	}
	return h.Type, h.Name, h.Namespace, nil
}

func readModel(compType, path string) (Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ctor, ok := modelRegistry[compType]
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", compType)
	}
	m := ctor()
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, err
	}
	return m, nil
}

func defaultModel(compType, domain, name string) ([]byte, error) {
	if domain == "" {
		domain = "Core"
	}
	switch compType {
	case "component":
		m := ComponentModel{
			Type: compType, Name: name, Namespace: domain,
			Fields: []FieldDef{{Name: "value", Type: "float"}},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "trait":
		m := TraitModel{
			Type: compType, Name: name, Namespace: domain,
			Components: []ModelRef{},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "entity":
		m := EntityModel{
			Type: compType, Name: name, Namespace: domain,
			Traits: []ModelRef{},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "archetype":
		m := ArchetypeModel{
			Type: compType, Name: name, Namespace: domain,
			Entity: ModelRef{Name: name},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "state_machine":
		m := StateMachineModel{
			Type: compType, Name: name, Namespace: domain,
			Entity: ModelRef{Name: "PlayerEntity"}, Initial: "Idle",
			States: []StateDef{
				{Name: "Idle", Transitions: []TransitionDef{{Target: "Running", Event: "StartEvent"}}},
				{Name: "Running", Transitions: []TransitionDef{{Target: "Idle", Event: "StopEvent"}}},
			},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "event":
		m := EventModel{
			Type: compType, Name: name, Namespace: domain,
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "system":
		m := SystemModel{
			Type: compType, Name: name, Namespace: domain,
			Entities: []ModelRef{}, Access: []ModelRef{},
		}
		out, err := yaml.Marshal(&m)
		if err != nil {
			return nil, err
		}
		return out, nil
	case "asset":
		m := AssetModel{
			Type: compType, Name: name, Namespace: domain,
			Kind: "texture",
			Path: "Assets/Textures/" + name + ".bmp",
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

func modelPath(compType, name string) string {
	return filepath.Join(".", "Models", TypeDir[compType], name+".yaml")
}
