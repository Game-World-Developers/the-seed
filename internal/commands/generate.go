package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

var allModels bool

const generateLongHelp = `Create a new YAML model definition.
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

Use --all to create one model of each type with sensible defaults.`

func runGenerate(cmd *cobra.Command, args []string) error {
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
}

var generateCmd = &cobra.Command{
	Use:     "generate <type> <name>",
	Aliases: []string{"g"},
	Short:   "Create a new YAML model definition",
	Long:    generateLongHelp,
	Args:    cobra.MaximumNArgs(2),
	RunE:    runGenerate,
}

// topLevelGenerateCmd is the Rails-convention `seed generate ...` alias for
// `seed model generate ...` — both run the exact same logic (runGenerate),
// registered as two *cobra.Command instances since a command can only have
// one parent. Kept alongside `model generate` rather than replacing it: an
// existing script or muscle-memory invocation of `seed model generate`
// keeps working, matching "normalize the command vocabulary" without a
// breaking rename.
var topLevelGenerateCmd = &cobra.Command{
	Use:     "generate <type> <name>",
	Aliases: []string{"g"},
	Short:   "Create a new YAML model definition (alias for 'model generate')",
	Long:    generateLongHelp,
	Args:    cobra.MaximumNArgs(2),
	RunE:    runGenerate,
}

func init() {
	generateCmd.Flags().BoolVar(&allModels, "all", false, "Generate one model of each type")
	modelCmd.AddCommand(generateCmd)

	topLevelGenerateCmd.Flags().BoolVar(&allModels, "all", false, "Generate one model of each type")
	rootCmd.AddCommand(topLevelGenerateCmd)
}
