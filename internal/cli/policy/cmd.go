package policy

import (
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

// T-0456 — tableHeader (lipgloss bold-dim style) was removed when
// `aps policy list` migrated to listing.RenderList; the styled table
// renderer now applies header styling from the active kit/cli theme.
var (
	headerStyle  = styles.Title
	dimStyle     = styles.Dim
	boldStyle    = styles.Bold
	successStyle = styles.Success
	warnStyle    = styles.Warn
)

// NewPolicyCmd creates the policy command group.
func NewPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "policy",
		Aliases: []string{"policies"},
		Short:   "Manage workspace access policies",
		Long: `Manage access control policies for workspaces.

Policies control which devices can access a workspace and what
they can do. Modes:
  allow-all   All linked devices have access (default)
  allow-list  Only specified devices have access
  deny-list   All devices except specified ones have access`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newTrustCmd())

	return cmd
}
