package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/session"

	"github.com/google/uuid"
)

type APSAdapter struct {
	runRegistry     map[string]*RunState
	runMutex        sync.RWMutex
	sessionRegistry *session.SessionRegistry
	storeDir        string
}

func NewAPSAdapter() (*APSAdapter, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	storeDir := filepath.Join(home, core.ApsHomeDir, "store")
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	return &APSAdapter{
		runRegistry:     make(map[string]*RunState),
		sessionRegistry: session.GetRegistry(),
		storeDir:        storeDir,
	}, nil
}

func (a *APSAdapter) ExecuteRun(ctx context.Context, input RunInput, stream StreamWriter) (*RunState, error) {
	if err := input.Validate(); err != nil {
		return nil, NewInvalidInputError("run_input", err.Error())
	}

	profile, loadErr := core.LoadProfile(input.ProfileID)
	if loadErr != nil {
		return nil, NewNotFoundError(input.ProfileID)
	}

	_, actionErr := core.GetAction(input.ProfileID, input.ActionID)
	if actionErr != nil {
		return nil, NewNotFoundError(input.ActionID)
	}

	runID := uuid.New().String()
	now := time.Now()

	state := &RunState{
		RunID:      runID,
		ProfileID:  input.ProfileID,
		ActionID:   input.ActionID,
		ThreadID:   input.ThreadID,
		Status:     RunStatusPending,
		StartTime:  now,
		OutputSize: 0,
	}

	a.runMutex.Lock()
	a.runRegistry[runID] = state
	a.runMutex.Unlock()

	var cmd *exec.Cmd
	var stdoutBuffer bytes.Buffer
	var stdoutPipe *os.File
	var stdoutReader *os.File
	var err error

	if stream != nil {
		stdoutReader, stdoutPipe, err = os.Pipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		defer stdoutReader.Close()
		defer func() {
			if stdoutPipe != nil {
				_ = stdoutPipe.Close()
			}
		}()

		cmd, err = a.createActionCommand(profile, input, stdoutPipe)
		if err != nil {
			return nil, err
		}

		streamDone := make(chan struct{})
		defer func() {
			if stdoutPipe != nil {
				_ = stdoutPipe.Close()
				stdoutPipe = nil
			}
			<-streamDone
		}()
		go func() {
			defer close(streamDone)
			a.streamOutput(stdoutReader, stream, runID)
		}()
	} else {
		cmd, err = a.createActionCommand(profile, input, nil)
		if err != nil {
			return nil, err
		}
		cmd.Stdout = &stdoutBuffer
	}

	a.updateRunState(runID, func(state *RunState) {
		state.Status = RunStatusRunning
	})

	if err := cmd.Start(); err != nil {
		a.updateRunState(runID, func(state *RunState) {
			state.Status = RunStatusFailed
			state.Error = err.Error()
		})
		return state, nil
	}
	if stdoutPipe != nil {
		_ = stdoutPipe.Close()
		stdoutPipe = nil
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			a.updateRunState(runID, func(state *RunState) {
				state.Error = fmt.Sprintf("failed to kill process: %v", err)
			})
		}
		a.updateRunState(runID, func(state *RunState) {
			state.Status = RunStatusCancelled
			state.Error = "cancelled by client"
		})
		return state, nil
	case err := <-done:
		a.updateRunState(runID, func(state *RunState) {
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode := exitErr.ExitCode()
					state.ExitCode = &exitCode
				}
				state.Status = RunStatusFailed
				state.Error = err.Error()
			} else {
				state.Status = RunStatusCompleted
				exitCode := 0
				state.ExitCode = &exitCode
			}
		})
	}

	now = time.Now()
	a.updateRunState(runID, func(state *RunState) {
		state.EndTime = &now
		if stream == nil {
			state.Output = stdoutBuffer.String()
			state.OutputSize = int64(stdoutBuffer.Len())
		}
	})

	return state, nil
}

