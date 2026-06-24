package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all YAML model definitions",
	Long: `List all YAML model files grouped by type.
Shows component, trait, entity, archetype, state_machine, event, asset, and system models.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		models, err := generators.ListModels()
		if err != nil {
			return err
		}
		if len(models) == 0 {
			fmt.Println("No models found.")
			return nil
		}
		typeCount := map[string]int{}
		for _, m := range models {
			fmt.Printf("  %-15s %s\n", m.Type, m.Name)
			typeCount[m.Type]++
		}
		fmt.Println()
		total := 0
		for _, t := range []string{"component", "trait", "entity", "archetype", "state_machine", "event", "asset", "system"} {
			c := typeCount[t]
			fmt.Printf("  %s: %d\n", t, c)
			total += c
		}
		fmt.Printf("  total: %d\n", total)
		return nil
	},
}

func init() {
	modelCmd.AddCommand(listCmd)
}
