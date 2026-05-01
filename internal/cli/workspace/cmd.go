package workspace

import (
	"github.com/spf13/cobra"
)

// NewWorkspaceCmd creates the workspace command group.
//
// Workspace is the canonical noun for collaboration, audit, and
// conflict surfaces. Subcommands previously hosted under aps collab,
// aps audit, and aps conflict have been merged here.
func NewWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Manage multi-agent workspaces",
		Long: `Manage workspaces for multi-agent collaboration.

A workspace is a shared context where agents coordinate, exchange
tasks, share context, and resolve conflicts. Set an active
workspace to avoid repeating its name:

  aps workspace use my-team
  aps workspace members      # uses active workspace`,
	}

	// Lifecycle
	cmd.AddCommand(NewNewCmd())
	cmd.AddCommand(NewShowCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewJoinCmd())
	cmd.AddCommand(NewLeaveCmd())
	cmd.AddCommand(NewRemoveCmd())
	cmd.AddCommand(NewRoleCmd())
	cmd.AddCommand(NewMembersCmd())
	cmd.AddCommand(NewUseCmd())
	cmd.AddCommand(NewArchiveCmd())

	// Tasks
	cmd.AddCommand(NewSendCmd())
	cmd.AddCommand(NewTasksCmd())
	cmd.AddCommand(NewTaskCmd())

	// Context
	cmd.AddCommand(NewCtxCmd())

	// Capabilities
	cmd.AddCommand(NewAgentsCmd())
	cmd.AddCommand(NewCapsCmd())

	// Conflicts (merged from internal/cli/conflict)
	cmd.AddCommand(NewConflictsCmd())
	cmd.AddCommand(NewPolicyCmd())

	// Monitoring
	cmd.AddCommand(NewAuditCmd())

	// Multi-adapter workspace surface
	cmd.AddCommand(NewActivityCmd())
	cmd.AddCommand(NewSyncCmd())

	return cmd
}

// NewConflictsCmd creates the "workspace conflicts" sub-group with
// list, show, and resolve subcommands.
func NewConflictsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "conflicts",
		Aliases: []string{"conflict"},
		Short:   "Manage workspace conflicts",
		Long: `Manage conflicts that occur when multiple devices modify
the same workspace resource concurrently.

Conflicts are detected automatically during sync. Use these
commands to list, inspect, and resolve them.`,
	}

	cmd.AddCommand(newConflictsListCmd())
	cmd.AddCommand(newConflictsShowCmd())
	cmd.AddCommand(newConflictsResolveCmd())

	return cmd
}
