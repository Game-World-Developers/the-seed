package cmd

import "github.com/spf13/cobra"

func RootCmd() *cobra.Command { return rootCmd }

func NewCmd() *cobra.Command { return newCmd }
