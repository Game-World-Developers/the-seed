package generators

import (
	"fmt"
	"os"
	"path/filepath"
)

type ProblemLevel string

const (
	ProblemError   ProblemLevel = "error"
	ProblemWarning ProblemLevel = "warning"
)

type Problem struct {
	Level     ProblemLevel
	ModelType string
	ModelName string
	Message   string
}

type ModelDetail struct {
	Type         string
	Name         string
	Namespace    string
	YAML         string
	SyncStatus   string
	Dependents   []ModelRef
	Dependencies []ModelRef
}

func buildRegistry() (*ModelRegistry, error) {
	models, err := scanAllModels()
	if err != nil {
		return nil, err
	}
	reg := NewModelRegistry()
	for _, m := range models {
		reg.Register(m.Type, m.Name, m.Namespace)
	}
	return reg, nil
}

func GetSyncStatus(compType, name string) string {
	path := modelPath(compType, name)
	yamlInfo, err := os.Stat(path)
	if err != nil {
		return "missing"
	}

	model, err := readModel(compType, path)
	if err != nil {
		return "error"
	}

	outDir := filepath.Join(".", "Include", model.ModelNamespace())
	hppPath := filepath.Join(outDir, name+".hpp")
	if model.HasParts() {
		parts, _ := model.ExtractParts()
		if len(parts) > 0 {
			hppPath = filepath.Join(outDir, name, parts[0]+".hpp")
		}
	}

	hppInfo, err := os.Stat(hppPath)
	if os.IsNotExist(err) {
		return "missing"
	}
	if hppInfo.ModTime().Before(yamlInfo.ModTime()) {
		return "stale"
	}
	return "ok"
}

