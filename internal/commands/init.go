package commands

import (
	"fmt"

	"Game-Developers-World/seed/internal/project"

	"github.com/spf13/cobra"
)

var initOverwrite bool
var initMode string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize or update an existing project with Seed files",
	Long: `Scaffold missing Seed files into an existing project directory.
Creates project structure files that don't already exist:
  .clang-format, .clang-tidy, .cppcheck-suppressions,
  Seed headers (context.hpp, window.hpp, renderer_*.hpp),
  bootstrap.hpp, bootstrap.cpp, xmake.lua, main.cpp, .gitignore

Use --overwrite to replace existing files with fresh templates.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Initializing project...")
		return project.ScaffoldInit(project.InitOpts{
			Mode:      initMode,
			Overwrite: initOverwrite,
		})
	},
}

func init() {
	initCmd.Flags().BoolVarP(&initOverwrite, "overwrite", "o", false, "Overwrite existing files")
	initCmd.Flags().StringVar(&initMode, "mode", "3d", "Project mode: 2d or 3d")
	rootCmd.AddCommand(initCmd)
}
