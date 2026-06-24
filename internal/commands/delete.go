package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <type> <name>",
	Short: "Delete a YAML model and its generated C++ files",
	Long: `Remove a YAML model definition and the corresponding generated C++
header (.hpp) and source (.cpp for systems) files.

The bootstrap.cpp registration is also updated automatically.

Types: component, trait, entity, archetype, state_machine, event, asset, system`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		compType := args[0]
		name := args[1]
		fmt.Printf("Deleting %s %s...\n", compType, name)
		return generators.DeleteModel(compType, name)
	},
}

func init() {
	modelCmd.AddCommand(deleteCmd)
}
