package testing

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	eventqueue "github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"oss-aps-cli/internal/a2a/transport"
	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/session"
)

// MockTaskStore implements a2asrv.TaskStore interface for testing
type MockTaskStore struct {
	mu            sync.RWMutex
	tasks         map[a2a.TaskID]*a2a.Task
	versions      map[a2a.TaskID]a2a.TaskVersion
	events        map[a2a.TaskID][]a2a.Event
	agentCards    map[string]*a2a.AgentCard
	messages      map[a2a.TaskID][]*a2a.Message
	savedCalls    int
	getCalls      int
	listCalls     int
	errorOnSave   bool
	errorOnGet    bool
	errorOnList   bool
}

// NewMockTaskStore creates a new mock task store
func NewMockTaskStore() *MockTaskStore {
	return &MockTaskStore{
		tasks:      make(map[a2a.TaskID]*a2a.Task),
		versions:   make(map[a2a.TaskID]a2a.TaskVersion),
		events:     make(map[a2a.TaskID][]a2a.Event),
		agentCards: make(map[string]*a2a.AgentCard),
		messages:   make(map[a2a.TaskID][]*a2a.Message),
	}
}

// Save implements a2asrv.TaskStore interface
func (m *MockTaskStore) Save(ctx context.Context, task *a2a.Task, event a2a.Event, prev a2a.TaskVersion) (a2a.TaskVersion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.savedCalls++

	if m.errorOnSave {
		return 0, fmt.Errorf("mock save error")
	}

	m.tasks[task.ID] = task
	m.events[task.ID] = append(m.events[task.ID], event)

	version := prev + 1
	if version == 0 {
		version = 1
	}
	m.versions[task.ID] = version

	return version, nil
}

// Get implements a2asrv.TaskStore interface
func (m *MockTaskStore) Get(ctx context.Context, taskID a2a.TaskID) (*a2a.Task, a2a.TaskVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.getCalls++

	if m.errorOnGet {
		return nil, 0, fmt.Errorf("mock get error")
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, 0, a2a.ErrTaskNotFound
	}

	version := m.versions[taskID]
	return task, version, nil
}

// List implements a2asrv.TaskStore interface
func (m *MockTaskStore) List(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.listCalls++

	if m.errorOnList {
		return nil, fmt.Errorf("mock list error")
	}

	tasks := make([]*a2a.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}

	return &a2a.ListTasksResponse{
		Tasks:         tasks,
		NextPageToken: "",
	}, nil
}

// SaveAgentCard saves an agent card
func (m *MockTaskStore) SaveAgentCard(agentID string, card *a2a.AgentCard) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agentCards[agentID] = card
	return nil
}

// GetAgentCard retrieves an agent card
func (m *MockTaskStore) GetAgentCard(agentID string) (*a2a.AgentCard, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	card, exists := m.agentCards[agentID]
	if !exists {
		return nil, fmt.Errorf("agent card not found")
	}

	return card, nil
}

// CreateMessageFile saves a message
func (m *MockTaskStore) CreateMessageFile(taskID a2a.TaskID, message *a2a.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages[taskID] = append(m.messages[taskID], message)
	return nil
}

// GetTaskEvents returns all events for a task
func (m *MockTaskStore) GetTaskEvents(taskID a2a.TaskID) []a2a.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.events[taskID]
}

// GetTaskMessages returns all messages for a task
func (m *MockTaskStore) GetTaskMessages(taskID a2a.TaskID) []*a2a.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.messages[taskID]
}

// GetSavedCalls returns the number of Save calls
func (m *MockTaskStore) GetSavedCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.savedCalls
}

// GetGetCalls returns the number of Get calls
func (m *MockTaskStore) GetGetCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getCalls
}

// GetListCalls returns the number of List calls
func (m *MockTaskStore) GetListCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.listCalls
}

// SetErrorOnSave sets whether Save should return an error
func (m *MockTaskStore) SetErrorOnSave(shouldError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorOnSave = shouldError
}

