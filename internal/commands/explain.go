package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain a specific piece of the compiled model set",
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
