package Cmd

import (
	"GameWorldDevelopers/The-Seed/Internal/Project"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new [project name]",
	Short: "Create a new The Seed project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return Project.Create(name)
	},
}
