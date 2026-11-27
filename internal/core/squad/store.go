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

const manifestName = "manifest.yaml"

// SquadsDir returns the directory squads persist under:
// <data dir>/squads (see core.GetDataDir).
func SquadsDir() (string, error) {
	dataDir, err := core.GetDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}
	return filepath.Join(dataDir, "squads"), nil
}

// NewFileManager creates a Manager persisting one <id>/manifest.yaml per
// squad under dir. The directory is created lazily on first write.
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

// dirOf returns the squad's directory, <dir>/<id>, after validating id.
func (m *Manager) dirOf(id string) (string, error) {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\`+"\x00") {
		return "", fmt.Errorf("invalid squad ID for storage: %q", id)
	}
	d := filepath.Join(m.dir, id)
	if filepath.Dir(d) != filepath.Clean(m.dir) {
		return "", fmt.Errorf("invalid squad ID for storage: %q", id)
	}
	return d, nil
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
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(m.dir, e.Name(), manifestName)
		data, err := os.ReadFile(p) // #nosec G304 -- under squads dir
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err == nil {
			var s Squad
			if err = yaml.Unmarshal(data, &s); err == nil && s.ID != e.Name() {
				err = fmt.Errorf("id %q does not match directory", s.ID)
			}
			if err == nil {
				m.squads[s.ID] = &s
				continue
			}
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("load squad dir %s: %w", e.Name(), err)
		}
	}
	return firstErr
}

// save writes the squad atomically. No-op for in-memory managers.
func (m *Manager) save(s *Squad) error {
	if m.dir == "" {
		return nil
	}
	sd, err := m.dirOf(s.ID)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal squad: %w", err)
	}
	if err := os.MkdirAll(sd, 0o750); err != nil {
		return fmt.Errorf("create squad dir: %w", err)
	}
	tmp, err := os.CreateTemp(sd, ".manifest-*.tmp")
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
	if err := os.Rename(tmp.Name(), filepath.Join(sd, manifestName)); err != nil {
		return fmt.Errorf("write squad: %w", err)
	}
	return nil
}

// remove deletes the squad's directory. No-op for in-memory managers. The
// target is exactly <dir>/<validated id>; anything else is refused.
func (m *Manager) remove(id string) error {
	if m.dir == "" {
		return nil
	}
	sd, err := m.dirOf(id)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(sd); err != nil {
		return fmt.Errorf("remove squad: %w", err)
	}
	return nil
}
