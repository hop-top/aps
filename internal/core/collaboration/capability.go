package collaboration

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// CapabilityRegistry maintains a workspace-level index of agent capabilities.
type CapabilityRegistry struct {
	mu    sync.RWMutex
	index map[string][]string // capability -> []profileID
}

// NewCapabilityRegistry creates an empty capability registry.
func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		index: make(map[string][]string),
	}
}

// Register adds an agent's capabilities to the index.
func (cr *CapabilityRegistry) Register(_ context.Context, _ string, profileID string, capabilities []string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Remove existing entries for this agent
	cr.removeAgent(profileID)

	for _, cap := range capabilities {
		cr.index[cap] = append(cr.index[cap], profileID)
	}
	return nil
}

// Refresh re-registers capabilities for an agent. In a full implementation this
// would fetch from the agent's A2A Agent Card. For now it delegates to Register.
func (cr *CapabilityRegistry) Refresh(ctx context.Context, workspaceID, profileID string) error {
	// In production, this would fetch the Agent Card and extract capabilities.
	// For now, the caller must provide capabilities via Register.
	return nil
}

// Unregister removes all capabilities for an agent.
func (cr *CapabilityRegistry) Unregister(profileID string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.removeAgent(profileID)
}

// FindAgents returns agents matching a capability query.
func (cr *CapabilityRegistry) FindAgents(_ context.Context, workspaceID string, query CapabilityQuery) ([]AgentMatch, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	if query.Capability != "" {
		return cr.findByCapability(query.Capability), nil
	}
	if query.Task != "" {
		return cr.findByTaskDescription(query.Task), nil
	}
	return nil, fmt.Errorf("query must specify capability or task")
}

// ListCapabilities returns capabilities for a specific agent, or all capabilities if profileID is empty.
func (cr *CapabilityRegistry) ListCapabilities(_ context.Context, _ string, profileID string) ([]string, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	if profileID == "" {
		caps := make([]string, 0, len(cr.index))
		for cap := range cr.index {
			caps = append(caps, cap)
		}
		return caps, nil
	}

	var caps []string
	for cap, agents := range cr.index {
		for _, a := range agents {
			if a == profileID {
				caps = append(caps, cap)
				break
			}
		}
	}
	return caps, nil
}

// findByCapability returns agents with an exact capability match.
func (cr *CapabilityRegistry) findByCapability(capability string) []AgentMatch {
	agents, ok := cr.index[capability]
	if !ok {
		return nil
	}

	matches := make([]AgentMatch, 0, len(agents))
	for _, a := range agents {
		matches = append(matches, AgentMatch{
			Agent: AgentInfo{ProfileID: a},
			Score: 1.0,
			Match: capability,
		})
	}
	return matches
}

// findByTaskDescription uses fuzzy matching to find agents whose capabilities
// match a task description. Scores by keyword overlap.
func (cr *CapabilityRegistry) findByTaskDescription(task string) []AgentMatch {
	taskWords := strings.Fields(strings.ToLower(task))
	if len(taskWords) == 0 {
		return nil
	}

	type scored struct {
		profileID  string
		capability string
		score      float64
	}

	var candidates []scored

	for cap, agents := range cr.index {
		capLower := strings.ToLower(cap)
		matchCount := 0
		for _, word := range taskWords {
			if strings.Contains(capLower, word) {
				matchCount++
			}
		}
		if matchCount == 0 {
			continue
		}

		score := float64(matchCount) / float64(len(taskWords))
		for _, a := range agents {
			candidates = append(candidates, scored{
				profileID:  a,
				capability: cap,
				score:      score,
			})
		}
	}

	// Deduplicate by agent, keeping highest score
	best := make(map[string]scored)
	for _, c := range candidates {
		if existing, ok := best[c.profileID]; !ok || c.score > existing.score {
			best[c.profileID] = c
		}
	}

	matches := make([]AgentMatch, 0, len(best))
	for _, s := range best {
		matches = append(matches, AgentMatch{
			Agent: AgentInfo{ProfileID: s.profileID},
			Score: s.score,
			Match: s.capability,
		})
	}
	return matches
}

// removeAgent removes all index entries for a profile.
func (cr *CapabilityRegistry) removeAgent(profileID string) {
	for cap, agents := range cr.index {
		var remaining []string
		for _, a := range agents {
			if a != profileID {
				remaining = append(remaining, a)
			}
		}
		if len(remaining) == 0 {
			delete(cr.index, cap)
		} else {
			cr.index[cap] = remaining
		}
	}
}
