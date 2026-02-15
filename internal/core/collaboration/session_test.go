package collaboration_test

import (
	"testing"
	"time"

	"oss-aps-cli/internal/core/collaboration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionManager_CreateSession(t *testing.T) {
	sm := collaboration.NewSessionManager()

	session, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "ws-1", session.WorkspaceID)
	assert.Equal(t, "agent-1", session.ProfileID)
	assert.Equal(t, collaboration.SessionActive, session.State)
	assert.False(t, session.JoinedAt.IsZero())
	assert.False(t, session.LastSeen.IsZero())
	assert.True(t, session.ExpiresAt.After(time.Now()))
}

func TestSessionManager_CreateSession_Duplicate(t *testing.T) {
	sm := collaboration.NewSessionManager()

	_, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	_, err = sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already has an active session")
}

func TestSessionManager_Heartbeat(t *testing.T) {
	sm := collaboration.NewSessionManager()

	session, err := sm.CreateSession("ws-1", "agent-1", 5*time.Second)
	require.NoError(t, err)

	originalLastSeen := session.LastSeen
	originalExpires := session.ExpiresAt

	// Small sleep to ensure time progresses.
	time.Sleep(10 * time.Millisecond)

	err = sm.Heartbeat(session.ID, 30*time.Second)
	require.NoError(t, err)

	updated, err := sm.GetSession(session.ID)
	require.NoError(t, err)
	assert.True(t, updated.LastSeen.After(originalLastSeen) || updated.LastSeen.Equal(originalLastSeen))
	assert.True(t, updated.ExpiresAt.After(originalExpires))
}

func TestSessionManager_Heartbeat_NotFound(t *testing.T) {
	sm := collaboration.NewSessionManager()

	err := sm.Heartbeat("nonexistent", 30*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionManager_Heartbeat_Closed(t *testing.T) {
	sm := collaboration.NewSessionManager()

	session, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	err = sm.CloseSession(session.ID)
	require.NoError(t, err)

	err = sm.Heartbeat(session.ID, 30*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestSessionManager_CloseSession(t *testing.T) {
	sm := collaboration.NewSessionManager()

	session, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	err = sm.CloseSession(session.ID)
	require.NoError(t, err)

	got, err := sm.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, collaboration.SessionClosed, got.State)
}

func TestSessionManager_CloseSession_NotFound(t *testing.T) {
	sm := collaboration.NewSessionManager()

	err := sm.CloseSession("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionManager_GetSession(t *testing.T) {
	sm := collaboration.NewSessionManager()

	session, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	got, err := sm.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, got.ID)
	assert.Equal(t, "ws-1", got.WorkspaceID)
	assert.Equal(t, "agent-1", got.ProfileID)
}

func TestSessionManager_GetSession_NotFound(t *testing.T) {
	sm := collaboration.NewSessionManager()

	_, err := sm.GetSession("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionManager_GetAgentSession(t *testing.T) {
	sm := collaboration.NewSessionManager()

	created, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	got, err := sm.GetAgentSession("ws-1", "agent-1")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestSessionManager_GetAgentSession_NotFound(t *testing.T) {
	sm := collaboration.NewSessionManager()

	_, err := sm.GetAgentSession("ws-1", "agent-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active session")
}

func TestSessionManager_ListSessions(t *testing.T) {
	sm := collaboration.NewSessionManager()

	_, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	_, err = sm.CreateSession("ws-1", "agent-2", 30*time.Second)
	require.NoError(t, err)

	// Different workspace -- should not appear in ws-1 list.
	_, err = sm.CreateSession("ws-2", "agent-3", 30*time.Second)
	require.NoError(t, err)

	sessions := sm.ListSessions("ws-1")
	assert.Len(t, sessions, 2)

	profileIDs := make(map[string]bool)
	for _, s := range sessions {
		profileIDs[s.ProfileID] = true
	}
	assert.True(t, profileIDs["agent-1"])
	assert.True(t, profileIDs["agent-2"])
}

func TestSessionManager_ListActiveSessions(t *testing.T) {
	sm := collaboration.NewSessionManager()

	s1, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	_, err = sm.CreateSession("ws-1", "agent-2", 30*time.Second)
	require.NoError(t, err)

	// Close one session.
	err = sm.CloseSession(s1.ID)
	require.NoError(t, err)

	active := sm.ListActiveSessions("ws-1")
	assert.Len(t, active, 1)
	assert.Equal(t, "agent-2", active[0].ProfileID)
}

func TestSessionManager_ExpireSessions(t *testing.T) {
	sm := collaboration.NewSessionManager()

	// Create with a very short timeout so it expires almost immediately.
	_, err := sm.CreateSession("ws-1", "agent-1", 1*time.Millisecond)
	require.NoError(t, err)

	// Create with a long timeout so it stays active.
	_, err = sm.CreateSession("ws-1", "agent-2", 1*time.Hour)
	require.NoError(t, err)

	// Wait for the short timeout to pass.
	time.Sleep(5 * time.Millisecond)

	expired := sm.ExpireSessions("ws-1")
	assert.Len(t, expired, 1)
	assert.Contains(t, expired, "agent-1")

	// Only agent-2 should remain active.
	active := sm.ListActiveSessions("ws-1")
	assert.Len(t, active, 1)
	assert.Equal(t, "agent-2", active[0].ProfileID)
}

func TestSessionManager_CleanupClosed(t *testing.T) {
	sm := collaboration.NewSessionManager()

	s1, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	s2, err := sm.CreateSession("ws-1", "agent-2", 30*time.Second)
	require.NoError(t, err)

	_, err = sm.CreateSession("ws-1", "agent-3", 30*time.Second)
	require.NoError(t, err)

	// Close two sessions.
	err = sm.CloseSession(s1.ID)
	require.NoError(t, err)

	err = sm.CloseSession(s2.ID)
	require.NoError(t, err)

	removed := sm.CleanupClosed("ws-1")
	assert.Equal(t, 2, removed)

	// Only the active session should remain.
	all := sm.ListSessions("ws-1")
	assert.Len(t, all, 1)
	assert.Equal(t, "agent-3", all[0].ProfileID)
}
