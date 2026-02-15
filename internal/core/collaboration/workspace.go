package collaboration

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Workspace represents a shared execution context where multiple agents coordinate.
type Workspace struct {
	mu sync.RWMutex

	ID        string         `json:"id" yaml:"id"`
	Config    WorkspaceConfig `json:"config" yaml:"config"`
	State     WorkspaceState `json:"state" yaml:"state"`
	Agents    []AgentInfo    `json:"agents" yaml:"agents"`
	Context   *WorkspaceContext `json:"-" yaml:"-"` // managed separately
	Policy    PolicyConfig   `json:"policy" yaml:"policy"`
	CreatedAt time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" yaml:"updated_at"`
}

// NewWorkspace creates a workspace from validated config.
func NewWorkspace(config WorkspaceConfig) (*Workspace, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid workspace config: %w", err)
	}

	now := time.Now()
	ws := &Workspace{
		ID:     uuid.New().String(),
		Config: config,
		State:  StateActive,
		Agents: []AgentInfo{
			{
				ProfileID:   config.OwnerProfileID,
				Role:        RoleOwner,
				JoinedAt:    now,
				LastSeen:    now,
				Status:      "online",
				Capabilities: []string{},
			},
		},
		Context: NewWorkspaceContext(),
		Policy: PolicyConfig{
			Default: config.DefaultPolicy,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	return ws, nil
}

// AddAgent registers a new agent in the workspace.
func (ws *Workspace) AddAgent(profileID string, role AgentRole) (*AgentInfo, error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.State != StateActive {
		return nil, fmt.Errorf("workspace %q is not active (state: %s)", ws.ID, ws.State)
	}

	for _, a := range ws.Agents {
		if a.ProfileID == profileID {
			return nil, fmt.Errorf("agent %q already in workspace", profileID)
		}
	}

	if ws.Config.MaxAgents > 0 && len(ws.Agents) >= ws.Config.MaxAgents {
		return nil, fmt.Errorf("workspace at capacity (%d agents)", ws.Config.MaxAgents)
	}

	now := time.Now()
	agent := AgentInfo{
		ProfileID:    profileID,
		Role:         role,
		JoinedAt:     now,
		LastSeen:     now,
		Status:       "online",
		Capabilities: []string{},
	}
	ws.Agents = append(ws.Agents, agent)
	ws.UpdatedAt = now
	return &agent, nil
}

// RemoveAgent deregisters an agent from the workspace.
func (ws *Workspace) RemoveAgent(profileID string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	idx := -1
	for i, a := range ws.Agents {
		if a.ProfileID == profileID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("agent %q not in workspace", profileID)
	}

	// Prevent removing last owner
	if ws.Agents[idx].Role == RoleOwner {
		ownerCount := 0
		for _, a := range ws.Agents {
			if a.Role == RoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return fmt.Errorf("cannot remove the last owner from workspace")
		}
	}

	ws.Agents = append(ws.Agents[:idx], ws.Agents[idx+1:]...)
	ws.UpdatedAt = time.Now()
	return nil
}

// GetAgent returns agent info by profile ID.
func (ws *Workspace) GetAgent(profileID string) (*AgentInfo, error) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	for i := range ws.Agents {
		if ws.Agents[i].ProfileID == profileID {
			return &ws.Agents[i], nil
		}
	}
	return nil, fmt.Errorf("agent %q not in workspace", profileID)
}

// SetAgentRole changes an agent's role.
func (ws *Workspace) SetAgentRole(profileID string, role AgentRole) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if err := role.Validate(); err != nil {
		return err
	}

	for i := range ws.Agents {
		if ws.Agents[i].ProfileID == profileID {
			oldRole := ws.Agents[i].Role

			// Prevent demoting the last owner
			if oldRole == RoleOwner && role != RoleOwner {
				ownerCount := 0
				for _, a := range ws.Agents {
					if a.Role == RoleOwner {
						ownerCount++
					}
				}
				if ownerCount <= 1 {
					return fmt.Errorf("cannot demote the last owner")
				}
			}

			ws.Agents[i].Role = role
			ws.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("agent %q not in workspace", profileID)
}

// UpdateAgentStatus updates an agent's online status and last-seen time.
func (ws *Workspace) UpdateAgentStatus(profileID, status string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	for i := range ws.Agents {
		if ws.Agents[i].ProfileID == profileID {
			ws.Agents[i].Status = status
			ws.Agents[i].LastSeen = time.Now()
			return nil
		}
	}
	return fmt.Errorf("agent %q not in workspace", profileID)
}

// SetState transitions the workspace to a new state.
func (ws *Workspace) SetState(state WorkspaceState) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if err := state.Validate(); err != nil {
		return err
	}

	// Validate transitions
	switch ws.State {
	case StateCreating:
		if state != StateActive && state != StateArchived {
			return fmt.Errorf("cannot transition from %s to %s", ws.State, state)
		}
	case StateActive:
		if state != StateClosing && state != StateArchived {
			return fmt.Errorf("cannot transition from %s to %s", ws.State, state)
		}
	case StateClosing:
		if state != StateArchived {
			return fmt.Errorf("cannot transition from %s to %s", ws.State, state)
		}
	case StateArchived:
		return fmt.Errorf("archived workspace cannot transition")
	}

	ws.State = state
	ws.UpdatedAt = time.Now()
	return nil
}

// OnlineAgentCount returns the number of agents with "online" status.
func (ws *Workspace) OnlineAgentCount() int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	count := 0
	for _, a := range ws.Agents {
		if a.Status == "online" {
			count++
		}
	}
	return count
}

// HasAgent checks if a profile is a member of this workspace.
func (ws *Workspace) HasAgent(profileID string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	for _, a := range ws.Agents {
		if a.ProfileID == profileID {
			return true
		}
	}
	return false
}

// IsOwner checks if a profile is an owner of this workspace.
func (ws *Workspace) IsOwner(profileID string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	for _, a := range ws.Agents {
		if a.ProfileID == profileID && a.Role == RoleOwner {
			return true
		}
	}
	return false
}
