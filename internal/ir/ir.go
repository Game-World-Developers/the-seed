// Package ir defines Seed's inspectable intermediate representation: a
// versioned, resolved snapshot of a project's compiled models, built from
// generators.CompileResult and independent of both the source YAML and the
// GameAK/SDL3 backend that eventually consumes it.
//
// Unlike the YAML models, every reference in the IR is a fully qualified
// Ref (namespace + name) rather than a possibly-ambiguous bare string —
// see docs/semantics.md §5. The IR is produced by Build and is safe to
// serialize (json tags on every type) for editor and CI tooling.
package ir

import (
	"fmt"
	"sort"

	"Game-Developers-World/seed/internal/generators"
)

// Version identifies the shape of this package's types. Bump the patch
// component for documentation-only changes, the minor component for
// additive, backward-compatible fields, and the major component for any
// breaking change to an existing field's meaning or type. Tooling reading
// a machine-readable IR dump should reject a major version it does not
// recognize rather than guess at a mapping.
const Version = "0.1.0"

// Ref is a fully qualified pointer to a declared model: the only form of
// reference that appears in the IR. Unlike generators.ModelRef, a Ref's
// Namespace is always resolved — it is never empty when the referenced
// name exists, because Build refuses to run over a CompileResult that has
// errors.
type Ref struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// Qualified renders the reference the same way generated C++ namespaces it.
func (r Ref) Qualified() string { return r.Namespace + "::" + r.Name }

func lessRef(a, b Ref) bool {
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	return a.Name < b.Name
}

// Field is a resolved (name, type) slot inside a Component or Event. Type
// is passed through verbatim from the model — the IR does not yet carry a
// formal Seed type system, matching the gap documented in
// docs/semantics.md and internal/generators/semantic.go's validateFields.
type Field struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func convertFields(fields []generators.FieldDef) []Field {
	out := make([]Field, 0, len(fields))
	for _, f := range fields {
		out = append(out, Field{Name: f.Name, Type: f.Type})
	}
	return out
}

type Component struct {
	Ref
	Fields []Field `json:"fields"`
}

type Trait struct {
	Ref
	Components []Ref `json:"components"`
}

// Entity is a declared entity kind. Components is the storage requirement
// implied by the entity's traits: the deduplicated, resolved set of every
// component reachable through Traits — this is what a GameAK archetype
// signature needs, computed once here instead of re-derived by every
// backend consumer.
type Entity struct {
	Ref
	Traits     []Ref `json:"traits"`
	Components []Ref `json:"components"`
}

type Archetype struct {
	Ref
	Entity Ref `json:"entity"`
}

// Transition is a resolved state-machine transition. Event is set only
// when the transition's event string matched one of the machine's declared
// Events; EventName always carries the raw string. This mirrors the known
// schema gap in docs/semantics.md §5: transition event/target fields are
// plain strings in YAML, not typed ModelRefs, so a transition can name an
// event the IR cannot resolve even though Compile's state validation
// already guarantees Target names an existing state.
type Transition struct {
	EventName string `json:"event_name"`
	Event     *Ref   `json:"event,omitempty"`
	Target    string `json:"target"`
}

type State struct {
	Name        string       `json:"name"`
	Transitions []Transition `json:"transitions"`
}

type StateMachine struct {
	Ref
	Entity   Ref     `json:"entity"`
	Initial  string  `json:"initial"`
	States   []State `json:"states"`
	Events   []Ref   `json:"events"`
	Priority int     `json:"priority"`
}

type Event struct {
	Ref
	Fields []Field `json:"fields"`
}

// System is a resolved system declaration: its query (Entities + Access)
// and its scheduling identity (Priority). Access carries no read/write
// mode and Emits is always empty today — both are the Phase 2 target
// semantics from docs/semantics.md §8 that have not been implemented in
// the YAML schema yet, so the IR cannot report what the schema does not
// carry. They are represented as empty rather than omitted so IR
// consumers can rely on the field always being present.
type System struct {
	Ref
	Entities []Ref `json:"entities"`
	Access   []Ref `json:"access"`
	Priority int   `json:"priority"`
	Emits    []Ref `json:"emits"`
}

