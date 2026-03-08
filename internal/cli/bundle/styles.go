package bundle

import (
	"hop.top/aps/internal/styles"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle  = styles.Title
	dimStyle     = styles.Dim
	boldStyle    = styles.Bold
	successStyle = styles.Success
	errorStyle   = styles.Error
	tableHeader  = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDim)
	builtinStyle = lipgloss.NewStyle().Foreground(styles.ColorBuiltin)
	userStyle    = lipgloss.NewStyle().Foreground(styles.ColorManaged)
)
