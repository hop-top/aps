package adapter

import "github.com/spf13/cobra"

func NewAdapterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "adapter",
		Aliases: []string{"adapters"},
		Short:   "Manage adapters (messengers, protocols, mobile, desktop)",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newLinkCmd())
	cmd.AddCommand(newUnlinkCmd())

	// Mobile device pairing commands
	cmd.AddCommand(newPairCmd())
	cmd.AddCommand(newRevokeCmd())
	cmd.AddCommand(newApproveCmd())
	cmd.AddCommand(newRejectCmd())
	cmd.AddCommand(newPendingCmd())

	// Workspace device management (Plan 7)
	cmd.AddCommand(newAttachCmd())
	cmd.AddCommand(newDetachCmd())
	cmd.AddCommand(newPresenceCmd())
	cmd.AddCommand(newSetPermissionsCmd())

	// Messenger device integration (Plan 8)
	cmd.AddCommand(newChannelsCmd())
	cmd.AddCommand(newLinksCmd())
	cmd.AddCommand(newTestMessengerCmd())

	return cmd
}
