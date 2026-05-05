// Tests for ContextVariable.Visibility (T-1309).
//
// Coverage:
//   - private/shared semantics on Get/List/Delete
//   - zero-value yaml round-trip reads as shared (no migration needed)
//   - WithVisibility opt-in; absent flag preserves prior visibility
//   - other-profile writes to a private var see "not found" (no leak)
package collaboration_test

import (
	"testing"
	"time"

	"hop.top/aps/internal/core/collaboration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVisibility_ZeroValueIsShared documents the zero-value contract
// that the on-disk migration story relies on: any ContextVariable
// loaded without a visibility field reads as shared.
func TestVisibility_ZeroValueIsShared(t *testing.T) {
	v := collaboration.ContextVariable{Key: "k", UpdatedBy: "alice"}

	assert.Equal(t, collaboration.VisibilityShared, v.EffectiveVisibility())
	assert.True(t, v.IsVisibleTo("alice"))
	assert.True(t, v.IsVisibleTo("bob"), "shared variables visible to every profile")
}

// TestVisibility_NormalizeHelper covers the explicit normalization
// helper used at every read seam.
func TestVisibility_NormalizeHelper(t *testing.T) {
	assert.Equal(t, collaboration.VisibilityShared, collaboration.NormalizeVisibility(""))
	assert.Equal(t, collaboration.VisibilityShared, collaboration.NormalizeVisibility(collaboration.VisibilityShared))
	assert.Equal(t, collaboration.VisibilityPrivate, collaboration.NormalizeVisibility(collaboration.VisibilityPrivate))
}

// TestVisibility_PrivateVarHiddenFromOthers exercises the headline
// guarantee: a private variable written by A is invisible to B on
// Get and List, with "not found" semantics so existence does not leak.
func TestVisibility_PrivateVarHiddenFromOthers(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()

	_, err := wc.Set("notes", "secret",
		"alice", collaboration.RoleOwner,
		collaboration.WithVisibility(collaboration.VisibilityPrivate))
	require.NoError(t, err)

	// Bob (different profile, owner role) gets nothing.
	v, ok := wc.GetForProfile("notes", "bob")
	assert.False(t, ok, "private var must be invisible to other profiles")
	assert.Nil(t, v)

	// And not even a hint via List.
	bobList := wc.ListForProfile("bob")
	assert.Empty(t, bobList, "private var must not appear in another profile's list")
}

// TestVisibility_PrivateVarVisibleToWriter is the writer-only
// counterpart: A still sees their own private var.
func TestVisibility_PrivateVarVisibleToWriter(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()

	_, err := wc.Set("notes", "secret",
		"alice", collaboration.RoleOwner,
		collaboration.WithVisibility(collaboration.VisibilityPrivate))
	require.NoError(t, err)

	v, ok := wc.GetForProfile("notes", "alice")
	require.True(t, ok)
	assert.Equal(t, "secret", v.Value)
	assert.Equal(t, collaboration.VisibilityPrivate, v.EffectiveVisibility())

	aliceList := wc.ListForProfile("alice")
	require.Len(t, aliceList, 1)
	assert.Equal(t, "notes", aliceList[0].Key)
}

// TestVisibility_SharedVarVisibleToBoth is the baseline: shared vars
// (the default) appear for every profile.
func TestVisibility_SharedVarVisibleToBoth(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()

	_, err := wc.Set("status", "ready", "alice", collaboration.RoleOwner)
	require.NoError(t, err)

	for _, who := range []string{"alice", "bob"} {
		v, ok := wc.GetForProfile("status", who)
		require.Truef(t, ok, "shared var must be visible to %q", who)
		assert.Equal(t, "ready", v.Value)
		assert.Equal(t, collaboration.VisibilityShared, v.EffectiveVisibility())

		list := wc.ListForProfile(who)
		require.Lenf(t, list, 1, "shared var must appear in %q's list", who)
	}
}

// TestVisibility_FromStateWithoutFieldReadsAsShared simulates an
// existing yaml/json record that was persisted before T-1309: the
// Visibility field is the zero value but reads as shared, no migration.
func TestVisibility_FromStateWithoutFieldReadsAsShared(t *testing.T) {
	preExisting := []collaboration.ContextVariable{
		{
			Key:       "build.status",
			Value:     "passing",
			Version:   3,
			UpdatedBy: "alice",
			UpdatedAt: time.Now(),
			// Visibility intentionally omitted — simulates pre-T-1309 state.
		},
	}
	wc := collaboration.NewWorkspaceContextFromState(preExisting, nil)

	// Visible to everyone.
	for _, who := range []string{"alice", "bob", "carol"} {
		v, ok := wc.GetForProfile("build.status", who)
		require.Truef(t, ok, "missing visibility must read as shared (caller=%q)", who)
		assert.Equal(t, collaboration.VisibilityShared, v.EffectiveVisibility())
	}
}

// TestVisibility_RawListAndGetIgnoreFilter verifies the legacy
// non-profile-aware accessors still see every variable. Storage
// round-trips and metrics rely on this.
func TestVisibility_RawListAndGetIgnoreFilter(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()
	_, err := wc.Set("private.note", "x",
		"alice", collaboration.RoleOwner,
		collaboration.WithVisibility(collaboration.VisibilityPrivate))
	require.NoError(t, err)

	v, ok := wc.Get("private.note")
	require.True(t, ok, "raw Get must surface every variable")
	assert.Equal(t, collaboration.VisibilityPrivate, v.EffectiveVisibility())

	all := wc.List()
	require.Len(t, all, 1, "raw List must surface every variable")
}

// TestVisibility_DeleteByOtherProfileLooksLikeNotFound: existence
// leak prevention extends to Delete — observers of B trying to
// remove A's private var receive the "not found" error string.
func TestVisibility_DeleteByOtherProfileLooksLikeNotFound(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()

	_, err := wc.Set("notes", "secret",
		"alice", collaboration.RoleOwner,
		collaboration.WithVisibility(collaboration.VisibilityPrivate))
	require.NoError(t, err)

	err = wc.Delete("notes", "bob", collaboration.RoleOwner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found",
		"delete by non-writer must surface as not-found, never permission-denied")

	// Variable still present for the writer.
	v, ok := wc.GetForProfile("notes", "alice")
	require.True(t, ok)
	assert.Equal(t, "secret", v.Value)
}

// TestVisibility_SetByOtherProfileLooksLikeNotFound: same
// existence-leak guard extends to Set — B can't probe by attempting
// a write either.
func TestVisibility_SetByOtherProfileLooksLikeNotFound(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()

	_, err := wc.Set("notes", "secret",
		"alice", collaboration.RoleOwner,
		collaboration.WithVisibility(collaboration.VisibilityPrivate))
	require.NoError(t, err)

	_, err = wc.Set("notes", "clobber", "bob", collaboration.RoleOwner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Original value untouched.
	v, ok := wc.GetForProfile("notes", "alice")
	require.True(t, ok)
	assert.Equal(t, "secret", v.Value)
	assert.Equal(t, 1, v.Version, "rejected write must not bump version")
}

// TestVisibility_UpdatePreservesExistingVisibility: a Set without
// WithVisibility on an existing private var leaves it private; the
// inverse holds for shared.
func TestVisibility_UpdatePreservesExistingVisibility(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()

	// Start private.
	_, err := wc.Set("k", "v1",
		"alice", collaboration.RoleOwner,
		collaboration.WithVisibility(collaboration.VisibilityPrivate))
	require.NoError(t, err)

	// Update without WithVisibility — should stay private.
	v, err := wc.Set("k", "v2", "alice", collaboration.RoleOwner)
	require.NoError(t, err)
	assert.Equal(t, collaboration.VisibilityPrivate, v.EffectiveVisibility(),
		"omitting WithVisibility must preserve prior visibility")

	// Bob still cannot see it.
	_, ok := wc.GetForProfile("k", "bob")
	assert.False(t, ok)
}

// TestVisibility_PromoteFromPrivateToShared: explicit
// WithVisibility(Shared) on a previously-private var releases it.
func TestVisibility_PromoteFromPrivateToShared(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()

	_, err := wc.Set("k", "v1",
		"alice", collaboration.RoleOwner,
		collaboration.WithVisibility(collaboration.VisibilityPrivate))
	require.NoError(t, err)

	v, err := wc.Set("k", "v2",
		"alice", collaboration.RoleOwner,
		collaboration.WithVisibility(collaboration.VisibilityShared))
	require.NoError(t, err)
	assert.Equal(t, collaboration.VisibilityShared, v.EffectiveVisibility())

	// Bob can now see it.
	got, ok := wc.GetForProfile("k", "bob")
	require.True(t, ok)
	assert.Equal(t, "v2", got.Value)
}

// TestVisibility_FirstWriteOmitsField: when an operator does not
// pass WithVisibility, the persisted entity must have the
// zero-value Visibility so existing yaml round-trips byte-stable.
// The field's `omitempty` json/yaml tag handles this on the wire.
func TestVisibility_FirstWriteOmitsField(t *testing.T) {
	wc := collaboration.NewWorkspaceContext()

	v, err := wc.Set("k", "v", "alice", collaboration.RoleOwner)
	require.NoError(t, err)

	assert.Equal(t, collaboration.ContextVisibility(""), v.Visibility,
		"first write without WithVisibility must leave Visibility zero so omitempty fires")
	assert.Equal(t, collaboration.VisibilityShared, v.EffectiveVisibility(),
		"zero value reads as shared via NormalizeVisibility")
}
