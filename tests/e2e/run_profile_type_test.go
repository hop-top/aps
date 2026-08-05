package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRejectsHumanProfile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", "human-agent")
	require.NoError(t, err)

	// Mark it as a human profile by appending the type field, the same
	// way TestSecretInjection edits secrets.env post-create.
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", "human-agent", "profile.yaml")
	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("\ntype: human\n")
	require.NoError(t, err)

	// Running a human profile must fail with a clear error.
	_, stderr, err := runAPS(t, home, "run", "human-agent", "--", "env")
	require.Error(t, err)
	assert.Contains(t, stderr, `profile "human-agent" is type human and cannot be run`)
}
