package a2a

import (
	"context"
	"testing"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
)

func TestNewClient_Success(t *testing.T) {
	profile := &core.Profile{
		ID:          "target-profile",
		DisplayName: "Target Agent",
		A2A: &core.A2AConfig{
			ListenAddr:      "127.0.0.1:8081",
			ProtocolBinding: "jsonrpc",
		},
		Capabilities: []string{"a2a", "execute"},
	}

	client, err := NewClient("target-profile", profile)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "target-profile", client.GetProfileID())
}

func TestNewClient_EmptyProfileID(t *testing.T) {
	profile := &core.Profile{
		ID:           "target-profile",
		Capabilities: []string{"a2a"},
		A2A:          &core.A2AConfig{},
	}

	client, err := NewClient("", profile)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Equal(t, ErrInvalidConfig, err)
}

func TestNewClient_NilProfile(t *testing.T) {
	client, err := NewClient("target-profile", nil)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_A2ADisabled(t *testing.T) {
	profile := &core.Profile{
		ID:  "target-profile",
		A2A: nil,
	}

	client, err := NewClient("target-profile", profile)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Equal(t, ErrA2ANotEnabled, err)
}

func TestNewClient_A2AConfigNil(t *testing.T) {
	profile := &core.Profile{
		ID:  "target-profile",
		A2A: nil,
	}

	client, err := NewClient("target-profile", profile)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_GetProfileID(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-target",
		DisplayName:  "Target Agent",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
	}

	client, err := NewClient("test-target", profile)
	require.NoError(t, err)
	assert.Equal(t, "test-target", client.GetProfileID())
}

func TestNewClient_GetAgentCard(t *testing.T) {
	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Agent",
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
		Capabilities: []string{"a2a", "execute"},
	}

	client, err := NewClient("test-profile", profile)
	require.NoError(t, err)

	card := client.GetAgentCard()
	assert.NotNil(t, card)
	assert.Equal(t, "Test Agent", card.Name)
	assert.NotEmpty(t, card.URL)
}

func TestValidateAgentCardTransport_ValidJSONRPC(t *testing.T) {
	card := &a2a.AgentCard{
		URL:                "http://127.0.0.1:8081",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		Skills:             []a2a.AgentSkill{{ID: "execute"}},
	}

	err := validateAgentCardTransport(card)
	assert.NoError(t, err)
}

func TestValidateAgentCardTransport_ValidGRPC(t *testing.T) {
	card := &a2a.AgentCard{
		URL:                "http://127.0.0.1:8081",
		PreferredTransport: a2a.TransportProtocolGRPC,
		Skills:             []a2a.AgentSkill{{ID: "execute"}},
	}

	err := validateAgentCardTransport(card)
	assert.NoError(t, err)
}

func TestValidateAgentCardTransport_NilCard(t *testing.T) {
	err := validateAgentCardTransport(nil)
	assert.Error(t, err)
}

func TestValidateAgentCardTransport_NoURL(t *testing.T) {
	card := &a2a.AgentCard{
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	err := validateAgentCardTransport(card)
	assert.Error(t, err)
}

func TestValidateAgentCardTransport_UnsupportedTransport(t *testing.T) {
	card := &a2a.AgentCard{
		URL:                "http://127.0.0.1:8081",
		PreferredTransport: a2a.TransportProtocol("unsupported"),
	}

	err := validateAgentCardTransport(card)
	assert.Error(t, err)
}

func TestValidateAgentCardTransport_NoTransportSetDefault(t *testing.T) {
	card := &a2a.AgentCard{
		URL: "http://127.0.0.1:8081",
	}

	err := validateAgentCardTransport(card)
	assert.NoError(t, err)
	assert.Equal(t, a2a.TransportProtocolJSONRPC, card.PreferredTransport)
}

func TestClient_SendMessage_NilMessage(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Agent",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
	}

	client, err := NewClient("test-profile", profile)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = client.SendMessage(ctx, nil)
	assert.Error(t, err)
}

func TestClient_GetTask_EmptyTaskID(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Agent",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
	}

	client, err := NewClient("test-profile", profile)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = client.GetTask(ctx, "")
	assert.Error(t, err)
}

func TestClient_CancelTask_EmptyTaskID(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Agent",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
	}

	client, err := NewClient("test-profile", profile)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.CancelTask(ctx, "")
	assert.Error(t, err)
}

func TestClient_SubscribeToTask_EmptyTaskID(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Agent",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
	}

	client, err := NewClient("test-profile", profile)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.SubscribeToTask(ctx, "", "http://example.com/webhook")
	assert.Error(t, err)
}

func TestClient_SubscribeToTask_EmptyWebhookURL(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Agent",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
	}

	client, err := NewClient("test-profile", profile)
	require.NoError(t, err)

	ctx := context.Background()
	taskID := a2a.NewTaskID()
	err = client.SubscribeToTask(ctx, taskID, "")
	assert.Error(t, err)
}

func TestClient_ListTasks_NotSupported(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Agent",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
	}

	client, err := NewClient("test-profile", profile)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = client.ListTasks(ctx, nil)
	assert.Error(t, err)
}

func TestClient_SendMessageStream_NotSupported(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Agent",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ListenAddr: "127.0.0.1:8081",
		},
	}

	client, err := NewClient("test-profile", profile)
	require.NoError(t, err)

	ctx := context.Background()
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "test"})
	_, err = client.SendMessageStream(ctx, msg)
	assert.Error(t, err)
}
