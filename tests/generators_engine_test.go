package tests

import (
	"os"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/generators"
	"github.com/caiolandgraf/gest/v2/gest"
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

			err = generators.CreateModel("state_machine", "PlayerController")
			gt.Expect(err).ToBeNil()

			err = generators.CreateModel("system", "Gravity")
			gt.Expect(err).ToBeNil()

			err = generators.Sync()
			gt.Expect(err).ToBeNil()

			compHpp := filepath.Join(tmpDir, "sync-full", "Include", "Core", "Component", "Position.hpp")
			traitHpp := filepath.Join(tmpDir, "sync-full", "Include", "Core", "Trait", "Movable.hpp")
			smHpp := filepath.Join(tmpDir, "sync-full", "Include", "Game", "PlayerController.hpp")
			sysHpp := filepath.Join(tmpDir, "sync-full", "Include", "Game", "Gravity.hpp")
			sysCpp := filepath.Join(tmpDir, "sync-full", "Src", "Game", "Gravity.cpp")

			_, err = os.Stat(compHpp)
			gt.Expect(err).ToBeNil()

			_, err = os.Stat(traitHpp)
			gt.Expect(err).ToBeNil()

			_, err = os.Stat(smHpp)
			gt.Expect(err).ToBeNil()

			_, err = os.Stat(sysHpp)
			gt.Expect(err).ToBeNil()

			_, err = os.Stat(sysCpp)
			gt.Expect(err).ToBeNil()

			data, err := os.ReadFile(compHpp)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(data)).ToContain("struct Position")
			gt.Expect(string(data)).ToContain("register_block_type")

			smData, err := os.ReadFile(smHpp)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(smData)).ToContain("register_controller")
			gt.Expect(string(smData)).ToContain("add_transition")

			sysData, err := os.ReadFile(sysHpp)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(sysData)).ToContain("CommandProducer")
			gt.Expect(string(sysData)).ToContain("register_controller")
		}).
		It("generates system .cpp only once (does not overwrite)", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "sync-cpp-once"))()

			err := generators.CreateModel("system", "MySystem")
			gt.Expect(err).ToBeNil()

			err = generators.Sync()
			gt.Expect(err).ToBeNil()

			cppPath := filepath.Join(tmpDir, "sync-cpp-once", "Src", "Game", "MySystem.cpp")
			data, err := os.ReadFile(cppPath)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(data)).ToContain("TODO: implement")

			// Second sync should not overwrite user code
			err = generators.Sync()
			gt.Expect(err).ToBeNil()

			data2, err := os.ReadFile(cppPath)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(data2)).ToEqual(string(data))
		}).
		It("handles empty model directories", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "sync-empty"))()

			err := generators.Sync()
			gt.Expect(err).ToBeNil()
		}).
		Run(t)
}
