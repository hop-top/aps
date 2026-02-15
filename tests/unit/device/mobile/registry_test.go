package mobile_test

import (
	"testing"
	"time"

	"oss-aps-cli/internal/core/device/mobile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRegistry(t *testing.T) *mobile.Registry {
	t.Helper()
	dir := t.TempDir()
	reg, err := mobile.NewRegistry(dir)
	require.NoError(t, err)
	return reg
}

func newTestDevice(id, profileID string) *mobile.MobileDevice {
	now := time.Now()
	return &mobile.MobileDevice{
		DeviceID:     id,
		ProfileID:    profileID,
		DeviceName:   "Test Device " + id,
		DeviceOS:     "iOS",
		RegisteredAt: now,
		LastSeenAt:   now,
		ExpiresAt:    now.Add(14 * 24 * time.Hour),
		TokenHash:    mobile.HashToken("test-token-" + id),
		Status:       mobile.PairingStateActive,
		Capabilities: []string{"run:stateless"},
	}
}

func TestRegistryRegisterDevice(t *testing.T) {
	t.Run("registers a new device", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestDevice("dev-1", "profile-a")

		err := reg.RegisterDevice(dev)
		require.NoError(t, err)

		got, err := reg.GetDevice("dev-1")
		require.NoError(t, err)
		assert.Equal(t, "dev-1", got.DeviceID)
		assert.Equal(t, "profile-a", got.ProfileID)
		assert.Equal(t, "Test Device dev-1", got.DeviceName)
	})

	t.Run("rejects duplicate device ID", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestDevice("dev-dup", "profile-a")

		err := reg.RegisterDevice(dev)
		require.NoError(t, err)

		err = reg.RegisterDevice(dev)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

func TestRegistryGetDevice(t *testing.T) {
	t.Run("returns error for missing device", func(t *testing.T) {
		reg := newTestRegistry(t)

		_, err := reg.GetDevice("nonexistent")
		assert.Error(t, err)
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeDeviceNotFound))
	})
}

func TestRegistryListDevices(t *testing.T) {
	reg := newTestRegistry(t)

	reg.RegisterDevice(newTestDevice("dev-a1", "profile-a"))
	reg.RegisterDevice(newTestDevice("dev-a2", "profile-a"))
	reg.RegisterDevice(newTestDevice("dev-b1", "profile-b"))

	t.Run("list all", func(t *testing.T) {
		devices, err := reg.ListDevices("")
		require.NoError(t, err)
		assert.Len(t, devices, 3)
	})

	t.Run("filter by profile", func(t *testing.T) {
		devices, err := reg.ListDevices("profile-a")
		require.NoError(t, err)
		assert.Len(t, devices, 2)
		for _, d := range devices {
			assert.Equal(t, "profile-a", d.ProfileID)
		}
	})

	t.Run("empty profile returns no results", func(t *testing.T) {
		devices, err := reg.ListDevices("nonexistent")
		require.NoError(t, err)
		assert.Empty(t, devices)
	})
}

func TestRegistryRevokeDevice(t *testing.T) {
	t.Run("revokes active device", func(t *testing.T) {
		reg := newTestRegistry(t)
		reg.RegisterDevice(newTestDevice("dev-revoke", "profile-a"))

		err := reg.RevokeDevice("dev-revoke")
		require.NoError(t, err)

		got, err := reg.GetDevice("dev-revoke")
		require.NoError(t, err)
		assert.Equal(t, mobile.PairingStateRevoked, got.Status)
	})

	t.Run("revoke nonexistent device returns error", func(t *testing.T) {
		reg := newTestRegistry(t)

		err := reg.RevokeDevice("nonexistent")
		assert.Error(t, err)
	})
}

func TestRegistryApproveDevice(t *testing.T) {
	t.Run("approves pending device", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestDevice("dev-approve", "profile-a")
		dev.Status = mobile.PairingStatePending
		reg.RegisterDevice(dev)

		err := reg.ApproveDevice("dev-approve")
		require.NoError(t, err)

		got, err := reg.GetDevice("dev-approve")
		require.NoError(t, err)
		assert.Equal(t, mobile.PairingStateActive, got.Status)
		assert.NotNil(t, got.ApprovedAt)
	})

	t.Run("cannot approve non-pending device", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestDevice("dev-active", "profile-a")
		dev.Status = mobile.PairingStateActive
		reg.RegisterDevice(dev)

		err := reg.ApproveDevice("dev-active")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")
	})
}

