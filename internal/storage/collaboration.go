package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"hop.top/aps/internal/core"
	collab "hop.top/aps/internal/core/collaboration"

	"gopkg.in/yaml.v3"
)

// workspaceState is the serializable form of a Workspace for state.json.
// The Workspace struct contains unexported fields (mutex, WorkspaceContext) that
// cannot be serialized directly, so we project the exportable fields here.
type workspaceState struct {
	ID        string                `json:"id"`
	State     collab.WorkspaceState `json:"state"`
	Agents    []collab.AgentInfo    `json:"agents"`
	Policy    collab.PolicyConfig   `json:"policy"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// CollaborationStorage implements the collaboration.Storage interface using
// the local filesystem. Data is stored under a configurable root directory
// (typically ~/.aps/collaboration/).
type CollaborationStorage struct {
	mu   sync.RWMutex
	root string
}

// NewCollaborationStorage returns a CollaborationStorage rooted at the given
// directory. If root is empty, the default ~/.aps/collaboration/ is used.
func NewCollaborationStorage(root string) (*CollaborationStorage, error) {
	if root == "" {
		dataDir, err := core.GetDataDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get data directory: %w", err)
		}
		root = filepath.Join(dataDir, "collaboration")
	}

	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("failed to create collaboration directory: %w", err)
	}

	return &CollaborationStorage{root: root}, nil
}

// dirNameEscape lists characters reserved in Windows file/directory names
// (< > : " / \ | ? *) plus the percent sign used as the escape marker.
// Workspace IDs are mostly UUIDs/slugs, but the collaboration package also
// uses synthetic sentinel IDs containing a colon (collab.GlobalAuditWorkspace
// = "aps:global"); a literal ":" in a Windows path component is reserved
// for drive letters/NTFS alternate data streams, so filepath.Join(root,
// "aps:global") produces a directory name Windows rejects outright
// ("The directory name is invalid").
const dirNameEscape = `<>:"/\|?*%`

// escapeWorkspaceID returns a filesystem-safe directory name for the given
// workspace ID. Reserved characters are percent-encoded so the mapping is
// unambiguous and reversible via unescapeWorkspaceID; ordinary IDs (the
// common case) are left untouched. A final dot or space is encoded too:
// Windows drops trailing dots and spaces from a path component, so "x.."
// and "x" would otherwise name the same directory there. "." and ".." are
// left as they are so the containment check still refuses them.
func escapeWorkspaceID(id string) string {
	var b strings.Builder
	last := -1
	if id != "." && id != ".." {
		last = len(id) - 1
	}
	for i, r := range id {
		if r < 0x20 || strings.ContainsRune(dirNameEscape, r) || (i == last && (r == '.' || r == ' ')) {
			fmt.Fprintf(&b, "%%%02X", r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// unescapeWorkspaceID reverses escapeWorkspaceID. Invalid escapes are left
// as-is rather than erroring, since directory names are only ever produced
// by escapeWorkspaceID in the first place.
func unescapeWorkspaceID(name string) string {
	if !strings.ContainsRune(name, '%') {
		return name
	}
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		if name[i] == '%' && i+2 < len(name) {
			if v, err := strconv.ParseUint(name[i+1:i+3], 16, 8); err == nil {
				b.WriteByte(byte(v))
				i += 2
				continue
			}
		}
		b.WriteByte(name[i])
	}
	return b.String()
}

// legacyEligible reports whether the verbatim ID can be a pre-escaping
// directory name. An escaped directory name contains "%" only as part of a
// valid %XX sequence, so an ID that unescapes to itself can never be another
// ID's escaped form. One that does not (e.g. "a%3Ab") can, and must not be
// treated as legacy or two workspaces would share a directory.
func legacyEligible(id string) bool {
	return unescapeWorkspaceID(id) == id
}

// containedJoin joins name under root and refuses any result that is not a
// strict child of root (root itself, its parent, or anything above or beside
// it). Pure path arithmetic after filepath.Clean, no filesystem access.
func containedJoin(root, name string) (string, bool) {
	root = filepath.Clean(root)
	p := filepath.Clean(filepath.Join(root, name))
	if !strings.HasPrefix(p, root+string(filepath.Separator)) {
		return "", false
	}
	return p, true
}

// validWorkspaceID reports whether id maps to a directory strictly inside
// the storage root.
func (s *CollaborationStorage) validWorkspaceID(id string) bool {
	_, err := s.workspaceDir(id)
	return err == nil
}

// workspaceDir returns the canonical (escaped) directory for a workspace.
// Pure path construction, no filesystem access — always the write target,
// so saves converge every workspace to the escaped layout over time. IDs
// whose directory is not a strict child of the root ("", ".", "..") yield
// an InvalidWorkspaceIDError; every operation goes through here or
// resolveWorkspaceDir, so no filesystem access happens for them.
func (s *CollaborationStorage) workspaceDir(id string) (string, error) {
	dir, ok := containedJoin(s.root, escapeWorkspaceID(id))
	if !ok {
		return "", &collab.InvalidWorkspaceIDError{ID: id}
	}
	return dir, nil
}

// resolveWorkspaceDir returns the directory to read/delete a workspace
// from: the escaped directory if present, otherwise the pre-escaping
// legacy directory (unescaped id) if that exists instead. Needed because
// escaping was introduced after workspaces could already exist on disk —
// notably collab.GlobalAuditWorkspace ("aps:global"), created
// unconditionally by the audit subscriber, not opt-in. A workspace ID with
// no reserved characters resolves to the same path either way. An ID that
// is itself the escaped form of another ID (e.g. "a%3Ab" for "a:b") has no
// legacy candidate: its verbatim directory is that other workspace's
// escaped directory (see legacyEligible). Both candidates are
// containment-checked before any filesystem access: an escaped candidate
// outside the root is an error; a legacy one is simply not a candidate.
func (s *CollaborationStorage) resolveWorkspaceDir(id string) (string, error) {
	dir, err := s.workspaceDir(id)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	legacyDir, ok := containedJoin(s.root, id)
	if !ok || legacyDir == dir || !legacyEligible(id) {
		return dir, nil
	}
	if _, err := os.Stat(legacyDir); err == nil {
		return legacyDir, nil
	}
	return dir, nil
}

// SaveWorkspace persists a workspace to disk as manifest.yaml + state.json.
func (s *CollaborationStorage) SaveWorkspace(ws *collab.Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.workspaceDir(ws.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Write manifest.yaml (workspace config)
	manifestData, err := yaml.Marshal(ws.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace config: %w", err)
	}
	if err := atomicWriteFile(filepath.Join(dir, "manifest.yaml"), manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest.yaml: %w", err)
	}

	// Write state.json (workspace state, agents, policy, timestamps)
	state := workspaceState{
		ID:        ws.ID,
		State:     ws.State,
		Agents:    ws.Agents,
		Policy:    ws.Policy,
		CreatedAt: ws.CreatedAt,
		UpdatedAt: ws.UpdatedAt,
	}
	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace state: %w", err)
	}
	if err := atomicWriteFile(filepath.Join(dir, "state.json"), stateData, 0644); err != nil {
		return fmt.Errorf("failed to write state.json: %w", err)
	}

	return nil
}

// LoadWorkspace loads a workspace from disk by reading manifest.yaml and state.json.
func (s *CollaborationStorage) LoadWorkspace(id string) (*collab.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir, err := s.resolveWorkspaceDir(id)
	if err != nil {
		return nil, err
	}

	// Read manifest.yaml
	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &collab.WorkspaceNotFoundError{ID: id}
		}
		return nil, fmt.Errorf("failed to read manifest.yaml: %w", err)
	}

	var config collab.WorkspaceConfig
	if err := yaml.Unmarshal(manifestData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse manifest.yaml: %w", err)
	}

	// Read state.json
	stateData, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("state.json missing for workspace %q: %w", id, err)
		}
		return nil, fmt.Errorf("failed to read state.json: %w", err)
	}

	var state workspaceState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state.json: %w", err)
	}

	ws := &collab.Workspace{
		ID:        state.ID,
		Config:    config,
		State:     state.State,
		Agents:    state.Agents,
		Policy:    state.Policy,
		CreatedAt: state.CreatedAt,
		UpdatedAt: state.UpdatedAt,
	}

	return ws, nil
}

