// Package gameak is Seed's explicit backend interface for the GameAK
// simulation layer: it consumes internal/ir.IR (never raw YAML or a
// generators.Model) and reports the mapping/compatibility limitations
// GameAK imposes that Seed's own symbol table cannot see, because they
// come from how GameAK's C++ API is generated against, not from Seed's
// own namespace rules.
//
// This package intentionally does not (yet) generate GameAK C++ itself —
// that generation still lives in internal/generators/templates, driven
// directly from decoded YAML models rather than from the IR. Moving that
// generation to run off the IR through this package is future work; see
// docs/gameak-mapping.md's "Known gap" note for why it wasn't done in this
// pass. What this package does today is the other two Phase 5 checklist
// items that don't require that migration: giving GameAK an explicit
// interface over the IR, and reporting GameAK's backend limitations as
// diagnostics instead of letting them fail silently (or fail loudly only
// at C++ compile time, far from the YAML that caused them).
package gameak

import (
	"fmt"
	"sort"

	"Game-Developers-World/seed/internal/ir"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Diagnostic reports a GameAK-specific limitation found in an IR. Unlike
// generators.Diagnostic, it has no file location — the IR carries no
// source position — so it anchors to the resolved Ref instead.
type Diagnostic struct {
	Severity Severity
	Ref      ir.Ref
	Message  string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("gameak %s: %s: %s", d.Severity, d.Ref.Qualified(), d.Message)
}

// Backend is Seed's seam for a simulation backend: something that consumes
// a resolved IR and can report whether it is representable in that
// backend's own primitives. GameAK is the only implementation today; the
// interface exists so a future backend doesn't require touching the IR or
// the semantic compiler to plug in.
type Backend interface {
	// Name identifies the backend, e.g. for diagnostic prefixes or CLI
	// backend-selection flags once more than one backend exists.
	Name() string
	// Validate reports every diagnostic the backend has about model, given
	// its own primitives' constraints. It never mutates model.
	Validate(model *ir.IR) []Diagnostic
}

type backend struct{}

// New returns Seed's GameAK backend.
func New() Backend { return backend{} }

func (backend) Name() string { return "gameak" }

func (backend) Validate(model *ir.IR) []Diagnostic {
	var diags []Diagnostic
	diags = append(diags, checkBareNameCollisions(model)...)
	diags = append(diags, checkEventSupport(model)...)
	sort.SliceStable(diags, func(i, j int) bool {
		return diags[i].Ref.Qualified() < diags[j].Ref.Qualified()
	})
	return diags
}

// checkBareNameCollisions reports Components, Systems, and StateMachines
// whose Name collides with another declaration of the same kind in a
// different namespace. Seed's symbol table treats these as distinct
// (docs/semantics.md §2), but GameAK does not see namespaces at all here:
//
//   - component.hpp.tmpl registers block types with `rt.define("{{.Name}}")`
//     — the bare Name, not a qualified one. Two same-named components in
//     different namespaces silently overwrite each other's block type at
//     runtime registration; whichever runs last wins, with no error.
//   - system.hpp.tmpl and state_machine.hpp.tmpl both generate a global
//     C++ function `register_<Name>` (again bare Name only). Two same-named
//     systems or state machines in different namespaces produce a C++
//     symbol collision — a real compile error, but one that surfaces far
//     from the YAML that caused it and with a confusing linker/compiler
//     message instead of a Seed diagnostic.
//
// This is exactly the kind of GameAK-specific limitation generic model
// analysis cannot see (Seed's own rules allow these names) and that would
// otherwise degrade silently (the block-type case) or fail confusingly
// far downstream (the C++ symbol case) — reporting it here as a Seed
// diagnostic is what Phase 5's "report backend limitations as diagnostics
// instead of silently degrading" asks for.
func checkBareNameCollisions(model *ir.IR) []Diagnostic {
	var diags []Diagnostic

	byName := map[string][]ir.Ref{}
	for _, c := range model.Components {
		byName[c.Name] = append(byName[c.Name], c.Ref)
	}
	for name, refs := range byName {
		if len(refs) > 1 {
			diags = append(diags, collisionDiagnostic(refs, "component", name,
				fmt.Sprintf("GameAK registers block types by bare name (rt.define(%q)); these will silently overwrite each other's registration at runtime", name)))
		}
	}

	byControllerName := map[string][]ir.Ref{}
	for _, s := range model.Systems {
		byControllerName[s.Name] = append(byControllerName[s.Name], s.Ref)
	}
	for _, s := range model.StateMachines {
		byControllerName[s.Name] = append(byControllerName[s.Name], s.Ref)
	}
	for name, refs := range byControllerName {
		if len(refs) > 1 {
			diags = append(diags, collisionDiagnostic(refs, "controller", name,
				fmt.Sprintf("Seed generates a global C++ function register_%s() per declaration; these will collide as duplicate C++ symbols at compile time", name)))
		}
	}

	return diags
}

func collisionDiagnostic(refs []ir.Ref, kind, name, detail string) Diagnostic {
	sort.Slice(refs, func(i, j int) bool { return refs[i].Qualified() < refs[j].Qualified() })
	var qualified []string
	for _, r := range refs {
		qualified = append(qualified, r.Qualified())
	}
	return Diagnostic{
		Severity: SeverityError,
		Ref:      refs[0],
		Message: fmt.Sprintf("%s name %q is declared in %d namespaces (%v); %s",
			kind, name, len(refs), qualified, detail),
	}
}

// checkEventSupport warns that Seed's user-defined Event model has no
// GameAK primitive counterpart yet. GameAK's own EventBus (see
// docs/gameak-mapping.md) is a fixed, low-level lifecycle bus — TickBegin,
// TickEnd, BlockCreated, BlockDestroyed — not a generic typed-event
// mechanism. event.hpp.tmpl only emits a name and an FNV-1a id constant;
// nothing in the generated code delivers the event through GameAK, and
// state_machine.hpp.tmpl's fsm.add_transition call matches events by a
// raw string, never the generated id. This is a real capability gap, not
// a bug to silently work around, and belongs in front of the author as a
// warning rather than discovered only once they wonder why an event
// "does nothing."
func checkEventSupport(model *ir.IR) []Diagnostic {
	var diags []Diagnostic
	for _, e := range model.Events {
		diags = append(diags, Diagnostic{
			Severity: SeverityWarning,
			Ref:      e.Ref,
			Message:  "GameAK has no generic user-event delivery primitive yet; this Event is only usable as a bare transition-event name in a state_machine today, not dispatched through GameAK's EventBus (see docs/gameak-mapping.md)",
		})
	}
	return diags
}