func (a *APSAdapter) createActionCommand(profile *core.Profile, input RunInput, stdoutPipe *os.File) (*exec.Cmd, error) {
	action, err := core.GetAction(input.ProfileID, input.ActionID)
	if err != nil {
		return nil, err
	}

	var cmd *exec.Cmd
	switch action.Type {
	case "sh":
		cmd = exec.Command("sh", action.Path)
	case "py":
		cmd = exec.Command("python3", action.Path)
	case "js":
		cmd = exec.Command("node", action.Path)
	default:
		cmd = exec.Command(action.Path)
	}

	if err := core.InjectEnvironment(cmd, profile); err != nil {
		return nil, fmt.Errorf("failed to inject environment: %w", err)
	}

	if len(input.Payload) > 0 {
		pipe, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
		}
		go func() {
			defer pipe.Close()
			pipe.Write(input.Payload)
		}()
	}

	if stdoutPipe != nil {
		cmd.Stdout = stdoutPipe
	} else {
		cmd.Stdout = os.Stdout
	}

	cmd.Stderr = os.Stderr

	return cmd, nil
}

func (a *APSAdapter) streamOutput(stdoutReader *os.File, stream StreamWriter, runID string) {
	defer stream.Close()

	buf := make([]byte, 1024)
	for {
		n, err := stdoutReader.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			a.updateRunState(runID, func(state *RunState) {
				state.OutputSize += int64(n)
			})

			if err := stream.Write("output", data); err != nil {
				return
			}
		}
		if err != nil {
			break
		}
	}
}

func (a *APSAdapter) GetRun(runID string) (*RunState, error) {
	a.runMutex.RLock()
	defer a.runMutex.RUnlock()

	state, exists := a.runRegistry[runID]
	if !exists {
		return nil, fmt.Errorf("run not found: %s", runID)
	}

	return cloneRunState(state), nil
}

func (a *APSAdapter) CancelRun(ctx context.Context, runID string) error {
	a.runMutex.Lock()
	defer a.runMutex.Unlock()

	state, exists := a.runRegistry[runID]
	if !exists {
		return fmt.Errorf("run not found: %s", runID)
	}

	if state.Status != RunStatusRunning && state.Status != RunStatusPending {
		return fmt.Errorf("run is not cancellable: %s", state.Status)
	}

	state.Status = RunStatusCancelled
	state.Error = "cancelled"

	return nil
}

func (a *APSAdapter) updateRunState(runID string, update func(*RunState)) {
	a.runMutex.Lock()
	defer a.runMutex.Unlock()

	if state, exists := a.runRegistry[runID]; exists {
		update(state)
	}
}

func cloneRunState(state *RunState) *RunState {
	if state == nil {
		return nil
	}
	clone := *state
	if state.EndTime != nil {
		endTime := *state.EndTime
		clone.EndTime = &endTime
	}
	if state.ExitCode != nil {
		exitCode := *state.ExitCode
		clone.ExitCode = &exitCode
	}
	return &clone
}

func (a *APSAdapter) GetAgent(profileID string) (*AgentInfo, error) {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, NewNotFoundError(profileID)
	}

	return &AgentInfo{
		ID:           profile.ID,
		Name:         profile.DisplayName,
		Description:  profile.Persona.Tone + " " + profile.Persona.Style,
		Capabilities: profile.Capabilities,
	}, nil
}

func (a *APSAdapter) ListAgents() ([]AgentInfo, error) {
	profileIDs, err := core.ListProfiles()
	if err != nil {
		return nil, err
	}

	var agents []AgentInfo
	for _, id := range profileIDs {
		agent, err := a.GetAgent(id)
		if err != nil {
			continue
		}
		agents = append(agents, *agent)
	}

	return agents, nil
}

func (a *APSAdapter) GetAgentSchemas(profileID string) ([]ActionSchema, error) {
	actions, err := core.LoadActions(profileID)
	if err != nil {
		return nil, NewNotFoundError(profileID)
	}

	var schemas []ActionSchema
	for _, action := range actions {
		schema := ActionSchema{
			Name:        action.ID,
			Description: action.Title,
			Input: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type":        "string",
						"description": "JSON input for the action",
					},
				},
			},
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

func (a *APSAdapter) CreateSession(profileID string, metadata map[string]string) (*SessionState, error) {
	sessionID := uuid.New().String()
	now := time.Now()

	session := &session.SessionInfo{
		ID:          sessionID,
		ProfileID:   profileID,
		Status:      session.SessionActive,
		CreatedAt:   now,
		LastSeenAt:  now,
		Environment: metadata,
	}

	if err := a.sessionRegistry.Register(session); err != nil {
		return nil, fmt.Errorf("failed to register session: %w", err)
	}

	return &SessionState{
		SessionID:  sessionID,
		ProfileID:  profileID,
		CreatedAt:  now,
		LastSeenAt: now,
		Metadata:   metadata,
	}, nil
}

