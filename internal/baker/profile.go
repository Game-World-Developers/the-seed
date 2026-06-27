package baker

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Profile string

const (
	ProfileQuality     Profile = "best-quality"
	ProfilePerformance Profile = "best-performance"
)

type SeedProject struct {
	AssetProfile Profile `yaml:"asset_profile"`
}

// ReadProfile reads the .seed_project configuration and returns the asset profile.
// Defaults to "best-quality" if the file doesn't exist or profile is unset.
func ReadProfile(projectRoot string) Profile {
	path := fmt.Sprintf("%s/.seed_project", projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return ProfileQuality
	}

	var cfg SeedProject
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ProfileQuality
	}

	if cfg.AssetProfile != ProfilePerformance && cfg.AssetProfile != ProfileQuality {
		return ProfileQuality
	}

	return cfg.AssetProfile
}
