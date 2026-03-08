package adapter

import (
	"oss-aps-cli/internal/styles"

	"github.com/charmbracelet/lipgloss"
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