func (a *APSAdapter) GetSession(sessionID string) (*SessionState, error) {
	session, err := a.sessionRegistry.Get(sessionID)
	if err != nil {
		return nil, NewNotFoundError(sessionID)
	}

	return &SessionState{
		SessionID:  session.ID,
		ProfileID:  session.ProfileID,
		CreatedAt:  session.CreatedAt,
		LastSeenAt: session.LastSeenAt,
		Metadata:   session.Environment,
	}, nil
}

func (a *APSAdapter) UpdateSession(sessionID string, metadata map[string]string) error {
	if err := a.sessionRegistry.UpdateSessionMetadata(sessionID, metadata); err != nil {
		return fmt.Errorf("update session metadata: %w", err)
	}
	return nil
}

// HeartbeatSession marks the session as recently active by updating
// its LastSeenAt timestamp via the registry (persists to disk).
// Returns an error if the session does not exist or if updating the
// registry, including persisting the change to disk, fails.
func (a *APSAdapter) HeartbeatSession(sessionID string) error {
	if err := a.sessionRegistry.UpdateHeartbeat(sessionID); err != nil {
		return fmt.Errorf("heartbeat session: %w", err)
	}
	return nil
}

func (a *APSAdapter) DeleteSession(sessionID string) error {
	if _, err := a.sessionRegistry.Get(sessionID); err != nil {
		return fmt.Errorf("session %s: %w", sessionID, err)
	}
	return a.sessionRegistry.Unregister(sessionID)
}

func (a *APSAdapter) ListSessions(profileID string) ([]SessionState, error) {
	sessions := a.sessionRegistry.ListByProfile(profileID)

	var states []SessionState
	for _, sess := range sessions {
		state := SessionState{
			SessionID:  sess.ID,
			ProfileID:  sess.ProfileID,
			CreatedAt:  sess.CreatedAt,
			LastSeenAt: sess.LastSeenAt,
			Metadata:   sess.Environment,
		}
		states = append(states, state)
	}

	return states, nil
}

// storeFileNameEscape lists characters that are reserved in Windows
// filenames (< > : " / \ | ? *) plus the percent sign used as the escape
// marker itself. StorePut keys are arbitrary caller-chosen strings (e.g.
// "user:1") and are used directly as a filename component; on Windows a
// literal ":" is interpreted as a drive-letter/NTFS-alternate-data-stream
// separator, so an unescaped key can silently write into an ADS instead of
// a real file (invisible to os.ReadDir) or fail outright. Escaping keeps
// the on-disk name filesystem-safe on every platform without changing the
// logical key, which is preserved verbatim in the JSON payload.
const storeFileNameEscape = `<>:"/\|?*%`

