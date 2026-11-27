package squad

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Manager provides thread-safe CRUD and membership operations for squads.
// Squads are keyed by ID. A manager from NewManager is in-memory only; one
// from NewFileManager mirrors every mutation to disk and reloads before each
// operation, so separate managers (processes) over one directory agree.
type Manager struct {
	mu     sync.Mutex
	squads map[string]*Squad
	dir    string                 // empty = in-memory only
	dirFn  func() (string, error) // resolves dir per operation when set
}

// NewManager creates a Manager with an empty squad registry.
func NewManager() *Manager {
	return &Manager{
		squads: make(map[string]*Squad),
	}
}

// lock acquires the manager lock and, when file-backed, reloads state from
// disk. Callers: `unlock, err := m.lock()`.
func (m *Manager) lock() (func(), error) {
	m.mu.Lock()
	if m.dirFn != nil {
		dir, err := m.dirFn()
		if err != nil {
			m.mu.Unlock()
			return nil, err
		}
		m.dir = dir
	}
	if m.dir == "" {
		return m.mu.Unlock, nil
	}
	if err := m.reload(); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	return m.mu.Unlock, nil
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

	unlock, err := m.lock()
	if err != nil {
		return err
	}
	defer unlock()

	if _, exists := m.squads[s.ID]; exists {
		return fmt.Errorf("squad already exists: %s", s.ID)
	}

	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	m.squads[s.ID] = &s

	return m.save(&s)
}

// Get returns a copy of the squad with the given ID.
func (m *Manager) Get(id string) (*Squad, error) {
	unlock, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	s, exists := m.squads[id]
	if !exists {
		return nil, fmt.Errorf("squad not found: %s", id)
	}

	cp := *s
	cp.Members = make([]string, len(s.Members))
	copy(cp.Members, s.Members)

	return &cp, nil
}

// List returns copies of all squads. Load failures yield an empty result;
// use ListChecked to observe them.
func (m *Manager) List() []Squad {
	out, _ := m.ListChecked()
	return out
}

// ListChecked is List with load errors surfaced.
func (m *Manager) ListChecked() ([]Squad, error) {
	unlock, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	result := make([]Squad, 0, len(m.squads))
	for _, s := range m.squads {
		cp := *s
		cp.Members = make([]string, len(s.Members))
		copy(cp.Members, s.Members)
		result = append(result, cp)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })

	return result, nil
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

	unlock, err := m.lock()
	if err != nil {
		return err
	}
	defer unlock()

	existing, exists := m.squads[s.ID]
	if !exists {
		return fmt.Errorf("squad not found: %s", s.ID)
	}

	s.CreatedAt = existing.CreatedAt
	s.UpdatedAt = time.Now()
	m.squads[s.ID] = &s

	return m.save(&s)
}

// Delete removes a squad by ID. Returns an error if the squad does not exist.
func (m *Manager) Delete(id string) error {
	unlock, err := m.lock()
	if err != nil {
		return err
	}
	defer unlock()

	if _, exists := m.squads[id]; !exists {
		return fmt.Errorf("squad not found: %s", id)
	}

	if err := m.remove(id); err != nil {
		return err
	}
	delete(m.squads, id)
	return nil
}

// AddMember adds a profile to the squad's member list. If the profile is
// already a member, this is a no-op (returns nil).
func (m *Manager) AddMember(squadID, profileID string) error {
	unlock, err := m.lock()
	if err != nil {
		return err
	}
	defer unlock()

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

	return m.save(s)
}

// RemoveMember removes a profile from the squad's member list. Returns an
// error if the profile is not a member.
func (m *Manager) RemoveMember(squadID, profileID string) error {
	unlock, err := m.lock()
	if err != nil {
		return err
	}
	defer unlock()

	s, exists := m.squads[squadID]
	if !exists {
		return fmt.Errorf("squad not found: %s", squadID)
	}

	for i, member := range s.Members {
		if member == profileID {
			s.Members = append(s.Members[:i], s.Members[i+1:]...)
			s.UpdatedAt = time.Now()
			return m.save(s)
		}
	}

	return fmt.Errorf(
		"profile %s is not a member of squad %s", profileID, squadID,
	)
}

// GetSquadsForProfile returns copies of all squads where the given profile is
// a member.
func (m *Manager) GetSquadsForProfile(profileID string) []Squad {
	unlock, err := m.lock()
	if err != nil {
		return nil
	}
	defer unlock()

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
	unlock, err := m.lock()
	if err != nil {
		return false
	}
	defer unlock()

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
	unlock, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()

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
