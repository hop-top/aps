package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/logging"
	"hop.top/aps/internal/styles"
	"hop.top/aps/internal/tui"
	"hop.top/aps/internal/version"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/core/upgrade"
)

var root = kitcli.New(kitcli.Config{
	Name:    "aps",
	Version: version.Short(),
	Short:   "Agent Profile System CLI",
})

// rootCmd is an alias so other files can call rootCmd.AddCommand() in init().
var rootCmd = root.Cmd

func init() {
	logging.SetViper(root.Viper)

	// Note: kit/go/console/cli.New already calls output.RegisterFlags
	// and output.RegisterHintFlags by default (gated by Config.Disable.
	// Format and .Hints). This wires --format (table|json|yaml) and
	// --no-hints persistent flags on rootCmd, both bound to root.Viper.
	// Subcommands read with root.Viper.GetString("format") /
	// output.HintsEnabled(root.Viper). Tests in root_test.go assert this.

	rootCmd.Long = `Agent Profile System CLI

Run aps with no arguments to launch the interactive TUI.

Pass a profile ID to start a session for that profile, or pass a profile
ID followed by a command to run that command under the selected profile.`

	// ArbitraryArgs + profile dispatch
	rootCmd.Args = cobra.ArbitraryArgs
	rootCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		profiles, err := core.ListProfiles()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return profiles, cobra.ShellCompDirectiveNoFileComp
	}
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		// If no args, launch TUI
		if len(args) == 0 {
			tui.Run()
			return
		}

		// Check if first arg is a profile ID
		profileID := args[0]
		profile, err := core.LoadProfile(profileID)

		if err == nil {
			if len(args) == 1 {
				shell := profile.Preferences.Shell
				if shell == "" {
					shell = core.DetectShell()
				}
				fmt.Printf("Starting session for %s using %s...\n", profileID, shell)
				if err := core.RunCommand(profileID, shell, nil); err != nil {
					logging.GetLogger().Error("session ended with error", err)
					os.Exit(1)
				}
				return
			}

			commandName := args[1]
			commandArgs := args[2:]
			if err := core.RunCommand(profileID, commandName, commandArgs); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				logging.GetLogger().Error("command failed", err)
				os.Exit(1)
			}
			return
		}

		logging.GetLogger().Error("unknown command or profile",
			fmt.Errorf("%q", profileID))
		if err := cmd.Help(); err != nil {
			logging.GetLogger().Error("error rendering help", err)
		}
		os.Exit(1)
	}

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		if cmd.Name() == "upgrade" {
			return
		}
		upgrade.NotifyIfAvailable(cmd.Context(), newChecker(), os.Stderr)
	}

	// Register contextual post-command hints (T-0346).
	registerHints(root.Hints)

	// Render hints after command output. Hints auto-suppress on non-TTY,
	// json/yaml formats, and when --no-hints is set (kit handles this
	// inside output.RenderHints).
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		renderPostRunHintsFor(cmd, root)
		return nil
	}
}

// Execute runs the CLI through fang (styled help, version, etc.)
func Execute() error {
	rootCmd.SilenceErrors = true
	err := root.Execute(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, styles.Error.Render("Error: "+err.Error()))
	}
	return err
}