// storeFileName returns a filesystem-safe file name (without extension)
// for the given store key. Reserved characters are percent-encoded so the
// mapping is unambiguous; ordinary keys are left untouched.
func storeFileName(key string) string {
	var b strings.Builder
	for _, r := range key {
		if r < 0x20 || strings.ContainsRune(storeFileNameEscape, r) {
			fmt.Fprintf(&b, "%%%02X", r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// containedJoin joins elems under root and refuses any result that resolves
// outside root. namespace and key reach the store untrusted (the
// agentprotocol HTTP adapter only checks them for non-emptiness), and a
// namespace may legitimately contain separators (nested namespaces), so
// containment after filepath.Clean is the invariant, not "single path
// component". The error deliberately omits root so it cannot leak the
// on-disk layout through the HTTP error response.
func containedJoin(root string, elems ...string) (string, error) {
	root = filepath.Clean(root)
	p := filepath.Clean(filepath.Join(append([]string{root}, elems...)...))
	if !strings.HasPrefix(p, root+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid store path component: %q", filepath.Join(elems...))
	}
	return p, nil
}

func (a *APSAdapter) StorePut(namespace string, key string, value []byte) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}

	profileDir, err := containedJoin(a.storeDir, namespace)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(profileDir, 0o750); err != nil {
		return err
	}

	filePath, err := containedJoin(profileDir, storeFileName(key)+".json")
	if err != nil {
		return err
	}
	data, err := json.Marshal(StoreItem{
		Namespace: namespace,
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (a *APSAdapter) StoreGet(namespace string, key string) ([]byte, error) {
	profileDir, err := containedJoin(a.storeDir, namespace)
	if err != nil {
		return nil, err
	}
	filePath, err := containedJoin(profileDir, storeFileName(key)+".json")
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- filePath is containment-checked under profileDir by
	// containedJoin, and storeFileName leaves no raw separators anyway.
	data, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read store item: %w", err)
		}
		// Escaped file missing: fall back to the legacy unescaped name for
		// keys written by a pre-escaping build (only reachable for keys
		// containing a reserved character; storeFileName is a no-op
		// otherwise, so the two paths already coincide). The raw key goes
		// through the same containment check: a key that would resolve
		// outside its own namespace directory was never a legitimate
		// legacy record, so it is reported as not found rather than read.
		// On a hit, migrate by writing the escaped copy (via StorePut, so
		// both write paths agree on file mode/marshaling) and removing the
		// legacy file, so the key converges to the new layout after first
		// read.
		legacyPath, legacyErr := containedJoin(profileDir, key+".json")
		if legacyErr != nil || legacyPath == filePath {
			return nil, fmt.Errorf("key not found: %s/%s", namespace, key)
		}
		// #nosec G304 -- legacyPath is containment-checked under profileDir
		// by containedJoin.
		legacyData, readErr := os.ReadFile(legacyPath)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				return nil, fmt.Errorf("key not found: %s/%s", namespace, key)
			}
			return nil, fmt.Errorf("failed to read legacy store item: %w", readErr)
		}
		var legacyItem StoreItem
		if err := json.Unmarshal(legacyData, &legacyItem); err != nil {
			return nil, fmt.Errorf("failed to parse legacy store item: %w", err)
		}
		if putErr := a.StorePut(namespace, key, legacyItem.Value); putErr == nil {
			_ = os.Remove(legacyPath)
		}
		return legacyItem.Value, nil
	}

	var item StoreItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}

	return item.Value, nil
}

func (a *APSAdapter) StoreDelete(namespace string, key string) error {
	profileDir, err := containedJoin(a.storeDir, namespace)
	if err != nil {
		return err
	}
	filePath, err := containedJoin(profileDir, storeFileName(key)+".json")
	if err != nil {
		return err
	}
	err = os.Remove(filePath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete store item: %w", err)
	}
	// Escaped file missing: the key may still exist under its pre-escaping
	// legacy name (see StoreGet). Only worth trying when the names differ,
	// and never outside this namespace directory.
	legacyPath, legacyErr := containedJoin(profileDir, key+".json")
	if legacyErr != nil || legacyPath == filePath {
		return fmt.Errorf("failed to delete store item: %w", err)
	}
	if legacyErr := os.Remove(legacyPath); legacyErr == nil {
		return nil
	}
	return fmt.Errorf("failed to delete store item: %w", err)
}

func (a *APSAdapter) StoreSearch(namespace string, prefix string) (map[string][]byte, error) {
	profileDir, err := containedJoin(a.storeDir, namespace)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]byte), nil
		}
		return nil, err
	}

	// storeFileName is a deterministic escape of the logical key, so an
	// escaped-prefix match on the filename alone cheaply rules out most
	// non-matches without reading/unmarshaling every file. It can't be the
	// only filter: a legacy (pre-escaping) file's name isn't run through
	// storeFileName, so it wouldn't share the escaped prefix even when its
	// payload key does. The payload-based check below stays as the
	// authoritative filter; this is a prefilter, not a replacement.
	escapedPrefix := storeFileName(prefix)

	result := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, escapedPrefix) && !strings.HasPrefix(name, prefix) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(profileDir, name))
		if err != nil {
			continue
		}

		var item StoreItem
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}

		// Filter on the logical key from the payload, not the escaped
		// on-disk file name, so prefixes containing reserved characters
		// (e.g. "user:") still match correctly.
		if prefix != "" && !strings.HasPrefix(item.Key, prefix) {
			continue
		}

		result[item.Key] = item.Value
	}

	return result, nil
}

func (a *APSAdapter) StoreListNamespaces() ([]string, error) {
	entries, err := os.ReadDir(a.storeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var namespaces []string
	for _, entry := range entries {
		if entry.IsDir() {
			namespaces = append(namespaces, entry.Name())
		}
	}

	return namespaces, nil
}

// mapError converts generic errors to typed errors
// Deprecated: Use custom error types directly instead
func mapError(err error, defaultCode int) error {
	return err
}
