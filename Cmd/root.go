package Cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "seed",
	Short: "The Seed Project Generator",
	Long:  "The Seed is a simulator first project generator for ECS-to-DoD workflows.",
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "config file")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