// SetErrorOnGet sets whether Get should return an error
func (m *MockTaskStore) SetErrorOnGet(shouldError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorOnGet = shouldError
}

// SetErrorOnList sets whether List should return an error
func (m *MockTaskStore) SetErrorOnList(shouldError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorOnList = shouldError
}

// MockAgentExecutor implements a2asrv.AgentExecutor interface for testing
type MockAgentExecutor struct {
	mu             sync.RWMutex
	profile        *core.Profile
	executeCalls   int
	cancelCalls    int
	executedTasks  []a2a.TaskID
	canceledTasks  []a2a.TaskID
	executeErr     error
	cancelErr      error
	shouldFail     bool
}

// NewMockAgentExecutor creates a new mock agent executor
func NewMockAgentExecutor(profile *core.Profile) *MockAgentExecutor {
	return &MockAgentExecutor{
		profile:       profile,
		executedTasks: make([]a2a.TaskID, 0),
		canceledTasks: make([]a2a.TaskID, 0),
	}
}

// Execute implements a2asrv.AgentExecutor interface
func (m *MockAgentExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executeCalls++
	m.executedTasks = append(m.executedTasks, reqCtx.TaskID)

	if m.executeErr != nil {
		return m.executeErr
	}

	if m.shouldFail {
		event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateFailed, nil)
		event.Final = true
		return queue.Write(ctx, event)
	}

	// Emit successful execution events
	if reqCtx.StoredTask == nil {
		event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateSubmitted, nil)
		if err := queue.Write(ctx, event); err != nil {
			return err
		}
	}

	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, nil)
	if err := queue.Write(ctx, event); err != nil {
		return err
	}

	event = a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCompleted, nil)
	event.Final = true
	return queue.Write(ctx, event)
}

// Cancel implements a2asrv.AgentExecutor interface
func (m *MockAgentExecutor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cancelCalls++
	m.canceledTasks = append(m.canceledTasks, reqCtx.TaskID)

	if m.cancelErr != nil {
		return m.cancelErr
	}

	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCanceled, nil)
	event.Final = true
	return queue.Write(ctx, event)
}

// GetProfile returns the associated profile
func (m *MockAgentExecutor) GetProfile() *core.Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.profile
}

// GetExecuteCalls returns the number of Execute calls
func (m *MockAgentExecutor) GetExecuteCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.executeCalls
}

// GetCancelCalls returns the number of Cancel calls
func (m *MockAgentExecutor) GetCancelCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cancelCalls
}

// GetExecutedTasks returns all executed task IDs
func (m *MockAgentExecutor) GetExecutedTasks() []a2a.TaskID {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]a2a.TaskID, len(m.executedTasks))
	copy(tasks, m.executedTasks)
	return tasks
}

// GetCanceledTasks returns all canceled task IDs
func (m *MockAgentExecutor) GetCanceledTasks() []a2a.TaskID {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]a2a.TaskID, len(m.canceledTasks))
	copy(tasks, m.canceledTasks)
	return tasks
}

// SetExecuteError sets the error to return from Execute
func (m *MockAgentExecutor) SetExecuteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executeErr = err
}

// SetCancelError sets the error to return from Cancel
func (m *MockAgentExecutor) SetCancelError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cancelErr = err
}

// SetShouldFail sets whether execution should fail
func (m *MockAgentExecutor) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shouldFail = shouldFail
}

// MockTransport implements transport.Transport interface for testing
type MockTransport struct {
	mu           sync.RWMutex
	transportType transport.TransportType
	sentMessages   []*a2a.Message
	receivedMsgs  []*a2a.Message
	recvIndex     int
	healthy       bool
	closeErr      error
	sendErr       error
	recvErr       error
	closed        bool
}

// NewMockTransport creates a new mock transport
func NewMockTransport(tType transport.TransportType) *MockTransport {
	return &MockTransport{
		transportType: tType,
		sentMessages:  make([]*a2a.Message, 0),
		receivedMsgs:  make([]*a2a.Message, 0),
		healthy:       true,
	}
}

