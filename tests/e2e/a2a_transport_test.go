package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	a2asdk "github.com/a2aproject/a2a-go/a2a"

	"oss-aps-cli/internal/a2a/transport"
	"oss-aps-cli/internal/core"
)

type E2EMockHandler struct{}

func (e *E2EMockHandler) HandleMessage(ctx context.Context, message *a2asdk.Message) error {
	return nil
}

func TestCrossTierCommunication_IPC(t *testing.T) {
	ctx := context.Background()

	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Profile",
		Capabilities: []string{"a2a", "ipc-test"},
		A2A: &core.A2AConfig{
			ProtocolBinding: "jsonrpc",
			ListenAddr:      "127.0.0.1:8081",
			IsolationTier:   "process",
		},
	}

	config := transport.DefaultIPCConfig(profile.ID)
	handler := &E2EMockHandler{}

	ipcTransport, err := transport.NewIPCTransport(config, handler)
	require.NoError(t, err)
	assert.Equal(t, "ipc", string(ipcTransport.Type()))

	message := a2asdk.NewMessage(a2asdk.MessageRoleUser, a2asdk.TextPart{Text: "Cross-tier test"})

	err = ipcTransport.Send(ctx, message)
	assert.NoError(t, err)

	ipcTransport.Close()
}

func TestCrossTierCommunication_TransportSelection(t *testing.T) {
	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Profile",
		Isolation: core.IsolationConfig{
			Level: core.IsolationPlatform,
		},
	}

	transportType, err := transport.SelectTransport(profile.Isolation.Level)
	assert.NoError(t, err)
	assert.Equal(t, "http", string(transportType))
}

func TestCrossTierCommunication_Fallback(t *testing.T) {
	current := transport.TransportHTTP
	fallback, ok := transport.GetFallbackTransport(current)

	assert.True(t, ok)
	assert.Equal(t, "grpc", string(fallback))
}
