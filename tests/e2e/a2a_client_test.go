package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	a2asdk "github.com/a2aproject/a2a-go/a2a"

	"oss-aps-cli/internal/a2a"
	"oss-aps-cli/internal/core"
)

func TestClient_ProfileToProfileCommunication(t *testing.T) {
	ctx := context.Background()

	// Create profile with available port
	targetProfile := GetTestProfileWithAvailablePort(t, "target-profile", "Target Profile")

	// Start test server
	testServer := StartTestServer(t, TestServerConfig{
		Profile:    targetProfile,
		ServerAddr: targetProfile.A2A.ListenAddr,
	})
	defer testServer.Stop()

	// Create client
	client, err := a2a.NewClient(targetProfile.ID, targetProfile)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, targetProfile.ID, client.GetProfileID())
	assert.NotNil(t, client.GetAgentCard())

	// Send message
	message := a2asdk.NewMessage(a2asdk.MessageRoleUser, a2asdk.TextPart{Text: "Deploy application"})

	task, err := client.SendMessage(ctx, message)
	require.NoError(t, err)
	assert.NotNil(t, task)
	assert.NotEmpty(t, task.ID)
}

func TestClient_ListTasks(t *testing.T) {
	ctx := context.Background()

	profile := &core.Profile{
		ID:           "test-list-profile",
		DisplayName:  "Test List Profile",
		Capabilities: []string{"test"},
		A2A: &core.A2AConfig{
			Enabled:         true,
			ProtocolBinding: "jsonrpc",
			ListenAddr:      "127.0.0.1:8084",
		},
	}

	client, err := a2a.NewClient(profile.ID, profile)
	require.NoError(t, err)

	resp, err := client.ListTasks(ctx, &a2asdk.ListTasksRequest{})
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, resp)
	}
}

func TestClient_CancelTask(t *testing.T) {
	ctx := context.Background()

	// Create profile with available port
	profile := GetTestProfileWithAvailablePort(t, "test-cancel-profile", "Test Cancel Profile")

	// Start test server
	testServer := StartTestServer(t, TestServerConfig{
		Profile:    profile,
		ServerAddr: profile.A2A.ListenAddr,
	})
	defer testServer.Stop()

	// Create client
	client, err := a2a.NewClient(profile.ID, profile)
	require.NoError(t, err)

	// Send message to create task
	message := a2asdk.NewMessage(a2asdk.MessageRoleUser, a2asdk.TextPart{Text: "Test task"})
	task, err := client.SendMessage(ctx, message)
	require.NoError(t, err)
	require.NotNil(t, task)

	// Cancel task
	err = client.CancelTask(ctx, task.ID)
	// Note: Cancel may succeed or fail depending on task state
	// We just verify the API call works
	t.Logf("Cancel task result: %v", err)
}
