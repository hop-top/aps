package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileLifecycle(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// 1. List - should be empty
	stdout, _, err := runAPS(t, home, "profile", "list")
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(stdout))

	// 2. Create profile
	stdout, _, err = runAPS(t, home, "profile", "new", "agent-1", "--display-name", "Agent One")
	require.NoError(t, err)
	assert.Contains(t, stdout, "created successfully")

	// 3. List - should contain agent-1
	stdout, _, err = runAPS(t, home, "profile", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, "agent-1")

	// 4. Show - should contain details
	stdout, _, err = runAPS(t, home, "profile", "show", "agent-1")
	require.NoError(t, err)
	assert.Contains(t, stdout, "display_name: Agent One")
	assert.Contains(t, stdout, "id: agent-1")
	assert.Contains(t, stdout, "Secrets: present") // Created by default
}

func TestProfileOverwrite(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create initial
	_, _, err := runAPS(t, home, "profile", "new", "agent-x")
	require.NoError(t, err)

	// Try to overwrite without force - should fail
	_, stderr, err := runAPS(t, home, "profile", "new", "agent-x")
	require.Error(t, err)
	assert.Contains(t, stderr, "already exists")

	// Overwrite with force - should succeed
	stdout, _, err := runAPS(t, home, "profile", "new", "agent-x", "--force", "--display-name", "Agent X Force")
	require.NoError(t, err)
	assert.Contains(t, stdout, "created successfully")

	// Verify change
	stdout, _, err = runAPS(t, home, "profile", "show", "agent-x")
	require.NoError(t, err)
	assert.Contains(t, stdout, "display_name: Agent X Force")
}
