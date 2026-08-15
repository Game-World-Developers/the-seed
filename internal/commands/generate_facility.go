package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Facility generators scaffold hand-written C++ using the Seed facilities
// Phase 7 built (seed::Scene, AssetManager, ...) — not YAML models, and
// not run through the semantic compiler. "seed generate <facility> <name>"
// is deliberately a separate command tree from "seed generate <type>
// <name>" (model generation): a Scene or asset pack has no semantic model
// backing it (docs/facilities.md §6 — Scene is hand-authored C++, there is
// no `scene` YAML model type), so there is nothing for generators.Compile
// to validate here. Conflating the two under one command would imply a
// validation guarantee this output doesn't have.
//
// Not every facility Phase 7 built gets a generator here. seed::ActionMap
// (input map), seed::Bus (audio bus), and the Renderer2D/Renderer3D
// selection (render feature) are each a single, already-directly-usable
// type or enum value — an author writes `seed::ActionMap map; map.bind_key(...)`
// or picks `seed::Bus::Music` inline, there's no per-instance boilerplate
// a generator would remove. Scaffolding an empty file for the sake of
// covering the word "generator" for each would add a file with no content
// worth generating, so those three are intentionally not implemented —
// recorded here as a reasoned scope decision, not an oversight.
const sceneHppTemplate = `#pragma once

#include <Seed/scene.hpp>

namespace game {

// Spawn/despawn intent for the %[1]s scene — see seed::Scene's doc
// comment (Seed/scene.hpp) for why this holds no live GameAK handles.
seed::Scene Make%[1]sScene();

} // namespace game
`

const sceneCppTemplate = `#include "%[1]s.hpp"

namespace game {

seed::Scene Make%[1]sScene() {
    seed::Scene scene("%[1]s");

    scene.on_enter([](gameak::runtime::Runtime<>& rt) {
        (void)rt;
        // TODO: spawn this scene's entities via rt.create_block/submit_command.
    });

    scene.on_exit([](gameak::runtime::Runtime<>& rt) {
        (void)rt;
        // TODO: despawn/cleanup on leaving this scene.
    });

    return scene;
}

} // namespace game
`

const sceneShortHelp = "Scaffold a seed::Scene factory (Src/Game/Scenes/<name>.hpp/.cpp)"
const sceneLongHelp = `Create a starter seed::Scene factory function under Src/Game/Scenes/.
This is hand-written C++, not a YAML model — there is no 'scene' model
type (docs/facilities.md §6), so nothing here is validated by
'seed compile'/'seed sync'; wire the generated Make<Name>Scene() into your
own SceneManager::transition_to call where your game decides scenes
change.`

func runGenerateScene(cmd *cobra.Command, args []string) error {
	if err := requireSeedProject(); err != nil {
		return err
	}
	name := args[0]
	dir := filepath.Join(".", "Src", "Game", "Scenes")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	if err := writeIfAbsent(filepath.Join(dir, name+".hpp"), fmt.Sprintf(sceneHppTemplate, name)); err != nil {
		return err
	}
	return writeIfAbsent(filepath.Join(dir, name+".cpp"), fmt.Sprintf(sceneCppTemplate, name))
}

const assetPackShortHelp = "Scaffold an asset directory for a named group of assets"
const assetPackLongHelp = `Create Assets/<name>/ for a related group of assets (e.g. one
character's textures, one level's audio). This only creates the
directory — add individual 'asset' YAML models under Models/Asset/
pointing into it with 'seed generate asset <name>', which are what
'seed compile'/'seed sync'/'seed package' actually see. There is no
'asset_pack' YAML model type; a pack is a filesystem convention (a
shared directory), not a declared, validated grouping.`

func runGenerateAssetPack(cmd *cobra.Command, args []string) error {
	if err := requireSeedProject(); err != nil {
		return err
	}
	name := args[0]
	dir := filepath.Join(".", "Assets", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	keep := filepath.Join(dir, ".gitkeep")
	if _, err := os.Stat(keep); os.IsNotExist(err) {
		if err := os.WriteFile(keep, nil, 0644); err != nil {
			return err
		}
		fmt.Printf("  %11s  %s\n", "create", dir+"/")
	} else {
		fmt.Printf("  %11s  %s\n", "identical", dir+"/")
	}
	fmt.Printf("Add assets with: seed generate asset <name>  (set its 'path' under %s/)\n", dir)
	return nil
}

func writeIfAbsent(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  %11s  %s\n", "identical", path)
		return nil
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	fmt.Printf("  %11s  %s\n", "create", path)
	return nil
}

func init() {
	generateCmd.AddCommand(&cobra.Command{
		Use: "scene <name>", Short: sceneShortHelp, Long: sceneLongHelp,
		Args: cobra.ExactArgs(1), RunE: runGenerateScene,
	})
	generateCmd.AddCommand(&cobra.Command{
		Use: "asset_pack <name>", Short: assetPackShortHelp, Long: assetPackLongHelp,
		Args: cobra.ExactArgs(1), RunE: runGenerateAssetPack,
	})
	topLevelGenerateCmd.AddCommand(&cobra.Command{
		Use: "scene <name>", Short: sceneShortHelp, Long: sceneLongHelp,
		Args: cobra.ExactArgs(1), RunE: runGenerateScene,
	})
	topLevelGenerateCmd.AddCommand(&cobra.Command{
		Use: "asset_pack <name>", Short: assetPackShortHelp, Long: assetPackLongHelp,
		Args: cobra.ExactArgs(1), RunE: runGenerateAssetPack,
	})
}
