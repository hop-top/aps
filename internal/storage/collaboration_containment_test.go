package storage_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// containmentFixture builds a storage root two levels below a temp dir with
// sentinel data outside the root (parent and sibling), a real workspace
// inside it, and a stray file in the root. Even if containment regressed,
// the worst case removes only outer, inside the test's temp dir.
type containmentFixture struct {
	outer string
	root  string
	s     *storage.CollaborationStorage
}

func newContainmentFixture(t *testing.T) *containmentFixture {
	t.Helper()
	outer := filepath.Join(t.TempDir(), "outer")
	root := filepath.Join(outer, "root")
	require.NoError(t, os.MkdirAll(filepath.Join(outer, "sibling"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outer, "sentinel.txt"), []byte("outer"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(outer, "sibling", "sentinel.txt"), []byte("sib"), 0o644))

	s, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, s.SaveWorkspace(&collab.Workspace{
		ID:        "keep",
		Config:    collab.WorkspaceConfig{Name: "keep", OwnerProfileID: "o"},
		State:     collab.StateActive,
		CreatedAt: now,
		UpdatedAt: now,
	}))
	require.NoError(t, os.WriteFile(filepath.Join(root, "stray.txt"), []byte("stray"), 0o644))
	return &containmentFixture{outer: outer, root: root, s: s}
}

// snapshot maps every path under outer to its content ("" for dirs).
func (f *containmentFixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	snap := map[string]string{}
	require.NoError(t, filepath.WalkDir(f.outer, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			snap[p] = ""
			return nil
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		snap[p] = string(b)
		return nil
	}))
	return snap
}

func invalidIDs() []string { return []string{"", ".", ".."} }

func TestCollaborationStorage_RejectsUncontainedIDs(t *testing.T) {
	for _, id := range invalidIDs() {
		t.Run("id="+id, func(t *testing.T) {
			f := newContainmentFixture(t)
			before := f.snapshot(t)

			var inv *collab.InvalidWorkspaceIDError
			ws := &collab.Workspace{ID: id, Config: collab.WorkspaceConfig{Name: "x", OwnerProfileID: "o"}}

			assert.ErrorAs(t, f.s.SaveWorkspace(ws), &inv, "SaveWorkspace")
			_, err := f.s.LoadWorkspace(id)
			assert.ErrorAs(t, err, &inv, "LoadWorkspace")
			assert.ErrorAs(t, f.s.DeleteWorkspace(id), &inv, "DeleteWorkspace")

			assert.ErrorAs(t, f.s.SaveTasks(id, []collab.TaskInfo{}), &inv, "SaveTasks")
			_, err = f.s.LoadTasks(id)
			assert.ErrorAs(t, err, &inv, "LoadTasks")
			assert.ErrorAs(t, f.s.SaveConflicts(id, []collab.Conflict{}), &inv, "SaveConflicts")
			_, err = f.s.LoadConflicts(id)
			assert.ErrorAs(t, err, &inv, "LoadConflicts")
			assert.ErrorAs(t, f.s.SaveContext(id, []collab.ContextVariable{}), &inv, "SaveContext")
			_, err = f.s.LoadContext(id)
			assert.ErrorAs(t, err, &inv, "LoadContext")
			assert.ErrorAs(t, f.s.SaveAuditEvents(id, []collab.AuditEvent{}), &inv, "SaveAuditEvents")
			_, err = f.s.LoadAuditEvents(id)
			assert.ErrorAs(t, err, &inv, "LoadAuditEvents")

			assert.Equal(t, before, f.snapshot(t), "filesystem must be untouched")
		})
	}
}

// A manifest planted where "." / ".." resolve must not make them loadable.
func TestCollaborationStorage_LoadDoesNotReadOutsideRoot(t *testing.T) {
	f := newContainmentFixture(t)
	for _, dir := range []string{f.outer, f.root} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte("name: planted\nowner_profile_id: o\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"id":"planted"}`), 0o644))
	}
	for _, id := range []string{".", ".."} {
		_, err := f.s.LoadWorkspace(id)
		var inv *collab.InvalidWorkspaceIDError
		assert.ErrorAs(t, err, &inv, id)
	}
}

func TestCollaborationStorage_ValidIDsStillWork(t *testing.T) {
	f := newContainmentFixture(t)
	now := time.Now()
	for _, id := range []string{"0b9f5b1e-4c2d-4a8e-9f10-2d3c4b5a6e7f", "aps:global", "a.b", "..x", "x..", "..%2Fx", "a/..", "a%3Ab"} {
		ws := &collab.Workspace{
			ID:        id,
			Config:    collab.WorkspaceConfig{Name: id, OwnerProfileID: "o"},
			State:     collab.StateActive,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, f.s.SaveWorkspace(ws), id)
		got, err := f.s.LoadWorkspace(id)
		require.NoError(t, err, id)
		assert.Equal(t, id, got.ID)
		require.NoError(t, f.s.SaveTasks(id, []collab.TaskInfo{}), id)
	}
	// Nothing leaked outside the root.
	entries, err := os.ReadDir(f.outer)
	require.NoError(t, err)
	assert.Len(t, entries, 3, "outer holds only root, sibling, sentinel.txt")

	for _, id := range []string{"aps:global", "a%3Ab"} {
		require.NoError(t, f.s.DeleteWorkspace(id), id)
	}
}

func TestCollaborationStorage_LegacyDirStillResolvedAndContained(t *testing.T) {
	f := newContainmentFixture(t)
	// Legacy layout: verbatim directory for an id with reserved characters
	// (a colon is a legal directory name on POSIX).
	if filepath.Separator == '/' {
		legacy := filepath.Join(f.root, "legacy:id")
		require.NoError(t, os.MkdirAll(legacy, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(legacy, "tasks.json"), []byte(`[{"id":"t1"}]`), 0o644))
		tasks, err := f.s.LoadTasks("legacy:id")
		require.NoError(t, err)
		require.Len(t, tasks, 1)
		require.NoError(t, f.s.DeleteWorkspace("legacy:id"))
		_, err = os.Stat(legacy)
		assert.True(t, os.IsNotExist(err))
	}

	// An id whose verbatim (legacy) join leaves the root is never a legacy
	// candidate: it resolves to its escaped, contained directory only.
	require.NoError(t, os.MkdirAll(filepath.Join(f.outer, "victim"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(f.outer, "victim", "tasks.json"), []byte(`[{"id":"leak"}]`), 0o644))
	tasks, err := f.s.LoadTasks("../victim")
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestCollaborationStorage_ListWorkspacesSkipsStrays(t *testing.T) {
	f := newContainmentFixture(t)
	// Directories whose names unescape to uncontained ids, plus a stray
	// manifest-bearing dir named like "." once unescaped.
	for _, name := range []string{"%2E", "%2E%2E"} {
		d := filepath.Join(f.root, name)
		require.NoError(t, os.MkdirAll(d, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(d, "manifest.yaml"), []byte("name: x\n"), 0o644))
	}
	ids, err := f.s.ListWorkspaces()
	require.NoError(t, err)
	assert.Equal(t, []string{"keep"}, ids)
}
