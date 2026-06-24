package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"Game-Developers-World/seed/internal/project"

	"github.com/spf13/cobra"
)

var newOverwrite bool
var newMode string

func promptOverwrite(name string) bool {
	fmt.Printf("Project %q already exists. Overwrite files? [y/N] ", name)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text() == "y" || scanner.Text() == "Y"
}

var newCmd = &cobra.Command{
	Use:   "new [project-name]",
	Short: "Scaffold a new C++ game project with GameAK + SDL3",
	Long: `Create a new C++ game project in <project-name>/ directory.

If <project-name> is omitted and the current directory is already a Seed project,
missing Seed files are scaffolded into the existing project.

If the project directory already exists, you will be prompted before overwriting
(use --overwrite to skip the prompt).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			name := args[0]
			root := filepath.Join(".", name)

			if _, err := os.Stat(filepath.Join(root, ".seed_project")); err == nil {
				if !newOverwrite && !promptOverwrite(name) {
					fmt.Println("Aborted.")
					return nil
				}
				oldDir, _ := os.Getwd()
				os.Chdir(root)
				defer os.Chdir(oldDir)
				return project.ScaffoldInit(project.InitOpts{
					Mode:      newMode,
					Overwrite: true,
				})
			}

			return project.Scaffold(name, newMode)
		}

		if err := requireSeedProject(); err != nil {
			return err
		}
		return project.ScaffoldInit(project.InitOpts{
			Mode:      newMode,
			Overwrite: newOverwrite,
		})
	},
}

func init() {
	newCmd.Flags().StringVar(&newMode, "mode", "3d", "Project mode: 2d or 3d")
	newCmd.Flags().BoolVarP(&newOverwrite, "overwrite", "o", false, "Overwrite existing files without prompting")
	rootCmd.AddCommand(newCmd)
}
