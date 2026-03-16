package voice_test

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestTUIAdapter_AcceptConnection(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "v.sock")
	adapter, err := voice.NewTUIAdapter(socketPath, "profile-1")
	assert.NoError(t, err)

	sessions, err := adapter.Accept()
	assert.NoError(t, err)
	defer adapter.Close()

	// simulate TUI connecting
	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions
	assert.Equal(t, "tui", sess.Meta().ChannelType)
	assert.Equal(t, "profile-1", sess.Meta().ProfileID)
	sess.Close()
}

func TestTUIAdapter_CleanupStaleSocket(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "v.sock")

	// create a stale socket file
	l, err := net.Listen("unix", socketPath)
	assert.NoError(t, err)
	l.Close()

	// NewTUIAdapter should clean it up and succeed
	adapter, err := voice.NewTUIAdapter(socketPath, "profile-1")
	assert.NoError(t, err)
	adapter.Close()
}
