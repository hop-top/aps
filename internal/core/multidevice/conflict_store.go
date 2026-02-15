package multidevice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ConflictStore persists conflicts to the filesystem as individual JSON
// files under ~/.aps/workspaces/{workspace_id}/conflicts/.
type ConflictStore struct {
	workspaceID string
	mu          sync.RWMutex
}

// NewConflictStore creates a store for the given workspace.
func NewConflictStore(workspaceID string) *ConflictStore {
	return &ConflictStore{
		workspaceID: workspaceID,
	}
}

// conflictsDir returns the directory where conflicts are stored for this
// workspace.
func (s *ConflictStore) conflictsDir() (string, error) {
	wsDir, err := GetWorkspaceDir(s.workspaceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(wsDir, "conflicts"), nil
}

// ensureDir creates the conflicts directory if it does not exist.
func (s *ConflictStore) ensureDir() error {
	dir, err := s.conflictsDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// conflictPath returns the file path for a given conflict ID.
func (s *ConflictStore) conflictPath(conflictID string) (string, error) {
	dir, err := s.conflictsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, conflictID+".json"), nil
}

// Save persists a conflict to disk.
func (s *ConflictStore) Save(conflict *Conflict) error {
	if conflict == nil {
		return fmt.Errorf("conflict must not be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("creating conflicts directory: %w", err)
	}

	data, err := json.MarshalIndent(conflict, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling conflict %s: %w", conflict.ID, err)
	}

	path, err := s.conflictPath(conflict.ID)
	if err != nil {
		return fmt.Errorf("resolving conflict path: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing conflict %s: %w", conflict.ID, err)
	}

	return nil
}

// Load reads a conflict from disk by its ID.
func (s *ConflictStore) Load(conflictID string) (*Conflict, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path, err := s.conflictPath(conflictID)
	if err != nil {
		return nil, fmt.Errorf("resolving conflict path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("conflict %s not found", conflictID)
		}
		return nil, fmt.Errorf("reading conflict %s: %w", conflictID, err)
	}

	var conflict Conflict
	if err := json.Unmarshal(data, &conflict); err != nil {
		return nil, fmt.Errorf("unmarshaling conflict %s: %w", conflictID, err)
	}

	return &conflict, nil
}

// List returns all conflicts for the workspace. When includeResolved is
// false, only pending and manual conflicts are returned.
func (s *ConflictStore) List(includeResolved bool) ([]*Conflict, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir, err := s.conflictsDir()
	if err != nil {
		return nil, fmt.Errorf("resolving conflicts directory: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing conflicts directory: %w", err)
	}

	var conflicts []*Conflict
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue // skip unreadable files
		}

		var conflict Conflict
		if err := json.Unmarshal(data, &conflict); err != nil {
			continue // skip malformed files
		}

		if !includeResolved {
			if conflict.Status == ConflictResolved || conflict.Status == ConflictAutoResolved {
				continue
			}
		}

		conflicts = append(conflicts, &conflict)
	}

	return conflicts, nil
}

// Delete removes a conflict file from disk.
func (s *ConflictStore) Delete(conflictID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.conflictPath(conflictID)
	if err != nil {
		return fmt.Errorf("resolving conflict path: %w", err)
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("conflict %s not found", conflictID)
		}
		return fmt.Errorf("deleting conflict %s: %w", conflictID, err)
	}

	return nil
}
