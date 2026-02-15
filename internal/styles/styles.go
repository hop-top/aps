package styles

import "github.com/charmbracelet/lipgloss"

// Palette — consistent across TUI + CLI
var (
	ColorTitle     = lipgloss.Color("205") // pink — headings
	ColorAccent    = lipgloss.Color("170") // light pink — selections, highlights
	ColorSuccess   = lipgloss.Color("42")  // green — confirmations
	ColorError     = lipgloss.Color("196") // red — errors
	ColorWarn      = lipgloss.Color("214") // orange — warnings
	ColorDim       = lipgloss.Color("240") // grey — secondary info
	ColorBuiltin   = lipgloss.Color("75")  // blue — builtin badges
	ColorManaged   = lipgloss.Color("114") // teal — managed badges
	ColorRef       = lipgloss.Color("179") // gold — reference badges
	ColorMessenger = lipgloss.Color("201") // magenta — messenger device badges
	ColorProtocol  = lipgloss.Color("75")  // blue — protocol device badges
	ColorDesktop   = lipgloss.Color("114") // teal — desktop device badges
	ColorMobile    = lipgloss.Color("179") // gold — mobile device badges
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

func DeviceTypeBadge(deviceType string) string {
	switch deviceType {
	case "messenger":
		return lipgloss.NewStyle().Foreground(ColorMessenger).Render("messenger")
	case "protocol":
		return lipgloss.NewStyle().Foreground(ColorProtocol).Render("protocol")
	case "desktop":
		return lipgloss.NewStyle().Foreground(ColorDesktop).Render("desktop")
	case "mobile":
		return lipgloss.NewStyle().Foreground(ColorMobile).Render("mobile")
	case "sense", "actuator":
		return Dim.Render(deviceType)
	default:
		return Dim.Render(deviceType)
	}
}

func DeviceStateBadge(state string) string {
	switch state {
	case "running":
		return lipgloss.NewStyle().Foreground(ColorSuccess).Render("● running")
	case "stopped":
		return lipgloss.NewStyle().Foreground(ColorDim).Render("○ stopped")
	case "starting":
		return lipgloss.NewStyle().Foreground(ColorWarn).Render("◐ starting")
	case "failed":
		return lipgloss.NewStyle().Foreground(ColorError).Render("● failed")
	default:
		return lipgloss.NewStyle().Foreground(ColorDim).Render("○ unknown")
	}
}

func DeviceStateDot(state string) string {
	switch state {
	case "running":
		return lipgloss.NewStyle().Foreground(ColorSuccess).Render("●")
	case "stopped":
		return lipgloss.NewStyle().Foreground(ColorDim).Render("○")
	case "starting":
		return lipgloss.NewStyle().Foreground(ColorWarn).Render("◐")
	case "failed":
		return lipgloss.NewStyle().Foreground(ColorError).Render("●")
	default:
		return lipgloss.NewStyle().Foreground(ColorDim).Render("○")
	}
}

func HealthBadge(health string) string {
	switch health {
	case "healthy":
		return lipgloss.NewStyle().Foreground(ColorSuccess).Render("healthy")
	case "unhealthy":
		return lipgloss.NewStyle().Foreground(ColorError).Render("unhealthy")
	default:
		return Dim.Render("--")
	}
}

func StrategyBadge(strategy string) string {
	switch strategy {
	case "subprocess":
		return Dim.Render("process")
	case "script":
		return Dim.Render("script")
	case "builtin":
		return lipgloss.NewStyle().Foreground(ColorBuiltin).Render("builtin")
	default:
		return Dim.Render(strategy)
	}
}

func ScopeBadge(scope string) string {
	switch scope {
	case "global":
		return Dim.Render("global")
	default:
		return Accent.Render(scope)
	}
}
