package cmd

import (
	"testing"

	"github.com/caiolandgraf/gest/v2/gest"
)

func TestRootCommand(t *testing.T) {
	gest.Describe("root command").
		It("has Use set to seed", func(t *gest.T) {
			t.Expect(rootCmd.Use).ToEqual("seed")
		}).
		It("has a non-empty Short description", func(t *gest.T) {
			t.Expect(rootCmd.Short).Not().ToEqual("")
		}).
		It("has a non-empty Long description", func(t *gest.T) {
			t.Expect(rootCmd.Long).Not().ToEqual("")
		}).
		It("has a --config flag", func(t *gest.T) {
			flag := rootCmd.PersistentFlags().Lookup("config")
			t.Expect(flag).Not().ToBeNil()
			t.Expect(flag.DefValue).ToEqual("")
		}).
		It("finds the new subcommand", func(t *gest.T) {
			cmd, _, err := rootCmd.Find([]string{"new"})
			t.Expect(err).ToBeNil()
			t.Expect(cmd).Not().ToBeNil()
			t.Expect(cmd.Use).ToEqual("new [project name]")
		}).
		It("does not panic on execute with --help", func(t *gest.T) {
			rootCmd.SetArgs([]string{"--help"})
			err := rootCmd.Execute()
			t.Expect(err).ToBeNil()
		}).
		Run(t)
}
