package voice_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestSessionManager_CreateAndGet(t *testing.T) {
	sm := voice.NewSessionManager()
	sess := sm.Create("profile-1", "web")
	assert.NotEmpty(t, sess.ID)
	assert.Equal(t, "profile-1", sess.ProfileID)
	assert.Equal(t, "web", sess.ChannelType)
	assert.Equal(t, voice.SessionStateActive, sess.State)

	got, err := sm.Get(sess.ID)
	assert.NoError(t, err)
	assert.Equal(t, sess.ID, got.ID)
}

func TestSessionManager_List(t *testing.T) {
	sm := voice.NewSessionManager()
	sm.Create("p1", "web")
	sm.Create("p2", "tui")
	sessions := sm.List()
	assert.Len(t, sessions, 2)
}

func TestSessionManager_Close(t *testing.T) {
	sm := voice.NewSessionManager()
	sess := sm.Create("p1", "web")
	err := sm.Close(sess.ID)
	assert.NoError(t, err)
	got, err := sm.Get(sess.ID)
	assert.NoError(t, err)
	assert.Equal(t, voice.SessionStateClosed, got.State)
}

func TestSessionManager_GetUnknown(t *testing.T) {
	sm := voice.NewSessionManager()
	_, err := sm.Get("does-not-exist")
	assert.Error(t, err)
}

func TestSessionManager_SwitchProfile(t *testing.T) {
	sm := voice.NewSessionManager()
	sess := sm.Create("p1", "web")
	err := sm.SwitchProfile(sess.ID, "p2")
	assert.NoError(t, err)
	got, _ := sm.Get(sess.ID)
	assert.Equal(t, "p2", got.ProfileID)
}
