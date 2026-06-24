package project

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	boldStyle   = lipgloss.NewStyle().Bold(true)
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

func logStatus(action, target string) {
	var style lipgloss.Style
	switch action {
	case "create", "generate":
		style = greenStyle
	case "identical", "overwrite":
		style = yellowStyle
	case "run":
		style = cyanStyle
	default:
		style = greenStyle
	}
	fmt.Printf("  %11s  %s\n", style.Render(action), boldStyle.Render(target))
}

func logDone(format string, args ...any) {
	fmt.Printf("  %11s  ", doneStyle.Render("done"))
	fmt.Printf(format+"\n", args...)
}
