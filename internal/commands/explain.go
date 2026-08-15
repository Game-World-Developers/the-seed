package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/ir"

	"github.com/spf13/cobra"
)

// gameakMappingNote is a one-line pointer to where each model type's
// GameAK mapping is documented in detail (docs/gameak-mapping.md) — kept
// here as a short summary rather than duplicating that document's content,
// consistent with how docs/cardinal.md's "Known gap" notes are referenced
// rather than restated elsewhere in this codebase.
var gameakMappingNote = map[string]string{
	"component":     "-> GameAK BlockTypeDescriptor/DataBlock, AoS layout (docs/gameak-mapping.md §1)",
	"trait":         "-> not a GameAK primitive; expanded into the entity's storage requirement at IR build time",
	"entity":        "-> GameAK archetype signature (its traits' components); see 'seed inspect storage'",
	"archetype":     "-> a spawn preset over the entity's GameAK signature, distinct from GameAK's own 'archetype' term (a layout strategy) — see docs/glossary.md",
	"state_machine": "-> gameak::runtime::Fsm<string,string> + register_controller (docs/gameak-mapping.md §4)",
	"event":         "-> no GameAK delivery primitive yet; only usable as a bare transition-event string (docs/cardinal.md §5)",
	"system":        "-> a GameAK controller via register_controller, Priority/Access mapped directly (docs/gameak-mapping.md §4)",
	"asset":         "-> loaded through seed::AssetManager, not a GameAK concept (docs/facilities.md §4-5)",
	"block":         "-> baked tile/atlas data plus a generated block registry (internal/baker)",
}

var explainCmd = &cobra.Command{
	Use:   "explain <type> <name>",
	Short: "Explain a model's semantic interpretation and GameAK mapping",
	Long: `Explain what a declared model means in Seed's own semantics (its
resolved references, per docs/semantics.md) and how it maps onto GameAK
(per docs/gameak-mapping.md) — the two things "seed explain <model>"
promises, kept separate because they're genuinely different questions:
one about Seed's own model, one about a specific backend's primitives.

"seed explain transition <machine> <from> <to>" is a separate, more
specific subcommand for one transition's detail — see its own --help.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		compType, name := args[0], args[1]

		model, err := buildIR()
		if err != nil {
			return err
		}

		found, refs := findModel(model, compType, name)
		if found == "" {
			return fmt.Errorf("%s %q not found", compType, name)
		}

		fmt.Printf("%s (%s)\n", found, compType)
		if len(refs) > 0 {
			fmt.Printf("  references: %v\n", refNames(refs))
		} else {
			fmt.Println("  references: (none)")
		}
		if note, ok := gameakMappingNote[compType]; ok {
			fmt.Printf("  gameak:     %s\n", note)
		}
		return nil
	},
}

// findModel looks up one model by (type, name) in the IR and returns its
// qualified identity plus the other models it directly references — the
// same relationship data "seed inspect models" prints for everything at
// once, reused here for a single lookup.
func findModel(model *ir.IR, compType, name string) (qualified string, refs []ir.Ref) {
	match := func(n, q string) bool { return n == name || q == name }
	switch compType {
	case "component":
		for _, m := range model.Components {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), nil
			}
		}
	case "trait":
		for _, m := range model.Traits {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), m.Components
			}
		}
	case "entity":
		for _, m := range model.Entities {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), m.Traits
			}
		}
	case "archetype":
		for _, m := range model.Archetypes {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), []ir.Ref{m.Entity}
			}
		}
	case "state_machine":
		for _, m := range model.StateMachines {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), append([]ir.Ref{m.Entity}, m.Events...)
			}
		}
	case "event":
		for _, m := range model.Events {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), nil
			}
		}
	case "system":
		for _, m := range model.Systems {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), append(append([]ir.Ref{}, m.Entities...), m.Access...)
			}
		}
	case "asset":
		for _, m := range model.Assets {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), nil
			}
		}
	case "block":
		for _, m := range model.Blocks {
			if match(m.Name, m.Qualified()) {
				return m.Qualified(), nil
			}
		}
	}
	return "", nil
}

var explainTransitionCmd = &cobra.Command{
	Use:   "transition <machine> <from> <to>",
	Short: "Explain the transition(s) between two states of a state machine",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		machine, from, to := args[0], args[1], args[2]

		model, err := buildIR()
		if err != nil {
			return err
		}

		for _, m := range model.StateMachines {
			if m.Name != machine && m.Qualified() != machine {
				continue
			}

			stateExists := false
			matches := 0
			for _, s := range m.States {
				if s.Name != from {
					continue
				}
				stateExists = true
				for _, t := range s.Transitions {
					if t.Target != to {
						continue
					}
					matches++
					fmt.Printf("%s: %s -[%s]-> %s\n", m.Qualified(), from, t.EventName, to)
					if t.Priority != 0 {
						fmt.Printf("  priority: %d\n", t.Priority)
					}
					if t.Guard != nil {
						fmt.Printf("  guard: %s\n", formatGuard(t.Guard))
					} else {
						fmt.Println("  guard: (none — always eligible when the event matches)")
					}
					if t.Event != nil {
						fmt.Printf("  event: %s (declared)\n", t.Event.Qualified())
					} else {
						fmt.Printf("  event: %q (not declared in this machine's events list)\n", t.EventName)
					}
					if len(t.Emits) > 0 {
						fmt.Printf("  emits: %v\n", refNames(t.Emits))
					}
				}
			}

			switch {
			case !stateExists:
				return fmt.Errorf("state machine %q has no state %q", m.Qualified(), from)
			case matches == 0:
				fmt.Printf("%s: no transition from %q to %q\n", m.Qualified(), from, to)
			}
			return nil
		}
		return fmt.Errorf("state machine %q not found", machine)
	},
}

func init() {
	explainCmd.AddCommand(explainTransitionCmd)
	rootCmd.AddCommand(explainCmd)
}
