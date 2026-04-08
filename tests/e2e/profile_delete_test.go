package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProfileDelete covers the happy path of `aps profile delete <id> --yes` (T7).
// It creates a profile, deletes it non-interactively, and asserts the profile
// no longer appears in `aps profile list`.
func TestProfileDelete(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create a profile.
	_, _, err := runAPS(t, home, "profile", "new", "doomed", "--display-name", "Doomed")
	require.NoError(t, err)

	// Sanity: it shows up in the list.
	stdout, _, err := runAPS(t, home, "profile", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, "doomed")

	// Delete it non-interactively.
	stdout, stderr, err := runAPS(t, home, "profile", "delete", "doomed", "--yes")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stdout, "Profile 'doomed' deleted.")

	// It is gone from the list.
	stdout, _, err = runAPS(t, home, "profile", "list")
	require.NoError(t, err)
	assert.NotContains(t, stdout, "doomed")

	// Verify the profile directory is actually gone on disk.
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "doomed")
	if _, err := os.Stat(profileDir); !os.IsNotExist(err) {
		t.Errorf("profile directory still exists on disk: %s (err: %v)", profileDir, err)
	}
}

// TestProfileDeleteMissing verifies that deleting a non-existent profile
// returns a non-zero exit and a useful error.
func TestProfileDeleteMissing(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "profile", "delete", "ghost", "--yes")
	require.Error(t, err)
	assert.True(t,
		strings.Contains(stderr, "loading profile") ||
			strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "no such"),
		"unexpected stderr: %s", stderr,
	)
}
