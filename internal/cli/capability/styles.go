package capability

import (
	"hop.top/aps/internal/styles"

	"charm.land/lipgloss/v2"
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
