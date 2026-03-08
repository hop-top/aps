package squad

import (
	"fmt"
	"sync"
	"time"
)

// Manager provides thread-safe CRUD and membership operations for squads.
// Squads are stored in-memory, keyed by ID.
type Manager struct {
	mu     sync.RWMutex
	squads map[string]*Squad
}

// NewManager creates a Manager with an empty squad registry.
func NewManager() *Manager {
	return &Manager{
		squads: make(map[string]*Squad),
	}
}

// Create validates and stores a new squad. Returns an error if a squad with the
// same ID already exists.
func (m *Manager) Create(s Squad) error {
	if s.ID == "" {
		return fmt.Errorf("squad ID is required")
	}
	if s.Name == "" {
		return fmt.Errorf("squad name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.squads[s.ID]; exists {
		return fmt.Errorf("squad already exists: %s", s.ID)
	}

	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	m.squads[s.ID] = &s

	return nil
}

// Get returns a copy of the squad with the given ID.
func (m *Manager) Get(id string) (*Squad, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, exists := m.squads[id]
	if !exists {
		return nil, fmt.Errorf("squad not found: %s", id)
	}

	cp := *s
	cp.Members = make([]string, len(s.Members))
	copy(cp.Members, s.Members)

	return &cp, nil
}

// List returns copies of all squads.
func (m *Manager) List() []Squad {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Squad, 0, len(m.squads))
	for _, s := range m.squads {
		cp := *s
		cp.Members = make([]string, len(s.Members))
		copy(cp.Members, s.Members)
		result = append(result, cp)
	}

	return result
}

// Update validates and replaces an existing squad. Returns an error if the
// squad does not exist.
func (m *Manager) Update(s Squad) error {
	if s.ID == "" {
		return fmt.Errorf("squad ID is required")
	}
	if s.Name == "" {
		return fmt.Errorf("squad name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.squads[s.ID]
	if !exists {
		return fmt.Errorf("squad not found: %s", s.ID)
	}

	s.CreatedAt = existing.CreatedAt
	s.UpdatedAt = time.Now()
	m.squads[s.ID] = &s

	return nil
}

// Delete removes a squad by ID. Returns an error if the squad does not exist.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.squads[id]; !exists {
		return fmt.Errorf("squad not found: %s", id)
	}

	delete(m.squads, id)
	return nil
}

// AddMember adds a profile to the squad's member list. If the profile is
// already a member, this is a no-op (returns nil).
func (m *Manager) AddMember(squadID, profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.squads[squadID]
	if !exists {
		return fmt.Errorf("squad not found: %s", squadID)
	}

	for _, member := range s.Members {
		if member == profileID {
			return nil
		}
	}

	s.Members = append(s.Members, profileID)
	s.UpdatedAt = time.Now()

	return nil
}

// RemoveMember removes a profile from the squad's member list. Returns an
// error if the profile is not a member.
func (m *Manager) RemoveMember(squadID, profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.squads[squadID]
	if !exists {
		return fmt.Errorf("squad not found: %s", squadID)
	}

	for i, member := range s.Members {
		if member == profileID {
			s.Members = append(s.Members[:i], s.Members[i+1:]...)
			s.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf(
		"profile %s is not a member of squad %s", profileID, squadID,
	)
}

// GetSquadsForProfile returns copies of all squads where the given profile is
// a member.
func (m *Manager) GetSquadsForProfile(profileID string) []Squad {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Squad
	for _, s := range m.squads {
		for _, member := range s.Members {
			if member == profileID {
				cp := *s
				cp.Members = make([]string, len(s.Members))
				copy(cp.Members, s.Members)
				result = append(result, cp)
				break
			}
		}
	}

	return result
}

// IsMember reports whether the given profile is a member of the squad.
func (m *Manager) IsMember(squadID, profileID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, exists := m.squads[squadID]
	if !exists {
		return false
	}

	for _, member := range s.Members {
		if member == profileID {
			return true
		}
	}

	return false
}

// GetInheritedConfig returns the configuration a member profile inherits from
// the squad. Keys: "domain", "type", "golden_path_defined".
func (m *Manager) GetInheritedConfig(squadID string) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, exists := m.squads[squadID]
	if !exists {
		return nil, fmt.Errorf("squad not found: %s", squadID)
	}

	return map[string]any{
		"domain":              s.Domain,
		"type":                string(s.Type),
		"golden_path_defined": s.GoldenPathDefined,
	}, nil
}
