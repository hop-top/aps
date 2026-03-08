package collaboration_test

import (
	"context"
	"sort"
	"testing"

	"hop.top/aps/internal/core/collaboration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilityRegistry_Register(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review", "testing", "refactoring"})
	require.NoError(t, err)

	caps, err := reg.ListCapabilities(ctx, wsID, "agent-1")
	require.NoError(t, err)

	sort.Strings(caps)
	assert.Equal(t, []string{"code-review", "refactoring", "testing"}, caps)
}

func TestCapabilityRegistry_Register_ReplacesExisting(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review", "testing"})
	require.NoError(t, err)

	// Re-register with different capabilities; old ones should be gone.
	err = reg.Register(ctx, wsID, "agent-1", []string{"deployment", "monitoring"})
	require.NoError(t, err)

	caps, err := reg.ListCapabilities(ctx, wsID, "agent-1")
	require.NoError(t, err)

	sort.Strings(caps)
	assert.Equal(t, []string{"deployment", "monitoring"}, caps)

	// The old capabilities should no longer resolve to agent-1.
	matches, err := reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Capability: "code-review"})
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestCapabilityRegistry_Unregister(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review", "testing"})
	require.NoError(t, err)

	reg.Unregister("agent-1")

	caps, err := reg.ListCapabilities(ctx, wsID, "agent-1")
	require.NoError(t, err)
	assert.Empty(t, caps)

	// Capability index should also be cleaned up.
	matches, err := reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Capability: "code-review"})
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestCapabilityRegistry_FindAgents_ByCapability(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review", "testing"})
	require.NoError(t, err)

	matches, err := reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Capability: "code-review"})
	require.NoError(t, err)
	require.Len(t, matches, 1)

	assert.Equal(t, "agent-1", matches[0].Agent.ProfileID)
	assert.Equal(t, 1.0, matches[0].Score)
	assert.Equal(t, "code-review", matches[0].Match)
}

func TestCapabilityRegistry_FindAgents_ByCapability_NoMatch(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review"})
	require.NoError(t, err)

	matches, err := reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Capability: "deployment"})
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestCapabilityRegistry_FindAgents_ByTask(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review"})
	require.NoError(t, err)

	// "review" is a substring of "code-review", so it should match.
	matches, err := reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Task: "review"})
	require.NoError(t, err)
	require.Len(t, matches, 1)

	assert.Equal(t, "agent-1", matches[0].Agent.ProfileID)
	assert.Equal(t, 1.0, matches[0].Score) // 1 word, 1 match -> 1.0
	assert.Equal(t, "code-review", matches[0].Match)
}

func TestCapabilityRegistry_FindAgents_ByTask_MultipleWords(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review", "testing"})
	require.NoError(t, err)

	// Task has two words: "code" and "analysis".
	// "code-review" contains "code" but not "analysis" -> score = 0.5
	matches, err := reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Task: "code analysis"})
	require.NoError(t, err)
	require.Len(t, matches, 1)

	assert.Equal(t, "agent-1", matches[0].Agent.ProfileID)
	assert.Equal(t, 0.5, matches[0].Score)
	assert.Equal(t, "code-review", matches[0].Match)
}

func TestCapabilityRegistry_FindAgents_EmptyQuery(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	_, err := reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query must specify capability or task")
}

func TestCapabilityRegistry_ListCapabilities_All(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review", "testing"})
	require.NoError(t, err)
	err = reg.Register(ctx, wsID, "agent-2", []string{"deployment", "monitoring"})
	require.NoError(t, err)

	// Empty profileID returns all capabilities across all agents.
	caps, err := reg.ListCapabilities(ctx, wsID, "")
	require.NoError(t, err)

	sort.Strings(caps)
	assert.Equal(t, []string{"code-review", "deployment", "monitoring", "testing"}, caps)
}

func TestCapabilityRegistry_ListCapabilities_ByAgent(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review", "testing"})
	require.NoError(t, err)
	err = reg.Register(ctx, wsID, "agent-2", []string{"deployment", "monitoring"})
	require.NoError(t, err)

	caps, err := reg.ListCapabilities(ctx, wsID, "agent-2")
	require.NoError(t, err)

	sort.Strings(caps)
	assert.Equal(t, []string{"deployment", "monitoring"}, caps)
}

func TestCapabilityRegistry_ListCapabilities_UnknownAgent(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	caps, err := reg.ListCapabilities(ctx, wsID, "nonexistent-agent")
	require.NoError(t, err)
	assert.Empty(t, caps)
}

func TestCapabilityRegistry_MultipleAgents(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	// Two agents share the "code-review" capability.
	err := reg.Register(ctx, wsID, "agent-1", []string{"code-review", "testing"})
	require.NoError(t, err)
	err = reg.Register(ctx, wsID, "agent-2", []string{"code-review", "deployment"})
	require.NoError(t, err)

	// Both agents should appear in a capability query for "code-review".
	matches, err := reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Capability: "code-review"})
	require.NoError(t, err)
	require.Len(t, matches, 2)

	profileIDs := []string{matches[0].Agent.ProfileID, matches[1].Agent.ProfileID}
	sort.Strings(profileIDs)
	assert.Equal(t, []string{"agent-1", "agent-2"}, profileIDs)

	// Both should have score 1.0 for exact capability match.
	for _, m := range matches {
		assert.Equal(t, 1.0, m.Score)
		assert.Equal(t, "code-review", m.Match)
	}

	// Unregister one agent; the other should still match.
	reg.Unregister("agent-1")

	matches, err = reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Capability: "code-review"})
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "agent-2", matches[0].Agent.ProfileID)

	// "testing" capability should be gone entirely.
	matches, err = reg.FindAgents(ctx, wsID, collaboration.CapabilityQuery{Capability: "testing"})
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestCapabilityRegistry_Refresh(t *testing.T) {
	reg := collaboration.NewCapabilityRegistry()
	ctx := context.Background()
	wsID := "ws-1"

	// Refresh is currently a no-op; it should return nil.
	err := reg.Refresh(ctx, wsID, "agent-1")
	assert.NoError(t, err)
}
