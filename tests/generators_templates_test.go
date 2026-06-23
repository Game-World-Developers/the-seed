package tests

import (
	"testing"

	"github.com/caiolandgraf/gest/v2/gest"
	"Game-Developers-World/seed/internal/generators/templates"
)

func TestGeneratorsTemplatesLoad(t *testing.T) {
	gest.Describe("templates.Load").
		It("loads component template", func(gt *gest.T) {
			tpl, err := templates.Load("component")
			gt.Expect(err).ToBeNil()
			gt.Expect(tpl).ToContain("struct {{.Name}}")
		}).
		It("loads trait template", func(gt *gest.T) {
			tpl, err := templates.Load("trait")
			gt.Expect(err).ToBeNil()
			gt.Expect(tpl).ToContain("struct {{.Name}}")
		}).
		It("loads entity template", func(gt *gest.T) {
			tpl, err := templates.Load("entity")
			gt.Expect(err).ToBeNil()
			gt.Expect(tpl).ToContain("struct {{.Name}}")
		}).
		It("loads archetype template", func(gt *gest.T) {
			tpl, err := templates.Load("archetype")
			gt.Expect(err).ToBeNil()
			gt.Expect(tpl).ToContain("struct {{.Name}}")
		}).
		It("returns error for unknown kind", func(gt *gest.T) {
			_, err := templates.Load("unknown")
			gt.Expect(err).Not().ToBeNil()
		}).
		Run(t)
}
