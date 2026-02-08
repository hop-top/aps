package a2a

import (
	"context"
	"testing"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"oss-aps-cli/internal/core"
)

func TestNewServer_A2ADisabled(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-disabled",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: false,
		},
	}

	server, err := NewServer(profile, config)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Equal(t, ErrA2ANotEnabled, err)
}

func TestNewServer_NilProfile(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-nil",
	}

	server, err := NewServer(nil, config)

	assert.Error(t, err)
	assert.Nil(t, server)
}

func TestNewServer_NilConfig(t *testing.T) {
	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, nil)

	assert.Error(t, err)
	assert.Nil(t, server)
}

func TestServer_Start(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-start",
	}

	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Profile",
		A2A: &core.A2AConfig{
			Enabled:         true,
			ListenAddr:      "127.0.0.1:8081",
			ProtocolBinding: "jsonrpc",
			SecurityScheme:  "apikey",
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx, nil)
	require.NoError(t, err)

	assert.True(t, server.IsRunning())

	err = server.Stop()
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestServer_Stop(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-stop",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled:         true,
			ListenAddr:      "127.0.0.1:8081",
			ProtocolBinding: "jsonrpc",
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx, nil)
	require.NoError(t, err)

	assert.True(t, server.IsRunning())

	err = server.Stop()
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestServer_GetAddress(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-addr",
	}

	t.Run("default address", func(t *testing.T) {
		profile := &core.Profile{
			ID: "test-profile",
			A2A: &core.A2AConfig{
				Enabled: true,
			},
		}

		server, err := NewServer(profile, config)
		require.NoError(t, err)
		assert.Contains(t, server.getAddress(), ":8081")
	})

	t.Run("custom address", func(t *testing.T) {
		profile := &core.Profile{
			ID: "test-profile",
			A2A: &core.A2AConfig{
				Enabled:    true,
				ListenAddr: "127.0.0.1:9999",
			},
		}

		server, err := NewServer(profile, config)
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1:9999", server.getAddress())
	})
}

func TestServer_Name(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-name",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)
	assert.Equal(t, "a2a", server.Name())
}

func TestServer_ProfileID(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-profileid",
	}

	profile := &core.Profile{
		ID: "my-profile-123",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)
	assert.Equal(t, "my-profile-123", server.ProfileID())
}

func TestServer_IsRunning(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-running",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestServer_Status(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-status",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)
	assert.Equal(t, "stopped", server.Status())
}

func TestServer_GetStorage(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-storage",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)
	assert.NotNil(t, server.GetStorage())
	assert.Equal(t, config.BasePath, server.GetStorage().GetBasePath())
}

func TestServer_DoubleStart(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-double-start",
	}

	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Profile",
		A2A: &core.A2AConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:8082",
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx, nil)
	require.NoError(t, err)
	defer server.Stop()

	err = server.Start(ctx, nil)
	assert.Error(t, err)
}

func TestServer_StopWhenNotRunning(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-stop-notrunning",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	err = server.Stop()
	assert.NoError(t, err)
}

func TestServer_OnGetTaskPushConfig(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-pushconfig",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test config not found
	params := &a2a.GetTaskPushConfigParams{
		TaskID: a2a.NewTaskID(),
	}
	_, err = server.OnGetTaskPushConfig(ctx, params)
	assert.Error(t, err)
}

func TestServer_OnSetTaskPushConfig(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-settaskpush",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	taskID := a2a.NewTaskID()

	pushConfig := &a2a.TaskPushConfig{
		TaskID: taskID,
		Config: a2a.PushConfig{
			URL: "http://example.com/webhook",
		},
	}

	result, err := server.OnSetTaskPushConfig(ctx, pushConfig)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, taskID, result.TaskID)
}

func TestServer_OnDeleteTaskPushConfig(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-deletetaskpush",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	taskID := a2a.NewTaskID()

	// Set config first
	pushConfig := &a2a.TaskPushConfig{
		TaskID: taskID,
		Config: a2a.PushConfig{
			URL: "http://example.com/webhook",
		},
	}
	_, err = server.OnSetTaskPushConfig(ctx, pushConfig)
	require.NoError(t, err)

	// Delete it
	err = server.OnDeleteTaskPushConfig(ctx, &a2a.DeleteTaskPushConfigParams{
		TaskID: taskID,
	})
	assert.NoError(t, err)

	// Verify it's gone
	_, err = server.OnGetTaskPushConfig(ctx, &a2a.GetTaskPushConfigParams{
		TaskID: taskID,
	})
	assert.Error(t, err)
}

