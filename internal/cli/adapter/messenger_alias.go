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
	// T-0398 — link parent (add/list/delete) replaces flat link/links/unlink.
	cmd.AddCommand(newLinkParentCmd())
	cmd.AddCommand(newChannelsCmd())
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
// Output format is controlled by kit's persistent --format flag (T-0345);
// the per-command --json flag was removed in T-0363.
func newMessengerListCmd() *cobra.Command {
	var profileFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messenger devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList("messenger", profileFilter, false)
		},
	}

	cmd.Flags().StringVarP(&profileFilter, "profile", "p", "", "Filter by linked profile")

	return cmd
}

// T-0398 removed newMessengerLinkCmd / newMessengerUnlinkCmd. The
// link parent (newLinkParentCmd) covers add/list/delete.
