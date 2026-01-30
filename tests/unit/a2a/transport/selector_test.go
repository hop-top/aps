package a2a_transport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"oss-aps-cli/internal/a2a/transport"
	"oss-aps-cli/internal/core"
)

func TestSelectTransport_ProcessTier(t *testing.T) {
	tier := core.IsolationProcess

	transportType, err := transport.SelectTransport(tier)
	assert.NoError(t, err)
	assert.Equal(t, "ipc", string(transportType))
}

func TestSelectTransport_PlatformTier(t *testing.T) {
	tier := core.IsolationPlatform

	transportType, err := transport.SelectTransport(tier)
	assert.NoError(t, err)
	assert.Equal(t, "http", string(transportType))
}

func TestSelectTransport_ContainerTier(t *testing.T) {
	tier := core.IsolationContainer

	transportType, err := transport.SelectTransport(tier)
	assert.NoError(t, err)
	assert.Equal(t, "grpc", string(transportType))
}

func TestGetFallbackTransport_IPCToHTTP(t *testing.T) {
	fallback, ok := transport.GetFallbackTransport("ipc")
	assert.True(t, ok)
	assert.Equal(t, "http", string(fallback))
}

func TestGetFallbackTransport_HTTPToGRPC(t *testing.T) {
	fallback, ok := transport.GetFallbackTransport("http")
	assert.True(t, ok)
	assert.Equal(t, "grpc", string(fallback))
}

func TestGetFallbackTransport_GRPCNoFallback(t *testing.T) {
	fallback, ok := transport.GetFallbackTransport("grpc")
	assert.False(t, ok)
	assert.Equal(t, "", string(fallback))
}
