package a2a_transport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	a2asdk "github.com/a2aproject/a2a-go/a2a"

	"hop.top/aps/internal/a2a/transport"
)

type GRPCMockHandler struct{}

func (m *GRPCMockHandler) HandleMessage(ctx context.Context, message *a2asdk.Message) error {
	return nil
}

func TestNewGRPCTransport_NilConfig(t *testing.T) {
	transport, err := transport.NewGRPCTransport(nil, nil)

	assert.Error(t, err)
	assert.Nil(t, transport)
}

func TestNewGRPCTransport_EmptyEndpoint(t *testing.T) {
	config := transport.DefaultGRPCConfig("")
	config.Endpoint = ""

	transport, err := transport.NewGRPCTransport(config, nil)

	assert.Error(t, err)
	assert.Nil(t, transport)
}

func TestNewGRPCTransport_ValidConfig(t *testing.T) {
	config := transport.DefaultGRPCConfig("127.0.0.1:9090")

	handler := &GRPCMockHandler{}
	grpcTransport, err := transport.NewGRPCTransport(config, handler)
	require.NoError(t, err)
	assert.NotNil(t, grpcTransport)
	assert.Equal(t, transport.TransportGRPC, grpcTransport.Type())

	grpcTransport.Close()
}

func TestNewGRPCTransport_SendMessage(t *testing.T) {
	ctx := context.Background()
	config := transport.DefaultGRPCConfig("127.0.0.1:9090")

	handler := &GRPCMockHandler{}
	grpcTransport, err := transport.NewGRPCTransport(config, handler)
	require.NoError(t, err)

	message := a2asdk.NewMessage(a2asdk.MessageRoleUser, a2asdk.TextPart{Text: "Test"})

	err = grpcTransport.Send(ctx, message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet fully implemented")

	grpcTransport.Close()
}

func TestNewGRPCTransport_IsHealthy(t *testing.T) {
	config := transport.DefaultGRPCConfig("127.0.0.1:9090")

	handler := &GRPCMockHandler{}
	grpcTransport, err := transport.NewGRPCTransport(config, handler)
	require.NoError(t, err)

	assert.False(t, grpcTransport.IsHealthy())

	grpcTransport.Close()
}
