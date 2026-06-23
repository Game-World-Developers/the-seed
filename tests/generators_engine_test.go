package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/caiolandgraf/gest/v2/gest"
	"Game-Developers-World/seed/internal/generators"
)

func TestGeneratorsSync(t *testing.T) {
	tmpDir := t.TempDir()

	gest.Describe("generators.Sync").
		It("generates C++ headers from YAML models", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "sync-full"))()

			err := generators.CreateModel("component", "Position")
			gt.Expect(err).ToBeNil()

			err = generators.CreateModel("trait", "Movable")
			gt.Expect(err).ToBeNil()

			err = generators.Sync()
			gt.Expect(err).ToBeNil()

			compHpp := filepath.Join(tmpDir, "sync-full", "Include", "Core", "Component", "Position.hpp")
			traitHpp := filepath.Join(tmpDir, "sync-full", "Include", "Core", "Trait", "Movable.hpp")

			_, err = os.Stat(compHpp)
			gt.Expect(err).ToBeNil()

			_, err = os.Stat(traitHpp)
			gt.Expect(err).ToBeNil()

			data, err := os.ReadFile(compHpp)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(data)).ToContain("struct Position")
		}).
		It("handles empty model directories", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "sync-empty"))()

			err := generators.Sync()
			gt.Expect(err).ToBeNil()
		}).
		Run(t)
}
