package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"hop.top/aps/internal/core"

	tea "charm.land/bubbletea/v2"
	kitcli "hop.top/kit/go/console/cli"
	kittui "hop.top/kit/go/console/tui"
)

// Run mounts the TUI on top of kit/console/tui.AppShell. The shell owns
// the bubbletea program loop, the WindowSize / quit / help keybindings,
// and the themed header / footer chrome; this package supplies the
// per-screen content rendering and the state machine via the Model
// type which implements kit's AppRenderer interface.
func Run(ctx context.Context, root *kitcli.Root) error {
	// Drop esc from the canonical quit keymap so it remains free for
	// per-screen "back" navigation. ctrl+c and q still quit globally.
	km := kittui.DefaultKeyMap()
	km.Quit = []string{"q", "ctrl+c"}

	shell := kittui.NewAppShellFromRoot(InitialModel(), root, kittui.WithKeyMap(km))
	if _, err := shell.Run(ctx); err != nil {
		return fmt.Errorf("appshell run: %w", err)
	}
	return nil
}

// Init satisfies kit's Initer interface — invoked once when AppShell
// boots. No initial commands needed; data is loaded eagerly in
// InitialModel.
func (m Model) Init() tea.Cmd { return nil }

// Resize satisfies kit's Resizer interface — AppShell forwards the
// post-chrome width and height (terminal size minus header / footer).
func (m Model) Resize(width, height int) kittui.AppRenderer {
	m.width = width
	m.height = height
	return m
}

// Update satisfies kit's Updater interface. The shell strips its
// canonical keys (ctrl+c, q, ?, h) before dispatch; per-screen handlers
// here own everything else, including esc.
func (m Model) Update(msg tea.Msg) (kittui.AppRenderer, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		var (
			next kittui.AppRenderer
			cmd  tea.Cmd
		)
		switch m.state {
		case StateProfileList:
			next = m.updateProfileList(msg)
		case StateProfileDetail:
			next = m.updateProfileDetail(msg)
		case StateCapabilityList:
			next = m.updateCapabilityList(msg)
		case StateActionList:
			next, cmd = m.updateActionList(msg)
		default:
			next = m
		}
		return next, cmd
	case errMsg:
		m.err = msg.err
	}
	return m, nil
}

func (m Model) updateProfileList(msg tea.KeyPressMsg) kittui.AppRenderer {
	switch msg.String() {
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
				return m
			}
			actions, err := core.LoadActions(profileID)
			if err != nil {
				m.err = err
				return m
			}
			m.profileDetail = profile
			m.actions = actions
			m.capabilities = loadCapabilities(profile)
			m.state = StateProfileDetail
		}
	}
	return m
}

func (m Model) updateProfileDetail(msg tea.KeyPressMsg) kittui.AppRenderer {
	switch msg.String() {
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
	return m
}

func (m Model) updateCapabilityList(msg tea.KeyPressMsg) kittui.AppRenderer {
	switch msg.String() {
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
	return m
}

func (m Model) updateActionList(msg tea.KeyPressMsg) (kittui.AppRenderer, tea.Cmd) {
	switch msg.String() {
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

var (
	_ kittui.AppRenderer = Model{}
	_ kittui.Initer      = Model{}
	_ kittui.Updater     = Model{}
	_ kittui.Resizer     = Model{}
)
