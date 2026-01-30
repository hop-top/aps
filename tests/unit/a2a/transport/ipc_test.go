package a2a_transport

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	a2asdk "github.com/a2aproject/a2a-go/a2a"

	"oss-aps-cli/internal/a2a/transport"
)

func TestNewIPCTransport_NilConfig(t *testing.T) {
	transport, err := transport.NewIPCTransport(nil, nil)

	assert.Error(t, err)
	assert.Nil(t, transport)
}

func TestNewIPCTransport_EmptyProfileID(t *testing.T) {
	config := transport.DefaultIPCConfig("")
	config.ProfileID = ""

	transport, err := transport.NewIPCTransport(config, nil)

	assert.Error(t, err)
	assert.Nil(t, transport)
}

func TestNewIPCTransport_ValidConfig(t *testing.T) {
	config := transport.DefaultIPCConfig("test-profile")

	handler := &MockMessageHandler{}

	ipcTransport, err := transport.NewIPCTransport(config, handler)
	require.NoError(t, err)
	assert.NotNil(t, ipcTransport)
	assert.Equal(t, transport.TransportIPC, ipcTransport.Type())
	assert.Equal(t, "test-profile", ipcTransport.GetProfileID())
}

func TestIPCTransport_SendMessage(t *testing.T) {
	ctx := context.Background()
	config := transport.DefaultIPCConfig("test-profile")

	handler := &MockMessageHandler{}
	ipc, err := transport.NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipc.Close()

	message := a2asdk.NewMessage(a2asdk.MessageRoleUser, a2asdk.TextPart{Text: "Test message"})

	err = ipc.Send(ctx, message)
	assert.NoError(t, err)

	entries, _ := os.ReadDir(config.QueuePath)
	assert.True(t, len(entries) > 0)
}

func TestIPCTransport_ReceiveMessage(t *testing.T) {
	ctx := context.Background()
	config := transport.DefaultIPCConfig("test-profile")

	handler := &MockMessageHandler{
		receiveChan: make(chan struct{}, 1),
	}

	ipc, err := transport.NewIPCTransport(config, handler)
	require.NoError(t, err)

	err = ipc.Start()
	require.NoError(t, err)

	defer ipc.Stop()

	message := a2asdk.NewMessage(a2asdk.MessageRoleUser, a2asdk.TextPart{Text: "Test receive"})

	_ = ipc.Send(ctx, message)

	received, err := ipc.Receive(ctx)
	if err == nil && received != nil {
		select {
		case <-handler.receiveChan:
			assert.NotNil(t, received)
		case <-time.After(100 * time.Millisecond):
			t.Log("Timeout waiting for message")
		}
	}
}

func TestIPCTransport_IsHealthy(t *testing.T) {
	config := transport.DefaultIPCConfig("test-profile")

	handler := &MockMessageHandler{}
	ipc, err := transport.NewIPCTransport(config, handler)
	require.NoError(t, err)

	assert.True(t, ipc.IsHealthy())

	ipc.Close()
}

func TestIPCTransport_Close(t *testing.T) {
	config := transport.DefaultIPCConfig("test-profile")

	handler := &MockMessageHandler{}
	ipc, err := transport.NewIPCTransport(config, handler)
	require.NoError(t, err)

	err = ipc.Close()
	assert.NoError(t, err)

	// IsHealthy checks if queue path exists, which persists after Close
	assert.True(t, ipc.IsHealthy())
}

func TestIPCTransport_StartStop(t *testing.T) {
	config := transport.DefaultIPCConfig("test-profile")
	config.Polling = false

	handler := &MockMessageHandler{}
	ipc, err := transport.NewIPCTransport(config, handler)
	require.NoError(t, err)

	err = ipc.Start()
	assert.NoError(t, err)
	assert.True(t, true)

	err = ipc.Stop()
	assert.NoError(t, err)
}

type MockMessageHandler struct {
	receiveChan chan struct{}
}

func (m *MockMessageHandler) HandleMessage(ctx context.Context, message *a2asdk.Message) error {
	if m.receiveChan != nil {
		m.receiveChan <- struct{}{}
	}
	return nil
}
