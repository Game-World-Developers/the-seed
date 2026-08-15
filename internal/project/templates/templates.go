package templates

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed *.tmpl
var templateFS embed.FS

var tmpls = template.Must(template.ParseFS(templateFS, "*.tmpl"))

type ProjectData struct {
	Name string
	Mode string
}

var StaticFiles = []string{
	"bootstrap.hpp",
	"bootstrap.cpp",
	"gitignore",
	"gameak-gitignore",
	"gameak-xmake.lua",
	"seed-context.hpp",
	"seed-window.hpp",
	"seed-audio.hpp",
	"seed-surface.hpp",
	"seed-texture.hpp",
	"seed-font.hpp",
	"seed-image.hpp",
	"seed-asset-manager.hpp",
	"seed-renderer_2d.hpp",
	"seed-renderer_3d.hpp",
	"seed-shaders.hpp",
	"seed-input.hpp",
	"seed-camera.hpp",
	"seed-scene.hpp",
	"seed-platform-services.hpp",
	"clang-format",
	"clang-tidy",
	"cppcheck-suppressions",
}

func Render(name string, data ProjectData) (string, error) {
	var buf bytes.Buffer
	if err := tmpls.ExecuteTemplate(&buf, name+".tmpl", data); err != nil {
		return "", fmt.Errorf("template %q: %w", name, err)
	}
	return buf.String(), nil
}

func RenderStatic(name string) (string, error) {
	data, err := templateFS.ReadFile(name + ".tmpl")
	if err != nil {
		return "", fmt.Errorf("static template %q: %w", name, err)
	}
	return string(data), nil
}