// ListWorkspaces returns all workspace IDs by listing subdirectories of the root.
func (s *CollaborationStorage) ListWorkspaces() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read collaboration directory: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Only include directories that contain a manifest.yaml
		id := unescapeWorkspaceID(entry.Name())
		if !s.validWorkspaceID(id) {
			continue
		}
		manifestPath := filepath.Join(s.root, entry.Name(), "manifest.yaml")
		if _, err := os.Stat(manifestPath); err == nil {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// DeleteWorkspace removes a workspace directory from disk.
func (s *CollaborationStorage) DeleteWorkspace(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.resolveWorkspaceDir(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return &collab.WorkspaceNotFoundError{ID: id}
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to delete workspace %q: %w", id, err)
	}
	return nil
}

// SaveTasks persists a task list for a workspace as tasks.json.
func (s *CollaborationStorage) SaveTasks(workspaceID string, tasks []collab.TaskInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveJSON(workspaceID, "tasks.json", tasks)
}

// LoadTasks loads the task list for a workspace from tasks.json.
func (s *CollaborationStorage) LoadTasks(workspaceID string) ([]collab.TaskInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []collab.TaskInfo
	if err := s.loadJSON(workspaceID, "tasks.json", &tasks); err != nil {
		return nil, err
	}
	if tasks == nil {
		return []collab.TaskInfo{}, nil
	}
	return tasks, nil
}

// SaveConflicts persists conflicts for a workspace as conflicts.json.
func (s *CollaborationStorage) SaveConflicts(workspaceID string, conflicts []collab.Conflict) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveJSON(workspaceID, "conflicts.json", conflicts)
}

