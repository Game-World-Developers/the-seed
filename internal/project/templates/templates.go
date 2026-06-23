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

func Render(name string, data ProjectData) (string, error) {
	var buf bytes.Buffer
	if err := tmpls.ExecuteTemplate(&buf, name+".tmpl", data); err != nil {
		return "", fmt.Errorf("template %q: %w", name, err)
	}
	return buf.String(), nil
}
