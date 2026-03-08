package multidevice

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestHome configures HOME and XDG_DATA_HOME to a temporary directory for test isolation.
func setupTestHome(t *testing.T) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
	t.Setenv("APS_DATA_PATH", "")
	return tmpHome
}

// ============================================================================
// Types Tests
// ============================================================================

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role  DeviceRole
		valid bool
	}{
		{RoleOwner, true},
		{RoleCollaborator, true},
		{RoleViewer, true},
		{"admin", false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidRole(tt.role))
		})
	}
}

func TestValidRoles(t *testing.T) {
	roles := ValidRoles()
	assert.Len(t, roles, 3)
	assert.Contains(t, roles, RoleOwner)
	assert.Contains(t, roles, RoleCollaborator)
	assert.Contains(t, roles, RoleViewer)
}

func TestPermissionsForRole_Owner(t *testing.T) {
	perms := PermissionsForRole(RoleOwner)
	assert.Equal(t, RoleOwner, perms.Role)
	assert.True(t, perms.CanRead)
	assert.True(t, perms.CanWrite)
	assert.True(t, perms.CanExecute)
	assert.True(t, perms.CanManage)
	assert.True(t, perms.CanSync)
}

func TestPermissionsForRole_Collaborator(t *testing.T) {
	perms := PermissionsForRole(RoleCollaborator)
	assert.Equal(t, RoleCollaborator, perms.Role)
	assert.True(t, perms.CanRead)
	assert.True(t, perms.CanWrite)
	assert.True(t, perms.CanExecute)
	assert.False(t, perms.CanManage)
	assert.True(t, perms.CanSync)
}

func TestPermissionsForRole_Viewer(t *testing.T) {
	perms := PermissionsForRole(RoleViewer)
	assert.Equal(t, RoleViewer, perms.Role)
	assert.True(t, perms.CanRead)
	assert.False(t, perms.CanWrite)
	assert.False(t, perms.CanExecute)
	assert.False(t, perms.CanManage)
	assert.True(t, perms.CanSync)
}

func TestPermissionsForRole_Unknown(t *testing.T) {
	perms := PermissionsForRole("unknown")
	assert.Equal(t, RoleViewer, perms.Role)
	assert.True(t, perms.CanRead)
	assert.True(t, perms.CanSync)
	assert.False(t, perms.CanWrite)
}

func TestDefaultPresenceConfig(t *testing.T) {
	cfg := DefaultPresenceConfig()
	assert.Equal(t, 10*time.Second, cfg.HeartbeatInterval)
	assert.Equal(t, 30*time.Second, cfg.AwayTimeout)
	assert.Equal(t, 120*time.Second, cfg.OfflineTimeout)
}

// ============================================================================
// Storage Tests
// ============================================================================

func TestSaveAndLoadLink(t *testing.T) {
	setupTestHome(t)

	link := &WorkspaceDeviceLink{
		WorkspaceID: "ws-1",
		DeviceID:    "dev-1",
		LinkedAt:    time.Now(),
		LinkedBy:    "dev-1",
		Permissions: PermissionsForRole(RoleOwner),
		Status:      PresenceLinking,
	}

	err := SaveLink(link)
	require.NoError(t, err)

	loaded, err := LoadLink("ws-1", "dev-1")
	require.NoError(t, err)
	assert.Equal(t, "ws-1", loaded.WorkspaceID)
	assert.Equal(t, "dev-1", loaded.DeviceID)
	assert.Equal(t, RoleOwner, loaded.Permissions.Role)
	assert.Equal(t, PresenceLinking, loaded.Status)
}

func TestSaveLink_MissingWorkspaceID(t *testing.T) {
	setupTestHome(t)

	link := &WorkspaceDeviceLink{DeviceID: "dev-1"}
	err := SaveLink(link)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace ID is required")
}

func TestSaveLink_MissingDeviceID(t *testing.T) {
	setupTestHome(t)

	link := &WorkspaceDeviceLink{WorkspaceID: "ws-1"}
	err := SaveLink(link)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device ID is required")
}

