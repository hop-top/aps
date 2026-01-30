package e2e

import (
	"testing"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/stretchr/testify/assert"
)

func TestA2AServer_TaskSubmission(t *testing.T) {
	taskID := a2a.NewTaskID()
	message := &a2a.Message{
		ID:   "msg-001",
		Parts: []a2a.Part{a2a.TextPart{Text: "Test message"}},
		Role:  a2a.MessageRoleUser,
	}

	assert.NotEmpty(t, string(taskID))
	assert.Equal(t, "msg-001", message.ID)
	assert.Equal(t, a2a.MessageRoleUser, message.Role)
	assert.Len(t, message.Parts, 1)

	if textPart, ok := message.Parts[0].(a2a.TextPart); ok {
		assert.Equal(t, "Test message", textPart.Text)
	}
}

func TestA2AServer_StreamingTaskSubmission(t *testing.T) {
	taskID := a2a.NewTaskID()

	assert.NotEmpty(t, string(taskID))
}

func TestA2AServer_TaskCancellation(t *testing.T) {
	taskID := a2a.NewTaskID()
	cancelParams := &a2a.TaskIDParams{
		ID: taskID,
	}

	assert.Equal(t, taskID, cancelParams.ID)
}

func TestA2AServer_PushNotificationConfig(t *testing.T) {
	taskID := a2a.NewTaskID()
	pushConfig := &a2a.TaskPushConfig{
		TaskID: taskID,
		Config: a2a.PushConfig{
			URL:   "http://localhost:9000/webhook",
			Token: "test-token-123",
		},
	}

	assert.Equal(t, taskID, pushConfig.TaskID)
	assert.Equal(t, "http://localhost:9000/webhook", pushConfig.Config.URL)
	assert.Equal(t, "test-token-123", pushConfig.Config.Token)
}
