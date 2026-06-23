package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/project"

	"github.com/spf13/cobra"
)

var mode string

var newCmd = &cobra.Command{
	Use:   "new <project-name>",
	Short: "Scaffold a new C++ game project with GameAK + SDL3",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("Creating project %s...\n", name)
		return project.Scaffold(name, mode)
	},
}

func init() {
	newCmd.Flags().StringVar(&mode, "mode", "3d", "Project mode: 2d or 3d")
	rootCmd.AddCommand(newCmd)
}
