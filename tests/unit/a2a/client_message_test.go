package a2a_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	a2asdk "github.com/a2aproject/a2a-go/a2a"

	"hop.top/aps/internal/a2a"
	"hop.top/aps/internal/core"
)

func TestClient_SendMessage_InvalidMessage(t *testing.T) {
	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Profile",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ProtocolBinding: "jsonrpc",
			ListenAddr:      freeAddr(t),
			IsolationTier:   "process",
		},
	}

	client, err := a2a.NewClient(profile.ID, profile)
	require.NoError(t, err)

	ctx := context.Background()

	task, err := client.SendMessage(ctx, nil)

	assert.Error(t, err)
	assert.Nil(t, task)
}

func TestClient_SendMessage_ValidMessage(t *testing.T) {
	// freeAddr binds :0, captures the assigned port, and closes the
	// listener so we have an address that is (very likely) free for
	// the duration of the test. SendMessage is expected to fail with
	// "connection refused" — without a kernel-assigned port, a sibling
	// test happening to listen on a hardcoded constant could make this
	// connect succeed and silently invert the assertion.
	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Profile",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ProtocolBinding: "jsonrpc",
			ListenAddr:      freeAddr(t),
			IsolationTier:   "process",
		},
	}

	client, err := a2a.NewClient(profile.ID, profile)
	require.NoError(t, err)

	ctx := context.Background()
	message := a2asdk.NewMessage(a2asdk.MessageRoleUser, a2asdk.TextPart{Text: "Hello from test"})

	task, err := client.SendMessage(ctx, message)

	// Note: This test expects connection refused since there's no server running
	// This is a unit test - for actual message sending, use integration tests
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "refused")
}