func TestRegistryRejectDevice(t *testing.T) {
	t.Run("rejects pending device", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestDevice("dev-reject", "profile-a")
		dev.Status = mobile.PairingStatePending
		reg.RegisterDevice(dev)

		err := reg.RejectDevice("dev-reject")
		require.NoError(t, err)

		// Device should be removed from registry
		_, err = reg.GetDevice("dev-reject")
		assert.Error(t, err)
	})

	t.Run("cannot reject non-pending device", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestDevice("dev-active2", "profile-a")
		dev.Status = mobile.PairingStateActive
		reg.RegisterDevice(dev)

		err := reg.RejectDevice("dev-active2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")
	})
}

func TestRegistryListPending(t *testing.T) {
	reg := newTestRegistry(t)

	dev1 := newTestDevice("dev-p1", "profile-a")
	dev1.Status = mobile.PairingStatePending
	reg.RegisterDevice(dev1)

	dev2 := newTestDevice("dev-p2", "profile-a")
	dev2.Status = mobile.PairingStatePending
	reg.RegisterDevice(dev2)

	reg.RegisterDevice(newTestDevice("dev-active3", "profile-a")) // active

	t.Run("lists pending only", func(t *testing.T) {
		pending, err := reg.ListPending("")
		require.NoError(t, err)
		assert.Len(t, pending, 2)
	})

	t.Run("filters pending by profile", func(t *testing.T) {
		pending, err := reg.ListPending("profile-a")
		require.NoError(t, err)
		assert.Len(t, pending, 2)

		pending, err = reg.ListPending("profile-b")
		require.NoError(t, err)
		assert.Empty(t, pending)
	})
}

func TestRegistryUpdateLastSeen(t *testing.T) {
	reg := newTestRegistry(t)
	dev := newTestDevice("dev-seen", "profile-a")
	dev.LastSeenAt = time.Now().Add(-1 * time.Hour)
	reg.RegisterDevice(dev)

	before, _ := reg.GetDevice("dev-seen")
	beforeSeen := before.LastSeenAt

	time.Sleep(10 * time.Millisecond)
	err := reg.UpdateLastSeen("dev-seen")
	require.NoError(t, err)

	after, _ := reg.GetDevice("dev-seen")
	assert.True(t, after.LastSeenAt.After(beforeSeen))
}

func TestRegistryCleanupExpired(t *testing.T) {
	reg := newTestRegistry(t)

	// Active, not expired
	reg.RegisterDevice(newTestDevice("dev-ok", "profile-a"))

	// Active, expired
	expired := newTestDevice("dev-expired", "profile-a")
	expired.ExpiresAt = time.Now().Add(-1 * time.Hour)
	reg.RegisterDevice(expired)

	removed, err := reg.CleanupExpired()
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	got, _ := reg.GetDevice("dev-expired")
	assert.Equal(t, mobile.PairingStateExpired, got.Status)
}

func TestRegistryCountActive(t *testing.T) {
	reg := newTestRegistry(t)

	reg.RegisterDevice(newTestDevice("dev-c1", "profile-a"))
	reg.RegisterDevice(newTestDevice("dev-c2", "profile-a"))

	revoked := newTestDevice("dev-c3", "profile-a")
	revoked.Status = mobile.PairingStateRevoked
	reg.RegisterDevice(revoked)

	reg.RegisterDevice(newTestDevice("dev-c4", "profile-b"))

	count, err := reg.CountActive("profile-a")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	count, err = reg.CountActive("profile-b")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRegistryPersistence(t *testing.T) {
	dir := t.TempDir()

	// Write with first registry
	reg1, err := mobile.NewRegistry(dir)
	require.NoError(t, err)
	reg1.RegisterDevice(newTestDevice("persist-1", "profile-a"))

	// Read with second registry (same dir)
	reg2, err := mobile.NewRegistry(dir)
	require.NoError(t, err)

	got, err := reg2.GetDevice("persist-1")
	require.NoError(t, err)
	assert.Equal(t, "persist-1", got.DeviceID)
}
