package tui

import (
	"oss-aps-cli/internal/styles"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = styles.Title.MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = styles.Accent.PaddingLeft(2).SetString("> ")
	capBadgeStyle     = lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
	boxStyle          = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(styles.ColorAccent).
				Padding(1, 2)
	footerStyle = styles.Dim
)
