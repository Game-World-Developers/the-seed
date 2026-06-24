package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

var checkSync bool

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

Use --check to show sync status without generating files.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
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
		fmt.Println("Syncing models...")
		return generators.Sync()
	},
}

func init() {
	syncCmd.Flags().BoolVarP(&checkSync, "check", "c", false, "Check sync status without generating files")
	rootCmd.AddCommand(syncCmd)
}
