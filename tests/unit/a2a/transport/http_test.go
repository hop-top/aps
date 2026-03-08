package a2a_transport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	a2asdk "github.com/a2aproject/a2a-go/a2a"

	"hop.top/aps/internal/a2a/transport"
)

type HTTPMockHandler struct{}

func (m *HTTPMockHandler) HandleMessage(ctx context.Context, message *a2asdk.Message) error {
	return nil
}

func TestNewHTTPTransport_NilConfig(t *testing.T) {
	transport, err := transport.NewHTTPTransport(nil, nil)

	assert.Error(t, err)
	assert.Nil(t, transport)
}

func TestNewHTTPTransport_EmptyEndpoint(t *testing.T) {
	config := transport.DefaultHTTPConfig("")
	config.Endpoint = ""

	transport, err := transport.NewHTTPTransport(config, nil)

	assert.Error(t, err)
	assert.Nil(t, transport)
}

func TestNewHTTPTransport_ValidConfig(t *testing.T) {
	config := transport.DefaultHTTPConfig("http://127.0.0.1:8081")

	handler := &HTTPMockHandler{}
	httpTransport, err := transport.NewHTTPTransport(config, handler)
	require.NoError(t, err)
	assert.NotNil(t, httpTransport)
	assert.Equal(t, transport.TransportHTTP, httpTransport.Type())

	httpTransport.Close()
}

func TestNewHTTPTransport_SendMessage(t *testing.T) {
	ctx := context.Background()
	config := transport.DefaultHTTPConfig("http://127.0.0.1:9999")
	config.Timeout = 100

	handler := &HTTPMockHandler{}
	httpTransport, err := transport.NewHTTPTransport(config, handler)
	require.NoError(t, err)

	message := a2asdk.NewMessage(a2asdk.MessageRoleUser, a2asdk.TextPart{Text: "Test"})

	err = httpTransport.Send(ctx, message)
	assert.Error(t, err)

	httpTransport.Close()
}

func TestNewHTTPTransport_IsHealthy(t *testing.T) {
	config := transport.DefaultHTTPConfig("http://127.0.0.1:9999")

	handler := &HTTPMockHandler{}
	httpTransport, err := transport.NewHTTPTransport(config, handler)
	require.NoError(t, err)

	// HTTP transport health check requires actual server - will return false without server
	assert.False(t, httpTransport.IsHealthy())

	httpTransport.Close()
	assert.False(t, httpTransport.IsHealthy())
}
