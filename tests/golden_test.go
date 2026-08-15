package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"Game-Developers-World/seed/internal/generators"
	"Game-Developers-World/seed/internal/ir"
)

func copyExampleDomain(t *testing.T, root, dst string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "docs", "examples"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		srcDir := filepath.Join(root, "docs", "examples", e.Name())
		files, err := os.ReadDir(srcDir)
		if err != nil {
			t.Fatal(err)
		}
		dstDir := filepath.Join(dst, "Models", e.Name())
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			data, err := os.ReadFile(filepath.Join(srcDir, f.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dstDir, f.Name()), data, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (go.mod not found)")
		}
		dir = parent
	}
}

var whitespaceRun = regexp.MustCompile(`\s+`)

// normalize strips all whitespace so golden comparisons don't depend on
// whether clang-format is installed on the machine running the test (see
// internal/generators/engine.go's formatCode, a no-op without it) or on
// exactly where clang-format chooses to wrap/space a long declaration —
// both change whitespace only, never token content, for the templates
// under test here.
func normalize(s string) string {
	return whitespaceRun.ReplaceAllString(s, "")
}

// TestGoldenExampleDomain compiles the representative domain in
// docs/examples (the same one docs/semantics.md's examples README
// describes) all the way from YAML through the IR (internal/ir) and
// through to generated GameAK C++ (internal/generators), and checks both
// stages against checked-in golden output — catching accidental
// regressions in either the IR shape or the generated code shape from a
// single, realistic, multi-model input.
func TestGoldenExampleDomain(t *testing.T) {
	root := findRepoRoot(t)
	dir := t.TempDir()
	copyExampleDomain(t, root, dir)
	defer inDir(t, dir)()

	t.Run("IR matches the golden JSON snapshot", func(t *testing.T) {
		compiled, err := generators.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if compiled.HasErrors() {
			t.Fatalf("unexpected semantic errors: %v", compiled.Errors())
		}
		built, err := ir.Build(compiled)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.MarshalIndent(built, "", "  ")
		if err != nil {
			t.Fatal(err)
		}

		want, err := os.ReadFile(filepath.Join(root, "tests", "testdata", "example-domain-ir.golden.json"))
		if err != nil {
			t.Fatal(err)
		}

		var gotNorm, wantNorm any
		if err := json.Unmarshal(got, &gotNorm); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(want, &wantNorm); err != nil {
			t.Fatal(err)
		}
		gotCanon, _ := json.Marshal(gotNorm)
		wantCanon, _ := json.Marshal(wantNorm)
		if string(gotCanon) != string(wantCanon) {
			t.Fatalf("IR does not match golden snapshot.\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("generated GameAK C++ matches golden content", func(t *testing.T) {
		if err := generators.Sync(); err != nil {
			t.Fatal(err)
		}

		cases := []struct {
			path   string
			golden string
		}{
			{
				path: filepath.Join(dir, "Include", "Core", "Position.hpp"),
				golden: `#pragma once
					#include <GameAk/Core/identity.h>
					#include <GameAk/Runtime/runtime.h>
					#include <cstddef>
					#include <cstdint>
					namespace Core {
					struct Position {
					float x;
					float y;
					float z;
					};
					constexpr size_t kPositionOffset_x = offsetof(Core::Position, x);
					constexpr size_t kPositionOffset_y = offsetof(Core::Position, y);
					constexpr size_t kPositionOffset_z = offsetof(Core::Position, z);
					} // namespace Core
					inline gameak::core::Result<void> register_Position(gameak::runtime::Runtime<>& rt) {
					auto builder = rt.define("Position");
					builder.template size_of<Core::Position>();
					builder.has("x", &Core::Position::x);
					builder.has("y", &Core::Position::y);
					builder.has("z", &Core::Position::z);
					return std::move(builder).done();
					}`,
			},
			{
				path: filepath.Join(dir, "Include", "Core", "MovementSystem.hpp"),
				golden: `#pragma once
					#include <Core/PlayerEntity.hpp>
					#include <GameAk/Runtime/runtime.h>
					namespace Core {
					struct MovementSystem {
					gameak::core::Result<void> operator()(
					gameak::runtime::StateView& view,
					gameak::runtime::CommandProducer& cmds,
					gameak::runtime::EphemeralProducer& ephemeral
					);
					};
					} // namespace Core
					inline gameak::core::Result<void> register_MovementSystem(gameak::runtime::Runtime<>& rt) {
					gameak::core::flat_vector<uint32_t, 4> type_access;
					for (const auto& [tid, desc] : rt.block_types()) {
					if (desc.name == "Position") { type_access.push_back(tid); continue; }
					if (desc.name == "Velocity") { type_access.push_back(tid); continue; }
					}
					return rt.register_controller(Core::MovementSystem{}, 0, type_access);
					}`,
			},
		}

		for _, c := range cases {
			data, err := os.ReadFile(c.path)
			if err != nil {
				t.Fatalf("reading %s: %v", c.path, err)
			}
			if normalize(string(data)) != normalize(c.golden) {
				t.Fatalf("generated %s does not match golden content.\ngot:\n%s\nwant:\n%s", c.path, data, c.golden)
			}
		}
	})
}
