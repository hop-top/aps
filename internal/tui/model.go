package tui

import (
	"fmt"
	"os"

	"oss-aps-cli/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

type State int

const (
	StateProfileList State = iota
	StateActionList
	StateExecution
)

type Model struct {
	state           State
	profiles        []string
	selectedProfile int
	actions         []core.Action
	selectedAction  int
	err             error
	width           int
	height          int
}

func InitialModel() Model {
	profiles, _ := core.ListProfiles() // ignore err for TUI init? Or show error state?
	return Model{
		state:    StateProfileList,
		profiles: profiles,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func Run() {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
