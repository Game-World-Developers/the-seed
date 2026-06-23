package tests

import (
	"os"
	"path/filepath"
	"testing"

	"Game-Developers-World/seed/internal/generators"
	"github.com/caiolandgraf/gest/v2/gest"
)

func TestGeneratorsCreateModel(t *testing.T) {
	tmpDir := t.TempDir()

	gest.Describe("generators.CreateModel").
		It("creates a component YAML file", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "component"))()

			err := generators.CreateModel("component", "Health")
			gt.Expect(err).ToBeNil()

			path := filepath.Join(tmpDir, "component", "Models", "Component", "Health.yaml")
			_, err = os.Stat(path)
			gt.Expect(err).ToBeNil()

			data, err := os.ReadFile(path)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(data)).ToContain("name: Health")
		}).
		It("creates a trait YAML file", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "trait"))()

			err := generators.CreateModel("trait", "Movable")
			gt.Expect(err).ToBeNil()

			path := filepath.Join(tmpDir, "trait", "Models", "Trait", "Movable.yaml")
			_, err = os.Stat(path)
			gt.Expect(err).ToBeNil()

			data, err := os.ReadFile(path)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(data)).ToContain("name: Movable")
		}).
		It("creates an entity YAML file", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "entity"))()

			err := generators.CreateModel("entity", "Player")
			gt.Expect(err).ToBeNil()

			path := filepath.Join(tmpDir, "entity", "Models", "Entity", "Player.yaml")
			_, err = os.Stat(path)
			gt.Expect(err).ToBeNil()

			data, err := os.ReadFile(path)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(data)).ToContain("name: Player")
		}).
		It("creates an archetype YAML file", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "archetype"))()

			err := generators.CreateModel("archetype", "PlayerArchetype")
			gt.Expect(err).ToBeNil()

			path := filepath.Join(tmpDir, "archetype", "Models", "Archetype", "PlayerArchetype.yaml")
			_, err = os.Stat(path)
			gt.Expect(err).ToBeNil()

			data, err := os.ReadFile(path)
			gt.Expect(err).ToBeNil()
			gt.Expect(string(data)).ToContain("name: PlayerArchetype")
		}).
		It("returns error for unknown type", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "unknown"))()

			err := generators.CreateModel("invalid", "Test")
			gt.Expect(err).Not().ToBeNil()
		}).
		It("does not error when file already exists", func(gt *gest.T) {
			defer inDir(t, filepath.Join(tmpDir, "exists"))()

			err := generators.CreateModel("component", "Health")
			gt.Expect(err).ToBeNil()

			err = generators.CreateModel("component", "Health")
			gt.Expect(err).ToBeNil()
		}).
		Run(t)
}
