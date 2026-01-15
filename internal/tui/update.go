package tui

import (
	"os"
	"os/exec"

	"oss-aps-cli/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// Update is implemented in view.go due to shared package but we need to fix the Exec logic there.
// Actually, I pasted Update in view.go in the previous step (my bad context tracking).
// I should move Update to update.go or fix it.
// Let's overwrite view.go with just View and put Update in update.go properly.

// Waiting... I wrote Update in view.go? Yes.
// Let's split them.

// This file will hold Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.state == StateProfileList {
				if m.selectedProfile > 0 {
					m.selectedProfile--
				}
			} else if m.state == StateActionList {
				if m.selectedAction > 0 {
					m.selectedAction--
				}
			}
		case "down", "j":
			if m.state == StateProfileList {
				if m.selectedProfile < len(m.profiles)-1 {
					m.selectedProfile++
				}
			} else if m.state == StateActionList {
				if m.selectedAction < len(m.actions)-1 {
					m.selectedAction++
				}
			}
		case "enter":
			if m.state == StateProfileList {
				if len(m.profiles) > 0 {
					profileID := m.profiles[m.selectedProfile]
					actions, err := core.LoadActions(profileID)
					if err != nil {
						m.err = err
					} else {
						m.actions = actions
						m.state = StateActionList
						m.selectedAction = 0
					}
				}
			} else if m.state == StateActionList {
				if len(m.actions) > 0 {
					action := m.actions[m.selectedAction]
					
					// Construct command to run "aps action run <profile> <action>"
					// using the current binary
					binary, _ := os.Executable()
					if binary == "" {
						binary = os.Args[0]
					}
					
					c := exec.Command(binary, "action", "run", m.profiles[m.selectedProfile], action.ID)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						if err != nil {
							return errMsg{err}
						}
						return nil
					})
				}
			}
		case "esc":
			if m.state == StateActionList {
				m.state = StateProfileList
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case errMsg:
		m.err = msg.err
	}
	return m, nil
}

type errMsg struct{ err error }
