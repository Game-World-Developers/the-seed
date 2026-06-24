package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

var allModels bool

var generateCmd = &cobra.Command{
	Use:   "generate <type> <name>",
	Short: "Create a new YAML model definition",
	Long: `Create a new YAML model definition.
If the file already exists, it prints a message and does nothing.

Types: component, trait, entity, archetype, state_machine, event, asset, system

- component     — POD struct with data fields, registered as a Data Block
- trait         — semantic grouping of components
- entity        — composition of traits
- archetype     — entity blueprint with factory
- state_machine — FSM with states and event-driven transitions
- event         — POD struct for inter-system communication
- asset         — game asset declaration (texture, font, etc.)
- system        — Controller callable with .hpp + .cpp skeleton

Use --all to create one model of each type with sensible defaults.`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		if allModels {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
			projectName := filepath.Base(wd)
			fmt.Println("Generating one of each type...")
			return generators.CreateAll(projectName)
		}
		if len(args) != 2 {
			return fmt.Errorf("exactly 2 arguments required (type and name), or use --all")
		}
		compType := args[0]
		name := args[1]

		if err := generators.CreateModel(compType, name); err != nil {
			return err
		}
		fmt.Printf("Use \"seed sync\" to generate C++ code.\n")
		return nil
	},
}

func init() {
	generateCmd.Flags().BoolVar(&allModels, "all", false, "Generate one model of each type")
	modelCmd.AddCommand(generateCmd)
}
