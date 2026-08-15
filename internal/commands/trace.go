package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"Game-Developers-World/seed/internal/trace"

	"github.com/spf13/cobra"
)

func readTraceFile(path string) (*trace.Trace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading trace file: %w", err)
	}
	var t trace.Trace
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing trace file: %w", err)
	}
	return &t, nil
}

var traceCmd = &cobra.Command{
	Use:   "trace <file>",
	Short: "Validate and summarize a causal trace file",
	Long: `Validate a causal trace file (internal/trace's format — see
docs/cardinal.md) for internal consistency: non-decreasing ticks, an
unbroken previous_state chain per machine, and no same-tick event
production/consumption.

No Seed-generated C++ produces a trace file yet — Cardinal's runtime
(guards, actions, and the FIFO Event Loop integration) hasn't been
implemented in the generated code, only specified. This command exists so
the trace format and its consumer are exercised by real, testable code
ahead of that: point it at a hand-written or externally produced trace
file conforming to the documented schema.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := readTraceFile(args[0])
		if err != nil {
			return err
		}

		diags := trace.Validate(t)
		for _, d := range diags {
			fmt.Fprintf(os.Stderr, "  %s\n", d)
		}
		if len(diags) > 0 {
			return fmt.Errorf("%d consistency problem(s) found", len(diags))
		}

		byMachine := map[string]int{}
		for _, e := range t.Entries {
			byMachine[e.Machine]++
		}
		fmt.Printf("Trace v%s: %d entries across %d machine(s), ticks %d-%d\n",
			t.Version, len(t.Entries), len(byMachine), firstTick(t), lastTick(t))
		for m, n := range byMachine {
			fmt.Printf("  %s: %d entries\n", m, n)
		}
		return nil
	},
}

func firstTick(t *trace.Trace) uint64 {
	if len(t.Entries) == 0 {
		return 0
	}
	return t.Entries[0].Tick
}

func lastTick(t *trace.Trace) uint64 {
	if len(t.Entries) == 0 {
		return 0
	}
	return t.Entries[len(t.Entries)-1].Tick
}

var replayCmd = &cobra.Command{
	Use:   "replay <file>",
	Short: "Deterministically walk a causal trace file, tick by tick",
	Long: `Validate a trace file (like "seed trace") and print its decision
sequence in tick order — a deterministic, offline reading of what
happened, not a live re-execution against GameAK. Replaying a trace back
into a running Runtime (so the game itself reproduces the recorded run)
is a Phase 7 runtime feature; this command works entirely from the trace
file's own recorded fields.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := readTraceFile(args[0])
		if err != nil {
			return err
		}
		if diags := trace.Validate(t); len(diags) > 0 {
			for _, d := range diags {
				fmt.Fprintf(os.Stderr, "  %s\n", d)
			}
			return fmt.Errorf("refusing to replay an inconsistent trace (%d problem(s) found)", len(diags))
		}

		for _, e := range t.Entries {
			if e.Rejected {
				fmt.Printf("[tick %d] %s: %s rejected event %q (%s)\n", e.Tick, e.Loop, e.Machine, e.Event, e.RejectReason)
				continue
			}
			fmt.Printf("[tick %d] %s: %s: %s -[%s]-> %s\n", e.Tick, e.Loop, e.Machine, e.PreviousState, e.Event, e.SelectedTransition)
			for _, c := range e.Commands {
				fmt.Printf("           command: %s %s\n", c.Type, c.Detail)
			}
			for _, ev := range e.ResultingEvents {
				fmt.Printf("           emits: %s\n", ev)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(traceCmd)
	rootCmd.AddCommand(replayCmd)
}
