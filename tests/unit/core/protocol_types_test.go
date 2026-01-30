package core

import (
	"testing"

	"oss-aps-cli/internal/core/protocol"

	"github.com/stretchr/testify/assert"
)

func TestValidateRunStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  protocol.RunStatus
		wantErr bool
	}{
		{
			name:    "valid pending",
			status:  protocol.RunStatusPending,
			wantErr: false,
		},
		{
			name:    "valid running",
			status:  protocol.RunStatusRunning,
			wantErr: false,
		},
		{
			name:    "valid completed",
			status:  protocol.RunStatusCompleted,
			wantErr: false,
		},
		{
			name:    "valid failed",
			status:  protocol.RunStatusFailed,
			wantErr: false,
		},
		{
			name:    "valid cancelled",
			status:  protocol.RunStatusCancelled,
			wantErr: false,
		},
		{
			name:    "invalid status",
			status:  protocol.RunStatus("invalid"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := protocol.ValidateRunStatus(tt.status)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAgentInfo(t *testing.T) {
	tests := []struct {
		name    string
		agent   *protocol.AgentInfo
		wantErr bool
	}{
		{
			name: "valid agent",
			agent: &protocol.AgentInfo{
				ID:   "test-agent",
				Name: "Test Agent",
			},
			wantErr: false,
		},
		{
			name: "missing id",
			agent: &protocol.AgentInfo{
				Name: "Test Agent",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			agent: &protocol.AgentInfo{
				ID: "test-agent",
			},
			wantErr: true,
		},
		{
			name: "empty id",
			agent: &protocol.AgentInfo{
				ID:   "",
				Name: "Test Agent",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			agent: &protocol.AgentInfo{
				ID:   "test-agent",
				Name: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := protocol.ValidateAgentInfo(tt.agent)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateActionSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  *protocol.ActionSchema
		wantErr bool
	}{
		{
			name: "valid schema",
			schema: &protocol.ActionSchema{
				Name:        "test-action",
				Description: "Test action description",
				Input:       map[string]interface{}{"type": "string"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			schema: &protocol.ActionSchema{
				Description: "Test action description",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			schema: &protocol.ActionSchema{
				Name: "test-action",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			schema: &protocol.ActionSchema{
				Name:        "",
				Description: "Test action description",
			},
			wantErr: true,
		},
		{
			name: "empty description",
			schema: &protocol.ActionSchema{
				Name:        "test-action",
				Description: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := protocol.ValidateActionSchema(tt.schema)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   protocol.RunInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: protocol.RunInput{
				ProfileID: "test-profile",
				ActionID:  "test-action",
			},
			wantErr: false,
		},
		{
			name: "missing profile_id",
			input: protocol.RunInput{
				ActionID: "test-action",
			},
			wantErr: true,
		},
		{
			name: "missing action_id",
			input: protocol.RunInput{
				ProfileID: "test-profile",
			},
			wantErr: true,
		},
		{
			name: "empty profile_id",
			input: protocol.RunInput{
				ProfileID: "",
				ActionID:  "test-action",
			},
			wantErr: true,
		},
		{
			name: "empty action_id",
			input: protocol.RunInput{
				ProfileID: "test-profile",
				ActionID:  "",
			},
			wantErr: true,
		},
		{
			name: "valid with payload",
			input: protocol.RunInput{
				ProfileID: "test-profile",
				ActionID:  "test-action",
				Payload:   []byte("test"),
			},
			wantErr: false,
		},
		{
			name: "valid with thread_id",
			input: protocol.RunInput{
				ProfileID: "test-profile",
				ActionID:  "test-action",
				ThreadID:  "thread-123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
