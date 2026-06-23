package tests

import (
	"os"
	"testing"
)

func inDir(t *testing.T, dir string) func() {
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() { os.Chdir(prev) }
}
