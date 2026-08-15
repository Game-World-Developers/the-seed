package commands

import (
	"fmt"
	"os"
	"sort"

	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/ir"

	"github.com/spf13/cobra"
)

// buildIR compiles the current project's models and builds the IR,
// printing diagnostics and returning an error if compilation failed. It's
// the shared entrypoint for every read-only inspection command
// (inspect/explain) — none of them generate C++ or need the GameAK
// backend gate `sync` uses, since they only read the resolved model set.
func buildIR() (*ir.IR, error) {
	result, err := generators.Compile()
	if err != nil {
		return nil, fmt.Errorf("compiling models: %w", err)
	}
	for _, d := range result.Diagnostics {
		fmt.Fprintf(os.Stderr, "  %s\n", d)
	}
	if result.HasErrors() {
		return nil, fmt.Errorf("%d semantic error(s) found", len(result.Errors()))
	}
	return ir.Build(result)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect the compiled model set",
	Long:  `Read-only inspection of the current project's resolved models — no C++ is generated.`,
}

var inspectLoopsCmd = &cobra.Command{
	Use:   "loops",
	Short: "Show which state machines consume each declared event",
	Long: `Show, for every declared Event, which state machines reference it in a
transition — the producer/consumer map an "Event Loop" (docs/cardinal.md)
would dispatch through. Seed has no standalone Loop model type yet; this
is computed from the state_machine/event relationships already in the IR.`,
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
		for _, sm := range model.StateMachines {
			seen := map[string]bool{}
			for _, s := range sm.States {
				for _, t := range s.Transitions {
					if seen[t.EventName] {
						continue
					}
					seen[t.EventName] = true
					consumers[t.EventName] = append(consumers[t.EventName], sm.Qualified())
				}
			}
		}

		if len(model.Events) == 0 {
			fmt.Println("No events declared.")
			return nil
		}
		for _, e := range model.Events {
			cs := consumers[e.Name]
			sort.Strings(cs)
			if len(cs) == 0 {
				fmt.Printf("  %s -> (no consumers)\n", e.Qualified())
				continue
			}
			fmt.Printf("  %s -> %v\n", e.Qualified(), cs)
		}
		return nil
	},
}

var inspectFSMCmd = &cobra.Command{
	Use:   "fsm <name>",
	Short: "Show a state machine's states, transitions, guards, and actions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		model, err := buildIR()
		if err != nil {
			return err
		}

		name := args[0]
		var found *ir.StateMachine
		for i := range model.StateMachines {
			sm := &model.StateMachines[i]
			if sm.Name == name || sm.Qualified() == name {
				found = sm
				break
			}
		}
		if found == nil {
			return fmt.Errorf("state machine %q not found", name)
		}

		fmt.Printf("%s (entity: %s, priority: %d, initial: %s)\n",
			found.Qualified(), found.Entity.Qualified(), found.Priority, found.Initial)
		for _, s := range found.States {
			marker := " "
			if s.Name == found.Initial {
				marker = "*"
			}
			fmt.Printf("  %s%s\n", marker, s.Name)
			for _, a := range s.Entry {
				fmt.Printf("      entry: %s\n", a)
			}
			for _, a := range s.Exit {
				fmt.Printf("      exit:  %s\n", a)
			}
			for _, t := range s.Transitions {
				line := fmt.Sprintf("      -> %s  on %s", t.Target, t.EventName)
				if t.Priority != 0 {
					line += fmt.Sprintf("  priority=%d", t.Priority)
				}
				if t.Guard != nil {
					line += "  guard=" + formatGuard(t.Guard)
				}
				if len(t.Emits) > 0 {
					line += fmt.Sprintf("  emits=%v", refNames(t.Emits))
				}
				fmt.Println(line)
			}
		}
		return nil
	},
}

func refNames(refs []ir.Ref) []string {
	names := make([]string, 0, len(refs))
	for _, r := range refs {
		names = append(names, r.Qualified())
	}
	return names
}

func formatGuard(g *ir.Guard) string {
	switch g.Kind {
	case "name":
		return g.Name
	case "not":
		return "not(" + formatGuard(g.Not) + ")"
	case "all":
		return "all(" + formatGuardList(g.All) + ")"
	case "any":
		return "any(" + formatGuardList(g.Any) + ")"
	default:
		return "?"
	}
}

func formatGuardList(gs []*ir.Guard) string {
	out := ""
	for i, g := range gs {
		if i > 0 {
			out += ", "
		}
		out += formatGuard(g)
	}
	return out
}

func init() {
	inspectCmd.AddCommand(inspectLoopsCmd)
	inspectCmd.AddCommand(inspectFSMCmd)
	rootCmd.AddCommand(inspectCmd)
}
