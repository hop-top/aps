package conflict

import (
	"hop.top/aps/internal/styles"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	headerStyle  = styles.Title
	dimStyle     = styles.Dim
	boldStyle    = styles.Bold
	errorStyle   = styles.Error
	successStyle = styles.Success
	warnStyle    = styles.Warn
	tableHeader  = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDim)
)

// NewConflictCmd creates the conflict command group.
func NewConflictCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "conflict",
		Aliases: []string{"conflicts"},
		Short:   "Manage workspace conflicts",
		Long: `Manage conflicts that occur when multiple devices modify
the same workspace resource concurrently.

Conflicts are detected automatically during sync. Use these
commands to list, inspect, and resolve them.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newResolveCmd())

	return cmd
}