// LoadConflicts loads conflicts for a workspace from conflicts.json.
func (s *CollaborationStorage) LoadConflicts(workspaceID string) ([]collab.Conflict, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var conflicts []collab.Conflict
	if err := s.loadJSON(workspaceID, "conflicts.json", &conflicts); err != nil {
		return nil, err
	}
	if conflicts == nil {
		return []collab.Conflict{}, nil
	}
	return conflicts, nil
}

// SaveContext persists context variables for a workspace as context.json.
func (s *CollaborationStorage) SaveContext(workspaceID string, variables []collab.ContextVariable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveJSON(workspaceID, "context.json", variables)
}

// LoadContext loads context variables for a workspace from context.json.
func (s *CollaborationStorage) LoadContext(workspaceID string) ([]collab.ContextVariable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var variables []collab.ContextVariable
	if err := s.loadJSON(workspaceID, "context.json", &variables); err != nil {
		return nil, err
	}
	if variables == nil {
		return []collab.ContextVariable{}, nil
	}
	return variables, nil
}

// SaveAuditEvents persists audit events for a workspace as audit.json.
func (s *CollaborationStorage) SaveAuditEvents(workspaceID string, events []collab.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveJSON(workspaceID, "audit.json", events)
}

// LoadAuditEvents loads audit events for a workspace from audit.json.
func (s *CollaborationStorage) LoadAuditEvents(workspaceID string) ([]collab.AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []collab.AuditEvent
	if err := s.loadJSON(workspaceID, "audit.json", &events); err != nil {
		return nil, err
	}
	if events == nil {
		return []collab.AuditEvent{}, nil
	}
	return events, nil
}

// SaveActiveWorkspace persists the active workspace ID as a plain text file.
func (s *CollaborationStorage) SaveActiveWorkspace(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.root, "active-workspace")
	if err := atomicWriteFile(path, []byte(id), 0644); err != nil {
		return fmt.Errorf("failed to write active-workspace: %w", err)
	}
	return nil
}

// LoadActiveWorkspace reads the active workspace ID from the plain text file.
func (s *CollaborationStorage) LoadActiveWorkspace() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.root, "active-workspace")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read active-workspace: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// saveJSON marshals v as indented JSON and writes it atomically to
// {workspaceDir}/{filename}. Caller must hold the write lock.
func (s *CollaborationStorage) saveJSON(workspaceID, filename string, v any) error {
	dir, err := s.workspaceDir(workspaceID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", filename, err)
	}

	if err := atomicWriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}
	return nil
}

// loadJSON reads {workspaceDir}/{filename} and unmarshals it into v.
// Returns nil (no error) if the file does not exist. Caller must hold the read lock.
func (s *CollaborationStorage) loadJSON(workspaceID, filename string, v any) error {
	dir, err := s.resolveWorkspaceDir(workspaceID)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	return nil
}

// atomicWriteFile writes data to a temporary file in the same directory, then
// renames it to the target path. This prevents partial writes from corrupting
// existing data.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
