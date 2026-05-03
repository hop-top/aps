package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkspaceMergedSurface validates that aps workspace is the canonical
// noun: merged surface from collab + audit + conflict. Top-level collab,
// audit, and conflict commands MUST be removed.
func TestWorkspaceMergedSurface(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Top-level collab/audit/conflict commands MUST NOT exist. The root
	// command falls back to "unknown command or profile" + exit 1 for
	// any non-registered top-level token.
	t.Run("collab not registered", func(t *testing.T) {
		_, stderr, err := runAPS(t, home, "collab")
		require.Error(t, err, "aps collab should not exist")
		assert.Contains(t, strings.ToLower(stderr), "unknown command or profile")
	})

	t.Run("audit not registered", func(t *testing.T) {
		_, stderr, err := runAPS(t, home, "audit")
		require.Error(t, err, "aps audit should not exist")
		assert.Contains(t, strings.ToLower(stderr), "unknown command or profile")
	})

	t.Run("conflict not registered", func(t *testing.T) {
		_, stderr, err := runAPS(t, home, "conflict")
		require.Error(t, err, "aps conflict should not exist")
		assert.Contains(t, strings.ToLower(stderr), "unknown command or profile")
	})

	// aps workspace --help should list merged subcommands.
	t.Run("workspace help lists merged subcommands", func(t *testing.T) {
		stdout, _, err := runAPS(t, home, "workspace", "--help")
		require.NoError(t, err)

		// Existing stub
		assert.Contains(t, stdout, "activity")
		assert.Contains(t, stdout, "sync")

		// Merged from collab
		// T-0394 — `new` was renamed to `create`.
		assert.Contains(t, stdout, "create")
		assert.Contains(t, stdout, "list")
		assert.Contains(t, stdout, "show")
		assert.Contains(t, stdout, "join")
		assert.Contains(t, stdout, "leave")
		assert.Contains(t, stdout, "members")
		assert.Contains(t, stdout, "audit")
		assert.Contains(t, stdout, "conflicts")
	})

	t.Run("workspace audit help works", func(t *testing.T) {
		stdout, _, err := runAPS(t, home, "workspace", "audit", "--help")
		require.NoError(t, err)
		// Richer collab/audit filter flags must be preserved
		assert.Contains(t, stdout, "--actor")
		assert.Contains(t, stdout, "--event")
		assert.Contains(t, stdout, "--since")
	})

	t.Run("workspace conflicts list help works", func(t *testing.T) {
		stdout, _, err := runAPS(t, home, "workspace", "conflicts", "list", "--help")
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(stdout), "list")
	})

	t.Run("workspace conflicts show help works", func(t *testing.T) {
		stdout, _, err := runAPS(t, home, "workspace", "conflicts", "show", "--help")
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(stdout), "show")
	})

	t.Run("workspace conflicts resolve help works", func(t *testing.T) {
		stdout, _, err := runAPS(t, home, "workspace", "conflicts", "resolve", "--help")
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(stdout), "resolve")
	})
}
