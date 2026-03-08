package collaboration_test

import (
	"testing"
	"time"

	"hop.top/aps/internal/core/collaboration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceContext_Set_Get(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	v, err := ctx.Set("build.status", "passing", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)
	require.NotNil(t, v)

	assert.Equal(t, "build.status", v.Key)
	assert.Equal(t, "passing", v.Value)
	assert.Equal(t, 1, v.Version)
	assert.Equal(t, "agent-1", v.UpdatedBy)
	assert.False(t, v.UpdatedAt.IsZero())

	got, ok := ctx.Get("build.status")
	require.True(t, ok)
	assert.Equal(t, "passing", got.Value)
}

func TestWorkspaceContext_Set_Versioning(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	v1, err := ctx.Set("counter", "1", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)
	assert.Equal(t, 1, v1.Version)

	v2, err := ctx.Set("counter", "2", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)
	assert.Equal(t, 2, v2.Version)

	v3, err := ctx.Set("counter", "3", "agent-2", collaboration.RoleContributor)
	require.NoError(t, err)
	assert.Equal(t, 3, v3.Version)
}

func TestWorkspaceContext_Get_NotFound(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	v, ok := ctx.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, v)
}

func TestWorkspaceContext_Delete(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	_, err := ctx.Set("temp", "value", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	err = ctx.Delete("temp", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	v, ok := ctx.Get("temp")
	assert.False(t, ok)
	assert.Nil(t, v)
}

func TestWorkspaceContext_Delete_NotFound(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	err := ctx.Delete("nonexistent", "agent-1", collaboration.RoleOwner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestWorkspaceContext_Delete_ACL_Denied(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	// Set a value as owner.
	_, err := ctx.Set("protected", "data", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	// Observer should not be able to delete (default ACL gives observers read-only).
	err = ctx.Delete("protected", "agent-2", collaboration.RoleObserver)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	// Verify the variable is still present.
	v, ok := ctx.Get("protected")
	assert.True(t, ok)
	assert.Equal(t, "data", v.Value)
}

func TestWorkspaceContext_List(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	_, err := ctx.Set("a", "1", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	_, err = ctx.Set("b", "2", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	_, err = ctx.Set("c", "3", "agent-2", collaboration.RoleContributor)
	require.NoError(t, err)

	vars := ctx.List()
	assert.Len(t, vars, 3)

	keys := make(map[string]string)
	for _, v := range vars {
		keys[v.Key] = v.Value
	}
	assert.Equal(t, "1", keys["a"])
	assert.Equal(t, "2", keys["b"])
	assert.Equal(t, "3", keys["c"])
}

func TestWorkspaceContext_Mutations(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	_, err := ctx.Set("key", "v1", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	_, err = ctx.Set("key", "v2", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	_, err = ctx.Set("other", "x", "agent-2", collaboration.RoleContributor)
	require.NoError(t, err)

	mutations := ctx.Mutations()
	assert.Len(t, mutations, 3)

	// First mutation: set "key" to "v1"
	assert.Equal(t, "key", mutations[0].Key)
	assert.Equal(t, "", mutations[0].OldValue)
	assert.Equal(t, "v1", mutations[0].NewValue)
	assert.Equal(t, 1, mutations[0].Version)

	// Second mutation: update "key" to "v2"
	assert.Equal(t, "key", mutations[1].Key)
	assert.Equal(t, "v1", mutations[1].OldValue)
	assert.Equal(t, "v2", mutations[1].NewValue)
	assert.Equal(t, 2, mutations[1].Version)

	// Third mutation: set "other" to "x"
	assert.Equal(t, "other", mutations[2].Key)
}

func TestWorkspaceContext_MutationsForKey(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	_, err := ctx.Set("target", "a", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	_, err = ctx.Set("noise", "b", "agent-2", collaboration.RoleContributor)
	require.NoError(t, err)

	_, err = ctx.Set("target", "c", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	mutations := ctx.MutationsForKey("target")
	assert.Len(t, mutations, 2)
	assert.Equal(t, "a", mutations[0].NewValue)
	assert.Equal(t, "c", mutations[1].NewValue)

	noiseMutations := ctx.MutationsForKey("noise")
	assert.Len(t, noiseMutations, 1)
}

func TestWorkspaceContext_SetACL_CustomPermissions(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	// Set a restrictive ACL: only owners can write to "secret".
	ctx.SetACL(collaboration.ACLEntry{
		Key: "secret",
		Permissions: map[collaboration.AgentRole][]collaboration.Permission{
			collaboration.RoleOwner:       {collaboration.PermRead, collaboration.PermWrite, collaboration.PermDelete},
			collaboration.RoleContributor: {collaboration.PermRead},
			collaboration.RoleObserver:    {collaboration.PermRead},
		},
	})

	// Owner can write.
	_, err := ctx.Set("secret", "classified", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	// Contributor should be denied write.
	_, err = ctx.Set("secret", "hacked", "agent-2", collaboration.RoleContributor)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	// Value should remain the owner's write.
	v, ok := ctx.Get("secret")
	require.True(t, ok)
	assert.Equal(t, "classified", v.Value)
}

func TestWorkspaceContext_GetACL_Default(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	acl := ctx.GetACL("any-key")
	assert.Equal(t, "any-key", acl.Key)

	// Default ACL: owner has all, contributor has read+write, observer has read.
	assert.True(t, acl.HasPermission(collaboration.RoleOwner, collaboration.PermAdmin))
	assert.True(t, acl.HasPermission(collaboration.RoleContributor, collaboration.PermWrite))
	assert.True(t, acl.HasPermission(collaboration.RoleObserver, collaboration.PermRead))
	assert.False(t, acl.HasPermission(collaboration.RoleObserver, collaboration.PermWrite))
}

func TestWorkspaceContext_Snapshot(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	_, err := ctx.Set("x", "1", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	_, err = ctx.Set("y", "2", "agent-2", collaboration.RoleContributor)
	require.NoError(t, err)

	ctx.SetACL(collaboration.ACLEntry{
		Key: "x",
		Permissions: map[collaboration.AgentRole][]collaboration.Permission{
			collaboration.RoleOwner: {collaboration.PermRead, collaboration.PermWrite},
		},
	})

	vars, acls := ctx.Snapshot()
	assert.Len(t, vars, 2)
	assert.Len(t, acls, 1)

	// Verify it is a copy: modifying the returned slice should not affect the context.
	vars[0].Value = "modified"
	original, ok := ctx.Get(vars[0].Key)
	require.True(t, ok)
	assert.NotEqual(t, "modified", original.Value)
}

func TestWorkspaceContext_ACL_ObserverCannotWrite(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	_, err := ctx.Set("data", "value", "observer-1", collaboration.RoleObserver)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestWorkspaceContext_ACL_ContributorCanWrite(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	v, err := ctx.Set("data", "value", "contributor-1", collaboration.RoleContributor)
	require.NoError(t, err)
	assert.Equal(t, "value", v.Value)
	assert.Equal(t, "contributor-1", v.UpdatedBy)
}

func TestWorkspaceContext_DetectWriteConflict(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	// Two different agents write to the same key within a short window.
	_, err := ctx.Set("shared", "v1", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	_, err = ctx.Set("shared", "v2", "agent-2", collaboration.RoleContributor)
	require.NoError(t, err)

	conflict := ctx.DetectWriteConflict("shared", 5*time.Second)
	require.NotNil(t, conflict)
	assert.Equal(t, collaboration.ConflictWrite, conflict.Type)
	assert.Equal(t, "shared", conflict.Resource)
	assert.Len(t, conflict.AgentIDs, 2)
	assert.Contains(t, conflict.AgentIDs, "agent-1")
	assert.Contains(t, conflict.AgentIDs, "agent-2")
}

func TestWorkspaceContext_DetectWriteConflict_NoConflict(t *testing.T) {
	ctx := collaboration.NewWorkspaceContext()

	// Single agent writes twice -- no conflict because only one agent.
	_, err := ctx.Set("solo", "v1", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	_, err = ctx.Set("solo", "v2", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	conflict := ctx.DetectWriteConflict("solo", 5*time.Second)
	assert.Nil(t, conflict)
}
