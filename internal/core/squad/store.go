package squad

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hop.top/aps/internal/core"

	"gopkg.in/yaml.v3"
)

const squadFileExt = ".yaml"

// SquadsDir returns the directory squads persist under:
// <data dir>/squads (see core.GetDataDir).
func SquadsDir() (string, error) {
	dataDir, err := core.GetDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}
	return filepath.Join(dataDir, "squads"), nil
}

// NewFileManager creates a Manager persisting one <id>.yaml per squad under
// dir. The directory is created lazily on first write.
func NewFileManager(dir string) *Manager {
	m := NewManager()
	m.dir = dir
	return m
}

// NewDefaultManager creates a file-backed Manager rooted at SquadsDir. The
// directory is resolved on every operation, so it follows APS_DATA_PATH /
// XDG_DATA_HOME as set at call time.
func NewDefaultManager() *Manager {
	m := NewManager()
	m.dirFn = SquadsDir
	return m
}

func (m *Manager) path(id string) (string, error) {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\`+"\x00") {
		return "", fmt.Errorf("invalid squad ID for storage: %q", id)
	}
	return filepath.Join(m.dir, id+squadFileExt), nil
}

// reload replaces the in-memory registry with the on-disk state. The first
// unreadable file aborts the operation with an error rather than silently
// hiding a squad. Caller holds m.mu.
func (m *Manager) reload() error {
	m.squads = make(map[string]*Squad)
	entries, err := os.ReadDir(m.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read squads dir: %w", err)
	}
	var firstErr error
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != squadFileExt {
			continue
		}
		p := filepath.Join(m.dir, e.Name())
		data, err := os.ReadFile(p) // #nosec G304 -- under squads dir
		if err == nil {
			var s Squad
			if err = yaml.Unmarshal(data, &s); err == nil && s.ID == "" {
				err = errors.New("missing squad id")
			}
			if err == nil {
				m.squads[s.ID] = &s
				continue
			}
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("load squad %s: %w", e.Name(), err)
		}
	}
	return firstErr
}

// save writes the squad atomically. No-op for in-memory managers.
func (m *Manager) save(s *Squad) error {
	if m.dir == "" {
		return nil
	}
	p, err := m.path(s.ID)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal squad: %w", err)
	}
	if err := os.MkdirAll(m.dir, 0o750); err != nil {
		return fmt.Errorf("create squads dir: %w", err)
	}
	tmp, err := os.CreateTemp(m.dir, ".squad-*.tmp")
	if err != nil {
		return fmt.Errorf("write squad: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write squad: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write squad: %w", err)
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return fmt.Errorf("write squad: %w", err)
	}
	return nil
}

// remove deletes the squad's file. No-op for in-memory managers.
func (m *Manager) remove(id string) error {
	if m.dir == "" {
		return nil
	}
	p, err := m.path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove squad: %w", err)
	}
	return nil
}
