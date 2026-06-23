package tests

import (
	"testing"

	projecttemplates "Game-Developers-World/seed/internal/project/templates"
	"github.com/caiolandgraf/gest/v2/gest"
)

func TestProjectTemplatesRender(t *testing.T) {
	data3d := projecttemplates.ProjectData{
		Name: "TestProject",
		Mode: "3d",
	}
	data2d := projecttemplates.ProjectData{
		Name: "Test2D",
		Mode: "2d",
	}

	gest.Describe("project/templates.Render").
		It("renders xmake.lua", func(gt *gest.T) {
			out, err := projecttemplates.Render("xmake.lua", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain(`target("TestProject")`)
		}).
		It("renders main.cpp with 3d mode", func(gt *gest.T) {
			out, err := projecttemplates.Render("main.cpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("seed::Renderer3D")
			gt.Expect(out).ToContain(`Window::create("TestProject"`)
		}).
		It("renders main.cpp with 2d mode", func(gt *gest.T) {
			out, err := projecttemplates.Render("main.cpp", data2d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("seed::Renderer2D")
			gt.Expect(out).ToContain(`Window::create("Test2D"`)
		}).
		It("renders bootstrap.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("bootstrap.hpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("namespace game")
		}).
		It("renders bootstrap.cpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("bootstrap.cpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("register_components")
		}).
		It("renders gitignore", func(gt *gest.T) {
			out, err := projecttemplates.Render("gitignore", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("Build/")
		}).
		It("renders gameak-gitignore", func(gt *gest.T) {
			out, err := projecttemplates.Render("gameak-gitignore", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("Build/")
		}).
		It("renders gameak-xmake.lua", func(gt *gest.T) {
			out, err := projecttemplates.Render("gameak-xmake.lua", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain(`target("gameak-core")`)
		}).
		It("renders Seed-Context.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("Seed-Context.hpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Context")
		}).
		It("renders Seed-Window.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("Seed-Window.hpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Window")
		}).
		It("renders Seed-Renderer3D.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("Seed-Renderer3D.hpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Renderer3D")
		}).
		It("renders Seed-Renderer2D.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("Seed-Renderer2D.hpp", data2d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Renderer2D")
		}).
		It("returns error for unknown template name", func(gt *gest.T) {
			_, err := projecttemplates.Render("nonexistent", data3d)
			gt.Expect(err).Not().ToBeNil()
		}).
		Run(t)
}
