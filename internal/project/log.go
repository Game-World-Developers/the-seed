package project

import "fmt"

func logStatus(action, target string) {
	fmt.Printf("  %-10s  %s\n", action, target)
}

func logDone(format string, args ...any) {
	fmt.Printf("  %-10s  ", "done")
	fmt.Printf(format+"\n", args...)
}
