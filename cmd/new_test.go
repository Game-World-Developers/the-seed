package cmd

import (
	"testing"

	"github.com/caiolandgraf/gest/v2/gest"
)

func TestNewCommand(t *testing.T) {
	gest.Describe("new command").
		It("has Use set to new [project name]", func(t *gest.T) {
			t.Expect(newCmd.Use).ToEqual("new [project name]")
		}).
		It("has a non-empty Short description", func(t *gest.T) {
			t.Expect(newCmd.Short).Not().ToEqual("")
		}).
		It("has RunE wired", func(t *gest.T) {
			t.Expect(newCmd.RunE).Not().ToBeNil()
		}).
		It("is registered as a child of rootCmd", func(t *gest.T) {
			var found bool
			for _, c := range rootCmd.Commands() {
				if c == newCmd {
					found = true
					break
				}
			}
			t.Expect(found).ToBeTrue()
		}).
		Describe("args validation", func(s *gest.Suite) {
			s.It("accepts exactly one argument", func(t *gest.T) {
				err := newCmd.Args(newCmd, []string{"myproject"})
				t.Expect(err).ToBeNil()
			})

			s.It("rejects zero arguments", func(t *gest.T) {
				err := newCmd.Args(newCmd, []string{})
				t.Expect(err).Not().ToBeNil()
			})

			s.It("rejects more than one argument", func(t *gest.T) {
				err := newCmd.Args(newCmd, []string{"a", "b"})
				t.Expect(err).Not().ToBeNil()
			})
		}).
		Run(t)
}
