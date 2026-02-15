package collab

import (
	"github.com/spf13/cobra"
)

// NewCollabCmd creates the collab command group.
func NewCollabCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "collab",
		Aliases: []string{"collaboration"},
		Short:   "Multi-agent collaboration workspaces",
		Long: `Manage multi-agent collaboration workspaces.

Create workspaces where multiple agents coordinate, communicate,
and resolve conflicts. Agents join workspaces, share context,
exchange capabilities, and send tasks to each other.

Set an active workspace to avoid repeating the workspace name:
  aps collab use my-team
  aps collab members      # uses active workspace`,
	}

	// Workspace lifecycle
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

	// Conflicts
	cmd.AddCommand(NewConflictsCmd())
	cmd.AddCommand(NewResolveCmd())
	cmd.AddCommand(NewPolicyCmd())

	// Monitoring
	cmd.AddCommand(NewAuditCmd())

	return cmd
}
