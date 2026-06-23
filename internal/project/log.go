package project

import "fmt"

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func logStatus(action, target string) {
	actionColor := colorGreen
	if action == "clone" {
		actionColor = colorCyan
	}
	fmt.Printf("  %s%-10s%s  %s%s%s\n",
		actionColor, action, colorReset,
		colorBold, target, colorReset)
}

func logDone(format string, args ...any) {
	fmt.Printf("  %s%-10s%s  ", colorGreen, "done", colorReset)
	fmt.Printf(format+"\n", args...)
}
