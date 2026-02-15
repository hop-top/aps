package audit

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
	tableHeader  = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDim)
)

// NewAuditCmd creates the audit command group.
func NewAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit workspace access",
		Long: `View audit logs for workspace access decisions.

Audit logs record every access check performed by devices,
including allowed and denied actions with timestamps and reasons.`,
	}

	cmd.AddCommand(newLogCmd())

	return cmd
}
