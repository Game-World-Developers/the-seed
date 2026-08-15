package generators

import (
	"fmt"
	"sort"
)

// Severity classifies a Diagnostic as blocking (Error) or informational
// (Warning). Only Error-level diagnostics stop code generation.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// SourceLocation identifies where a diagnostic originates: the file it was
// read from, the declaring model's type/name/namespace, and, when the
// diagnostic concerns a specific reference, the field that holds it.
type SourceLocation struct {
	File      string
	ModelType string
	ModelName string
	Namespace string
	Field     string
}

func (l SourceLocation) String() string {
	loc := l.File
	if l.ModelType != "" && l.ModelName != "" {
		loc = fmt.Sprintf("%s (%s %q)", loc, l.ModelType, l.ModelName)
	}
	if l.Field != "" {
		loc = fmt.Sprintf("%s field %q", loc, l.Field)
	}
	return loc
}

// Diagnostic is a single structured finding produced by Compile. It carries
// enough context to be reported without re-reading the source file.
type Diagnostic struct {
	Severity Severity
	Location SourceLocation
	Message  string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s: %s: %s", d.Severity, d.Location, d.Message)
}

// CompileResult is the output of the semantic compilation pass: every
// successfully decoded model plus every diagnostic collected while
// validating declarations and resolving references.
type CompileResult struct {
	Models      []Model
	Entries     []ModelEntry
	Diagnostics []Diagnostic

	resolver *symbolTable
}

// ResolveRef looks up the namespace that satisfies a (refType, name)
// reference, using the same project-wide symbol table Compile validated
// against. It lets downstream tooling (e.g. the IR builder) turn a
// possibly-unqualified ModelRef into a fully qualified one without
// re-implementing resolution. It only returns ok=true for references that
// were valid at compile time — callers should not call this when
// HasErrors() is true, since some references may be unresolved.
func (r *CompileResult) ResolveRef(refType, name, explicitNamespace string) (namespace string, ok bool) {
	if r.resolver == nil {
		return "", false
	}
	ns, cause := r.resolver.resolve(refType, name, explicitNamespace)
	if cause != nil {
		return "", false
	}
	return ns, true
}

// HasErrors reports whether any diagnostic is Error-level.
func (r *CompileResult) HasErrors() bool {
	return len(r.Errors()) > 0
}

// Errors returns only the Error-level diagnostics.
func (r *CompileResult) Errors() []Diagnostic {
	var out []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			out = append(out, d)
		}
	}
	return out
}

// symbolKey identifies a declaration uniquely by type, name, and namespace —
// see docs/semantics.md §2/§7 for why namespace must be part of the key.
type symbolKey struct{ Type, Name, Namespace string }

// symbolTable is the project-wide table of declared models, used to resolve
// references independently of code generation (docs/semantics.md §5).
type symbolTable struct {
	byKey  map[symbolKey]ModelEntry
	byType map[string]map[string][]string // type -> name -> namespaces declaring it
}

func newSymbolTable() *symbolTable {
	return &symbolTable{
		byKey:  make(map[symbolKey]ModelEntry),
		byType: make(map[string]map[string][]string),
	}
}

// register adds a declaration to the table, returning a diagnostic if a
// model of the same type, name, and namespace was already declared.
func (s *symbolTable) register(e ModelEntry) *Diagnostic {
	key := symbolKey{e.Type, e.Name, e.Namespace}
	if existing, ok := s.byKey[key]; ok {
		return &Diagnostic{
			Severity: SeverityError,
			Location: SourceLocation{File: e.Path, ModelType: e.Type, ModelName: e.Name, Namespace: e.Namespace},
			Message: fmt.Sprintf("duplicate %s %q in namespace %q; also declared in %s",
				e.Type, e.Name, e.Namespace, existing.Path),
		}
	}
	s.byKey[key] = e
	if s.byType[e.Type] == nil {
		s.byType[e.Type] = make(map[string][]string)
	}
	s.byType[e.Type][e.Name] = append(s.byType[e.Type][e.Name], e.Namespace)
	return nil
}

// resolve finds the namespace that satisfies an (refType, name) reference.
// An explicit namespace is checked directly; a bare reference must be
// unambiguous across the whole project or it is rejected rather than
// guessed (docs/semantics.md decision #1).
func (s *symbolTable) resolve(refType, name, explicitNS string) (string, *diagCause) {
	if explicitNS != "" {
		if _, ok := s.byKey[symbolKey{refType, name, explicitNS}]; !ok {
			return "", &diagCause{
				message: fmt.Sprintf("references missing %s %q in namespace %q", refType, name, explicitNS),
			}
		}
		return explicitNS, nil
	}
	namespaces := s.byType[refType][name]
	switch len(namespaces) {
	case 0:
		return "", &diagCause{message: fmt.Sprintf("references missing %s %q", refType, name)}
	case 1:
		return namespaces[0], nil
	default:
		return "", &diagCause{
			message: fmt.Sprintf("reference to %s %q is ambiguous across namespaces %v; qualify it with an explicit namespace",
				refType, name, namespaces),
		}
	}
}

