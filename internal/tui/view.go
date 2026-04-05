package tui

import (
	"fmt"
	"strings"

	"hop.top/aps/internal/styles"

	tea "charm.land/bubbletea/v2"
)

func (m Model) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("Error: %v\nPress q to quit.", m.err))
	}

	var s strings.Builder

	switch m.state {
	case StateProfileList:
		s.WriteString(titleStyle.Render("Select Profile"))
		s.WriteString("\n\n")
		for i, p := range m.profiles {
			if i == m.selectedProfile {
				s.WriteString(selectedItemStyle.Render(p))
			} else {
				s.WriteString(itemStyle.Render(p))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n" + footerStyle.Render("(q to quit)"))

	case StateProfileDetail:
		s.WriteString(m.viewProfileDetail())

	case StateCapabilityList:
		s.WriteString(m.viewCapabilityList())

	case StateActionList:
		s.WriteString(titleStyle.Render("Select Action"))
		s.WriteString("\n\n")
		for i, a := range m.actions {
			label := a.ID
			if a.Title != "" {
				label += " - " + a.Title
			}
			if i == m.selectedAction {
				s.WriteString(selectedItemStyle.Render(label))
			} else {
				s.WriteString(itemStyle.Render(label))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n" + footerStyle.Render("(esc to back, q to quit)"))
	}

	return tea.NewView(s.String())
}

func (m Model) viewProfileDetail() string {
	var s strings.Builder

	if m.profileDetail == nil {
		return "  Loading..."
	}

	p := m.profileDetail
	name := p.DisplayName
	if name == "" {
		name = p.ID
	}

	// Profile info box
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
		BorderForeground(styles.ColorAccent).
		Width(42).
		Render(content.String())
	s.WriteString("  " + titleStyle.Render(p.ID) + "\n\n")
	s.WriteString(box + "\n\n")
	s.WriteString(footerStyle.Render(
		"  [c] capabilities  [a] actions  [esc] back  [q] quit"))

	return s.String()
}

func (m Model) viewCapabilityList() string {
	var s strings.Builder

	profileName := ""
	if m.profileDetail != nil {
		profileName = m.profileDetail.ID
	}

	s.WriteString(titleStyle.Render(
		fmt.Sprintf("Capabilities — %s", profileName)))
	s.WriteString("\n\n")

	for i, cap := range m.capabilities {
		dot := styles.StatusDot(cap.Enabled)
		kind := styles.KindBadge(cap.Kind)
		typ := ""
		if cap.Kind == "external" {
			typ = "  " + styles.TypeBadge(cap.Type)
		}
		desc := styles.Dim.Render(cap.Description)
		line := fmt.Sprintf("%s %-18s %s%s  %s",
			dot, cap.Name, kind, typ, desc)

		if i == m.selectedCap {
			s.WriteString(selectedItemStyle.Render(line))
		} else {
			s.WriteString(itemStyle.Render(line))
		}
		s.WriteString("\n")
	}

	enabled := 0
	disabled := 0
	for _, c := range m.capabilities {
		if c.Enabled {
			enabled++
		} else {
			disabled++
		}
	}
	s.WriteString("\n" + footerStyle.Render(fmt.Sprintf(
		"  [space] toggle  [esc] back  [q] quit    %d enabled, %d disabled",
		enabled, disabled)))

	return s.String()
}
