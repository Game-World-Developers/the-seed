package commands

import (
	"github.com/spf13/cobra"
)

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage YAML model definitions",
	Long: `Create, list, and delete YAML model definitions.
Models are the source of truth for C++ code generation.`,
}

func init() {
	rootCmd.AddCommand(modelCmd)
}