type diagCause struct{ message string }

// compiler holds the mutable state of one Compile run.
type compiler struct {
	table       *symbolTable
	diagnostics []Diagnostic
	edges       []graphEdge
}

type graphEdge struct {
	from symbolKey
	to   symbolKey
}

func (c *compiler) checkRef(entry ModelEntry, refType, field string, ref ModelRef) {
	if ref.Name == "" {
		return
	}
	ns, cause := c.table.resolve(refType, ref.Name, ref.Namespace)
	if cause != nil {
		c.diagnostics = append(c.diagnostics, Diagnostic{
			Severity: SeverityError,
			Location: SourceLocation{File: entry.Path, ModelType: entry.Type, ModelName: entry.Name, Namespace: entry.Namespace, Field: field},
			Message:  cause.message,
		})
		return
	}
	c.edges = append(c.edges, graphEdge{
		from: symbolKey{entry.Type, entry.Name, entry.Namespace},
		to:   symbolKey{refType, ref.Name, ns},
	})
}

func (c *compiler) errorf(entry ModelEntry, field, format string, args ...any) {
	c.diagnostics = append(c.diagnostics, Diagnostic{
		Severity: SeverityError,
		Location: SourceLocation{File: entry.Path, ModelType: entry.Type, ModelName: entry.Name, Namespace: entry.Namespace, Field: field},
		Message:  fmt.Sprintf(format, args...),
	})
}

// Compile decodes every model under Models/, builds a project-wide symbol
// table, resolves cross-model references, and validates duplicates,
// missing references, ambiguous references, and invalid state-machine
// transitions. It performs no code generation and does not mutate the
// decoded models. Diagnostics are returned even when scanning succeeds;
// only a non-nil error indicates the compile pass itself could not run
// (e.g. the Models directory could not be read).
func Compile() (*CompileResult, error) {
	entries, err := scanAllModels()
	if err != nil {
		return nil, err
	}

	c := &compiler{table: newSymbolTable()}

	type decoded struct {
		entry ModelEntry
		model Model
	}
	var models []decoded

	// Decoding phase: turn YAML into typed models. Kept separate from the
	// validation/resolution phase below (docs/semantics.md §5).
	for _, e := range entries {
		m, err := readModel(e.Type, e.Path)
		if err != nil {
			c.diagnostics = append(c.diagnostics, Diagnostic{
				Severity: SeverityError,
				Location: SourceLocation{File: e.Path, ModelType: e.Type, ModelName: e.Name},
				Message:  fmt.Sprintf("failed to decode: %v", err),
			})
			continue
		}
		if diag := c.table.register(e); diag != nil {
			c.diagnostics = append(c.diagnostics, *diag)
			continue
		}
		models = append(models, decoded{e, m})
	}

	// Resolution and validation phase: walk each model's references against
	// the symbol table, independent of code generation.
	for _, d := range models {
		e, m := d.entry, d.model
		switch v := m.(type) {
		case *TraitModel:
			for _, ref := range v.Components {
				c.checkRef(e, "component", "components", ref)
			}
			for _, p := range v.Parts {
				for _, ref := range p.Components {
					c.checkRef(e, "component", fmt.Sprintf("parts[%s].components", p.Name), ref)
				}
			}

		case *ComponentModel:
			c.validateFields(e, v.Fields, "fields")
			for _, p := range v.Parts {
				c.validateFields(e, p.Fields, fmt.Sprintf("parts[%s].fields", p.Name))
			}

		case *EventModel:
			c.validateFields(e, v.Fields, "fields")

		case *EntityModel:
			for _, ref := range v.Traits {
				c.checkRef(e, "trait", "traits", ref)
			}

		case *ArchetypeModel:
			c.checkRef(e, "entity", "entity", v.Entity)

		case *StateMachineModel:
			c.checkRef(e, "entity", "entity", v.Entity)
			for _, ref := range v.Events {
				c.checkRef(e, "event", "events", ref)
			}
			c.validateStates(e, v.Name, v.Initial, v.States, "states")
			for _, p := range v.Parts {
				c.checkRef(e, "entity", fmt.Sprintf("parts[%s].entity", p.Name), p.Entity)
				c.validateStates(e, p.Name, p.Initial, p.States, fmt.Sprintf("parts[%s].states", p.Name))
			}

		case *SystemModel:
			for _, ref := range v.Entities {
				c.checkRef(e, "entity", "entities", ref)
			}
			for _, ref := range v.Access {
				c.checkRef(e, "component", "access", ref)
			}
			for _, p := range v.Parts {
				for _, ref := range p.Entities {
					c.checkRef(e, "entity", fmt.Sprintf("parts[%s].entities", p.Name), ref)
				}
				for _, ref := range p.Access {
					c.checkRef(e, "component", fmt.Sprintf("parts[%s].access", p.Name), ref)
				}
			}
		}
	}

	c.detectCycles()

	sort.SliceStable(c.diagnostics, func(i, j int) bool {
		a, b := c.diagnostics[i], c.diagnostics[j]
		if a.Location.File != b.Location.File {
			return a.Location.File < b.Location.File
		}
		return a.Location.Field < b.Location.Field
	})

	result := &CompileResult{Entries: entries, Diagnostics: c.diagnostics, resolver: c.table}
	for _, d := range models {
		result.Models = append(result.Models, d.model)
	}
	return result, nil
}

