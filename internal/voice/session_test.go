package voice_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/aps/internal/core/session"
	"hop.top/aps/internal/voice"
)

// TestRegisterSession_WritesVoiceTypedEntry registers a voice session
// and verifies it lands in the core SessionRegistry with
// Type=SessionTypeVoice and the channel preserved in Environment.
func TestRegisterSession_WritesVoiceTypedEntry(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	info, err := voice.RegisterSession("profile-1", "web")
	require.NoError(t, err)
	assert.NotEmpty(t, info.ID)
	assert.Equal(t, "profile-1", info.ProfileID)
	assert.Equal(t, session.SessionTypeVoice, info.Type)
	assert.Equal(t, session.SessionActive, info.Status)
	assert.Equal(t, "web", info.Environment[voice.ChannelMetaKey])

	// Verify visible via core registry.
	got, err := session.GetRegistry().Get(info.ID)
	require.NoError(t, err)
	assert.Equal(t, session.SessionTypeVoice, got.Type)
}

func TestRegisterSession_RequiresProfile(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	_, err := voice.RegisterSession("", "web")
	assert.Error(t, err)
}

func TestCloseSession_MarksInactive(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	info, err := voice.RegisterSession("p1", "web")
	require.NoError(t, err)

	require.NoError(t, voice.CloseSession(info.ID))

	got, err := session.GetRegistry().Get(info.ID)
	require.NoError(t, err)
	assert.Equal(t, session.SessionInactive, got.Status)
}
