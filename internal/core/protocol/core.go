package protocol

import (
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
	var stdoutPipe *os.File
	var stdoutReader *os.File
	var err error

	if stream != nil {
		stdoutReader, stdoutPipe, err = os.Pipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		defer stdoutReader.Close()
		defer stdoutPipe.Close()

		cmd, err = a.createActionCommand(profile, input, stdoutPipe)
		if err != nil {
			return nil, err
		}

		go a.streamOutput(ctx, cmd, stdoutReader, stream, state)
	} else {
		cmd, err = a.createActionCommand(profile, input, nil)
		if err != nil {
			return nil, err
		}
	}

	state.Status = RunStatusRunning

	if err := cmd.Start(); err != nil {
		state.Status = RunStatusFailed
		state.Error = err.Error()
		return state, nil
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			state.Error = fmt.Sprintf("failed to kill process: %v", err)
		}
		state.Status = RunStatusCancelled
		state.Error = "cancelled by client"
		return state, nil
	case err := <-done:
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
	}

	now = time.Now()
	state.EndTime = &now

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

func (a *APSAdapter) streamOutput(ctx context.Context, cmd *exec.Cmd, stdoutReader *os.File, stream StreamWriter, state *RunState) {
	defer stream.Close()

	buf := make([]byte, 1024)
	for {
		n, err := stdoutReader.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			state.OutputSize += int64(n)

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

	return state, nil
}

func (a *APSAdapter) CancelRun(ctx context.Context, runID string) error {
	a.runMutex.Lock()
	state, exists := a.runRegistry[runID]
	if !exists {
		a.runMutex.Unlock()
		return fmt.Errorf("run not found: %s", runID)
	}
	a.runMutex.Unlock()

	if state.Status != RunStatusRunning && state.Status != RunStatusPending {
		return fmt.Errorf("run is not cancellable: %s", state.Status)
	}

	state.Status = RunStatusCancelled
	state.Error = "cancelled"

	return nil
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

func (a *APSAdapter) StorePut(namespace string, key string, value []byte) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}

	profileDir := filepath.Join(a.storeDir, namespace)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(profileDir, key+".json")
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
	filePath := filepath.Join(a.storeDir, namespace, key+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key not found: %s/%s", namespace, key)
		}
		return nil, err
	}

	var item StoreItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}

	return item.Value, nil
}

func (a *APSAdapter) StoreDelete(namespace string, key string) error {
	filePath := filepath.Join(a.storeDir, namespace, key+".json")
	return os.Remove(filePath)
}

func (a *APSAdapter) StoreSearch(namespace string, prefix string) (map[string][]byte, error) {
	profileDir := filepath.Join(a.storeDir, namespace)
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]byte), nil
		}
		return nil, err
	}

	result := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		key := name[:len(name)-5]
		if prefix != "" && key[:len(prefix)] != prefix {
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
