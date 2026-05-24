package adapter

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func NewAdapterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "adapter",
		Aliases: []string{"adapters"},
		Short:   "Manage adapters (messengers, protocols, mobile, desktop)",
		Long: `Manage aps adapter devices — the external transports
(messengers, protocols, mobile, desktop, sense, actuator) the
runtime can talk to. Records live under $APS_DATA_PATH at either
global or profile scope; subcommands cover the full lifecycle:
create / start / stop / status / logs / list, the link parent
(add | list | delete) for profile binding, mobile pairing
(pair | approve | reject | revoke | pending), workspace device
management (attach | detach | presence | permissions), and
messenger-specific helpers (channels | test).

aps messenger is a type-scoped shorthand for the messenger subset
of this tree.`,
	}
	// T-0648 — adapter is an intermediate grouping node with depth-3
	// leaves underneath (link/messenger/permissions subtrees); kit's
	// signature-validator depth-hierarchical check requires the
	// annotation on every intermediate ancestor of a depth>=3 leaf.
	kitcli.SetHierarchical(cmd)

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newLogsCmd())

	// T-0398 — link parent (add/list/delete) replaces flat link/links/unlink.
	cmd.AddCommand(newLinkParentCmd())

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
	cmd.AddCommand(newPermissionsCmd())

	// Messenger device integration (Plan 8)
	cmd.AddCommand(newChannelsCmd())
	cmd.AddCommand(newTestMessengerCmd())

	// Script adapter execution
	cmd.AddCommand(newExecCmd())

	// Messenger alias (T-0363) — type-scoped shorthand for messenger devices.
	cmd.AddCommand(NewMessengerCmd())

	return cmd
}