// validateStates checks that a state machine's Initial state and every
// transition Target refer to a state actually declared in States — an
// invalid execution dependency the current StateDef/TransitionDef schema
// cannot catch on its own, since transitions are plain strings rather than
// resolved ModelRefs (docs/semantics.md §5's known gap).
func (c *compiler) validateStates(e ModelEntry, machineName, initial string, states []StateDef, field string) {
	if len(states) == 0 {
		return
	}
	declared := make(map[string]bool, len(states))
	for _, s := range states {
		if declared[s.Name] {
			c.errorf(e, field, "state %q is declared more than once in state machine %q", s.Name, machineName)
		}
		declared[s.Name] = true
	}
	if initial != "" && !declared[initial] {
		c.errorf(e, field, "initial state %q is not one of the declared states in state machine %q", initial, machineName)
	}
	for _, s := range states {
		for _, t := range s.Transitions {
			if t.Target != "" && !declared[t.Target] {
				c.errorf(e, field, "transition from %q targets undeclared state %q in state machine %q", s.Name, t.Target, machineName)
			}
		}
	}
}

// validateFields checks that a Component/Event's fields are well-formed:
// non-empty name and type, no duplicate names, and no type string containing
// whitespace or punctuation that cannot be a bare C++ type token.
//
// This is deliberately narrow. FieldDef.Type is emitted verbatim into
// generated C++ (see templates/component.hpp.tmpl) rather than checked
// against a Seed-level type system — Seed has no enumerated set of valid
// field types today, so true "incompatible types" validation (e.g. a System
// writing a float into a bool field) is not yet possible. That requires a
// formal type system decision this document does not make; this check only
// catches malformed declarations, not type mismatches between models.
func (c *compiler) validateFields(e ModelEntry, fields []FieldDef, field string) {
	seen := make(map[string]bool, len(fields))
	for _, f := range fields {
		if f.Name == "" {
			c.errorf(e, field, "field is missing a name")
			continue
		}
		if f.Type == "" {
			c.errorf(e, field, "field %q is missing a type", f.Name)
		} else if !isBareTypeToken(f.Type) {
			c.errorf(e, field, "field %q has an invalid type %q", f.Name, f.Type)
		}
		if seen[f.Name] {
			c.errorf(e, field, "field %q is declared more than once", f.Name)
		}
		seen[f.Name] = true
	}
}

// isBareTypeToken reports whether s could plausibly be a C++ type name:
// letters, digits, underscore, and "::" for namespaced types only.
func isBareTypeToken(s string) bool {
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9', ch == '_':
		case ch == ':':
			if i+1 >= len(s) || s[i+1] != ':' {
				return false
			}
			i++
		default:
			return false
		}
	}
	return len(s) > 0
}

// detectCycles walks the reference graph collected while resolving models
// and flags any cycle. The current schema has no back-references (component
// and event are always leaves), so this only ever fires on malformed or
// future schema extensions — it exists so a cycle is a compile error rather
// than infinite recursion in a downstream tool.
func (c *compiler) detectCycles() {
	adj := make(map[symbolKey][]symbolKey)
	for _, edge := range c.edges {
		adj[edge.from] = append(adj[edge.from], edge.to)
	}

	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := make(map[symbolKey]int)
	var stack []symbolKey

	var visit func(n symbolKey) bool
	visit = func(n symbolKey) bool {
		state[n] = visiting
		stack = append(stack, n)
		for _, next := range adj[n] {
			switch state[next] {
			case visiting:
				return true
			case unvisited:
				if visit(next) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = done
		return false
	}

	// Sort start nodes for deterministic diagnostic ordering.
	var starts []symbolKey
	for k := range adj {
		starts = append(starts, k)
	}
	sort.Slice(starts, func(i, j int) bool {
		if starts[i].Type != starts[j].Type {
			return starts[i].Type < starts[j].Type
		}
		if starts[i].Namespace != starts[j].Namespace {
			return starts[i].Namespace < starts[j].Namespace
		}
		return starts[i].Name < starts[j].Name
	})

	for _, n := range starts {
		if state[n] == unvisited {
			stack = nil
			if visit(n) {
				entry, ok := c.table.byKey[n]
				if !ok {
					continue
				}
				c.errorf(entry, "", "circular dependency detected starting from %s %q in namespace %q", n.Type, n.Name, n.Namespace)
			}
		}
	}
}
