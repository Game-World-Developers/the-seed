package tests

import (
	"testing"

	"Game-Developers-World/seed/internal/generators/templates"
	"github.com/caiolandgraf/gest/v2/gest"
)

func TestGeneratorsTemplatesLoad(t *testing.T) {
	gest.Describe("templates.Load").
		It("loads component template", func(gt *gest.T) {
			tpl, err := templates.Load("component")
			gt.Expect(err).ToBeNil()
			gt.Expect(tpl).ToContain("struct {{.Name}}")
			gt.Expect(tpl).ToContain("BlockTypeDescriptor")
			gt.Expect(tpl).ToContain("register_block_type")
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
		It("loads state_machine template", func(gt *gest.T) {
			tpl, err := templates.Load("state_machine")
			gt.Expect(err).ToBeNil()
			gt.Expect(tpl).ToContain("register_controller")
			gt.Expect(tpl).ToContain("add_transition")
			gt.Expect(tpl).ToContain("Fsm<>")
		}).
		It("loads system template", func(gt *gest.T) {
			tpl, err := templates.Load("system")
			gt.Expect(err).ToBeNil()
			gt.Expect(tpl).ToContain("CommandProducer")
			gt.Expect(tpl).ToContain("EphemeralProducer")
			gt.Expect(tpl).ToContain("register_controller")
		}).
		It("returns error for unknown kind", func(gt *gest.T) {
			_, err := templates.Load("unknown")
			gt.Expect(err).Not().ToBeNil()
		}).
		Run(t)
}

func TestGeneratorsTemplatesLoadCpp(t *testing.T) {
	gest.Describe("templates.LoadCpp").
		It("loads system .cpp template", func(gt *gest.T) {
			tpl, err := templates.LoadCpp("system")
			gt.Expect(err).ToBeNil()
			gt.Expect(tpl).ToContain("operator()")
			gt.Expect(tpl).ToContain("TODO: implement")
		}).
		It("returns error for unknown kind", func(gt *gest.T) {
			_, err := templates.LoadCpp("component")
			gt.Expect(err).Not().ToBeNil()
		}).
		Run(t)
}
