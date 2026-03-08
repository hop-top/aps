package adapter

import (
	"github.com/spf13/cobra"
)

// NewMessengerCmd creates the "aps messenger" alias command.
// It delegates to device subcommands that apply to messenger devices,
// providing a convenient shorthand for messenger-focused workflows.
func NewMessengerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "messenger",
		Aliases: []string{"messengers"},
		Short:   "Messenger device commands (alias for 'aps device' with messenger context)",
		Long: `Messenger commands provide shortcuts for common messenger device operations.

These commands are equivalent to their 'aps device' counterparts but
pre-filtered for messenger-type devices.`,
	}

	// Core messenger operations
	cmd.AddCommand(newMessengerListCmd())
	cmd.AddCommand(newMessengerLinkCmd())
	cmd.AddCommand(newMessengerUnlinkCmd())
	cmd.AddCommand(newChannelsCmd())
	cmd.AddCommand(newLinksCmd())
	cmd.AddCommand(newTestMessengerCmd())

	// Lifecycle commands
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newCreateCmd())

	return cmd
}

// newMessengerListCmd wraps the list command with --type=messenger preset.
func newMessengerListCmd() *cobra.Command {
	var profileFilter string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messenger devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList("messenger", profileFilter, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&profileFilter, "profile", "p", "", "Filter by linked profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

// newMessengerLinkCmd wraps the link command for messenger context.
func newMessengerLinkCmd() *cobra.Command {
	return newLinkCmd()
}

// newMessengerUnlinkCmd wraps the unlink command for messenger context.
func newMessengerUnlinkCmd() *cobra.Command {
	return newUnlinkCmd()
}
