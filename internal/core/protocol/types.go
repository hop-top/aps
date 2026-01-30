package protocol

import (
	"errors"
	"fmt"
	"time"
)

type StreamEvent struct {
	Event string
	Data  []byte
}

type RunInput struct {
	ProfileID string
	ActionID  string
	Payload   []byte
	ThreadID  string
}

func (r *RunInput) Validate() error {
	if r.ProfileID == "" {
		return errors.New("profile_id is required")
	}
	if r.ActionID == "" {
		return errors.New("action_id is required")
	}
	return nil
}

type RunState struct {
	RunID      string      `json:"run_id"`
	ProfileID  string      `json:"profile_id"`
	ActionID   string      `json:"action_id"`
	ThreadID   string      `json:"thread_id,omitempty"`
	Status     RunStatus   `json:"status"`
	StartTime  time.Time   `json:"start_time"`
	EndTime    *time.Time  `json:"end_time,omitempty"`
	ExitCode   *int        `json:"exit_code,omitempty"`
	OutputSize int64       `json:"output_size"`
	Error      string      `json:"error,omitempty"`
	Metadata   interface{} `json:"metadata,omitempty"`
}

type SessionState struct {
	SessionID  string            `json:"session_id"`
	ProfileID  string            `json:"profile_id"`
	CreatedAt  time.Time         `json:"created_at"`
	LastSeenAt time.Time         `json:"last_seen_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type AgentInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

type ActionSchema struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Input       interface{} `json:"input"`
}

type StoreItem struct {
	Namespace string    `json:"namespace"`
	Key       string    `json:"key"`
	Value     []byte    `json:"value,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ValidateRunStatus(status RunStatus) error {
	switch status {
	case RunStatusPending, RunStatusRunning, RunStatusCompleted, RunStatusFailed, RunStatusCancelled:
		return nil
	default:
		return fmt.Errorf("invalid run status: %s", status)
	}
}

func ValidateAgentInfo(info *AgentInfo) error {
	if info.ID == "" {
		return errors.New("agent id is required")
	}
	if info.Name == "" {
		return errors.New("agent name is required")
	}
	return nil
}

func ValidateActionSchema(schema *ActionSchema) error {
	if schema.Name == "" {
		return errors.New("action schema name is required")
	}
	if schema.Description == "" {
		return errors.New("action schema description is required")
	}
	return nil
}
