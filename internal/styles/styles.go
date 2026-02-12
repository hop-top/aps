package styles

import "github.com/charmbracelet/lipgloss"

// Palette — consistent across TUI + CLI
var (
	ColorTitle   = lipgloss.Color("205") // pink — headings
	ColorAccent  = lipgloss.Color("170") // light pink — selections, highlights
	ColorSuccess = lipgloss.Color("42")  // green — confirmations
	ColorError   = lipgloss.Color("196") // red — errors
	ColorWarn    = lipgloss.Color("214") // orange — warnings
	ColorDim     = lipgloss.Color("240") // grey — secondary info
	ColorBuiltin = lipgloss.Color("75")  // blue — builtin badges
	ColorManaged = lipgloss.Color("114") // teal — managed badges
	ColorRef     = lipgloss.Color("179") // gold — reference badges
)

// Renderers
var (
	Title   = lipgloss.NewStyle().Bold(true).Foreground(ColorTitle)
	Accent  = lipgloss.NewStyle().Foreground(ColorAccent)
	Success = lipgloss.NewStyle().Foreground(ColorSuccess)
	Error   = lipgloss.NewStyle().Bold(true).Foreground(ColorError)
	Warn    = lipgloss.NewStyle().Foreground(ColorWarn)
	Dim     = lipgloss.NewStyle().Foreground(ColorDim)
	Bold    = lipgloss.NewStyle().Bold(true)
)

// KindBadge renders a colored badge for capability kind
func KindBadge(kind string) string {
	switch kind {
	case "builtin":
		return lipgloss.NewStyle().Foreground(ColorBuiltin).Render("builtin")
	case "external":
		return lipgloss.NewStyle().Foreground(ColorDim).Render("external")
	default:
		return lipgloss.NewStyle().Foreground(ColorDim).Render(kind)
	}
}

// TypeBadge renders a colored badge for capability type (managed/reference)
func TypeBadge(typ string) string {
	switch typ {
	case "managed":
		return lipgloss.NewStyle().Foreground(ColorManaged).Render("managed")
	case "reference":
		return lipgloss.NewStyle().Foreground(ColorRef).Render("reference")
	default:
		return Dim.Render("--")
	}
}

// StatusDot renders a colored dot for enabled/disabled state
func StatusDot(enabled bool) string {
	if enabled {
		return lipgloss.NewStyle().Foreground(ColorSuccess).Render("●")
	}
	return lipgloss.NewStyle().Foreground(ColorDim).Render("○")
}
