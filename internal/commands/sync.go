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
C++ headers into include/core/.

This processes every .yaml file in:
  Models/component/
  Models/trait/
  Models/entity/
  Models/archetype/`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Syncing models...")
		return generators.Sync()
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