func TestLoadLink_NotFound(t *testing.T) {
	setupTestHome(t)

	_, err := LoadLink("ws-1", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device link not found")
}

func TestDeleteLink(t *testing.T) {
	setupTestHome(t)

	link := &WorkspaceDeviceLink{
		WorkspaceID: "ws-1",
		DeviceID:    "dev-1",
		Permissions: PermissionsForRole(RoleOwner),
	}
	require.NoError(t, SaveLink(link))

	err := DeleteLink("ws-1", "dev-1")
	require.NoError(t, err)

	_, err = LoadLink("ws-1", "dev-1")
	require.Error(t, err)
}

func TestDeleteLink_NotFound(t *testing.T) {
	setupTestHome(t)

	err := DeleteLink("ws-1", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device link not found")
}

func TestListLinks(t *testing.T) {
	setupTestHome(t)

	// No links directory exists yet; should return empty.
	links, err := ListLinks("ws-1")
	require.NoError(t, err)
	assert.Len(t, links, 0)

	// Save two links.
	for _, id := range []string{"dev-1", "dev-2"} {
		require.NoError(t, SaveLink(&WorkspaceDeviceLink{
			WorkspaceID: "ws-1",
			DeviceID:    id,
			Permissions: PermissionsForRole(RoleViewer),
		}))
	}

	links, err = ListLinks("ws-1")
	require.NoError(t, err)
	assert.Len(t, links, 2)
}

// ============================================================================
// Linker Tests
// ============================================================================

func TestLinker_AttachDevice(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	link, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)
	assert.Equal(t, "ws-1", link.WorkspaceID)
	assert.Equal(t, "dev-1", link.DeviceID)
	assert.Equal(t, RoleOwner, link.Permissions.Role)
	assert.Equal(t, PresenceLinking, link.Status)
}

func TestLinker_AttachDevice_EmptyWorkspace(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("", "dev-1", RoleOwner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace ID is required")
}

func TestLinker_AttachDevice_EmptyDevice(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "", RoleOwner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device ID is required")
}

func TestLinker_AttachDevice_InvalidRole(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid device role")
}

func TestLinker_AttachDevice_AlreadyLinked(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	_, err = linker.AttachDevice("ws-1", "dev-1", RoleViewer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already linked")
}

func TestLinker_DetachDevice(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	err = linker.DetachDevice("ws-1", "dev-1")
	require.NoError(t, err)

	_, err = linker.GetLink("ws-1", "dev-1")
	require.Error(t, err)
}

func TestLinker_DetachDevice_EmptyIDs(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()

	err := linker.DetachDevice("", "dev-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace ID is required")

	err = linker.DetachDevice("ws-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device ID is required")
}

func TestLinker_DetachDevice_NotLinked(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	err := linker.DetachDevice("ws-1", "dev-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot detach")
}

func TestLinker_SetRole(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	err = linker.SetRole("ws-1", "dev-1", RoleViewer)
	require.NoError(t, err)

	link, err := linker.GetLink("ws-1", "dev-1")
	require.NoError(t, err)
	assert.Equal(t, RoleViewer, link.Permissions.Role)
	assert.False(t, link.Permissions.CanWrite)
}

func TestLinker_SetRole_InvalidRole(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	err = linker.SetRole("ws-1", "dev-1", "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid device role")
}

func TestLinker_UpdatePermissions(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	newPerms := DevicePermissions{
		Role:            RoleCollaborator,
		CanRead:         true,
		CanWrite:        true,
		CanExecute:      false,
		CanManage:       false,
		CanSync:         true,
		RateLimitPerMin: 100,
	}
	err = linker.UpdatePermissions("ws-1", "dev-1", newPerms)
	require.NoError(t, err)

	link, err := linker.GetLink("ws-1", "dev-1")
	require.NoError(t, err)
	assert.Equal(t, RoleCollaborator, link.Permissions.Role)
	assert.Equal(t, 100, link.Permissions.RateLimitPerMin)
	assert.False(t, link.Permissions.CanExecute)
}

func TestLinker_ListLinks(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)
	_, err = linker.AttachDevice("ws-1", "dev-2", RoleViewer)
	require.NoError(t, err)

	links, err := linker.ListLinks("ws-1")
	require.NoError(t, err)
	assert.Len(t, links, 2)
}

// ============================================================================
// Events Tests
// ============================================================================

func TestNewEvent(t *testing.T) {
	payload := map[string]interface{}{"key": "value"}
	event := NewEvent("ws-1", "dev-1", EventProfileCreated, payload)

	assert.NotEmpty(t, event.ID)
	assert.Contains(t, event.ID, "evt-")
	assert.Equal(t, "ws-1", event.WorkspaceID)
	assert.Equal(t, "dev-1", event.DeviceID)
	assert.Equal(t, EventProfileCreated, event.EventType)
	assert.Equal(t, "value", event.Payload["key"])
	assert.False(t, event.Timestamp.IsZero())
}

func TestEventType_Category(t *testing.T) {
	tests := []struct {
		et       EventType
		category string
	}{
		{EventProfileCreated, "profile"},
		{EventProfileUpdated, "profile"},
		{EventActionCreated, "action"},
		{EventActionExecuted, "action"},
		{EventWorkspaceAccessed, "workspace"},
		{EventDeviceLinked, "device"},
		{EventDeviceUnlinked, "device"},
		{EventConflictDetected, "conflict"},
		{EventConflictResolved, "conflict"},
	}

	for _, tt := range tests {
		t.Run(string(tt.et), func(t *testing.T) {
			assert.Equal(t, tt.category, tt.et.Category())
		})
	}
}

// ============================================================================
// EventStore Tests
// ============================================================================

func TestEventStore_StoreAndGetByID(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")
	event := NewEvent("ws-1", "dev-1", EventProfileCreated, map[string]interface{}{"name": "test"})

	err := store.Store(event)
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.Version)

	loaded, err := store.GetByID(event.ID)
	require.NoError(t, err)
	assert.Equal(t, event.ID, loaded.ID)
	assert.Equal(t, int64(1), loaded.Version)
}

func TestEventStore_GetByID_NotFound(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")
	_, err := store.GetByID("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event not found")
}

func TestEventStore_GetRange(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")

	for i := 0; i < 5; i++ {
		event := NewEvent("ws-1", "dev-1", EventProfileUpdated, nil)
		require.NoError(t, store.Store(event))
	}

	events, err := store.GetRange(2, 4)
	require.NoError(t, err)
	assert.Len(t, events, 3)
	assert.Equal(t, int64(2), events[0].Version)
	assert.Equal(t, int64(4), events[2].Version)
}

func TestEventStore_GetSince(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")
	pastTime := time.Now().Add(-1 * time.Second)

	event := NewEvent("ws-1", "dev-1", EventProfileCreated, nil)
	require.NoError(t, store.Store(event))

	events, err := store.GetSince(pastTime)
	require.NoError(t, err)
	assert.Len(t, events, 1)

	futureTime := time.Now().Add(1 * time.Hour)
	events, err = store.GetSince(futureTime)
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestEventStore_GetLatestVersion(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")

	version, err := store.GetLatestVersion()
	require.NoError(t, err)
	assert.Equal(t, int64(0), version)

	for i := 0; i < 3; i++ {
		require.NoError(t, store.Store(NewEvent("ws-1", "dev-1", EventProfileUpdated, nil)))
	}

	version, err = store.GetLatestVersion()
	require.NoError(t, err)
	assert.Equal(t, int64(3), version)
}

func TestEventStore_Count(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")

	count, err := store.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	for i := 0; i < 4; i++ {
		require.NoError(t, store.Store(NewEvent("ws-1", "dev-1", EventProfileUpdated, nil)))
	}

	count, err = store.Count()
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestEventStore_QueryByResource(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")

	// Event with explicit resource key.
	e1 := NewEvent("ws-1", "dev-1", EventProfileUpdated, map[string]interface{}{
		"resource": "profile:test-profile",
	})
	require.NoError(t, store.Store(e1))

	// Event with id-based resource.
	e2 := NewEvent("ws-1", "dev-2", EventActionCreated, map[string]interface{}{
		"id": "my-action",
	})
	require.NoError(t, store.Store(e2))

	// Query for the explicit resource.
	results, err := store.QueryByResource("ws-1", "profile:test-profile")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, e1.ID, results[0].ID)

	// Query for the id-based resource.
	results, err = store.QueryByResource("ws-1", "action:my-action")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, e2.ID, results[0].ID)
}

// ============================================================================
// Presence Tests
// ============================================================================

func TestPresenceTracker_RecordHeartbeat(t *testing.T) {
	tracker := NewPresenceTracker(DefaultPresenceConfig())

	err := tracker.RecordHeartbeat("dev-1", "ws-1")
	require.NoError(t, err)

	p, err := tracker.GetPresence("dev-1", "ws-1")
	require.NoError(t, err)
	assert.Equal(t, PresenceOnline, p.State)
	assert.Equal(t, "dev-1", p.DeviceID)
	assert.Equal(t, "ws-1", p.WorkspaceID)
}

func TestPresenceTracker_RecordHeartbeat_EmptyIDs(t *testing.T) {
	tracker := NewPresenceTracker(DefaultPresenceConfig())

	err := tracker.RecordHeartbeat("", "ws-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device ID is required")

	err = tracker.RecordHeartbeat("dev-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace ID is required")
}

func TestPresenceTracker_GetPresence_NotFound(t *testing.T) {
	tracker := NewPresenceTracker(DefaultPresenceConfig())

	_, err := tracker.GetPresence("dev-1", "ws-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no presence record")
}

func TestPresenceTracker_ListPresence(t *testing.T) {
	tracker := NewPresenceTracker(DefaultPresenceConfig())

	require.NoError(t, tracker.RecordHeartbeat("dev-1", "ws-1"))
	require.NoError(t, tracker.RecordHeartbeat("dev-2", "ws-1"))
	require.NoError(t, tracker.RecordHeartbeat("dev-3", "ws-2"))

	presences, err := tracker.ListPresence("ws-1")
	require.NoError(t, err)
	assert.Len(t, presences, 2)

	presences, err = tracker.ListPresence("ws-2")
	require.NoError(t, err)
	assert.Len(t, presences, 1)
}

func TestPresenceTracker_IsOnline(t *testing.T) {
	tracker := NewPresenceTracker(DefaultPresenceConfig())

	assert.False(t, tracker.IsOnline("dev-1", "ws-1"))

	require.NoError(t, tracker.RecordHeartbeat("dev-1", "ws-1"))
	assert.True(t, tracker.IsOnline("dev-1", "ws-1"))
}

func TestPresenceTracker_CheckTimeouts(t *testing.T) {
	cfg := PresenceConfig{
		HeartbeatInterval: 1 * time.Millisecond,
		AwayTimeout:       5 * time.Millisecond,
		OfflineTimeout:    10 * time.Millisecond,
	}
	tracker := NewPresenceTracker(cfg)

	require.NoError(t, tracker.RecordHeartbeat("dev-1", "ws-1"))

	// Wait long enough for the away timeout.
	time.Sleep(8 * time.Millisecond)
	transitions := tracker.CheckTimeouts()
	// Should transition to away.
	found := false
	for _, tr := range transitions {
		if tr.DeviceID == "dev-1" {
			found = true
			assert.Equal(t, PresenceOnline, tr.From)
			assert.Equal(t, PresenceAway, tr.To)
		}
	}
	assert.True(t, found, "expected a transition for dev-1")

	// Wait for offline timeout.
	time.Sleep(10 * time.Millisecond)
	transitions = tracker.CheckTimeouts()
	found = false
	for _, tr := range transitions {
		if tr.DeviceID == "dev-1" {
			found = true
			assert.Equal(t, PresenceAway, tr.From)
			assert.Equal(t, PresenceOffline, tr.To)
		}
	}
	assert.True(t, found, "expected offline transition for dev-1")
}

func TestPresenceTracker_TransitionState(t *testing.T) {
	tracker := NewPresenceTracker(DefaultPresenceConfig())

	// Transitioning a non-existing device creates a new record.
	tr, err := tracker.TransitionState("dev-1", "ws-1", PresenceOnline)
	require.NoError(t, err)
	assert.Equal(t, PresenceOffline, tr.From)
	assert.Equal(t, PresenceOnline, tr.To)

	// Transition to away.
	tr, err = tracker.TransitionState("dev-1", "ws-1", PresenceAway)
	require.NoError(t, err)
	assert.Equal(t, PresenceOnline, tr.From)
	assert.Equal(t, PresenceAway, tr.To)

	// Same state should error.
	_, err = tracker.TransitionState("dev-1", "ws-1", PresenceAway)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already in state")
}

func TestPresenceTracker_RecordHeartbeatRestoresOnline(t *testing.T) {
	tracker := NewPresenceTracker(DefaultPresenceConfig())

	// Start online, manually transition to away.
	require.NoError(t, tracker.RecordHeartbeat("dev-1", "ws-1"))
	_, err := tracker.TransitionState("dev-1", "ws-1", PresenceAway)
	require.NoError(t, err)

	// Heartbeat should bring it back online.
	require.NoError(t, tracker.RecordHeartbeat("dev-1", "ws-1"))
	p, err := tracker.GetPresence("dev-1", "ws-1")
	require.NoError(t, err)
	assert.Equal(t, PresenceOnline, p.State)
}

func TestPresenceTracker_RemovePresence(t *testing.T) {
	tracker := NewPresenceTracker(DefaultPresenceConfig())
	require.NoError(t, tracker.RecordHeartbeat("dev-1", "ws-1"))

	tracker.RemovePresence("dev-1", "ws-1")

	_, err := tracker.GetPresence("dev-1", "ws-1")
	require.Error(t, err)
}

// ============================================================================
// Access Control Tests
// ============================================================================

func TestAccessController_OwnerFullAccess(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	ac := NewAccessController(linker)
	for _, action := range []string{"read", "write", "execute", "manage", "sync"} {
		err := ac.EvaluatePermission("ws-1", "dev-1", action, "")
		assert.NoError(t, err, "owner should have %s permission", action)
	}
}

func TestAccessController_ViewerReadOnly(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleViewer)
	require.NoError(t, err)

	ac := NewAccessController(linker)

	err = ac.EvaluatePermission("ws-1", "dev-1", "read", "")
	assert.NoError(t, err)

	err = ac.EvaluatePermission("ws-1", "dev-1", "write", "")
	assert.Error(t, err)
	assert.True(t, IsAccessDenied(err))
}

