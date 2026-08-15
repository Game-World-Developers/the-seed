package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

const deleteLongHelp = `Remove a YAML model definition and the corresponding generated C++
header (.hpp) and source (.cpp for systems) files.

The bootstrap.cpp registration is also updated automatically.

Types: component, trait, entity, archetype, state_machine, event, asset, system`

func runDelete(cmd *cobra.Command, args []string) error {
	if err := requireSeedProject(); err != nil {
		return err
	}
	compType := args[0]
	name := args[1]
	fmt.Printf("Deleting %s %s...\n", compType, name)
	return generators.DeleteModel(compType, name)
}

var deleteCmd = &cobra.Command{
	Use:   "delete <type> <name>",
	Short: "Delete a YAML model and its generated C++ files",
	Long:  deleteLongHelp,
	Args:  cobra.ExactArgs(2),
	RunE:  runDelete,
}

// topLevelDestroyCmd is the Rails-convention `seed destroy ...` counterpart
// to `seed generate ...` — same rationale as generate.go's
// topLevelGenerateCmd: an additive alias, not a replacement for `model
// delete`, which keeps working unchanged.
var topLevelDestroyCmd = &cobra.Command{
	Use:     "destroy <type> <name>",
	Aliases: []string{"d"},
	Short:   "Delete a YAML model and its generated C++ files (alias for 'model delete')",
	Long:    deleteLongHelp,
	Args:    cobra.ExactArgs(2),
	RunE:    runDelete,
}

func init() {
	modelCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(topLevelDestroyCmd)
}
