package policy

import (
	"oss-aps-cli/internal/styles"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	headerStyle  = styles.Title
	dimStyle     = styles.Dim
	boldStyle    = styles.Bold
	successStyle = styles.Success
	warnStyle    = styles.Warn
	tableHeader  = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDim)
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
