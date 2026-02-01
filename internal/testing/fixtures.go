package testing

import (
	"testing"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/stretchr/testify/require"
)

// SampleProfileYAML returns a sample profile in YAML format
// Example:
//   yaml := SampleProfileYAML("test-agent")
//   profile, err := ParseProfileYAML(yaml)
const SampleProfileYAML = `
id: %s
display_name: Test Agent
persona:
  tone: professional
  style: concise
  risk: low
capabilities:
  - execute
  - query
  - analyze
accounts:
  default:
    username: test-user
preferences:
  language: en
  timezone: UTC
  shell: /bin/sh
limits:
  max_concurrency: 10
  max_runtime_minutes: 60
isolation:
  level: process
  strict: false
  fallback: true
a2a:
  enabled: true
  protocol_binding: grpc
  listen_addr: "127.0.0.1:8081"
  public_endpoint: "http://127.0.0.1:8081"
acp:
  enabled: true
  transport: stdio
  listen_addr: "127.0.0.1:3000"
  port: 3000
`

// SampleAgentCardJSON returns a sample agent card in JSON format
func SampleAgentCardJSON(t *testing.T, agentID string) *a2a.AgentCard {
	t.Helper()

	card := &a2a.AgentCard{
		Name:        "Test Agent: " + agentID,
		Description: "A test agent for unit testing",
		Version:     "1.0.0",
		URL:         "http://127.0.0.1:8081",
		Skills: []a2a.AgentSkill{
			{
				ID:          "execute",
				Name:        "Execute Commands",
				Description: "Execute shell commands",
			},
			{
				ID:          "query",
				Name:        "Query Data",
				Description: "Query data from systems",
			},
		},
	}

	return card
}

// SampleConfigurationJSON returns sample A2A configuration in JSON format
func SampleConfigurationJSON(t *testing.T) map[string]interface{} {
	t.Helper()

	return map[string]interface{}{
		"protocol": "a2a",
		"version":  "1.0.0",
		"transport": map[string]interface{}{
			"type":    "grpc",
			"address": "127.0.0.1:8081",
		},
		"isolation": map[string]interface{}{
			"level": "process",
			"strict": false,
		},
		"authentication": map[string]interface{}{
			"type": "none",
		},
		"logging": map[string]interface{}{
			"level": "info",
		},
	}
}

// SamplePermissionRules returns sample permission rules for testing
func SamplePermissionRules(t *testing.T) map[string][]string {
	t.Helper()

	return map[string][]string{
		"execute": {
			"allow_all",
		},
		"query": {
			"profiles.read",
			"actions.read",
		},
		"admin": {
			"profiles.write",
			"actions.write",
			"profiles.delete",
		},
	}
}

// SampleTaskJSON returns a sample A2A task in JSON format
func SampleTaskJSON(t *testing.T) *a2a.Task {
	t.Helper()

	return &a2a.Task{
		ID: a2a.NewTaskID(),
		Status: a2a.TaskStatus{
			State: a2a.TaskStateSubmitted,
		},
	}
}

// SampleMessageJSON returns a sample A2A message in JSON format
func SampleMessageJSON(t *testing.T, text string) *a2a.Message {
	t.Helper()

	if text == "" {
		text = "Test message content"
	}

	return a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: text})
}

// SampleEventJSON returns a sample A2A event in JSON format
func SampleEventJSON(t *testing.T) a2a.Event {
	t.Helper()

	reqCtx := &a2asrv.RequestContext{
		TaskID: a2a.NewTaskID(),
	}

	return a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, nil)
}

// ValidateAgentCard validates that an agent card has the required fields
func ValidateAgentCard(t *testing.T, card *a2a.AgentCard) {
	t.Helper()

	require.NotNil(t, card)
	require.NotEmpty(t, card.Name)
	require.NotEmpty(t, card.URL)
	require.Greater(t, len(card.Skills), 0)
}

// ValidateTask validates that a task has the required fields
func ValidateTask(t *testing.T, task *a2a.Task) {
	t.Helper()

	require.NotNil(t, task)
	require.NotEmpty(t, task.ID)
	require.NotEmpty(t, task.Status.State)
}

// ValidateMessage validates that a message has the required fields
func ValidateMessage(t *testing.T, message *a2a.Message) {
	t.Helper()

	require.NotNil(t, message)
	require.NotEmpty(t, message.ID)
	require.NotEmpty(t, message.Role)
}

// CreateSampleAgentCards creates multiple sample agent cards
func CreateSampleAgentCards(t *testing.T, count int) []*a2a.AgentCard {
	t.Helper()

	cards := make([]*a2a.AgentCard, count)
	for i := 0; i < count; i++ {
		cards[i] = SampleAgentCardJSON(t, "agent-"+string(rune('0'+i)))
	}

	return cards
}

// CreateSampleTasks creates multiple sample tasks
func CreateSampleTasks(t *testing.T, count int) []*a2a.Task {
	t.Helper()

	tasks := make([]*a2a.Task, count)
	for i := 0; i < count; i++ {
		tasks[i] = SampleTaskJSON(t)
	}

	return tasks
}

// CreateSampleMessages creates multiple sample messages
func CreateSampleMessages(t *testing.T, count int) []*a2a.Message {
	t.Helper()

	messages := make([]*a2a.Message, count)
	for i := 0; i < count; i++ {
		messages[i] = SampleMessageJSON(t, "Message "+string(rune('0'+i)))
	}

	return messages
}

// FixtureManager manages test fixtures and ensures cleanup
type FixtureManager struct {
	t           *testing.T
	profiles    []*Profile
	cards       []*a2a.AgentCard
	tasks       []*a2a.Task
	messages    []*a2a.Message
	cleanup     []func()
}

// NewFixtureManager creates a new fixture manager
func NewFixtureManager(t *testing.T) *FixtureManager {
	t.Helper()

	fm := &FixtureManager{
		t:        t,
		profiles: make([]*Profile, 0),
		cards:    make([]*a2a.AgentCard, 0),
		tasks:    make([]*a2a.Task, 0),
		messages: make([]*a2a.Message, 0),
		cleanup:  make([]func(), 0),
	}

	t.Cleanup(func() {
		fm.Cleanup()
	})

	return fm
}

// Cleanup runs all registered cleanup functions
func (fm *FixtureManager) Cleanup() {
	for i := len(fm.cleanup) - 1; i >= 0; i-- {
		fm.cleanup[i]()
	}
}

// RegisterCleanup registers a cleanup function
func (fm *FixtureManager) RegisterCleanup(fn func()) {
	fm.cleanup = append(fm.cleanup, fn)
}

// Profile represents a test profile fixture
type Profile struct {
	Name      string
	IsolationLevel string
	Config    map[string]interface{}
}
