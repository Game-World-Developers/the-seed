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
			gt.Expect(out).ToContain("Runtime<>{}")
		}).
		It("renders main.cpp with 2d mode", func(gt *gest.T) {
			out, err := projecttemplates.Render("main.cpp", data2d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("seed::Renderer2D")
			gt.Expect(out).ToContain(`Window::create("Test2D"`)
			gt.Expect(out).ToContain("Runtime<>{}")
		}).
		It("renders bootstrap.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("bootstrap.hpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("runtime.h")
			gt.Expect(out).ToContain("Runtime<>&")
		}).
		It("renders bootstrap.cpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("bootstrap.cpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("@seed:begin(components)")
			gt.Expect(out).ToContain("@seed:begin(controllers)")
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
		It("renders seed-context.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("seed-context.hpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Context")
		}).
		It("renders seed-window.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("seed-window.hpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Window")
		}).
		It("renders seed-renderer_3d.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("seed-renderer_3d.hpp", data3d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Renderer3D")
			gt.Expect(out).Not().ToContain("SDL_Renderer")
		}).
		It("renders seed-renderer_2d.hpp", func(gt *gest.T) {
			out, err := projecttemplates.Render("seed-renderer_2d.hpp", data2d)
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Renderer2D")
		}).
		It("returns error for unknown template name", func(gt *gest.T) {
			_, err := projecttemplates.Render("nonexistent", data3d)
			gt.Expect(err).Not().ToBeNil()
		}).
		Run(t)
}

func TestProjectTemplatesRenderStatic(t *testing.T) {
	gest.Describe("project/templates.RenderStatic").
		It("copies bootstrap.hpp directly", func(gt *gest.T) {
			out, err := projecttemplates.RenderStatic("bootstrap.hpp")
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("runtime.h")
		}).
		It("copies bootstrap.cpp directly", func(gt *gest.T) {
			out, err := projecttemplates.RenderStatic("bootstrap.cpp")
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("@seed:begin(components)")
		}).
		It("copies gitignore directly", func(gt *gest.T) {
			out, err := projecttemplates.RenderStatic("gitignore")
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("Build/")
		}).
		It("copies seed-context.hpp directly", func(gt *gest.T) {
			out, err := projecttemplates.RenderStatic("seed-context.hpp")
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Context")
		}).
		It("copies seed-renderer_3d.hpp directly", func(gt *gest.T) {
			out, err := projecttemplates.RenderStatic("seed-renderer_3d.hpp")
			gt.Expect(err).ToBeNil()
			gt.Expect(out).ToContain("class Renderer3D")
			gt.Expect(out).Not().ToContain("SDL_Renderer")
		}).
		It("returns error for unknown static template", func(gt *gest.T) {
			_, err := projecttemplates.RenderStatic("nonexistent")
			gt.Expect(err).Not().ToBeNil()
		}).
		Run(t)
}
