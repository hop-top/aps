package squad

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

// maxIDLen caps ids at the common filename limit so storage never fails
// halfway with a filesystem error.
const maxIDLen = 255

// windowsReserved are device names Windows refuses as a file's base name,
// with or without an extension.
var windowsReserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// validStorageID reports whether id can be used as a directory name on
// every supported platform, whatever the host OS: no separators or
// traversal, nothing Windows rejects, nothing that escapes the squads dir.
func validStorageID(id string) error {
	reject := func(why string) error {
		return fmt.Errorf("invalid squad ID for storage: %q (%s)", id, why)
	}
	switch {
	case id == "" || id == "." || id == "..":
		return reject("empty or relative")
	case len(id) > maxIDLen:
		return reject("too long")
	case strings.ContainsAny(id, `/\:*?"<>|`):
		return reject("contains a path or reserved character")
	case strings.HasSuffix(id, ".") || strings.HasSuffix(id, " "):
		return reject("trailing dot or space")
	}
	for _, r := range id {
		if r < 0x20 || r == 0x7f {
			return reject("contains a control character")
		}
	}
	base, _, _ := strings.Cut(id, ".")
	if windowsReserved[strings.ToUpper(base)] {
		return reject("reserved device name")
	}
	return nil
}

// dirOf returns the squad's directory, <dir>/<id>, after validating id.
func (m *Manager) dirOf(id string) (string, error) {
	if err := validStorageID(id); err != nil {
		return "", err
	}
	d := filepath.Join(m.dir, id)
	if filepath.Dir(d) != filepath.Clean(m.dir) {
		return "", fmt.Errorf("invalid squad ID for storage: %q", id)
	}
	return d, nil
}

// readManifest loads <dir>/<name>/manifest.yaml. Every error names the file.
func (m *Manager) readManifest(name string) (*Squad, error) {
	p := filepath.Join(m.dir, name, manifestName)
	data, err := os.ReadFile(p) // #nosec G304 -- under squads dir
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no manifest %s: %w", p, err)
		}
		return nil, fmt.Errorf("load squad manifest %s: %w", p, err)
	}
	var s Squad
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("load squad manifest %s: %w", p, err)
	}
	if s.ID != name {
		return nil, fmt.Errorf(
			"load squad manifest %s: id %q does not match directory %q", p, s.ID, name)
	}
	return &s, nil
}

// reload replaces the in-memory registry with the on-disk state. A squad
// whose manifest cannot be loaded is recorded in m.broken, keyed by its
// directory name, and left out of m.squads: one bad record never hides or
// blocks the others (profile listing behaves the same way). Only a squads
// directory that cannot be read at all aborts. Caller holds m.mu.
func (m *Manager) reload() error {
	m.squads = make(map[string]*Squad)
	m.broken = make(map[string]error)
	entries, err := os.ReadDir(m.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read squads dir %s: %w", m.dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := m.readManifest(e.Name())
		switch {
		case errors.Is(err, os.ErrNotExist):
			continue
		case err != nil:
			m.broken[e.Name()] = err
		default:
			m.squads[s.ID] = s
		}
	}
	return nil
}

// SkippedError reports squads left out of a result because their manifests
// could not be loaded. Each wrapped error names its file. Results returned
// alongside it are valid and complete for the squads that did load.
type SkippedError struct{ Errs []error }

func (e *SkippedError) Error() string { return errors.Join(e.Errs...).Error() }

// Unwrap exposes the individual load errors to errors.Is / errors.As.
func (e *SkippedError) Unwrap() []error { return e.Errs }

// brokenErr reports the load failures found by the last reload in
// directory order, or nil when every manifest loaded.
func (m *Manager) brokenErr() error {
	if len(m.broken) == 0 {
		return nil
	}
	names := make([]string, 0, len(m.broken))
	for n := range m.broken {
		names = append(names, n)
	}
	sort.Strings(names)
	errs := make([]error, 0, len(names))
	for _, n := range names {
		errs = append(errs, m.broken[n])
	}
	return &SkippedError{Errs: errs}
}

// find returns the loaded squad for id. A squad that exists on disk but
// failed to load yields that load error, never "not found", so callers
// cannot mistake a damaged record for an absent one and overwrite it.
func (m *Manager) find(id string) (*Squad, error) {
	if s, ok := m.squads[id]; ok {
		return s, nil
	}
	if err, ok := m.broken[id]; ok {
		return nil, err
	}
	return nil, fmt.Errorf("squad not found: %s", id)
}

// save writes the squad atomically: a temp file in the squad's directory
// is synced and renamed over manifest.yaml, so a crash leaves either the
// old or the new record and never a truncated one. Directory and file
// modes follow the profile store (0755 / 0644, before umask). No-op for
// in-memory managers.
//
// There is no cross-process lock. Two processes saving the same squad at
// once each write a complete manifest and the later rename wins, so a
// concurrent member add can be lost; the store itself is never corrupted.
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
	if err := os.MkdirAll(sd, 0o755); err != nil { //nolint:gosec // squad data is not secret; mode matches the profile store
		return fmt.Errorf("create squad dir: %w", err)
	}
	tmpPath, err := writeTemp(sd, data)
	if err != nil {
		return err
	}
	if err := os.Rename(tmpPath, filepath.Join(sd, manifestName)); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write squad: %w", err)
	}
	return nil
}

// writeTemp writes data to a new, uniquely named, synced file in dir and
// returns its path. The file is opened with 0644 so umask applies as for
// os.WriteFile in the sibling stores.
func writeTemp(dir string, data []byte) (string, error) {
	for i := 0; ; i++ {
		name := fmt.Sprintf(".manifest-%d-%d.tmp", os.Getpid(), time.Now().UnixNano()+int64(i))
		p := filepath.Join(dir, name)
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) //nolint:gosec // under squads dir; mode matches the profile store
		if errors.Is(err, os.ErrExist) && i < 16 {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("write squad: %w", err)
		}
		_, werr := f.Write(data)
		if werr == nil {
			werr = f.Sync()
		}
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			_ = os.Remove(p)
			return "", fmt.Errorf("write squad: %w", werr)
		}
		return p, nil
	}
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