func TestAccessController_DeviceNotLinked(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	ac := NewAccessController(linker)

	err := ac.EvaluatePermission("ws-1", "dev-1", "read", "")
	assert.Error(t, err)
	assert.True(t, IsAccessDenied(err))

	ade := err.(*AccessDeniedError)
	assert.Equal(t, "link_existence", ade.Step)
}

func TestAccessController_UnlinkedStatus(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	// Manually set the status to unlinked.
	link, err := LoadLink("ws-1", "dev-1")
	require.NoError(t, err)
	link.Status = PresenceUnlinked
	require.NoError(t, SaveLink(link))

	ac := NewAccessController(linker)
	err = ac.EvaluatePermission("ws-1", "dev-1", "read", "")
	require.Error(t, err)
	assert.True(t, IsAccessDenied(err))

	ade := err.(*AccessDeniedError)
	assert.Equal(t, "link_status", ade.Step)
}

func TestAccessController_DeniedActions(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	// Add "write" to denied actions list.
	link, err := LoadLink("ws-1", "dev-1")
	require.NoError(t, err)
	link.Permissions.DeniedActions = []string{"write"}
	require.NoError(t, SaveLink(link))

	ac := NewAccessController(linker)

	err = ac.EvaluatePermission("ws-1", "dev-1", "read", "")
	assert.NoError(t, err)

	err = ac.EvaluatePermission("ws-1", "dev-1", "write", "")
	require.Error(t, err)
	ade := err.(*AccessDeniedError)
	assert.Equal(t, "denied_actions", ade.Step)
}

func TestAccessController_AllowedActions(t *testing.T) {
	setupTestHome(t)

	linker := NewLinker()
	_, err := linker.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	// Set allowed actions to only "read".
	link, err := LoadLink("ws-1", "dev-1")
	require.NoError(t, err)
	link.Permissions.AllowedActions = []string{"read"}
	require.NoError(t, SaveLink(link))

	ac := NewAccessController(linker)

	err = ac.EvaluatePermission("ws-1", "dev-1", "read", "")
	assert.NoError(t, err)

	err = ac.EvaluatePermission("ws-1", "dev-1", "write", "")
	require.Error(t, err)
	ade := err.(*AccessDeniedError)
	assert.Equal(t, "allowed_actions", ade.Step)
}

func TestIsAccessDenied(t *testing.T) {
	assert.True(t, IsAccessDenied(&AccessDeniedError{}))
	assert.False(t, IsAccessDenied(nil))
}

func TestAccessDeniedError_Error(t *testing.T) {
	ade := &AccessDeniedError{
		DeviceID:    "dev-1",
		WorkspaceID: "ws-1",
		Action:      "write",
		Resource:    "profile:main",
		Reason:      "not allowed",
		Step:        "role_permission",
		Suggestion:  "upgrade role",
	}

	msg := ade.Error()
	assert.Contains(t, msg, "access denied")
	assert.Contains(t, msg, "dev-1")
	assert.Contains(t, msg, "ws-1")
	assert.Contains(t, msg, "write")
	assert.Contains(t, msg, "profile:main")
	assert.Contains(t, msg, "not allowed")
	assert.Contains(t, msg, "upgrade role")
}

// ============================================================================
// Rate Limiter Tests
// ============================================================================

func TestRateLimiter_Check(t *testing.T) {
	rl := NewRateLimiter()

	allowed, remaining, retryAfter := rl.Check("dev-1", "ws-1", 10)
	assert.True(t, allowed)
	assert.Equal(t, 10, remaining)
	assert.Equal(t, time.Duration(0), retryAfter)
}

func TestRateLimiter_Check_ZeroLimit(t *testing.T) {
	rl := NewRateLimiter()

	allowed, _, _ := rl.Check("dev-1", "ws-1", 0)
	assert.True(t, allowed, "zero limit should always allow")
}

func TestRateLimiter_Consume(t *testing.T) {
	rl := NewRateLimiter()

	allowed, remaining, _ := rl.Consume("dev-1", "ws-1", 2)
	assert.True(t, allowed)
	assert.Equal(t, 1, remaining)

	allowed, remaining, _ = rl.Consume("dev-1", "ws-1", 2)
	assert.True(t, allowed)
	assert.Equal(t, 0, remaining)

	// Third consume should be denied (no tokens left right away).
	allowed, _, retryAfter := rl.Consume("dev-1", "ws-1", 2)
	assert.False(t, allowed)
	assert.True(t, retryAfter > 0)
}

