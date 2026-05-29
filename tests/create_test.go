package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/caiolandgraf/gest/v2/gest"

	"gameworlddevelopers/the-seed/internal/project"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func copySeedYml(dst string) {
	src := filepath.Join("..", "config", "project.seed.yml")
	data, err := os.ReadFile(src)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Join(dst, "config"), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "config", "project.seed.yml"), data, 0o644); err != nil {
		panic(err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func createDir(root, name string, subdirs ...string) {
	for _, dir := range subdirs {
		if err := os.MkdirAll(filepath.Join(root, name, dir), 0o755); err != nil {
			panic(err)
		}
	}
}

// ---------------------------------------------------------------------------
// FindGameAK
// ---------------------------------------------------------------------------

func TestFindGameAK_returnsEmptyWhenNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	gest.Describe("FindGameAK").
		It("returns empty", func(t *gest.T) {
			t.Expect(project.FindGameAK()).ToEqual("")
		}).
		Run(t)
}

func TestFindGameAK_findsInHomeProjectsGameAK(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	headerPath := filepath.Join(tmp, "Projects", "GameAK", "Include", "AK", "Core", "Types.hpp")
	if err := os.MkdirAll(filepath.Dir(headerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(headerPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	gest.Describe("FindGameAK").
		It("finds GameAK in HOME/Projects/GameAK", func(t *gest.T) {
			expected := filepath.Join(tmp, "Projects", "GameAK")
			t.Expect(project.FindGameAK()).ToEqual(expected)
		}).
		Run(t)
}

func TestFindGameAK_ignoresDirWithoutTypesHpp(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := os.MkdirAll(filepath.Join(tmp, "Projects", "GameAK"), 0o755); err != nil {
		t.Fatal(err)
	}

	gest.Describe("FindGameAK").
		It("ignores directories without Types.hpp", func(t *gest.T) {
			t.Expect(project.FindGameAK()).ToEqual("")
		}).
		Run(t)
}

// ---------------------------------------------------------------------------
// GenerateYaml
// ---------------------------------------------------------------------------

func TestGenerateYaml(t *testing.T) {
	tmp := t.TempDir()
	copySeedYml(tmp)
	chdir(t, tmp)

	gest.Describe("GenerateYaml").
		It("generates project.yaml with correct fields", func(t *gest.T) {
			name := "testproj"
			createDir(tmp, name, "Config")

			data := project.TemplateData{Name: name}
			if err := project.GenerateYaml(name, data); err != nil {
				panic(err)
			}

			content, err := os.ReadFile(filepath.Join(tmp, name, "Config", "project.yaml"))
			if err != nil {
				panic(err)
			}

			s := string(content)
			t.Expect(s).ToContain(`name: "` + name + `"`)
			t.Expect(s).ToContain("deterministic: true")
			t.Expect(s).ToContain("tick_rate: 60")
			t.Expect(s).ToContain("rollback: true")
			t.Expect(s).ToContain("execution: bits")
			t.Expect(s).ToContain("scheduler: graph")
			t.Expect(s).ToContain("replication: delta")
		}).
		It("creates yaml with 0644 permissions", func(t *gest.T) {
			name := "permtest"
			createDir(tmp, name, "Config")

			data := project.TemplateData{Name: name}
			if err := project.GenerateYaml(name, data); err != nil {
				panic(err)
			}

			info, err := os.Stat(filepath.Join(tmp, name, "Config", "project.yaml"))
			if err != nil {
				panic(err)
			}
			t.Expect(info.Mode() & 0o777).ToEqual(os.FileMode(0o644))
		}).
		Run(t)
}

// ---------------------------------------------------------------------------
// Create (full integration)
// ---------------------------------------------------------------------------

func TestCreate(t *testing.T) {
	tmp := t.TempDir()
	copySeedYml(tmp)
	chdir(t, tmp)

	gest.Describe("Create").
		It("creates a project with expected directories and project.yaml", func(t *gest.T) {
			name := "fullproject"
			if err := project.Create(name); err != nil {
				panic(err)
			}

			info, err := os.Stat(filepath.Join(tmp, name))
			t.Expect(err).ToBeNil()
			if err == nil {
				t.Expect(info.IsDir()).ToBeTrue()
			}

			yamlContent, err := os.ReadFile(filepath.Join(tmp, name, "Config", "project.yaml"))
			if err != nil {
				panic(err)
			}
			t.Expect(string(yamlContent)).ToContain(`name: "` + name + `"`)
		}).
		It("succeeds when target directory already exists", func(t *gest.T) {
			name := "existingproj"
			if err := os.MkdirAll(filepath.Join(tmp, name), 0o755); err != nil {
				panic(err)
			}
			err := project.Create(name)
			t.Expect(err).ToBeNil()
		}).
		It("creates root directory with 0755 permissions", func(t *gest.T) {
			name := "dirperm"
			if err := project.Create(name); err != nil {
				panic(err)
			}
			info, err := os.Stat(filepath.Join(tmp, name))
			if err != nil {
				panic(err)
			}
			t.Expect(info.Mode() & 0o777).ToEqual(os.FileMode(0o755))
		}).
		Run(t)
}
