package tui

import (
	"os"
	"os/exec"

	"oss-aps-cli/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateProfileList:
			return m.updateProfileList(msg)
		case StateProfileDetail:
			return m.updateProfileDetail(msg)
		case StateCapabilityList:
			return m.updateCapabilityList(msg)
		case StateActionList:
			return m.updateActionList(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case errMsg:
		m.err = msg.err
	}
	return m, nil
}

func (m Model) updateProfileList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.selectedProfile > 0 {
			m.selectedProfile--
		}
	case "down", "j":
		if m.selectedProfile < len(m.profiles)-1 {
			m.selectedProfile++
		}
	case "enter":
		if len(m.profiles) > 0 {
			profileID := m.profiles[m.selectedProfile]
			profile, err := core.LoadProfile(profileID)
			if err != nil {
				m.err = err
				return m, nil
			}
			actions, err := core.LoadActions(profileID)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.profileDetail = profile
			m.actions = actions
			m.capabilities = loadCapabilities(profile)
			m.state = StateProfileDetail
		}
	}
	return m, nil
}

func (m Model) updateProfileDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "c":
		m.state = StateCapabilityList
		m.selectedCap = 0
	case "a", "enter":
		m.state = StateActionList
		m.selectedAction = 0
	case "esc":
		m.state = StateProfileList
		m.profileDetail = nil
	}
	return m, nil
}

func (m Model) updateCapabilityList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.selectedCap > 0 {
			m.selectedCap--
		}
	case "down", "j":
		if m.selectedCap < len(m.capabilities)-1 {
			m.selectedCap++
		}
	case " ":
		if len(m.capabilities) > 0 && m.profileDetail != nil {
			cap := &m.capabilities[m.selectedCap]
			profileID := m.profileDetail.ID
			if cap.Enabled {
				_ = core.RemoveCapabilityFromProfile(profileID, cap.Name)
				cap.Enabled = false
			} else {
				_ = core.AddCapabilityToProfile(profileID, cap.Name)
				cap.Enabled = true
			}
			// Reload profile to keep in sync
			if p, err := core.LoadProfile(profileID); err == nil {
				m.profileDetail = p
			}
		}
	case "esc":
		m.state = StateProfileDetail
	}
	return m, nil
}

func (m Model) updateActionList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.selectedAction > 0 {
			m.selectedAction--
		}
	case "down", "j":
		if m.selectedAction < len(m.actions)-1 {
			m.selectedAction++
		}
	case "enter":
		if len(m.actions) > 0 {
			action := m.actions[m.selectedAction]
			binary, _ := os.Executable()
			if binary == "" {
				binary = os.Args[0]
			}
			c := exec.Command(binary, "action", "run",
				m.profiles[m.selectedProfile], action.ID)
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				if err != nil {
					return errMsg{err}
				}
				return nil
			})
		}
	case "esc":
		if m.profileDetail != nil {
			m.state = StateProfileDetail
		} else {
			m.state = StateProfileList
		}
	}
	return m, nil
}

type errMsg struct{ err error }