func TestRateLimiter_Consume_ZeroLimit(t *testing.T) {
	rl := NewRateLimiter()

	allowed, _, _ := rl.Consume("dev-1", "ws-1", 0)
	assert.True(t, allowed, "zero limit should always allow")
}

// ============================================================================
// Policy Tests
// ============================================================================

func TestDefaultPolicy(t *testing.T) {
	policy := DefaultPolicy("ws-1")
	assert.Equal(t, "ws-1", policy.WorkspaceID)
	assert.Equal(t, PolicyAllowAll, policy.Mode)
}

func TestPolicy_IsDeviceAllowed_AllowAll(t *testing.T) {
	policy := &Policy{Mode: PolicyAllowAll}
	assert.True(t, policy.IsDeviceAllowed("any-device"))
}

func TestPolicy_IsDeviceAllowed_AllowList(t *testing.T) {
	policy := &Policy{
		Mode:         PolicyAllowList,
		AllowDevices: []string{"dev-1", "dev-2"},
	}
	assert.True(t, policy.IsDeviceAllowed("dev-1"))
	assert.True(t, policy.IsDeviceAllowed("dev-2"))
	assert.False(t, policy.IsDeviceAllowed("dev-3"))
}

func TestPolicy_IsDeviceAllowed_DenyList(t *testing.T) {
	policy := &Policy{
		Mode:        PolicyDenyList,
		DenyDevices: []string{"bad-device"},
	}
	assert.True(t, policy.IsDeviceAllowed("dev-1"))
	assert.False(t, policy.IsDeviceAllowed("bad-device"))
}

func TestPolicy_IsDeviceAllowed_UnknownMode(t *testing.T) {
	policy := &Policy{Mode: "mystery"}
	assert.True(t, policy.IsDeviceAllowed("dev-1"), "unknown mode defaults to allow")
}

func TestSaveAndLoadPolicy(t *testing.T) {
	setupTestHome(t)

	policy := &Policy{
		WorkspaceID:  "ws-1",
		Mode:         PolicyAllowList,
		AllowDevices: []string{"dev-1"},
	}

	err := SavePolicy("ws-1", policy)
	require.NoError(t, err)

	loaded, err := LoadPolicy("ws-1")
	require.NoError(t, err)
	assert.Equal(t, PolicyAllowList, loaded.Mode)
	assert.Contains(t, loaded.AllowDevices, "dev-1")
}

func TestLoadPolicy_Default(t *testing.T) {
	setupTestHome(t)

	// No policy file; LoadPolicy should return a default allow-all policy.
	policy, err := LoadPolicy("ws-1")
	require.NoError(t, err)
	assert.Equal(t, PolicyAllowAll, policy.Mode)
}

// ============================================================================
// Broker Tests
// ============================================================================

