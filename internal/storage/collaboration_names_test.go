package storage_test

import (
	"os"
	"strings"
	"testing"

	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Workspace IDs such as "aps:global" are not valid directory names on
// Windows; on-disk names must stay free of reserved characters on every OS.
func TestCollaborationStorage_ReservedCharsInWorkspaceID(t *testing.T) {
	root := t.TempDir()
	s, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)

	const id = "aps:global"
	require.NoError(t, s.SaveAuditEvents(id, []collab.AuditEvent{}))

	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.ContainsAny(e.Name(), `<>:"/\|?*`), "dir %q", e.Name())
	}

	got, err := s.LoadAuditEvents(id)
	require.NoError(t, err)
	assert.Empty(t, got)
}
