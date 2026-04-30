// Package-level styles wired to kit's themed TUI palette
// (hop.top/kit/go/console/tui + tui/styles). Domain-specific badges
// (capability/role/device) remain in hop.top/aps/internal/styles.
package tui

import (
	"charm.land/lipgloss/v2"
	"hop.top/kit/go/console/cli"
	tuistyles "hop.top/kit/go/console/tui/styles"
)

// theme is the kit Theme used to derive aps's TUI styling. The aps CLI is
// constructed with default (Neon) palette in internal/cli/root.go; we
// reproduce that here so the tui can be invoked without dragging the cobra
// root in. Visual parity is preserved — both paths feed the same palette.
var theme = cli.New(cli.Config{Name: "aps"}).Theme

// kitStyles is the central themed style bundle. Layout-region styles
// (Header, Sidebar, Main, Footer) come pre-built; semantic colors
// (Accent, Muted, etc.) are reused for component-level overrides.
var kitStyles = tuistyles.NewStyles(theme)

var (
	// titleStyle keeps the existing 2-cell left margin while drawing
	// from kit's themed Title foreground.
	titleStyle = kitStyles.Title.MarginLeft(2)

	// itemStyle: unselected list rows.
	itemStyle = lipgloss.NewStyle().PaddingLeft(4)

	// selectedItemStyle: selected list row prefixed with the kit accent.
	selectedItemStyle = kitStyles.Accent.PaddingLeft(2).SetString("> ")

	// boxStyle: rounded panel framing the profile-detail view, painted
	// with kit's accent color.
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Accent).
			Padding(1, 2)

	// footerStyle: muted helper-key line.
	footerStyle = kitStyles.Muted
)
