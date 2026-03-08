package mobile_test

import (
	"testing"
	"time"

	"oss-aps-cli/internal/core/adapter/mobile"

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

func newTestAdapter(id, profileID string) *mobile.MobileAdapter {
	now := time.Now()
	return &mobile.MobileAdapter{
		AdapterID:     id,
		ProfileID:    profileID,
		AdapterName:   "Test Adapter " + id,
		AdapterOS:     "iOS",
		RegisteredAt: now,
		LastSeenAt:   now,
		ExpiresAt:    now.Add(14 * 24 * time.Hour),
		TokenHash:    mobile.HashToken("test-token-" + id),
		Status:       mobile.PairingStateActive,
		Capabilities: []string{"run:stateless"},
	}
}

func TestRegistryRegisterAdapter(t *testing.T) {
	t.Run("registers a new adapter", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestAdapter("dev-1", "profile-a")

		err := reg.RegisterAdapter(dev)
		require.NoError(t, err)

		got, err := reg.GetAdapter("dev-1")
		require.NoError(t, err)
		assert.Equal(t, "dev-1", got.AdapterID)
		assert.Equal(t, "profile-a", got.ProfileID)
		assert.Equal(t, "Test Adapter dev-1", got.AdapterName)
	})

	t.Run("rejects duplicate adapter ID", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestAdapter("dev-dup", "profile-a")

		err := reg.RegisterAdapter(dev)
		require.NoError(t, err)

		err = reg.RegisterAdapter(dev)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

func TestRegistryGetAdapter(t *testing.T) {
	t.Run("returns error for missing adapter", func(t *testing.T) {
		reg := newTestRegistry(t)

		_, err := reg.GetAdapter("nonexistent")
		assert.Error(t, err)
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeAdapterNotFound))
	})
}

func TestRegistryListAdapters(t *testing.T) {
	reg := newTestRegistry(t)

	reg.RegisterAdapter(newTestAdapter("dev-a1", "profile-a"))
	reg.RegisterAdapter(newTestAdapter("dev-a2", "profile-a"))
	reg.RegisterAdapter(newTestAdapter("dev-b1", "profile-b"))

	t.Run("list all", func(t *testing.T) {
		adapters, err := reg.ListAdapters("")
		require.NoError(t, err)
		assert.Len(t, adapters, 3)
	})

	t.Run("filter by profile", func(t *testing.T) {
		adapters, err := reg.ListAdapters("profile-a")
		require.NoError(t, err)
		assert.Len(t, adapters, 2)
		for _, a := range adapters {
			assert.Equal(t, "profile-a", a.ProfileID)
		}
	})

	t.Run("empty profile returns no results", func(t *testing.T) {
		adapters, err := reg.ListAdapters("nonexistent")
		require.NoError(t, err)
		assert.Empty(t, adapters)
	})
}

func TestRegistryRevokeAdapter(t *testing.T) {
	t.Run("revokes active adapter", func(t *testing.T) {
		reg := newTestRegistry(t)
		reg.RegisterAdapter(newTestAdapter("dev-revoke", "profile-a"))

		err := reg.RevokeAdapter("dev-revoke")
		require.NoError(t, err)

		got, err := reg.GetAdapter("dev-revoke")
		require.NoError(t, err)
		assert.Equal(t, mobile.PairingStateRevoked, got.Status)
	})

	t.Run("revoke nonexistent adapter returns error", func(t *testing.T) {
		reg := newTestRegistry(t)

		err := reg.RevokeAdapter("nonexistent")
		assert.Error(t, err)
	})
}

func TestRegistryApproveAdapter(t *testing.T) {
	t.Run("approves pending adapter", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestAdapter("dev-approve", "profile-a")
		dev.Status = mobile.PairingStatePending
		reg.RegisterAdapter(dev)

		err := reg.ApproveAdapter("dev-approve")
		require.NoError(t, err)

		got, err := reg.GetAdapter("dev-approve")
		require.NoError(t, err)
		assert.Equal(t, mobile.PairingStateActive, got.Status)
		assert.NotNil(t, got.ApprovedAt)
	})

	t.Run("cannot approve non-pending adapter", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestAdapter("dev-active", "profile-a")
		dev.Status = mobile.PairingStateActive
		reg.RegisterAdapter(dev)

		err := reg.ApproveAdapter("dev-active")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")
	})
}

func TestRegistryRejectAdapter(t *testing.T) {
	t.Run("rejects pending adapter", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestAdapter("dev-reject", "profile-a")
		dev.Status = mobile.PairingStatePending
		reg.RegisterAdapter(dev)

		err := reg.RejectAdapter("dev-reject")
		require.NoError(t, err)

		// Adapter should be removed from registry
		_, err = reg.GetAdapter("dev-reject")
		assert.Error(t, err)
	})

	t.Run("cannot reject non-pending adapter", func(t *testing.T) {
		reg := newTestRegistry(t)
		dev := newTestAdapter("dev-active2", "profile-a")
		dev.Status = mobile.PairingStateActive
		reg.RegisterAdapter(dev)

		err := reg.RejectAdapter("dev-active2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")
	})
}

func TestRegistryListPending(t *testing.T) {
	reg := newTestRegistry(t)

	dev1 := newTestAdapter("dev-p1", "profile-a")
	dev1.Status = mobile.PairingStatePending
	reg.RegisterAdapter(dev1)

	dev2 := newTestAdapter("dev-p2", "profile-a")
	dev2.Status = mobile.PairingStatePending
	reg.RegisterAdapter(dev2)

	reg.RegisterAdapter(newTestAdapter("dev-active3", "profile-a")) // active

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
	dev := newTestAdapter("dev-seen", "profile-a")
	dev.LastSeenAt = time.Now().Add(-1 * time.Hour)
	reg.RegisterAdapter(dev)

	before, _ := reg.GetAdapter("dev-seen")
	beforeSeen := before.LastSeenAt

	time.Sleep(10 * time.Millisecond)
	err := reg.UpdateLastSeen("dev-seen")
	require.NoError(t, err)

	after, _ := reg.GetAdapter("dev-seen")
	assert.True(t, after.LastSeenAt.After(beforeSeen))
}

func TestRegistryCleanupExpired(t *testing.T) {
	reg := newTestRegistry(t)

	// Active, not expired
	reg.RegisterAdapter(newTestAdapter("dev-ok", "profile-a"))

	// Active, expired
	expired := newTestAdapter("dev-expired", "profile-a")
	expired.ExpiresAt = time.Now().Add(-1 * time.Hour)
	reg.RegisterAdapter(expired)

	removed, err := reg.CleanupExpired()
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	got, _ := reg.GetAdapter("dev-expired")
	assert.Equal(t, mobile.PairingStateExpired, got.Status)
}

func TestRegistryCountActive(t *testing.T) {
	reg := newTestRegistry(t)

	reg.RegisterAdapter(newTestAdapter("dev-c1", "profile-a"))
	reg.RegisterAdapter(newTestAdapter("dev-c2", "profile-a"))

	revoked := newTestAdapter("dev-c3", "profile-a")
	revoked.Status = mobile.PairingStateRevoked
	reg.RegisterAdapter(revoked)

	reg.RegisterAdapter(newTestAdapter("dev-c4", "profile-b"))

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
	reg1.RegisterAdapter(newTestAdapter("persist-1", "profile-a"))

	// Read with second registry (same dir)
	reg2, err := mobile.NewRegistry(dir)
	require.NoError(t, err)

	got, err := reg2.GetAdapter("persist-1")
	require.NoError(t, err)
	assert.Equal(t, "persist-1", got.AdapterID)
}
