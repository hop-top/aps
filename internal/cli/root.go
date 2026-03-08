package cli

import (
	"fmt"
	"os"
	"os/exec"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/tui"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aps",
	Short: "Agent Profile System CLI",
	Long:  `APS is a local-first Agent Profile System that enables running commands and agent workflows under isolated profiles.`,
	Args:  cobra.ArbitraryArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		profiles, err := core.ListProfiles()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return profiles, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		// If no args, launch TUI
		if len(args) == 0 {
			tui.Run()
			return
		}

		// Check if first arg is a profile ID
		profileID := args[0]
		profile, err := core.LoadProfile(profileID)

		if err == nil {
			// It is a valid profile!
			// Dispatch as shorthand execution

			// Case 1: Session (no other args) -> aps run <profile> -- <shell>
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

			// Case 2: Command execution -> aps run <profile> -- <cmd> <args>
			// args[1] is the command, args[2:] are args
			commandName := args[1]
			commandArgs := args[2:]

			if err := core.RunCommand(profileID, commandName, commandArgs); err != nil {
				// We might want to pass through exit code here
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Not a profile, and not a valid subcommand (or Cobra would have routed it)
		// Cobra handles "unknown command" usually, but since we use ArbitraryArgs,
		// we are intercepting everything that isn't a defined subcommand.
		// If LoadProfile failed, it means it's not a profile.
		// So we should print help or error.

		fmt.Fprintf(os.Stderr, "Error: unknown command or profile '%s'\n", profileID)
		cmd.Help()
		os.Exit(1)
	},
}

func Execute() error {
	return rootCmd.Execute()
}
