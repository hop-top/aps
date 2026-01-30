package protocol

import "context"

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

type StreamMode string

const (
	StreamModeNone   StreamMode = "none"
	StreamModeStdout StreamMode = "stdout"
	StreamModeFull   StreamMode = "full"
)

type StreamWriter interface {
	Write(event string, data []byte) error
	Close() error
}

type APSCore interface {
	ExecuteRun(ctx context.Context, input RunInput, stream StreamWriter) (*RunState, error)
	GetRun(runID string) (*RunState, error)
	CancelRun(ctx context.Context, runID string) error

	GetAgent(profileID string) (*AgentInfo, error)
	ListAgents() ([]AgentInfo, error)
	GetAgentSchemas(profileID string) ([]ActionSchema, error)

	CreateSession(profileID string, metadata map[string]string) (*SessionState, error)
	GetSession(sessionID string) (*SessionState, error)
	UpdateSession(sessionID string, metadata map[string]string) error
	DeleteSession(sessionID string) error
	ListSessions(profileID string) ([]SessionState, error)

	StorePut(namespace string, key string, value []byte) error
	StoreGet(namespace string, key string) ([]byte, error)
	StoreDelete(namespace string, key string) error
	StoreSearch(namespace string, prefix string) (map[string][]byte, error)
	StoreListNamespaces() ([]string, error)
}