func ReadModelYAML(compType, name string) (string, error) {
	path := modelPath(compType, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetModelDetail(compType, name, ns string) (*ModelDetail, error) {
	yaml, err := ReadModelYAML(compType, name)
	if err != nil {
		return nil, err
	}

	syncStatus := GetSyncStatus(compType, name)

	dependents, err := GetDependents(compType, name)
	if err != nil {
		return nil, err
	}

	dependencies, err := GetDependencies(compType, name)
	if err != nil {
		return nil, err
	}

	return &ModelDetail{
		Type:         compType,
		Name:         name,
		Namespace:    ns,
		YAML:         yaml,
		SyncStatus:   syncStatus,
		Dependents:   dependents,
		Dependencies: dependencies,
	}, nil
}

func GetDependents(compType, name string) ([]ModelRef, error) {
	models, err := scanAllModels()
	if err != nil {
		return nil, err
	}

	var refs []ModelRef

	for _, m := range models {
		model, err := readModel(m.Type, m.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not read %s: %v\n", m.Path, err)
			continue
		}

		switch v := model.(type) {
		case *TraitModel:
			for _, c := range v.Components {
				if compType == "component" && c.Name == name {
					refs = append(refs, ModelRef{Name: m.Name, Namespace: m.Namespace})
				}
			}
		case *EntityModel:
			for _, t := range v.Traits {
				if compType == "trait" && t.Name == name {
					refs = append(refs, ModelRef{Name: m.Name, Namespace: m.Namespace})
				}
			}
		case *ArchetypeModel:
			if compType == "entity" && v.Entity.Name == name {
				refs = append(refs, ModelRef{Name: m.Name, Namespace: m.Namespace})
			}
		case *StateMachineModel:
			if compType == "entity" && v.Entity.Name == name {
				refs = append(refs, ModelRef{Name: m.Name, Namespace: m.Namespace})
			}
		case *SystemModel:
			for _, e := range v.Entities {
				if compType == "entity" && e.Name == name {
					refs = append(refs, ModelRef{Name: m.Name, Namespace: m.Namespace})
				}
			}
		}
	}

	return refs, nil
}

func GetDependencies(compType, name string) ([]ModelRef, error) {
	path := modelPath(compType, name)
	model, err := readModel(compType, path)
	if err != nil {
		return nil, err
	}

	var refs []ModelRef

	switch v := model.(type) {
	case *TraitModel:
		refs = append(refs, v.Components...)
	case *EntityModel:
		refs = append(refs, v.Traits...)
	case *ArchetypeModel:
		if v.Entity.Name != "" {
			refs = append(refs, v.Entity)
		}
	case *StateMachineModel:
		if v.Entity.Name != "" {
			refs = append(refs, v.Entity)
		}
		refs = append(refs, v.Events...)
	case *SystemModel:
		refs = append(refs, v.Entities...)
		refs = append(refs, v.Access...)
	}

	// Deduplicate
	seen := map[string]bool{}
	var unique []ModelRef
	for _, r := range refs {
		key := r.Namespace + ":" + r.Name
		if r.Name != "" && !seen[key] {
			seen[key] = true
			unique = append(unique, r)
		}
	}

	return unique, nil
}

func GetProblems() ([]Problem, error) {
	models, err := scanAllModels()
	if err != nil {
		return nil, err
	}

	reg := NewModelRegistry()
	for _, m := range models {
		reg.Register(m.Type, m.Name, m.Namespace)
	}

	compUsers := map[string][]string{}
	traitUsers := map[string][]string{}
	entityUsers := map[string][]string{}
	systemUsers := map[string][]string{}
	eventUsers := map[string][]string{}

	for _, m := range models {
		model, err := readModel(m.Type, m.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not read %s: %v\n", m.Path, err)
			continue
		}

		switch v := model.(type) {
		case *TraitModel:
			for _, c := range v.Components {
				compUsers[c.Name] = append(compUsers[c.Name], m.Name)
			}
		case *EntityModel:
			for _, t := range v.Traits {
				traitUsers[t.Name] = append(traitUsers[t.Name], m.Name)
			}
		case *ArchetypeModel:
			entityUsers[v.Entity.Name] = append(entityUsers[v.Entity.Name], m.Name)
		case *StateMachineModel:
			if v.Entity.Name != "" {
				entityUsers[v.Entity.Name] = append(entityUsers[v.Entity.Name], m.Name)
			}
			for _, ev := range v.Events {
				eventUsers[ev.Name] = append(eventUsers[ev.Name], m.Name)
			}
		case *SystemModel:
			for _, e := range v.Entities {
				entityUsers[e.Name] = append(entityUsers[e.Name], m.Name)
			}
			for _, c := range v.Access {
				systemUsers[c.Name] = append(systemUsers[c.Name], m.Name)
			}
		}
	}

	modelExists := func(t, n string) bool {
		for _, m := range models {
			if m.Type == t && m.Name == n {
				return true
			}
		}
		return false
	}

	var problems []Problem

	for _, m := range models {
		model, err := readModel(m.Type, m.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not read %s: %v\n", m.Path, err)
			continue
		}

		syncStatus := GetSyncStatus(m.Type, m.Name)

		switch m.Type {
		case "component":
			if len(compUsers[m.Name]) == 0 && len(systemUsers[m.Name]) == 0 {
				problems = append(problems, Problem{
					Level: ProblemWarning, ModelType: m.Type, ModelName: m.Name,
					Message: "Component is not used by any Trait or System",
				})
			}

		case "trait":
			trait := model.(*TraitModel)
			for _, c := range trait.Components {
				if !modelExists("component", c.Name) {
					problems = append(problems, Problem{
						Level: ProblemError, ModelType: m.Type, ModelName: m.Name,
						Message: fmt.Sprintf("References missing Component %q in namespace %q", c.Name, c.Namespace),
					})
				}
			}
			if len(traitUsers[m.Name]) == 0 {
				problems = append(problems, Problem{
					Level: ProblemWarning, ModelType: m.Type, ModelName: m.Name,
					Message: "Trait is not used by any Entity",
				})
			}

		case "entity":
			entity := model.(*EntityModel)
			for _, t := range entity.Traits {
				if !modelExists("trait", t.Name) {
					problems = append(problems, Problem{
						Level: ProblemError, ModelType: m.Type, ModelName: m.Name,
						Message: fmt.Sprintf("References missing Trait %q in namespace %q", t.Name, t.Namespace),
					})
				}
			}
			if len(entityUsers[m.Name]) == 0 {
				problems = append(problems, Problem{
					Level: ProblemWarning, ModelType: m.Type, ModelName: m.Name,
					Message: "Entity is not referenced by any Archetype, State Machine, or System",
				})
			}

		case "archetype":
			arch := model.(*ArchetypeModel)
			if arch.Entity.Name != "" && !modelExists("entity", arch.Entity.Name) {
				problems = append(problems, Problem{
					Level: ProblemError, ModelType: m.Type, ModelName: m.Name,
					Message: fmt.Sprintf("References missing Entity %q in namespace %q", arch.Entity.Name, arch.Entity.Namespace),
				})
			}

		case "state_machine":
			sm := model.(*StateMachineModel)
			if sm.Entity.Name != "" && !modelExists("entity", sm.Entity.Name) {
				problems = append(problems, Problem{
					Level: ProblemError, ModelType: m.Type, ModelName: m.Name,
					Message: fmt.Sprintf("References missing Entity %q in namespace %q", sm.Entity.Name, sm.Entity.Namespace),
				})
			}
			for _, ev := range sm.Events {
				if !modelExists("event", ev.Name) {
					problems = append(problems, Problem{
						Level: ProblemError, ModelType: m.Type, ModelName: m.Name,
						Message: fmt.Sprintf("References missing Event %q in namespace %q", ev.Name, ev.Namespace),
					})
				}
			}

		case "system":
			sys := model.(*SystemModel)
			for _, e := range sys.Entities {
				if !modelExists("entity", e.Name) {
					problems = append(problems, Problem{
						Level: ProblemError, ModelType: m.Type, ModelName: m.Name,
						Message: fmt.Sprintf("References missing Entity %q in namespace %q", e.Name, e.Namespace),
					})
				}
			}
			for _, c := range sys.Access {
				if !modelExists("component", c.Name) {
					problems = append(problems, Problem{
						Level: ProblemError, ModelType: m.Type, ModelName: m.Name,
						Message: fmt.Sprintf("References missing Component %q in namespace %q", c.Name, c.Namespace),
					})
				}
			}
		}

		if syncStatus == "stale" {
			problems = append(problems, Problem{
				Level: ProblemWarning, ModelType: m.Type, ModelName: m.Name,
				Message: "Generated C++ is out of date (run seed sync)",
			})
		} else if syncStatus == "missing" {
			problems = append(problems, Problem{
				Level: ProblemWarning, ModelType: m.Type, ModelName: m.Name,
				Message: "No C++ header generated (run seed sync)",
			})
		}
	}

	// ── Cycle detection (cross-type) ─────────────────────────────────
	for _, m := range models {
		if m.Type != "trait" && m.Type != "entity" {
			continue
		}
		visited := map[string]bool{}
		var detectCycle func(currentType, currentName string, path []string) bool
		detectCycle = func(ct, cn string, path []string) bool {
			key := ct + ":" + cn
			for _, p := range path {
				if p == key {
					return true
				}
			}
			if visited[key] {
				return false
			}
			visited[key] = true
			path = append(path, key)

			for _, candidate := range models {
				if candidate.Type != "trait" && candidate.Type != "entity" {
					continue
				}
				cm, err := readModel(candidate.Type, candidate.Path)
				if err != nil {
					continue
				}
				switch v := cm.(type) {
				case *TraitModel:
					for _, c := range v.Components {
						if c.Name == cn && ct == "component" {
							// Trait -> Component (end of chain for cycles through components)
						}
					}
				case *EntityModel:
					for _, t := range v.Traits {
						if t.Name == cn && ct == "trait" {
							if detectCycle("entity", candidate.Name, path) {
								return true
							}
						}
					}
				}
			}
			return false
		}
		if detectCycle(m.Type, m.Name, nil) {
			problems = append(problems, Problem{
				Level: ProblemError, ModelType: m.Type, ModelName: m.Name,
				Message: "Circular dependency detected via Trait/Entity chain",
			})
		}
	}

	return problems, nil
}
