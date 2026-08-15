package commands

import (
	"fmt"
	"os"
	"strings"

	"Game-Developers-World/seed/internal/backend/gameak"
	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/ir"

	"github.com/spf13/cobra"
)

// checkGameAKBackend re-validates the current model set against GameAK's
// backend-specific limitations (bare-name collisions, unsupported event
// delivery — see internal/backend/gameak) before generation runs. It
// duplicates the semantic Compile pass generators.Sync already performs
// internally, but that pass has no notion of GameAK's own constraints —
// see docs/gameak-mapping.md's "Known gap" note on why this check lives
// in the CLI layer rather than inside internal/generators (importing
// internal/ir there would be a package import cycle, since ir already
// depends on generators for its Model/CompileResult types).
func checkGameAKBackend() error {
	result, err := generators.Compile()
	if err != nil {
		return fmt.Errorf("compiling models: %w", err)
	}
	if result.HasErrors() {
		// generators.Sync will report these the same way; nothing further
		// to check here since the IR can't be built from a failed compile.
		return nil
	}
	built, err := ir.Build(result)
	if err != nil {
		return err
	}
	diags := gameak.New().Validate(built)
	hasErrors := false
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "  %s\n", d)
		if s := suggestFor(d.Message); s != "" {
			fmt.Fprintf(os.Stderr, "    %s\n", s)
		}
		if d.Severity == gameak.SeverityError {
			hasErrors = true
		}
	}
	if hasErrors {
		return fmt.Errorf("gameak backend rejected the model set")
	}
	return nil
}

var checkSync bool
var dryRunSync bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Generate C++ headers from all YAML model files",
	Long: `Read all YAML model files from Models/ and generate corresponding
C++ headers into include/.

This processes every .yaml file in:
  Models/component/     → Include/<ns>/Component/<name>.hpp
  Models/trait/         → Include/<ns>/Trait/<name>.hpp
  Models/entity/        → Include/<ns>/Entity/<name>.hpp
  Models/archetype/     → Include/<ns>/Archetype/<name>.hpp
  Models/state_machine/ → Include/<ns>/<name>.hpp
  Models/system/        → Include/<ns>/<name>.hpp + Src/Game/<name>.cpp

It also updates registration calls in Src/Game/bootstrap.cpp.

Use --check to show sync status without generating files.
Use --dry-run to preview exactly what would change (create/update/
unchanged, with a diff for updates) without writing anything.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		if dryRunSync {
			if err := checkGameAKBackend(); err != nil {
				return err
			}
			changes, err := generators.SyncPlan()
			if err != nil {
				return err
			}
			if len(changes) == 0 {
				fmt.Println("No models found.")
				return nil
			}
			create, update, unchanged := 0, 0, 0
			for _, c := range changes {
				switch c.Action {
				case generators.ActionCreate:
					fmt.Printf("  %11s  %s\n", "create", c.Path)
					create++
				case generators.ActionUpdate:
					fmt.Printf("  %11s  %s\n", "update", c.Path)
					fmt.Print(indentDiff(c.Diff))
					update++
				case generators.ActionUnchanged:
					unchanged++
				}
			}
			fmt.Printf("\n%d to create, %d to update, %d unchanged. No files were written (--dry-run).\n", create, update, unchanged)
			return nil
		}
		if checkSync {
			results, err := generators.CheckSync()
			if err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Println("No models found.")
				return nil
			}
			for _, r := range results {
				switch r.Status {
				case "ok":
					fmt.Printf("  [OK]      %s %s\n", r.Type, r.Name)
				case "stale":
					fmt.Printf("  [STALE]   %s %s (needs sync)\n", r.Type, r.Name)
				case "missing":
					fmt.Printf("  [MISSING] %s %s\n", r.Type, r.Name)
				}
			}
			return nil
		}
		if err := checkGameAKBackend(); err != nil {
			return err
		}
		fmt.Println("Syncing models...")
		return generators.Sync()
	},
}

func indentDiff(diff string) string {
	if diff == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
		b.WriteString("      ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func init() {
	syncCmd.Flags().BoolVarP(&checkSync, "check", "c", false, "Check sync status without generating files")
	syncCmd.Flags().BoolVar(&dryRunSync, "dry-run", false, "Preview changes without writing any files")
	rootCmd.AddCommand(syncCmd)
}
