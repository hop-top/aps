package voice_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestSessionMeta_Fields(t *testing.T) {
	meta := voice.SessionMeta{
		ProfileID:   "my-profile",
		ChannelType: "web",
		CallerID:    "user-123",
	}
	assert.Equal(t, "my-profile", meta.ProfileID)
	assert.Equal(t, "web", meta.ChannelType)
	assert.Equal(t, "user-123", meta.CallerID)
}

// mockChannelSession implements ChannelSession for testing.
type mockChannelSession struct {
	audioIn  chan []byte
	audioOut chan []byte
	textOut  chan string
	meta     voice.SessionMeta
	closed   bool
}

func (m *mockChannelSession) AudioIn() <-chan []byte   { return m.audioIn }
func (m *mockChannelSession) AudioOut() chan<- []byte  { return m.audioOut }
func (m *mockChannelSession) TextOut() chan<- string   { return m.textOut }
func (m *mockChannelSession) Meta() voice.SessionMeta { return m.meta }
func (m *mockChannelSession) Close() error            { m.closed = true; return nil }

func TestMockChannelSession_ImplementsInterface(t *testing.T) {
	var _ voice.ChannelSession = &mockChannelSession{}
	assert.True(t, true) // compile-time check
}