// Type implements transport.Transport interface
func (m *MockTransport) Type() transport.TransportType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.transportType
}

// Send implements transport.Transport interface
func (m *MockTransport) Send(ctx context.Context, message *a2a.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendErr != nil {
		return m.sendErr
	}

	if m.closed {
		return fmt.Errorf("transport closed")
	}

	m.sentMessages = append(m.sentMessages, message)
	return nil
}

// Receive implements transport.Transport interface
func (m *MockTransport) Receive(ctx context.Context) (*a2a.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.recvErr != nil {
		return nil, m.recvErr
	}

	if m.closed {
		return nil, fmt.Errorf("transport closed")
	}

	if m.recvIndex >= len(m.receivedMsgs) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil, fmt.Errorf("no messages available")
		}
	}

	msg := m.receivedMsgs[m.recvIndex]
	m.recvIndex++
	return msg, nil
}

// Close implements transport.Transport interface
func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return m.closeErr
}

// IsHealthy implements transport.Transport interface
func (m *MockTransport) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.healthy
}

// GetSentMessages returns all sent messages
func (m *MockTransport) GetSentMessages() []*a2a.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msgs := make([]*a2a.Message, len(m.sentMessages))
	copy(msgs, m.sentMessages)
	return msgs
}

// SetReceivedMessages sets messages to be received
func (m *MockTransport) SetReceivedMessages(messages []*a2a.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.receivedMsgs = messages
	m.recvIndex = 0
}

// SetSendError sets the error to return from Send
func (m *MockTransport) SetSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sendErr = err
}

// SetReceiveError sets the error to return from Receive
func (m *MockTransport) SetReceiveError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recvErr = err
}

// SetHealthy sets whether the transport is healthy
func (m *MockTransport) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.healthy = healthy
}

// IsClosed returns whether the transport is closed
func (m *MockTransport) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.closed
}

// MockCore implements a core interface for testing
type MockCore struct {
	mu           sync.RWMutex
	profiles     map[string]*core.Profile
	sessions     map[string]*session.SessionInfo
	store        map[string]map[string][]byte
	runs         map[string]*RunState
	testLogger   *testing.T
}

// GetProfile returns a profile by ID
func (m *MockCore) GetProfile(profileID string) (*core.Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	profile, exists := m.profiles[profileID]
	if !exists {
		return nil, fmt.Errorf("profile not found: %s", profileID)
	}

	return profile, nil
}

// SaveProfile saves a profile
func (m *MockCore) SaveProfile(profile *core.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if profile.ID == "" {
		return fmt.Errorf("profile ID cannot be empty")
	}

	m.profiles[profile.ID] = profile
	return nil
}

// CreateSession creates a session
func (m *MockCore) CreateSession(sessionInfo *session.SessionInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionInfo.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	m.sessions[sessionInfo.ID] = sessionInfo
	return nil
}

// GetSession returns a session by ID
func (m *MockCore) GetSession(sessionID string) (*session.SessionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// PutStore stores a value in the store
func (m *MockCore) PutStore(namespace, key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.store[namespace]; !exists {
		m.store[namespace] = make(map[string][]byte)
	}

	m.store[namespace][key] = value
	return nil
}

// GetStore retrieves a value from the store
func (m *MockCore) GetStore(namespace, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ns, exists := m.store[namespace]
	if !exists {
		return nil, fmt.Errorf("namespace not found: %s", namespace)
	}

	val, exists := ns[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s/%s", namespace, key)
	}

	return val, nil
}

// DeleteStore deletes a value from the store
func (m *MockCore) DeleteStore(namespace, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ns, exists := m.store[namespace]
	if !exists {
		return nil
	}

	delete(ns, key)
	return nil
}

// RunState represents the state of a run
type RunState struct {
	RunID      string
	ProfileID  string
	ActionID   string
	ThreadID   string
	Status     string
	StartTime  time.Time
	EndTime    *time.Time
	OutputSize int64
	ExitCode   *int
	Error      string
}
