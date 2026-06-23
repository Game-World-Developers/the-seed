package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate <type> <name>",
	Short: "Create a new YAML model definition",
	Long: `Create a new YAML model definition.
If the file already exists, it prints a message and does nothing.

Types: component, trait, entity, archetype

Use "seed sync" to generate C++ headers from all model files.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compType := args[0]
		name := args[1]
		fmt.Printf("Generating %s %s...\n", compType, name)
		return generators.CreateModel(compType, name)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
