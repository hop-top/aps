package collaboration_test

import (
	"context"
	"testing"

	"oss-aps-cli/internal/core/collaboration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPolicy_Priority(t *testing.T) {
	policy, err := collaboration.NewPolicy(collaboration.StrategyPriority)
	require.NoError(t, err)
	assert.NotNil(t, policy)
}

func TestNewPolicy_KeepFirst(t *testing.T) {
	policy, err := collaboration.NewPolicy(collaboration.StrategyKeepFirst)
	require.NoError(t, err)
	assert.NotNil(t, policy)
}

func TestNewPolicy_KeepLast(t *testing.T) {
	policy, err := collaboration.NewPolicy(collaboration.StrategyKeepLast)
	require.NoError(t, err)
	assert.NotNil(t, policy)
}

func TestNewPolicy_Rollback(t *testing.T) {
	policy, err := collaboration.NewPolicy(collaboration.StrategyRollback)
	require.NoError(t, err)
	assert.NotNil(t, policy)
}

func TestNewPolicy_Unsupported(t *testing.T) {
	policy, err := collaboration.NewPolicy(collaboration.ResolutionStrategy("consensus"))
	assert.Error(t, err)
	assert.Nil(t, policy)
}

func TestPriorityPolicy_OwnerWins(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	_, err = ws.AddAgent("agent-a", collaboration.RoleOwner)
	require.NoError(t, err)
	_, err = ws.AddAgent("agent-b", collaboration.RoleContributor)
	require.NoError(t, err)

	ws.Context.Set("deploy-key", "owner-val", "agent-a", collaboration.RoleOwner)
	ws.Context.Set("deploy-key", "contrib-val", "agent-b", collaboration.RoleContributor)

	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{"agent-a", "agent-b"},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyPriority)
	require.NoError(t, err)

	resolution, err := policy.Resolve(context.Background(), conflict, ws)
	require.NoError(t, err)
	require.NotNil(t, resolution)
	assert.Contains(t, resolution.Details, "agent-a")
}

func TestPriorityPolicy_NilWorkspace(t *testing.T) {
	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{"agent-a", "agent-b"},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyPriority)
	require.NoError(t, err)

	_, err = policy.Resolve(context.Background(), conflict, nil)
	assert.Error(t, err)
}

func TestPriorityPolicy_NoAgents(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyPriority)
	require.NoError(t, err)

	_, err = policy.Resolve(context.Background(), conflict, ws)
	assert.Error(t, err)
}

func TestKeepFirstPolicy_KeepsEarliestValue(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	ws.Context.Set("deploy-key", "first-val", "agent-a", collaboration.RoleOwner)
	ws.Context.Set("deploy-key", "second-val", "agent-b", collaboration.RoleContributor)

	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{"agent-a", "agent-b"},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyKeepFirst)
	require.NoError(t, err)

	resolution, err := policy.Resolve(context.Background(), conflict, ws)
	require.NoError(t, err)
	require.NotNil(t, resolution)
	assert.Contains(t, resolution.Details, "first writer")
}

func TestKeepFirstPolicy_NilWorkspace(t *testing.T) {
	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{"agent-a", "agent-b"},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyKeepFirst)
	require.NoError(t, err)

	_, err = policy.Resolve(context.Background(), conflict, nil)
	assert.Error(t, err)
}

func TestKeepLastPolicy_KeepsLatestValue(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	ws.Context.Set("deploy-key", "first-val", "agent-a", collaboration.RoleOwner)
	ws.Context.Set("deploy-key", "latest-val", "agent-b", collaboration.RoleContributor)

	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{"agent-a", "agent-b"},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyKeepLast)
	require.NoError(t, err)

	resolution, err := policy.Resolve(context.Background(), conflict, ws)
	require.NoError(t, err)
	require.NotNil(t, resolution)
	assert.Contains(t, resolution.Details, "last writer")
}

func TestKeepLastPolicy_NilWorkspace(t *testing.T) {
	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{"agent-a", "agent-b"},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyKeepLast)
	require.NoError(t, err)

	_, err = policy.Resolve(context.Background(), conflict, nil)
	assert.Error(t, err)
}

func TestRollbackPolicy_RestoresOldValue(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	ws.Context.Set("deploy-key", "original", "agent-a", collaboration.RoleOwner)
	ws.Context.Set("deploy-key", "changed", "agent-b", collaboration.RoleContributor)

	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{"agent-a", "agent-b"},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyRollback)
	require.NoError(t, err)

	resolution, err := policy.Resolve(context.Background(), conflict, ws)
	require.NoError(t, err)
	require.NotNil(t, resolution)
	// Rollback restores the OldValue from the earliest mutation (pre-conflict state)
	assert.Contains(t, resolution.Details, "rolled back")
}

func TestRollbackPolicy_NilWorkspace(t *testing.T) {
	conflict := collaboration.Conflict{
		Resource: "deploy-key",
		AgentIDs: []string{"agent-a", "agent-b"},
	}

	policy, err := collaboration.NewPolicy(collaboration.StrategyRollback)
	require.NoError(t, err)

	_, err = policy.Resolve(context.Background(), conflict, nil)
	assert.Error(t, err)
}
