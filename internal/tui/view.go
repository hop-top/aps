package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().MarginLeft(2).Bold(true).Foreground(lipgloss.Color("205"))
	itemStyle  = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).SetString("> ")
)

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", m.err)
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
		s.WriteString("\n(q to quit)")

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
		s.WriteString("\n(esc to back, q to quit)")
	}

	return s.String()
}