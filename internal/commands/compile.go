package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"Game-Developers-World/seed/internal/backend/gameak"
	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/ir"

	"github.com/spf13/cobra"
)

var compileJSON bool

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Validate models and produce the IR, without generating C++",
	Long: `Validate every YAML model under Models/, resolve cross-model
references, and build the intermediate representation (IR) — without
writing any C++ output.

This is the same semantic compiler pass "seed sync" runs before
generating code, exposed standalone so models can be validated (or
inspected) in editors and CI without touching Include/ or Src/.

Use --json to print the resolved IR as machine-readable JSON instead of
a human-readable summary.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}

		result, err := generators.Compile()
		if err != nil {
			return fmt.Errorf("compiling models: %w", err)
		}

		for _, d := range result.Diagnostics {
			fmt.Fprintf(os.Stderr, "  %s\n", d)
			if s := suggestFor(d.Message); s != "" {
				fmt.Fprintf(os.Stderr, "    %s\n", s)
			}
		}

		if result.HasErrors() {
			return fmt.Errorf("%d semantic error(s) found", len(result.Errors()))
		}

		built, err := ir.Build(result)
		if err != nil {
			return err
		}

		backendDiags := gameak.New().Validate(built)
		hasBackendErrors := false
		for _, d := range backendDiags {
			fmt.Fprintf(os.Stderr, "  %s\n", d)
			if s := suggestFor(d.Message); s != "" {
				fmt.Fprintf(os.Stderr, "    %s\n", s)
			}
			if d.Severity == gameak.SeverityError {
				hasBackendErrors = true
			}
		}
		if hasBackendErrors {
			return fmt.Errorf("gameak backend rejected the model set")
		}

		if compileJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(built)
		}

		fmt.Printf("Compiled OK (IR v%s): %d components, %d traits, %d entities, %d archetypes, %d state_machines, %d events, %d systems, %d assets, %d blocks.\n",
			built.Version, len(built.Components), len(built.Traits), len(built.Entities),
			len(built.Archetypes), len(built.StateMachines), len(built.Events),
			len(built.Systems), len(built.Assets), len(built.Blocks))
		if len(built.Schedule) > 0 {
			fmt.Println("Schedule:")
			for _, s := range built.Schedule {
				fmt.Printf("  [%3d] %s\n", s.Priority, s.Qualified())
			}
		}
		return nil
	},
}

func init() {
	compileCmd.Flags().BoolVar(&compileJSON, "json", false, "Print the resolved IR as JSON")
	rootCmd.AddCommand(compileCmd)
}
