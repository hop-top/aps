package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/tui"
	"hop.top/aps/internal/version"
	"hop.top/upgrade"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/cli"
)

var (
	root    *kitcli.Root
	rootCmd *cobra.Command
)

func init() {
	root = kitcli.New(kitcli.Config{
		Name:    "aps",
		Version: version.Short(),
		Short:   "Agent Profile System CLI",
	})

	// Keep rootCmd for backward compat with init() AddCommand calls
	rootCmd = root.Cmd

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
					fmt.Fprintf(os.Stderr, "Session ended with error: %v\n", err)
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
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		fmt.Fprintf(os.Stderr, "Error: unknown command or profile '%s'\n", profileID)
		cmd.Help()
		os.Exit(1)
	}

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		if cmd.Name() == "upgrade" {
			return
		}
		upgrade.NotifyIfAvailable(cmd.Context(), newChecker(), os.Stderr)
	}
}

// Execute runs the CLI through fang (styled help, version, etc.)
func Execute() error {
	return root.Execute(context.Background())
}
