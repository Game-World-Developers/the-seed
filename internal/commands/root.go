package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func requireSeedProject() error {
	if _, err := os.Stat(".seed_project"); os.IsNotExist(err) {
		return fmt.Errorf("not a seed project (no .seed_project found). Run 'seed new --overwrite' first")
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "seed",
	Short: "The Seed — C++ game project scaffold and code generator",
	Long: `The Seed is a high performance, deterministic runtime SDK
for building 3D Games and virtual worlds.

It scaffolds new C++ projects and generates game components.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = "0.1.0"
}
