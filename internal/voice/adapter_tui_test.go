package voice_test

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestTUISession_AudioIn_ReceivesFrame(t *testing.T) {
	dir, err := os.MkdirTemp("", "tui")
	assert.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	socketPath := filepath.Join(dir, "v.sock")

	adapter, err := voice.NewTUIAdapter(socketPath, "profile-1")
	assert.NoError(t, err)
	sessions, _ := adapter.Accept()
	defer adapter.Close()

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("pcm-frame-data")
	_, err = conn.Write(frame)
	assert.NoError(t, err)

	select {
	case received := <-sess.AudioIn():
		assert.Equal(t, frame, received)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for audio frame")
	}
	sess.Close()
}

func TestTUISession_AudioOut_WritesToConn(t *testing.T) {
	dir, err := os.MkdirTemp("", "tui")
	assert.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	socketPath := filepath.Join(dir, "v.sock")

	adapter, err := voice.NewTUIAdapter(socketPath, "profile-1")
	assert.NoError(t, err)
	sessions, _ := adapter.Accept()
	defer adapter.Close()

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("response-audio")
	sess.AudioOut() <- frame

	buf := make([]byte, len(frame))
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err = io.ReadFull(conn, buf)
	assert.NoError(t, err)
	assert.Equal(t, frame, buf)
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
