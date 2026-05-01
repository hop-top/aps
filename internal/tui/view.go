package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	kittui "hop.top/kit/go/console/tui"

	"hop.top/aps/internal/styles"
)

// View composes screens via hop.top/kit/go/console/tui primitives:
// kittui.List manages the visible slice for every list screen,
// kittui.NewModel-equivalent layout (header/footer/main) is supplied by
// the kitFrame field on Model. This file no longer hand-rolls scrolling
// or row layout — only state-specific content composition remains.
func (m Model) View() tea.View {
	if m.err != nil {
		v := tea.NewView(fmt.Sprintf("Error: %v\nPress q to quit.", m.err))
		v.AltScreen = true
		return v
	}

	var content string
	switch m.state {
	case StateProfileList:
		content = renderListScreen("Select Profile",
			profileItemsView(m), "(q to quit)")
	case StateProfileDetail:
		content = renderProfileDetail(m)
	case StateCapabilityList:
		content = renderCapabilityScreen(m)
	case StateActionList:
		content = renderListScreen("Select Action",
			actionItemsView(m), "(esc to back, q to quit)")
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// renderListScreen composes a titled list screen using a pre-rendered
// kit/tui.List View output and a footer hint.
func renderListScreen(title, listView, footer string) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render(title))
	s.WriteString("\n\n")
	s.WriteString(listView)
	s.WriteString("\n\n")
	s.WriteString(footerStyle.Render(footer))
	return s.String()
}

func profileItemsView(m Model) string {
	items := make([]kittui.Item, 0, len(m.profiles))
	for i, p := range m.profiles {
		items = append(items, profileListItem{id: p, selected: i == m.selectedProfile})
	}
	list := kittui.NewList(maxListHeight(len(items), m.height)).SetItems(items)
	return list.View(m.width)
}

func actionItemsView(m Model) string {
	items := make([]kittui.Item, 0, len(m.actions))
	for i, a := range m.actions {
		items = append(items, actionListItem{action: a, selected: i == m.selectedAction})
	}
	list := kittui.NewList(maxListHeight(len(items), m.height)).SetItems(items)
	return list.View(m.width)
}

func renderCapabilityScreen(m Model) string {
	var s strings.Builder
	profileName := ""
	if m.profileDetail != nil {
		profileName = m.profileDetail.ID
	}
	s.WriteString(titleStyle.Render(
		fmt.Sprintf("Capabilities — %s", profileName)))
	s.WriteString("\n\n")

	items := make([]kittui.Item, 0, len(m.capabilities))
	for i, c := range m.capabilities {
		items = append(items, capabilityListItem{cap: c, selected: i == m.selectedCap})
	}
	list := kittui.NewList(maxListHeight(len(items), m.height)).SetItems(items)
	s.WriteString(list.View(m.width))

	enabled, disabled := 0, 0
	for _, c := range m.capabilities {
		if c.Enabled {
			enabled++
		} else {
			disabled++
		}
	}
	s.WriteString("\n\n" + footerStyle.Render(fmt.Sprintf(
		"  [space] toggle  [esc] back  [q] quit    %d enabled, %d disabled",
		enabled, disabled)))
	return s.String()
}

func renderProfileDetail(m Model) string {
	if m.profileDetail == nil {
		return "  Loading..."
	}
	p := m.profileDetail
	name := p.DisplayName
	if name == "" {
		name = p.ID
	}

	var content strings.Builder
	content.WriteString(styles.Bold.Render("Display Name") + ": " + name + "\n")
	if p.Preferences.Shell != "" {
		content.WriteString(styles.Bold.Render("Shell") +
			":        " + p.Preferences.Shell + "\n")
	}
	content.WriteString(styles.Bold.Render("Isolation") +
		":    " + string(p.Isolation.Level) + "\n")

	content.WriteString("\n" + styles.Bold.Render(
		fmt.Sprintf("Capabilities (%d)", len(p.Capabilities))) + "\n")
	for _, cap := range m.capabilities {
		dot := styles.StatusDot(cap.Enabled)
		kind := styles.KindBadge(cap.Kind)
		content.WriteString(fmt.Sprintf("%s %-18s %s\n", dot, cap.Name, kind))
	}
	content.WriteString(fmt.Sprintf("\n"+styles.Bold.Render("Actions")+
		": %d available\n", len(m.actions)))

	box := boxStyle.
		BorderForeground(theme.Accent).
		Width(42).
		Render(content.String())

	var s strings.Builder
	s.WriteString("  " + titleStyle.Render(p.ID) + "\n\n")
	s.WriteString(box + "\n\n")
	s.WriteString(footerStyle.Render(
		"  [c] capabilities  [a] actions  [esc] back  [q] quit"))
	return s.String()
}

// maxListHeight clamps the list visible height so it never exceeds the
// terminal or the item count. With width/height unset (e.g. unit tests)
// it falls back to a sane default.
func maxListHeight(n, termHeight int) int {
	if n < 1 {
		return 1
	}
	if termHeight <= 0 {
		return n
	}
	avail := termHeight - 6 // title + footer + margins
	if avail < 1 {
		avail = 1
	}
	if avail > n {
		return n
	}
	return avail
}
