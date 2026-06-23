package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

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

It also updates registration calls in Src/Game/bootstrap.cpp.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Syncing models...")
		return generators.Sync()
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
