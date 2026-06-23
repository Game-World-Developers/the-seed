package tests

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/caiolandgraf/gest/v2/gest"
)

func TestCommandsBinary(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "seed")

	gest.Describe("seed binary").
		BeforeAll(func(gt *gest.T) {
			build := exec.Command("go", "build", "-o", binPath, "../cmd/seed/")
			out, err := build.CombinedOutput()
			gt.Expect(err).ToBeNil()
			_ = out
		}).
		It("prints help with --help", func(gt *gest.T) {
			cmd := exec.Command(binPath, "--help")
			out, err := cmd.Output()
			gt.Expect(err).ToBeNil()
			gt.Expect(string(out)).ToContain("The Seed")
		}).
		It("prints version with --version", func(gt *gest.T) {
			cmd := exec.Command(binPath, "--version")
			out, err := cmd.Output()
			gt.Expect(err).ToBeNil()
			gt.Expect(string(out)).ToContain("0.1.0")
		}).
		It("responds to new --help", func(gt *gest.T) {
			cmd := exec.Command(binPath, "new", "--help")
			out, err := cmd.Output()
			gt.Expect(err).ToBeNil()
			gt.Expect(string(out)).ToContain("<project-name>")
		}).
		It("responds to generate --help", func(gt *gest.T) {
			cmd := exec.Command(binPath, "generate", "--help")
			out, err := cmd.Output()
			gt.Expect(err).ToBeNil()
			gt.Expect(string(out)).ToContain("<type> <name>")
		}).
		It("responds to sync --help", func(gt *gest.T) {
			cmd := exec.Command(binPath, "sync", "--help")
			out, err := cmd.Output()
			gt.Expect(err).ToBeNil()
			gt.Expect(string(out)).ToContain("YAML model files")
		}).
		Run(t)
}
