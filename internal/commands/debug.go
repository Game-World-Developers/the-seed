package commands

import (
	"fmt"
	"os"

	"Game-Developers-World/seed/internal/tui"

	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Interactive project dashboard (TUI)",
	Long: `Launch an interactive terminal UI to browse models,
check sync status, and inspect model definitions.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		if err := tui.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)
}