func TestServer_OnListTaskPushConfig(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-listtaskpush",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Set multiple configs
	for i := 0; i < 3; i++ {
		pushConfig := &a2a.TaskPushConfig{
			TaskID: a2a.NewTaskID(),
			Config: a2a.PushConfig{
				URL: "http://example.com/webhook",
			},
		}
		_, err = server.OnSetTaskPushConfig(ctx, pushConfig)
		require.NoError(t, err)
	}

	// List them
	configs, err := server.OnListTaskPushConfig(ctx, &a2a.ListTaskPushConfigParams{})
	assert.NoError(t, err)
	assert.Len(t, configs, 3)
}

func TestServer_OnGetExtendedAgentCard(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-agentcard",
	}

	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Agent",
		A2A: &core.A2AConfig{
			Enabled:         true,
			ProtocolBinding: "jsonrpc",
		},
		Capabilities: []string{"execute", "search"},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()

	card, err := server.OnGetExtendedAgentCard(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, card)
	assert.Equal(t, "Test Agent", card.Name)
	assert.NotEmpty(t, card.Skills)
}

func TestServer_Before(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-before",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	callCtx := &a2asrv.CallContext{}
	req := &a2asrv.Request{}

	newCtx, err := server.Before(ctx, callCtx, req)
	assert.NoError(t, err)
	assert.NotNil(t, newCtx)
}

func TestServer_After(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-after",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	callCtx := &a2asrv.CallContext{}
	resp := &a2asrv.Response{}

	err = server.After(ctx, callCtx, resp)
	assert.NoError(t, err)
}

func TestServer_OnGetTask(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create and save a task
	taskID := a2a.NewTaskID()
	task := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)
	_, err = server.storage.Save(ctx, task, event, 0)
	require.NoError(t, err)

	// Get the task
	query := &a2a.TaskQueryParams{
		ID: taskID,
	}
	retrievedTask, err := server.OnGetTask(ctx, query)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)
	assert.Equal(t, taskID, retrievedTask.ID)
}

func TestServer_OnGetTask_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	query := &a2a.TaskQueryParams{
		ID: a2a.NewTaskID(),
	}

	_, err = server.OnGetTask(ctx, query)
	assert.Error(t, err)
}

func TestServer_OnGetTask_WithHistoryLength(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a task with empty history
	taskID := a2a.NewTaskID()
	task := &a2a.Task{
		ID:      taskID,
		Status:  a2a.TaskStatus{State: a2a.TaskStateSubmitted},
		History: []*a2a.Message{},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)
	_, err = server.storage.Save(ctx, task, event, 0)
	require.NoError(t, err)

	// Get task with limited history
	historyLen := int(2)
	query := &a2a.TaskQueryParams{
		ID:            taskID,
		HistoryLength: &historyLen,
	}
	retrievedTask, err := server.OnGetTask(ctx, query)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)
}

func TestServer_OnCancelTask(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a task
	taskID := a2a.NewTaskID()
	task := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateWorking},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateWorking, nil)
	_, err = server.storage.Save(ctx, task, event, 0)
	require.NoError(t, err)

	// Cancel the task
	params := &a2a.TaskIDParams{
		ID: taskID,
	}
	cancelledTask, err := server.OnCancelTask(ctx, params)
	assert.NoError(t, err)
	assert.NotNil(t, cancelledTask)
}

func TestServer_OnCancelTask_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	params := &a2a.TaskIDParams{
		ID: a2a.NewTaskID(),
	}

	_, err = server.OnCancelTask(ctx, params)
	assert.Error(t, err)
}

func TestServer_OnSendMessage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Pre-create and save a task with a specific ID
	taskID := a2a.NewTaskID()
	task := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)
	_, err = server.storage.Save(ctx, task, event, 0)
	require.NoError(t, err)

	// Create a message with the same task ID
	message := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "test message"})
	message.TaskID = taskID

	params := &a2a.MessageSendParams{
		Message: message,
	}

	result, err := server.OnSendMessage(ctx, params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestServer_OnSendMessage_NilMessage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: nil,
	}

	_, err = server.OnSendMessage(ctx, params)
	assert.Error(t, err)
}

func TestServer_OnSendMessage_WithTaskID(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a task first
	taskID := a2a.NewTaskID()
	task := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)
	_, err = server.storage.Save(ctx, task, event, 0)
	require.NoError(t, err)

	// Send a message to that task
	message := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "test message"})
	message.TaskID = taskID

	params := &a2a.MessageSendParams{
		Message: message,
	}

	result, err := server.OnSendMessage(ctx, params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