type Asset struct {
	Ref
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type Block struct {
	Ref
	TileIndex   uint8   `json:"tile_index"`
	Solid       bool    `json:"solid"`
	Transparent bool    `json:"transparent"`
	Fluid       bool    `json:"fluid"`
	Hardness    float64 `json:"hardness"`
	Drop        string  `json:"drop"`
	AtlasPath   string  `json:"atlas_path,omitempty"`
}

// ScheduleEntry is one system's position in deterministic execution order,
// matching GameAK's actual controller ordering
// (Runtime::execute_single_tick stable-sorts by `a.priority > b.priority`,
// i.e. higher priority runs first — see docs/gameak-mapping.md): highest
// Priority first, then declaration order (Ref.Name, since scanAllModels
// already walks Models/System in lexicographic file-name order) — the
// tie-break rule decided in docs/semantics.md §8.
type ScheduleEntry struct {
	Ref
	Priority int `json:"priority"`
}

// IR is the complete resolved snapshot of one project's compiled models.
type IR struct {
	Version       string          `json:"version"`
	Components    []Component     `json:"components"`
	Traits        []Trait         `json:"traits"`
	Entities      []Entity        `json:"entities"`
	Archetypes    []Archetype     `json:"archetypes"`
	StateMachines []StateMachine  `json:"state_machines"`
	Events        []Event         `json:"events"`
	Systems       []System        `json:"systems"`
	Assets        []Asset         `json:"assets"`
	Blocks        []Block         `json:"blocks"`
	Schedule      []ScheduleEntry `json:"schedule"`
}

func resolveRef(compiled *generators.CompileResult, refType string, ref generators.ModelRef) Ref {
	if ref.Name == "" {
		return Ref{}
	}
	ns, ok := compiled.ResolveRef(refType, ref.Name, ref.Namespace)
	if !ok {
		// Build already refuses to run when compiled.HasErrors() is true,
		// so every reference reaching here was validated successfully;
		// this branch only guards against a future caller skipping that check.
		ns = ref.Namespace
	}
	return Ref{Type: refType, Name: ref.Name, Namespace: ns}
}

// Build turns a successful CompileResult into an IR. It returns an error
// if the CompileResult has any Error-level diagnostic — the IR only ever
// represents a fully resolved, valid model set, never a partial one.
func Build(compiled *generators.CompileResult) (*IR, error) {
	if compiled == nil {
		return nil, fmt.Errorf("cannot build IR: nil compile result")
	}
	if compiled.HasErrors() {
		return nil, fmt.Errorf("cannot build IR: %d semantic error(s) found", len(compiled.Errors()))
	}

	traitByRef := map[Ref]*generators.TraitModel{}
	for _, m := range compiled.Models {
		if t, ok := m.(*generators.TraitModel); ok {
			traitByRef[Ref{"trait", t.Name, t.Namespace}] = t
		}
	}

	out := &IR{Version: Version}

	for _, m := range compiled.Models {
		switch v := m.(type) {
		case *generators.ComponentModel:
			out.Components = append(out.Components, Component{
				Ref:    Ref{"component", v.Name, v.Namespace},
				Fields: convertFields(v.Fields),
			})

		case *generators.TraitModel:
			var comps []Ref
			for _, c := range v.Components {
				comps = append(comps, resolveRef(compiled, "component", c))
			}
			out.Traits = append(out.Traits, Trait{
				Ref:        Ref{"trait", v.Name, v.Namespace},
				Components: comps,
			})

		case *generators.EntityModel:
			var traits []Ref
			var storage []Ref
			seen := map[Ref]bool{}
			for _, t := range v.Traits {
				tref := resolveRef(compiled, "trait", t)
				traits = append(traits, tref)
				if tm, ok := traitByRef[tref]; ok {
					for _, c := range tm.Components {
						cref := resolveRef(compiled, "component", c)
						if !seen[cref] {
							seen[cref] = true
							storage = append(storage, cref)
						}
					}
				}
			}
			sort.Slice(storage, func(i, j int) bool { return lessRef(storage[i], storage[j]) })
			out.Entities = append(out.Entities, Entity{
				Ref:        Ref{"entity", v.Name, v.Namespace},
				Traits:     traits,
				Components: storage,
			})

		case *generators.ArchetypeModel:
			out.Archetypes = append(out.Archetypes, Archetype{
				Ref:    Ref{"archetype", v.Name, v.Namespace},
				Entity: resolveRef(compiled, "entity", v.Entity),
			})

		case *generators.StateMachineModel:
			declared := map[string]bool{}
			var events []Ref
			for _, e := range v.Events {
				declared[e.Name] = true
				events = append(events, resolveRef(compiled, "event", e))
			}
			var states []State
			for _, s := range v.States {
				var transitions []Transition
				for _, t := range s.Transitions {
					tr := Transition{EventName: t.Event, Target: t.Target}
					if declared[t.Event] {
						resolved := resolveRef(compiled, "event", generators.ModelRef{Name: t.Event})
						tr.Event = &resolved
					}
					transitions = append(transitions, tr)
				}
				states = append(states, State{Name: s.Name, Transitions: transitions})
			}
			out.StateMachines = append(out.StateMachines, StateMachine{
				Ref:      Ref{"state_machine", v.Name, v.Namespace},
				Entity:   resolveRef(compiled, "entity", v.Entity),
				Initial:  v.Initial,
				States:   states,
				Events:   events,
				Priority: v.Priority,
			})

		case *generators.EventModel:
			out.Events = append(out.Events, Event{
				Ref:    Ref{"event", v.Name, v.Namespace},
				Fields: convertFields(v.Fields),
			})

		case *generators.SystemModel:
			var entities, access []Ref
			for _, e := range v.Entities {
				entities = append(entities, resolveRef(compiled, "entity", e))
			}
			for _, a := range v.Access {
				access = append(access, resolveRef(compiled, "component", a))
			}
			out.Systems = append(out.Systems, System{
				Ref:      Ref{"system", v.Name, v.Namespace},
				Entities: entities,
				Access:   access,
				Priority: v.Priority,
				Emits:    []Ref{},
			})

		case *generators.AssetModel:
			out.Assets = append(out.Assets, Asset{
				Ref:  Ref{"asset", v.Name, v.Namespace},
				Kind: v.Kind,
				Path: v.Path,
			})

		case *generators.BlockModel:
			out.Blocks = append(out.Blocks, Block{
				Ref:         Ref{"block", v.Name, v.Namespace},
				TileIndex:   v.TileIndex,
				Solid:       v.Solid,
				Transparent: v.Transparent,
				Fluid:       v.Fluid,
				Hardness:    v.Hardness,
				Drop:        v.Drop,
				AtlasPath:   v.AtlasPath,
			})
		}
	}

	sortAll(out)
	out.Schedule = buildSchedule(out.Systems)
	return out, nil
}

// buildSchedule orders systems the way GameAK's Runtime actually will:
// highest Priority first (see the ScheduleEntry doc comment), then name as
// the deterministic tie-break decided in docs/semantics.md §8. Systems,
// not just their names, are duplicated here as ScheduleEntry so IR
// consumers have a single place to read final execution order without
// re-sorting Systems themselves.
func buildSchedule(systems []System) []ScheduleEntry {
	schedule := make([]ScheduleEntry, 0, len(systems))
	for _, s := range systems {
		schedule = append(schedule, ScheduleEntry{Ref: s.Ref, Priority: s.Priority})
	}
	sort.SliceStable(schedule, func(i, j int) bool {
		if schedule[i].Priority != schedule[j].Priority {
			return schedule[i].Priority > schedule[j].Priority
		}
		return lessRef(schedule[i].Ref, schedule[j].Ref)
	})
	return schedule
}

// sortAll orders every top-level slice by (namespace, name) so IR output is
// stable and deterministic regardless of filesystem directory-walk order or
// Go map iteration — required for tests and any diffable CI/editor output.
func sortAll(out *IR) {
	sort.Slice(out.Components, func(i, j int) bool { return lessRef(out.Components[i].Ref, out.Components[j].Ref) })
	sort.Slice(out.Traits, func(i, j int) bool { return lessRef(out.Traits[i].Ref, out.Traits[j].Ref) })
	sort.Slice(out.Entities, func(i, j int) bool { return lessRef(out.Entities[i].Ref, out.Entities[j].Ref) })
	sort.Slice(out.Archetypes, func(i, j int) bool { return lessRef(out.Archetypes[i].Ref, out.Archetypes[j].Ref) })
	sort.Slice(out.StateMachines, func(i, j int) bool { return lessRef(out.StateMachines[i].Ref, out.StateMachines[j].Ref) })
	sort.Slice(out.Events, func(i, j int) bool { return lessRef(out.Events[i].Ref, out.Events[j].Ref) })
	sort.Slice(out.Systems, func(i, j int) bool { return lessRef(out.Systems[i].Ref, out.Systems[j].Ref) })
	sort.Slice(out.Assets, func(i, j int) bool { return lessRef(out.Assets[i].Ref, out.Assets[j].Ref) })
	sort.Slice(out.Blocks, func(i, j int) bool { return lessRef(out.Blocks[i].Ref, out.Blocks[j].Ref) })
}
