package commands

import (
	"fmt"
	"sort"

	"Game-Developers-World/seed/internal/ir"

	"github.com/spf13/cobra"
)

var inspectModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show every resolved model declaration and its relationships",
	Long: `List every declared model with its fully qualified identity and the
other models it references — all already resolved by the semantic
compiler (generators.Compile) and the IR (internal/ir), so every
reference shown here is a real, validated declaration, never a bare
possibly-dangling YAML string.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		model, err := buildIR()
		if err != nil {
			return err
		}

		line := func(kind, name string, refs []ir.Ref) {
			if len(refs) == 0 {
				fmt.Printf("  %-14s %s\n", kind, name)
				return
			}
			fmt.Printf("  %-14s %s -> %v\n", kind, name, refNames(refs))
		}

		for _, c := range model.Components {
			line("component", c.Qualified(), nil)
		}
		for _, t := range model.Traits {
			line("trait", t.Qualified(), t.Components)
		}
		for _, e := range model.Entities {
			line("entity", e.Qualified(), e.Traits)
		}
		for _, a := range model.Archetypes {
			line("archetype", a.Qualified(), []ir.Ref{a.Entity})
		}
		for _, sm := range model.StateMachines {
			refs := append([]ir.Ref{sm.Entity}, sm.Events...)
			line("state_machine", sm.Qualified(), refs)
		}
		for _, e := range model.Events {
			line("event", e.Qualified(), nil)
		}
		for _, s := range model.Systems {
			refs := append(append([]ir.Ref{}, s.Entities...), s.Access...)
			line("system", s.Qualified(), refs)
		}
		for _, a := range model.Assets {
			line("asset", a.Qualified(), nil)
		}
		for _, b := range model.Blocks {
			line("block", b.Qualified(), nil)
		}
		return nil
	},
}

var inspectScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Show system execution order and priorities",
	Long: `Print the deterministic system execution schedule: highest Priority
first, ties broken by qualified name — matching GameAK's actual
Runtime::execute_single_tick ordering (see docs/gameak-mapping.md §4 and
docs/semantics.md §8's correction). "Phases" and cross-system
dependencies beyond Priority are not part of the schema yet (that
decision remains undelivered — see docs/semantics.md §8), so this is
the complete ordering picture today, not a partial view of a richer one.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		model, err := buildIR()
		if err != nil {
			return err
		}
		if len(model.Schedule) == 0 {
			fmt.Println("No systems declared.")
			return nil
		}
		for _, s := range model.Schedule {
			fmt.Printf("  [%3d] %s\n", s.Priority, s.Qualified())
		}
		return nil
	},
}

var inspectEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show every event's producers, consumers, and delivery model",
	Long: `For each declared Event, list which state machines consume it (a
transition matching its name) and which transitions declare it as a
possible resulting event (Emits). See docs/cardinal.md §5 for why this
is a producer/consumer *declaration* map, not a live delivery report:
GameAK has no generic user-event delivery primitive yet, so nothing here
claims an event was actually dispatched at runtime — only that the
model set declares the relationship.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		model, err := buildIR()
		if err != nil {
			return err
		}

		consumers := map[string][]string{}
		producers := map[string][]string{}
		for _, sm := range model.StateMachines {
			for _, s := range sm.States {
				for _, t := range s.Transitions {
					consumers[t.EventName] = appendUnique(consumers[t.EventName], sm.Qualified())
					for _, emitted := range t.Emits {
						producers[emitted.Qualified()] = appendUnique(producers[emitted.Qualified()], sm.Qualified())
					}
				}
			}
		}

		if len(model.Events) == 0 {
			fmt.Println("No events declared.")
			return nil
		}
		for _, e := range model.Events {
			q := e.Qualified()
			fmt.Printf("%s\n", q)
			if len(e.Fields) > 0 {
				var fields []string
				for _, f := range e.Fields {
					fields = append(fields, f.Name+" "+f.Type)
				}
				fmt.Printf("  fields:    %v\n", fields)
			}
			ps := producers[e.Name]
			sort.Strings(ps)
			cs := consumers[e.Name]
			sort.Strings(cs)
			fmt.Printf("  producers: %v\n", ps)
			fmt.Printf("  consumers: %v\n", cs)
		}
		return nil
	},
}

func appendUnique(list []string, item string) []string {
	for _, existing := range list {
		if existing == item {
			return list
		}
	}
	return append(list, item)
}

var inspectStorageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Show inferred entity storage layouts and their rationale",
	Long: `For each declared Entity, show the deduplicated set of components its
traits require it to store — GameAK's Data Block layout for that entity
(see docs/gameak-mapping.md §1). Every component is laid out AoS
(Array-of-Structs) today: there is no Seed schema field to request
SoA/AoSoA/Archetype layout, even though GameAK supports all four — that
gap, and its performance implications, are documented in
docs/gameak-mapping.md §1, not repeated per-entity here.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		model, err := buildIR()
		if err != nil {
			return err
		}
		if len(model.Entities) == 0 {
			fmt.Println("No entities declared.")
			return nil
		}
		for _, e := range model.Entities {
			fmt.Printf("%s\n", e.Qualified())
			fmt.Printf("  traits:    %v\n", refNames(e.Traits))
			fmt.Printf("  storage:   %v (layout: AoS, the only strategy Seed's schema can request today)\n", refNames(e.Components))
		}
		return nil
	},
}

var inspectCardinalCmd = &cobra.Command{
	Use:   "cardinal",
	Short: "Summarize declared state machines, guards, priorities, and loops",
	Long: `Static summary of Cardinal's declared structure: every state machine's
state/transition/guard/priority shape, and the event -> consumer map
("loops" — see "seed inspect loops"). This is deliberately not "active
loops, pending events, recent decisions" in the live sense the roadmap
describes: nothing in this codebase runs a live GameAK Runtime to
observe (docs/cardinal.md's Known gap). For a *recorded* run's actual
decisions, use "seed trace"/"seed replay" against a trace file — this
command only ever reads the compiled model set, not a live or recorded
execution.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		model, err := buildIR()
		if err != nil {
			return err
		}
		if len(model.StateMachines) == 0 {
			fmt.Println("No state machines declared.")
			return nil
		}
		for _, sm := range model.StateMachines {
			guarded, withPriority := 0, 0
			transitions := 0
			for _, s := range sm.States {
				for _, t := range s.Transitions {
					transitions++
					if t.Guard != nil {
						guarded++
					}
					if t.Priority != 0 {
						withPriority++
					}
				}
			}
			fmt.Printf("%s\n", sm.Qualified())
			fmt.Printf("  entity:      %s\n", sm.Entity.Qualified())
			fmt.Printf("  states:      %d, transitions: %d (%d guarded, %d with explicit priority)\n",
				len(sm.States), transitions, guarded, withPriority)
			fmt.Printf("  events:      %v\n", refNames(sm.Events))
		}
		fmt.Println()
		fmt.Println("(static declarations only — no live or recorded run to summarize; see 'seed trace'/'seed replay')")
		return nil
	},
}

func init() {
	inspectCmd.AddCommand(inspectModelsCmd)
	inspectCmd.AddCommand(inspectScheduleCmd)
	inspectCmd.AddCommand(inspectEventsCmd)
	inspectCmd.AddCommand(inspectStorageCmd)
	inspectCmd.AddCommand(inspectCardinalCmd)
}