func TestBroker_SubscribeAndPublish(t *testing.T) {
	broker := NewBroker()
	ch := broker.Subscribe("test-channel")

	event := NewEvent("ws-1", "dev-1", EventProfileCreated, nil)
	broker.Publish("test-channel", event)

	select {
	case received := <-ch:
		assert.Equal(t, event.ID, received.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}
}

func TestBroker_Unsubscribe(t *testing.T) {
	broker := NewBroker()
	ch := broker.Subscribe("test-channel")

	assert.Equal(t, 1, broker.SubscriberCount("test-channel"))

	broker.Unsubscribe("test-channel", ch)
	assert.Equal(t, 0, broker.SubscriberCount("test-channel"))
}

func TestBroker_PublishToNonexistentChannel(t *testing.T) {
	broker := NewBroker()
	event := NewEvent("ws-1", "dev-1", EventProfileCreated, nil)
	// Should not panic.
	broker.Publish("no-such-channel", event)
}

func TestBroker_SubscriberCount(t *testing.T) {
	broker := NewBroker()
	assert.Equal(t, 0, broker.SubscriberCount("ch"))

	broker.Subscribe("ch")
	broker.Subscribe("ch")
	assert.Equal(t, 2, broker.SubscriberCount("ch"))
}

func TestBroker_MultipleSubscribers(t *testing.T) {
	broker := NewBroker()
	ch1 := broker.Subscribe("ch")
	ch2 := broker.Subscribe("ch")

	event := NewEvent("ws-1", "dev-1", EventProfileCreated, nil)
	broker.Publish("ch", event)

	select {
	case r := <-ch1:
		assert.Equal(t, event.ID, r.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch1 timed out")
	}

	select {
	case r := <-ch2:
		assert.Equal(t, event.ID, r.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch2 timed out")
	}
}

func TestChannelNameHelpers(t *testing.T) {
	assert.Equal(t, "workspace:ws-1", WorkspaceChannel("ws-1"))
	assert.Equal(t, "workspace:ws-1:device:dev-1", DeviceChannel("ws-1", "dev-1"))
	assert.Equal(t, "device:dev-1:presence", PresenceChannel("dev-1"))
}

// ============================================================================
// VectorClock (Causality) Tests
// ============================================================================

func TestVectorClock_IncrementAndGet(t *testing.T) {
	vc := NewVectorClock()
	assert.Equal(t, int64(0), vc.Get("dev-1"))

	vc.Increment("dev-1")
	assert.Equal(t, int64(1), vc.Get("dev-1"))

	vc.Increment("dev-1")
	assert.Equal(t, int64(2), vc.Get("dev-1"))
}

func TestVectorClock_Set(t *testing.T) {
	vc := NewVectorClock()
	vc.Set("dev-1", 42)
	assert.Equal(t, int64(42), vc.Get("dev-1"))
}

func TestVectorClock_Merge(t *testing.T) {
	a := NewVectorClock()
	a.Set("dev-1", 3)
	a.Set("dev-2", 1)

	b := NewVectorClock()
	b.Set("dev-1", 1)
	b.Set("dev-2", 5)
	b.Set("dev-3", 2)

	a.Merge(b)

	assert.Equal(t, int64(3), a.Get("dev-1")) // max(3,1)
	assert.Equal(t, int64(5), a.Get("dev-2")) // max(1,5)
	assert.Equal(t, int64(2), a.Get("dev-3")) // from b
}

func TestVectorClock_MergeNil(t *testing.T) {
	a := NewVectorClock()
	a.Set("dev-1", 1)
	a.Merge(nil) // should not panic
	assert.Equal(t, int64(1), a.Get("dev-1"))
}

func TestVectorClock_Copy(t *testing.T) {
	vc := NewVectorClock()
	vc.Set("dev-1", 5)
	vc.Set("dev-2", 3)

	cp := vc.Copy()
	assert.Equal(t, int64(5), cp.Get("dev-1"))
	assert.Equal(t, int64(3), cp.Get("dev-2"))

	// Mutating the copy should not affect the original.
	cp.Set("dev-1", 99)
	assert.Equal(t, int64(5), vc.Get("dev-1"))
}

func TestCompare_Equal(t *testing.T) {
	a := NewVectorClock()
	a.Set("dev-1", 2)
	a.Set("dev-2", 3)

	b := NewVectorClock()
	b.Set("dev-1", 2)
	b.Set("dev-2", 3)

	assert.Equal(t, "equal", Compare(a, b))
}

func TestCompare_Before(t *testing.T) {
	a := NewVectorClock()
	a.Set("dev-1", 1)
	a.Set("dev-2", 2)

	b := NewVectorClock()
	b.Set("dev-1", 2)
	b.Set("dev-2", 3)

	assert.Equal(t, "before", Compare(a, b))
}

func TestCompare_After(t *testing.T) {
	a := NewVectorClock()
	a.Set("dev-1", 3)
	a.Set("dev-2", 4)

	b := NewVectorClock()
	b.Set("dev-1", 2)
	b.Set("dev-2", 3)

	assert.Equal(t, "after", Compare(a, b))
}

func TestCompare_Concurrent(t *testing.T) {
	a := NewVectorClock()
	a.Set("dev-1", 3)
	a.Set("dev-2", 1)

	b := NewVectorClock()
	b.Set("dev-1", 1)
	b.Set("dev-2", 3)

	assert.Equal(t, "concurrent", Compare(a, b))
}

func TestCompare_NilClocks(t *testing.T) {
	a := NewVectorClock()
	assert.Equal(t, "concurrent", Compare(nil, a))
	assert.Equal(t, "concurrent", Compare(a, nil))
	assert.Equal(t, "concurrent", Compare(nil, nil))
}

// ============================================================================
// ConflictDetector Tests
// ============================================================================

func TestConflictDetector_NoConflictWithoutResource(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")
	detector := NewConflictDetector(store)

	event := NewEvent("ws-1", "dev-1", EventProfileUpdated, nil) // no resource in payload
	conflict, err := detector.Detect(event)
	require.NoError(t, err)
	assert.Nil(t, conflict)
}

func TestConflictDetector_NilEvent(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")
	detector := NewConflictDetector(store)

	_, err := detector.Detect(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event must not be nil")
}

func TestConflictDetector_DetectsConcurrentWrite(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")
	detector := NewConflictDetector(store)

	// Store an event from device A.
	now := time.Now()
	e1 := &WorkspaceEvent{
		ID:          "evt-1",
		WorkspaceID: "ws-1",
		DeviceID:    "dev-A",
		Timestamp:   now,
		EventType:   EventProfileUpdated,
		Payload:     map[string]interface{}{"resource": "profile:main"},
	}
	require.NoError(t, store.Store(e1))

	// Incoming event from device B on the same resource, close in time.
	e2 := &WorkspaceEvent{
		ID:          "evt-2",
		WorkspaceID: "ws-1",
		DeviceID:    "dev-B",
		Timestamp:   now.Add(1 * time.Second),
		EventType:   EventProfileUpdated,
		Version:     1, // same version as e1 will have
		Payload:     map[string]interface{}{"resource": "profile:main"},
	}

	conflict, err := detector.Detect(e2)
	require.NoError(t, err)
	require.NotNil(t, conflict)
	assert.Equal(t, ConflictConcurrentWrite, conflict.Type)
	assert.Equal(t, "profile:main", conflict.Resource)
	assert.Equal(t, ConflictPending, conflict.Status)
}

func TestConflictDetector_NoConflictSameDevice(t *testing.T) {
	setupTestHome(t)

	store := NewEventStore("ws-1")
	detector := NewConflictDetector(store)

	e1 := NewEvent("ws-1", "dev-1", EventProfileUpdated, map[string]interface{}{"resource": "profile:main"})
	require.NoError(t, store.Store(e1))

	// Same device, same resource - should not conflict.
	e2 := &WorkspaceEvent{
		ID:          "evt-2",
		WorkspaceID: "ws-1",
		DeviceID:    "dev-1",
		Timestamp:   time.Now(),
		EventType:   EventProfileUpdated,
		Version:     1,
		Payload:     map[string]interface{}{"resource": "profile:main"},
	}

	conflict, err := detector.Detect(e2)
	require.NoError(t, err)
	assert.Nil(t, conflict)
}

// ============================================================================
// LWW Resolver Tests
// ============================================================================

func TestLWWResolver_Resolve(t *testing.T) {
	resolver := NewLWWResolver()

	now := time.Now()
	conflict := &Conflict{
		ID:     "cnfl-1",
		Status: ConflictPending,
		Events: []*WorkspaceEvent{
			{ID: "evt-1", DeviceID: "dev-A", Timestamp: now.Add(-1 * time.Second), Payload: map[string]interface{}{"v": "old"}},
			{ID: "evt-2", DeviceID: "dev-B", Timestamp: now, Payload: map[string]interface{}{"v": "new"}},
		},
	}

	resolution, err := resolver.Resolve(conflict)
	require.NoError(t, err)
	assert.Equal(t, "lww", resolution.Strategy)
	assert.Equal(t, "evt-2", resolution.WinnerEvent) // latest timestamp wins
	assert.Equal(t, "auto", resolution.ResolvedBy)
	assert.Equal(t, ConflictAutoResolved, conflict.Status)
	assert.NotNil(t, conflict.ResolvedAt)
}

func TestLWWResolver_Resolve_Tiebreaker(t *testing.T) {
	resolver := NewLWWResolver()

	now := time.Now()
	conflict := &Conflict{
		ID:     "cnfl-2",
		Status: ConflictPending,
		Events: []*WorkspaceEvent{
			{ID: "evt-1", DeviceID: "dev-A", Timestamp: now},
			{ID: "evt-2", DeviceID: "dev-B", Timestamp: now}, // same timestamp
		},
	}

	resolution, err := resolver.Resolve(conflict)
	require.NoError(t, err)
	// dev-B > dev-A lexicographically, so evt-2 should win.
	assert.Equal(t, "evt-2", resolution.WinnerEvent)
}

func TestLWWResolver_Resolve_NilConflict(t *testing.T) {
	resolver := NewLWWResolver()
	_, err := resolver.Resolve(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflict must not be nil")
}

func TestLWWResolver_Resolve_NoEvents(t *testing.T) {
	resolver := NewLWWResolver()
	conflict := &Conflict{ID: "cnfl-3", Events: []*WorkspaceEvent{}}
	_, err := resolver.Resolve(conflict)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no events to resolve")
}

// ============================================================================
// OfflineQueue Tests
// ============================================================================

func TestOfflineQueue_EnqueueDequeue(t *testing.T) {
	setupTestHome(t)

	queue := NewOfflineQueue("dev-1", "ws-1")

	e1 := NewEvent("ws-1", "dev-1", EventProfileUpdated, map[string]interface{}{"x": 1})
	e2 := NewEvent("ws-1", "dev-1", EventActionExecuted, map[string]interface{}{"x": 2})

	require.NoError(t, queue.Enqueue(e1))
	require.NoError(t, queue.Enqueue(e2))

	events, err := queue.Dequeue()
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, e1.ID, events[0].ID)
	assert.Equal(t, e2.ID, events[1].ID)

	// After dequeue, queue should be empty.
	events, err = queue.Dequeue()
	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestOfflineQueue_Enqueue_NilEvent(t *testing.T) {
	setupTestHome(t)

	queue := NewOfflineQueue("dev-1", "ws-1")
	err := queue.Enqueue(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event must not be nil")
}

func TestOfflineQueue_Peek(t *testing.T) {
	setupTestHome(t)

	queue := NewOfflineQueue("dev-1", "ws-1")
	event := NewEvent("ws-1", "dev-1", EventProfileCreated, nil)
	require.NoError(t, queue.Enqueue(event))

	events, err := queue.Peek()
	require.NoError(t, err)
	assert.Len(t, events, 1)

	// Peek should not remove events.
	events, err = queue.Peek()
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestOfflineQueue_Size(t *testing.T) {
	setupTestHome(t)

	queue := NewOfflineQueue("dev-1", "ws-1")

	size, err := queue.Size()
	require.NoError(t, err)
	assert.Equal(t, 0, size)

	require.NoError(t, queue.Enqueue(NewEvent("ws-1", "dev-1", EventProfileCreated, nil)))
	require.NoError(t, queue.Enqueue(NewEvent("ws-1", "dev-1", EventProfileUpdated, nil)))

	size, err = queue.Size()
	require.NoError(t, err)
	assert.Equal(t, 2, size)
}

func TestOfflineQueue_Clear(t *testing.T) {
	setupTestHome(t)

	queue := NewOfflineQueue("dev-1", "ws-1")
	require.NoError(t, queue.Enqueue(NewEvent("ws-1", "dev-1", EventProfileCreated, nil)))

	err := queue.Clear()
	require.NoError(t, err)

	size, err := queue.Size()
	require.NoError(t, err)
	assert.Equal(t, 0, size)
}

func TestOfflineQueue_Clear_NoFile(t *testing.T) {
	setupTestHome(t)

	queue := NewOfflineQueue("dev-1", "ws-1")
	err := queue.Clear()
	require.NoError(t, err)
}

// ============================================================================
// ConflictStore Tests
// ============================================================================

func TestConflictStore_SaveAndLoad(t *testing.T) {
	setupTestHome(t)

	store := NewConflictStore("ws-1")

	conflict := &Conflict{
		ID:          "cnfl-test-1",
		WorkspaceID: "ws-1",
		Type:        ConflictConcurrentWrite,
		Status:      ConflictPending,
		Resource:    "profile:main",
		DetectedAt:  time.Now(),
	}

	err := store.Save(conflict)
	require.NoError(t, err)

	loaded, err := store.Load("cnfl-test-1")
	require.NoError(t, err)
	assert.Equal(t, "cnfl-test-1", loaded.ID)
	assert.Equal(t, ConflictConcurrentWrite, loaded.Type)
	assert.Equal(t, ConflictPending, loaded.Status)
}

func TestConflictStore_Save_NilConflict(t *testing.T) {
	setupTestHome(t)

	store := NewConflictStore("ws-1")
	err := store.Save(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflict must not be nil")
}

func TestConflictStore_Load_NotFound(t *testing.T) {
	setupTestHome(t)

	store := NewConflictStore("ws-1")
	_, err := store.Load("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConflictStore_List(t *testing.T) {
	setupTestHome(t)

	store := NewConflictStore("ws-1")

	// Save pending and resolved conflicts.
	c1 := &Conflict{ID: "cnfl-1", WorkspaceID: "ws-1", Status: ConflictPending, DetectedAt: time.Now()}
	c2 := &Conflict{ID: "cnfl-2", WorkspaceID: "ws-1", Status: ConflictAutoResolved, DetectedAt: time.Now()}
	c3 := &Conflict{ID: "cnfl-3", WorkspaceID: "ws-1", Status: ConflictManual, DetectedAt: time.Now()}

	require.NoError(t, store.Save(c1))
	require.NoError(t, store.Save(c2))
	require.NoError(t, store.Save(c3))

	// Without resolved.
	conflicts, err := store.List(false)
	require.NoError(t, err)
	assert.Len(t, conflicts, 2) // pending + manual

	// With resolved.
	conflicts, err = store.List(true)
	require.NoError(t, err)
	assert.Len(t, conflicts, 3)
}

func TestConflictStore_List_NoDirectory(t *testing.T) {
	setupTestHome(t)

	store := NewConflictStore("ws-nonexistent")
	conflicts, err := store.List(true)
	require.NoError(t, err)
	assert.Nil(t, conflicts)
}

func TestConflictStore_Delete(t *testing.T) {
	setupTestHome(t)

	store := NewConflictStore("ws-1")

	conflict := &Conflict{ID: "cnfl-del", WorkspaceID: "ws-1", Status: ConflictPending, DetectedAt: time.Now()}
	require.NoError(t, store.Save(conflict))

	err := store.Delete("cnfl-del")
	require.NoError(t, err)

	_, err = store.Load("cnfl-del")
	require.Error(t, err)
}

func TestConflictStore_Delete_NotFound(t *testing.T) {
	setupTestHome(t)

	store := NewConflictStore("ws-1")
	err := store.Delete("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ============================================================================
// ResolutionManager Tests
// ============================================================================

func TestResolutionManager_ResolveConcurrentWrite(t *testing.T) {
	setupTestHome(t)

	rm := NewResolutionManager("ws-1")
	now := time.Now()

	conflict := &Conflict{
		ID:          "cnfl-rw",
		WorkspaceID: "ws-1",
		Type:        ConflictConcurrentWrite,
		Status:      ConflictPending,
		Events: []*WorkspaceEvent{
			{ID: "e1", DeviceID: "dev-A", Timestamp: now.Add(-1 * time.Second), Payload: map[string]interface{}{"v": "old"}},
			{ID: "e2", DeviceID: "dev-B", Timestamp: now, Payload: map[string]interface{}{"v": "new"}},
		},
		DetectedAt: now,
	}

	err := rm.ResolveConflict(conflict)
	require.NoError(t, err)
	assert.Equal(t, ConflictAutoResolved, conflict.Status)
	assert.Equal(t, "lww", conflict.Resolution.Strategy)
	assert.Equal(t, "e2", conflict.Resolution.WinnerEvent)
}

func TestResolutionManager_ResolveSemanticIsManual(t *testing.T) {
	setupTestHome(t)

	rm := NewResolutionManager("ws-1")
	now := time.Now()

	conflict := &Conflict{
		ID:          "cnfl-sem",
		WorkspaceID: "ws-1",
		Type:        ConflictSemantic,
		Status:      ConflictPending,
		Events: []*WorkspaceEvent{
			{ID: "e1", DeviceID: "dev-A", Timestamp: now},
		},
		DetectedAt: now,
	}

	err := rm.ResolveConflict(conflict)
	require.NoError(t, err)
	assert.Equal(t, ConflictManual, conflict.Status)
}

func TestResolutionManager_ResolveConflict_NilConflict(t *testing.T) {
	setupTestHome(t)

	rm := NewResolutionManager("ws-1")
	err := rm.ResolveConflict(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflict must not be nil")
}

func TestResolutionManager_ResolveManually(t *testing.T) {
	setupTestHome(t)

	rm := NewResolutionManager("ws-1")

	// First save a pending conflict.
	conflict := &Conflict{
		ID:          "cnfl-manual",
		WorkspaceID: "ws-1",
		Type:        ConflictSemantic,
		Status:      ConflictManual,
		Events: []*WorkspaceEvent{
			{ID: "e1", DeviceID: "dev-A", Timestamp: time.Now()},
		},
		DetectedAt: time.Now(),
	}
	require.NoError(t, rm.store.Save(conflict))

	err := rm.ResolveManually("cnfl-manual", "manual", map[string]interface{}{"chosen": "e1"})
	require.NoError(t, err)

	loaded, err := rm.GetConflict("cnfl-manual")
	require.NoError(t, err)
	assert.Equal(t, ConflictResolved, loaded.Status)
	assert.Equal(t, "manual", loaded.Resolution.Strategy)
}

func TestResolutionManager_ResolveManually_AlreadyResolved(t *testing.T) {
	setupTestHome(t)

	rm := NewResolutionManager("ws-1")

	now := time.Now()
	conflict := &Conflict{
		ID:          "cnfl-done",
		WorkspaceID: "ws-1",
		Status:      ConflictResolved,
		ResolvedAt:  &now,
		DetectedAt:  now,
	}
	require.NoError(t, rm.store.Save(conflict))

	err := rm.ResolveManually("cnfl-done", "manual", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already resolved")
}

func TestResolutionManager_ListConflicts(t *testing.T) {
	setupTestHome(t)

	rm := NewResolutionManager("ws-1")

	c1 := &Conflict{ID: "c1", WorkspaceID: "ws-1", Status: ConflictPending, DetectedAt: time.Now()}
	c2 := &Conflict{ID: "c2", WorkspaceID: "ws-1", Status: ConflictResolved, DetectedAt: time.Now()}
	require.NoError(t, rm.store.Save(c1))
	require.NoError(t, rm.store.Save(c2))

	conflicts, err := rm.ListConflicts("ws-1", false)
	require.NoError(t, err)
	assert.Len(t, conflicts, 1)

	conflicts, err = rm.ListConflicts("ws-1", true)
	require.NoError(t, err)
	assert.Len(t, conflicts, 2)
}

// ============================================================================
// Publisher Tests
// ============================================================================

func TestPublisher_PublishEvent(t *testing.T) {
	setupTestHome(t)

	pub := NewPublisher("ws-1")

	// Subscribe to the workspace channel via the publisher's broker.
	ch := pub.Broker().Subscribe(WorkspaceChannel("ws-1"))

	event := NewEvent("ws-1", "dev-1", EventProfileCreated, map[string]interface{}{"name": "test"})
	err := pub.PublishEvent(event)
	require.NoError(t, err)

	select {
	case received := <-ch:
		assert.Equal(t, event.ID, received.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for published event")
	}

	// Event should also be in the store.
	loaded, err := pub.Store().GetByID(event.ID)
	require.NoError(t, err)
	assert.Equal(t, event.ID, loaded.ID)
}

func TestPublisher_PublishEventBroadcastsToDeviceChannel(t *testing.T) {
	setupTestHome(t)

	pub := NewPublisher("ws-1")
	deviceCh := pub.Broker().Subscribe(DeviceChannel("ws-1", "dev-1"))

	event := NewEvent("ws-1", "dev-1", EventProfileCreated, nil)
	require.NoError(t, pub.PublishEvent(event))

	select {
	case received := <-deviceCh:
		assert.Equal(t, event.ID, received.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for device channel event")
	}
}

func TestPublisher_PublishProfileUpdate(t *testing.T) {
	setupTestHome(t)

	pub := NewPublisher("ws-1")
	err := pub.PublishProfileUpdate("ws-1", "prof-1", map[string]interface{}{"display_name": "Test"})
	require.NoError(t, err)

	count, err := pub.Store().Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPublisher_PublishActionExecution(t *testing.T) {
	setupTestHome(t)

	pub := NewPublisher("ws-1")
	err := pub.PublishActionExecution("ws-1", "my-action", map[string]interface{}{"status": "success"})
	require.NoError(t, err)

	count, err := pub.Store().Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestNewPublisherWithBroker(t *testing.T) {
	setupTestHome(t)

	broker := NewBroker()
	pub := NewPublisherWithBroker("ws-1", broker)
	assert.Equal(t, broker, pub.Broker())
}

// ============================================================================
// AuditLogger Tests
// ============================================================================

func TestAuditLogger_LogAndListEntries(t *testing.T) {
	setupTestHome(t)

	logger := NewAuditLogger("ws-1")

	entry1 := &AuditEntry{
		DeviceID:    "dev-1",
		WorkspaceID: "ws-1",
		Action:      "read",
		Result:      "allow",
	}
	entry2 := &AuditEntry{
		DeviceID:    "dev-2",
		WorkspaceID: "ws-1",
		Action:      "write",
		Result:      "deny",
		Reason:      "no permission",
	}

	require.NoError(t, logger.Log(entry1))
	require.NoError(t, logger.Log(entry2))

	// List all entries.
	entries, err := logger.ListEntries(time.Time{}, "", "", "", 0)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Filter by device.
	entries, err = logger.ListEntries(time.Time{}, "dev-1", "", "", 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "dev-1", entries[0].DeviceID)

	// Filter by result.
	entries, err = logger.ListEntries(time.Time{}, "", "deny", "", 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "deny", entries[0].Result)

	// Filter by action.
	entries, err = logger.ListEntries(time.Time{}, "", "", "read", 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestAuditLogger_ListEntries_Limit(t *testing.T) {
	setupTestHome(t)

	logger := NewAuditLogger("ws-1")

	for i := 0; i < 10; i++ {
		require.NoError(t, logger.Log(&AuditEntry{
			DeviceID: "dev-1",
			Action:   "read",
			Result:   "allow",
		}))
	}

	entries, err := logger.ListEntries(time.Time{}, "", "", "", 3)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestAuditLogger_ListEntries_NoFile(t *testing.T) {
	setupTestHome(t)

	logger := NewAuditLogger("ws-1")
	entries, err := logger.ListEntries(time.Time{}, "", "", "", 0)
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestAuditLogger_Query(t *testing.T) {
	setupTestHome(t)

	logger := NewAuditLogger("ws-1")
	now := time.Now()

	require.NoError(t, logger.Log(&AuditEntry{
		Timestamp: now.Add(-2 * time.Hour),
		DeviceID:  "dev-1",
		Action:    "read",
		Result:    "allow",
	}))
	require.NoError(t, logger.Log(&AuditEntry{
		Timestamp: now,
		DeviceID:  "dev-2",
		Action:    "write",
		Result:    "deny",
	}))

	// Query by device.
	results, err := logger.Query(AuditFilters{DeviceID: "dev-2"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "dev-2", results[0].DeviceID)

	// Query by result.
	results, err = logger.Query(AuditFilters{Result: "allow"})
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Query by action.
	results, err = logger.Query(AuditFilters{Action: "write"})
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Query with time range.
	results, err = logger.Query(AuditFilters{Since: now.Add(-1 * time.Hour)})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "dev-2", results[0].DeviceID)

	// Query with limit.
	results, err = logger.Query(AuditFilters{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestAuditLogger_Query_NoFile(t *testing.T) {
	setupTestHome(t)

	logger := NewAuditLogger("ws-empty")
	results, err := logger.Query(AuditFilters{})
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestMatchesAuditFilters(t *testing.T) {
	now := time.Now()

	entry := &AuditEntry{
		Timestamp: now,
		DeviceID:  "dev-1",
		Action:    "read",
		Result:    "allow",
	}

	// All empty filters should match.
	assert.True(t, matchesAuditFilters(entry, &AuditFilters{}))

	// Device filter.
	assert.True(t, matchesAuditFilters(entry, &AuditFilters{DeviceID: "dev-1"}))
	assert.False(t, matchesAuditFilters(entry, &AuditFilters{DeviceID: "dev-2"}))

	// Action filter.
	assert.True(t, matchesAuditFilters(entry, &AuditFilters{Action: "read"}))
	assert.False(t, matchesAuditFilters(entry, &AuditFilters{Action: "write"}))

	// Result filter.
	assert.True(t, matchesAuditFilters(entry, &AuditFilters{Result: "allow"}))
	assert.False(t, matchesAuditFilters(entry, &AuditFilters{Result: "deny"}))

	// Time filters.
	assert.True(t, matchesAuditFilters(entry, &AuditFilters{Since: now.Add(-1 * time.Hour)}))
	assert.False(t, matchesAuditFilters(entry, &AuditFilters{Since: now.Add(1 * time.Hour)}))

	assert.True(t, matchesAuditFilters(entry, &AuditFilters{Until: now.Add(1 * time.Hour)}))
	assert.False(t, matchesAuditFilters(entry, &AuditFilters{Until: now.Add(-1 * time.Hour)}))
}

// ============================================================================
// SyncManager Tests
// ============================================================================

func TestSyncManager_InitiateSync_AlreadyUpToDate(t *testing.T) {
	setupTestHome(t)

	sm := NewSyncManager("ws-1")

	result, err := sm.InitiateSync("dev-1", "ws-1", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.LatestVersion)
	assert.Equal(t, 0, result.EventsSynced)
}

func TestSyncManager_InitiateSync_WithMissedEvents(t *testing.T) {
	setupTestHome(t)

	sm := NewSyncManager("ws-1")

	// Store some events.
	for i := 0; i < 5; i++ {
		require.NoError(t, sm.store.Store(NewEvent("ws-1", "dev-1", EventProfileUpdated, nil)))
	}

	result, err := sm.InitiateSync("dev-2", "ws-1", 2)
	require.NoError(t, err)
	assert.Equal(t, 3, result.EventsSynced) // events 3, 4, 5
	assert.Equal(t, int64(5), result.LatestVersion)
}

func TestSyncManager_GetMissedEvents(t *testing.T) {
	setupTestHome(t)

	sm := NewSyncManager("ws-1")

	for i := 0; i < 3; i++ {
		require.NoError(t, sm.store.Store(NewEvent("ws-1", "dev-1", EventProfileUpdated, nil)))
	}

	events, err := sm.GetMissedEvents("ws-1", 1)
	require.NoError(t, err)
	assert.Len(t, events, 2) // versions 2 and 3

	// Already up to date.
	events, err = sm.GetMissedEvents("ws-1", 3)
	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestSyncManager_ProcessOfflineEvents(t *testing.T) {
	setupTestHome(t)

	sm := NewSyncManager("ws-1")

	events := []*WorkspaceEvent{
		NewEvent("ws-1", "dev-2", EventProfileUpdated, map[string]interface{}{"name": "offline-1"}),
		NewEvent("ws-1", "dev-2", EventProfileUpdated, map[string]interface{}{"name": "offline-2"}),
	}

	result, err := sm.ProcessOfflineEvents("dev-2", "ws-1", events)
	require.NoError(t, err)
	assert.Equal(t, 2, result.EventsSynced)
	assert.Equal(t, 2, result.EventsProcessed)
	assert.Equal(t, int64(2), result.LatestVersion)
}

// ============================================================================
// Manager Integration Tests
// ============================================================================

func TestManager_NewManager(t *testing.T) {
	m := NewManager()
	assert.NotNil(t, m)
}

func TestManager_AttachAndDetachDevice(t *testing.T) {
	setupTestHome(t)

	m := NewManager()

	link, err := m.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)
	assert.Equal(t, "ws-1", link.WorkspaceID)
	assert.Equal(t, "dev-1", link.DeviceID)
	assert.Equal(t, RoleOwner, link.Permissions.Role)

	err = m.DetachDevice("ws-1", "dev-1")
	require.NoError(t, err)

	_, err = m.GetDeviceLink("ws-1", "dev-1")
	require.Error(t, err)
}

func TestManager_ListDeviceLinks(t *testing.T) {
	setupTestHome(t)

	m := NewManager()
	_, err := m.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)
	_, err = m.AttachDevice("ws-1", "dev-2", RoleViewer)
	require.NoError(t, err)

	links, err := m.ListDeviceLinks("ws-1")
	require.NoError(t, err)
	assert.Len(t, links, 2)
}

func TestManager_SetRole(t *testing.T) {
	setupTestHome(t)

	m := NewManager()
	_, err := m.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	err = m.SetRole("ws-1", "dev-1", RoleViewer)
	require.NoError(t, err)

	link, err := m.GetDeviceLink("ws-1", "dev-1")
	require.NoError(t, err)
	assert.Equal(t, RoleViewer, link.Permissions.Role)
}

func TestManager_UpdatePermissions(t *testing.T) {
	setupTestHome(t)

	m := NewManager()
	_, err := m.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	perms := PermissionsForRole(RoleCollaborator)
	perms.RateLimitPerMin = 50
	err = m.UpdatePermissions("ws-1", "dev-1", perms)
	require.NoError(t, err)

	link, err := m.GetDeviceLink("ws-1", "dev-1")
	require.NoError(t, err)
	assert.Equal(t, 50, link.Permissions.RateLimitPerMin)
}

func TestManager_CheckAccess(t *testing.T) {
	setupTestHome(t)

	m := NewManager()
	_, err := m.AttachDevice("ws-1", "dev-1", RoleViewer)
	require.NoError(t, err)

	// Viewer can read.
	err = m.CheckAccess("ws-1", "dev-1", "read", "")
	assert.NoError(t, err)

	// Viewer cannot write.
	err = m.CheckAccess("ws-1", "dev-1", "write", "")
	assert.Error(t, err)
}

func TestManager_RecordHeartbeat(t *testing.T) {
	setupTestHome(t)

	m := NewManager()
	_, err := m.AttachDevice("ws-1", "dev-1", RoleOwner)
	require.NoError(t, err)

	err = m.RecordHeartbeat("dev-1", "ws-1")
	require.NoError(t, err)
}

func TestManager_GetPublisher(t *testing.T) {
	m := NewManager()

	pub1 := m.GetPublisher("ws-1")
	pub2 := m.GetPublisher("ws-1")
	assert.Same(t, pub1, pub2, "same workspace should return the same publisher")

	pub3 := m.GetPublisher("ws-2")
	assert.NotSame(t, pub1, pub3, "different workspace should return different publisher")
}

func TestManager_GetEventStore(t *testing.T) {
	m := NewManager()

	s1 := m.GetEventStore("ws-1")
	s2 := m.GetEventStore("ws-1")
	assert.Same(t, s1, s2)
}

func TestManager_GetSyncManager(t *testing.T) {
	m := NewManager()

	sm1 := m.GetSyncManager("ws-1")
	sm2 := m.GetSyncManager("ws-1")
	assert.Same(t, sm1, sm2)
}

func TestManager_GetResolutionManager(t *testing.T) {
	m := NewManager()

	rm1 := m.GetResolutionManager("ws-1")
	rm2 := m.GetResolutionManager("ws-1")
	assert.Same(t, rm1, rm2)
}

func TestManager_GetAuditLogger(t *testing.T) {
	m := NewManager()

	al1 := m.GetAuditLogger("ws-1")
	al2 := m.GetAuditLogger("ws-1")
	assert.Same(t, al1, al2)
}

func TestManager_GetEvents(t *testing.T) {
	setupTestHome(t)

	m := NewManager()
	pub := m.GetPublisher("ws-1")

	past := time.Now().Add(-1 * time.Second)
	require.NoError(t, pub.PublishEvent(NewEvent("ws-1", "dev-1", EventProfileCreated, nil)))
	require.NoError(t, pub.PublishEvent(NewEvent("ws-1", "dev-1", EventProfileUpdated, nil)))

	events, err := m.GetEvents("ws-1", past, 0)
	require.NoError(t, err)
	assert.Len(t, events, 2)

	// With limit.
	events, err = m.GetEvents("ws-1", past, 1)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestManager_AccessControl_PolicyDenyList(t *testing.T) {
	setupTestHome(t)

	m := NewManager()
	_, err := m.AttachDevice("ws-1", "dev-blocked", RoleOwner)
	require.NoError(t, err)

	// Save a deny-list policy that blocks dev-blocked.
	policy := &Policy{
		WorkspaceID: "ws-1",
		Mode:        PolicyDenyList,
		DenyDevices: []string{"dev-blocked"},
	}
	require.NoError(t, SavePolicy("ws-1", policy))

	err = m.CheckAccess("ws-1", "dev-blocked", "read", "")
	require.Error(t, err)
	assert.True(t, IsAccessDenied(err))

	ade := err.(*AccessDeniedError)
	assert.Equal(t, "policy", ade.Step)
}
