package commands

import (
	"fmt"
	"os"
	"os/exec"

	"Game-Developers-World/seed/internal/generators"

	"github.com/spf13/cobra"
)

// runXmake shells out to xmake with a consistent --profile/-y translation
// across build/run/test, so a project author only has to learn Seed's own
// flag once instead of xmake's per-subcommand conventions. Unlike
// `xmake build`/`run`/`test`, xmake's build-mode flag (`-m debug|release`)
// is only recognized by its config step (`xmake f`/`xmake config`), not by
// build/run/test directly — so --profile runs that config step first,
// only when given, leaving xmake's own current configuration untouched
// otherwise.
func runXmake(profile string, xmakeArgs ...string) error {
	if _, err := exec.LookPath("xmake"); err != nil {
		return fmt.Errorf("xmake not found in PATH: install it from https://xmake.io")
	}
	if profile != "" {
		if profile != "debug" && profile != "release" {
			return fmt.Errorf("invalid --profile %q: must be debug or release", profile)
		}
		if err := runXmakeCommand("f", "-m", profile, "-y"); err != nil {
			return fmt.Errorf("configuring profile %s: %w", profile, err)
		}
	}
	return runXmakeCommand(append(xmakeArgs, "-y")...)
}

func runXmakeCommand(args ...string) error {
	cmd := exec.Command("xmake", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

var buildProfile string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the project's C++ binary with xmake",
	Long: `Build the current project with xmake. This does not run "seed sync"
first — run that separately (or use "seed run", which does) if models
changed since the last generation.

--profile selects xmake's build mode (debug or release); see
docs/platforms.md §5 for what each implies (symbols, optimization).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		return runXmake(buildProfile, "build")
	},
}

var runProfile string
var runSkipSync bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Sync models, build, and run the project's binary",
	Long: `Runs "seed sync" (unless --no-sync), then builds and runs the project's
binary via xmake — the single command for "make my latest models show up
running," matching how Rails' equivalent commands don't make an author
remember a multi-step sequence.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		if !runSkipSync {
			if err := checkGameAKBackend(); err != nil {
				return err
			}
			fmt.Println("Syncing models...")
			if err := generators.Sync(); err != nil {
				return err
			}
		}
		return runXmake(runProfile, "run")
	},
}

var testProfile string

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run the project's C++ tests with xmake",
	Long: `Runs "xmake test" against the current project. A freshly scaffolded
Seed project has no test targets defined in its xmake.lua — this command
doesn't add any (deciding what a project's C++ tests look like is a game
author's choice, not a Seed convention imposed on them) — so it succeeds
with "no tests to run" until a project adds test targets of its own.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSeedProject(); err != nil {
			return err
		}
		return runXmake(testProfile, "test")
	},
}

func init() {
	buildCmd.Flags().StringVar(&buildProfile, "profile", "", "Build profile: debug or release (defaults to xmake's current config)")
	rootCmd.AddCommand(buildCmd)

	runCmd.Flags().StringVar(&runProfile, "profile", "", "Build profile: debug or release (defaults to xmake's current config)")
	runCmd.Flags().BoolVar(&runSkipSync, "no-sync", false, "Skip running 'seed sync' before building")
	rootCmd.AddCommand(runCmd)

	testCmd.Flags().StringVar(&testProfile, "profile", "", "Build profile: debug or release (defaults to xmake's current config)")
	rootCmd.AddCommand(testCmd)
}
