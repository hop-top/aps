package tui

import (
	"fmt"
	"os"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/capability"

	tea "github.com/charmbracelet/bubbletea"
)

type State int

const (
	StateProfileList   State = iota
	StateProfileDetail       // profile info + capabilities
	StateCapabilityList      // manage capabilities for selected profile
	StateActionList
	StateExecution
)

type capItem struct {
	Name        string
	Kind        string // "builtin" or "external"
	Type        string // "managed", "reference", or "--"
	Description string
	Enabled     bool // whether profile has this capability
}

type Model struct {
	state           State
	profiles        []string
	selectedProfile int
	actions         []core.Action
	selectedAction  int
	err             error
	width           int
	height          int

	// Capability management
	capabilities  []capItem
	selectedCap   int
	profileDetail *core.Profile
}

func InitialModel() Model {
	profiles, _ := core.ListProfiles()
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

// loadCapabilities builds the unified capability list for a profile
func loadCapabilities(profile *core.Profile) []capItem {
	items := make([]capItem, 0)
	seen := make(map[string]bool)

	// Add all builtins
	for _, b := range capability.ListBuiltins() {
		enabled := core.ProfileHasCapability(profile, b.Name)
		items = append(items, capItem{
			Name:        b.Name,
			Kind:        string(capability.KindBuiltin),
			Type:        "--",
			Description: b.Description,
			Enabled:     enabled,
		})
		seen[b.Name] = true
	}

	// Add external capabilities from the profile
	for _, capName := range profile.Capabilities {
		if seen[capName] {
			continue
		}
		item := capItem{
			Name:        capName,
			Kind:        string(capability.KindExternal),
			Description: capName,
			Enabled:     true,
		}
		if ext, err := capability.LoadCapability(capName); err == nil {
			item.Type = string(ext.Type)
			if ext.Description != "" {
				item.Description = ext.Description
			}
		}
		items = append(items, item)
		seen[capName] = true
	}

	// Add installed externals not yet in the profile
	if caps, err := capability.List(); err == nil {
		for _, ext := range caps {
			if seen[ext.Name] {
				continue
			}
			items = append(items, capItem{
				Name:        ext.Name,
				Kind:        string(capability.KindExternal),
				Type:        string(ext.Type),
				Description: ext.Description,
				Enabled:     false,
			})
			seen[ext.Name] = true
		}
	}

	return items
}
