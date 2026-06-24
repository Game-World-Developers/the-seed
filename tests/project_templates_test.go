package tests

import (
	"testing"

	projecttemplates "Game-Developers-World/seed/internal/project/templates"
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

	t.Run("renders xmake.lua", func(t *testing.T) {
		out, err := projecttemplates.Render("xmake.lua", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, `target("TestProject")`) {
			t.Fatalf("expected xmake.lua to contain target, got: %s", out)
		}
	})

	t.Run("renders main.cpp with 3d mode", func(t *testing.T) {
		out, err := projecttemplates.Render("main.cpp", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "seed::Renderer3D") {
			t.Fatalf("expected main.cpp 3d to contain 'seed::Renderer3D', got: %s", out)
		}
		if !contains(out, "SDL_WINDOW_RESIZABLE") {
			t.Fatalf("expected main.cpp to contain window flags, got: %s", out)
		}
		if !contains(out, "Runtime<>::configure().build()") {
			t.Fatalf("expected main.cpp to contain 'Runtime<>::configure().build()', got: %s", out)
		}
		if !contains(out, "detect_refresh_rate") {
			t.Fatalf("expected main.cpp to contain display detection, got: %s", out)
		}
	})

	t.Run("renders main.cpp with 2d mode", func(t *testing.T) {
		out, err := projecttemplates.Render("main.cpp", data2d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "seed::Renderer2D") {
			t.Fatalf("expected main.cpp 2d to contain 'seed::Renderer2D', got: %s", out)
		}
		if !contains(out, "SDL_WINDOW_RESIZABLE") {
			t.Fatalf("expected main.cpp to contain window flags, got: %s", out)
		}
		if !contains(out, "Runtime<>::configure().build()") {
			t.Fatalf("expected main.cpp to contain 'Runtime<>::configure().build()', got: %s", out)
		}
		if !contains(out, "detect_refresh_rate") {
			t.Fatalf("expected main.cpp to contain display detection, got: %s", out)
		}
	})

	t.Run("renders bootstrap.hpp", func(t *testing.T) {
		out, err := projecttemplates.Render("bootstrap.hpp", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "runtime.h") {
			t.Fatalf("expected bootstrap.hpp to contain 'runtime.h', got: %s", out)
		}
		if !contains(out, "Runtime<>&") {
			t.Fatalf("expected bootstrap.hpp to contain 'Runtime<>&', got: %s", out)
		}
	})

	t.Run("renders bootstrap.cpp", func(t *testing.T) {
		out, err := projecttemplates.Render("bootstrap.cpp", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "@seed:begin(components_include)") {
			t.Fatalf("expected bootstrap.cpp to contain '@seed:begin(components_include)', got: %s", out)
		}
		if !contains(out, "@seed:begin(components_reg)") {
			t.Fatalf("expected bootstrap.cpp to contain '@seed:begin(components_reg)', got: %s", out)
		}
		if !contains(out, "@seed:begin(controllers_include)") {
			t.Fatalf("expected bootstrap.cpp to contain '@seed:begin(controllers_include)', got: %s", out)
		}
		if !contains(out, "@seed:begin(controllers_reg)") {
			t.Fatalf("expected bootstrap.cpp to contain '@seed:begin(controllers_reg)', got: %s", out)
		}
	})

	t.Run("renders gitignore", func(t *testing.T) {
		out, err := projecttemplates.Render("gitignore", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "Build/") {
			t.Fatalf("expected gitignore to contain 'Build/', got: %s", out)
		}
	})

	t.Run("renders gameak-gitignore", func(t *testing.T) {
		out, err := projecttemplates.Render("gameak-gitignore", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "Build/") {
			t.Fatalf("expected gameak-gitignore to contain 'Build/', got: %s", out)
		}
	})

	t.Run("renders gameak-xmake.lua", func(t *testing.T) {
		out, err := projecttemplates.Render("gameak-xmake.lua", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, `target("gameak-core")`) {
			t.Fatalf("expected gameak-xmake.lua to contain target, got: %s", out)
		}
	})

	t.Run("renders seed-context.hpp", func(t *testing.T) {
		out, err := projecttemplates.Render("seed-context.hpp", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "class Context") {
			t.Fatalf("expected seed-context.hpp to contain 'class Context', got: %s", out)
		}
	})

	t.Run("renders seed-window.hpp", func(t *testing.T) {
		out, err := projecttemplates.Render("seed-window.hpp", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "class Window") {
			t.Fatalf("expected seed-window.hpp to contain 'class Window', got: %s", out)
		}
	})

	t.Run("renders seed-renderer_3d.hpp", func(t *testing.T) {
		out, err := projecttemplates.Render("seed-renderer_3d.hpp", data3d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "class Renderer3D") {
			t.Fatalf("expected seed-renderer_3d.hpp to contain 'class Renderer3D', got: %s", out)
		}
		if contains(out, "SDL_Renderer") {
			t.Fatalf("expected seed-renderer_3d.hpp to NOT contain 'SDL_Renderer', got: %s", out)
		}
	})

	t.Run("renders seed-renderer_2d.hpp", func(t *testing.T) {
		out, err := projecttemplates.Render("seed-renderer_2d.hpp", data2d)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "class Renderer2D") {
			t.Fatalf("expected seed-renderer_2d.hpp to contain 'class Renderer2D', got: %s", out)
		}
	})

	t.Run("returns error for unknown template name", func(t *testing.T) {
		_, err := projecttemplates.Render("nonexistent", data3d)
		if err == nil {
			t.Fatal("expected error for unknown template name")
		}
	})
}

func TestProjectTemplatesRenderStatic(t *testing.T) {
	t.Run("copies bootstrap.hpp directly", func(t *testing.T) {
		out, err := projecttemplates.RenderStatic("bootstrap.hpp")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "runtime.h") {
			t.Fatalf("expected bootstrap.hpp to contain 'runtime.h', got: %s", out)
		}
	})

	t.Run("copies bootstrap.cpp directly", func(t *testing.T) {
		out, err := projecttemplates.RenderStatic("bootstrap.cpp")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "@seed:begin(components_include)") {
			t.Fatalf("expected bootstrap.cpp to contain '@seed:begin(components_include)', got: %s", out)
		}
	})

	t.Run("copies gitignore directly", func(t *testing.T) {
		out, err := projecttemplates.RenderStatic("gitignore")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "Build/") {
			t.Fatalf("expected gitignore to contain 'Build/', got: %s", out)
		}
	})

	t.Run("copies seed-context.hpp directly", func(t *testing.T) {
		out, err := projecttemplates.RenderStatic("seed-context.hpp")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "class Context") {
			t.Fatalf("expected seed-context.hpp to contain 'class Context', got: %s", out)
		}
	})

	t.Run("copies seed-renderer_3d.hpp directly", func(t *testing.T) {
		out, err := projecttemplates.RenderStatic("seed-renderer_3d.hpp")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(out, "class Renderer3D") {
			t.Fatalf("expected seed-renderer_3d.hpp to contain 'class Renderer3D', got: %s", out)
		}
		if contains(out, "SDL_Renderer") {
			t.Fatalf("expected seed-renderer_3d.hpp to NOT contain 'SDL_Renderer', got: %s", out)
		}
	})

	t.Run("returns error for unknown static template", func(t *testing.T) {
		_, err := projecttemplates.RenderStatic("nonexistent")
		if err == nil {
			t.Fatal("expected error for unknown static template")
		}
	})
}
