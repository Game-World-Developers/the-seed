package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"Game-Developers-World/seed/internal/dist"

	"github.com/spf13/cobra"
)

var packagePlatform string
var packageOutput string

var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Build a distributable asset bundle (manifest + zip)",
	Long: `Cook a manifest of every declared Asset (identity, kind, size, checksum)
and bundle the referenced files into a zip archive under dist/.

This packages assets, not the compiled game binary — building the binary
itself for a given target is xmake's job (see the generated xmake.lua and
docs/platforms.md's target matrix). --platform labels which target the
bundle is for; it does not cross-compile or transform assets per-target
today (see docs/platforms.md's "asset cooking" gap).

App identity (name, version, identifier) comes from app.yaml at the
project root if present, defaulting otherwise — see internal/dist's
AppMetadata doc comment for what's deliberately not handled (icons,
code signing).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}

		model, err := buildIR()
		if err != nil {
			return err
		}

		app, err := dist.LoadAppMetadata(".")
		if err != nil {
			return err
		}

		platform := packagePlatform
		if platform == "" {
			platform = runtime.GOOS
		}

		manifest, err := dist.BuildManifest(model, app, ".", platform)
		if err != nil {
			return fmt.Errorf("building manifest: %w", err)
		}

		outPath := packageOutput
		if outPath == "" {
			outPath = filepath.Join("dist", fmt.Sprintf("%s-%s.zip", app.Name, platform))
		}

		if err := dist.WriteBundle(manifest, ".", outPath); err != nil {
			return fmt.Errorf("writing bundle: %w", err)
		}

		info, err := os.Stat(outPath)
		if err != nil {
			return err
		}
		fmt.Printf("Packaged %s v%s (%s): %d asset(s), %d bytes -> %s\n",
			app.Name, app.Version, platform, len(manifest.Assets), info.Size(), outPath)
		return nil
	},
}

func init() {
	packageCmd.Flags().StringVar(&packagePlatform, "platform", "", "Target platform label for the manifest (defaults to the host OS)")
	packageCmd.Flags().StringVarP(&packageOutput, "output", "o", "", "Output zip path (defaults to dist/<app>-<platform>.zip)")
	rootCmd.AddCommand(packageCmd)
}
