package workspace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, adapter)
	defer adapter.Close()
}

func TestAdapterCreateAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	ws, err := adapter.Create(ctx, "test-project", CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, "test-project", ws.Name)
	assert.Equal(t, "active", ws.Status)

	got, err := adapter.Get(ctx, "test-project")
	require.NoError(t, err)
	assert.Equal(t, ws.Name, got.Name)
}

func TestAdapterList(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Create(ctx, "project-a", CreateOptions{})
	require.NoError(t, err)
	_, err = adapter.Create(ctx, "project-b", CreateOptions{})
	require.NoError(t, err)

	list, err := adapter.List(ctx, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestAdapterArchive(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Create(ctx, "archive-me", CreateOptions{})
	require.NoError(t, err)

	err = adapter.Archive(ctx, "archive-me")
	require.NoError(t, err)

	ws, err := adapter.Get(ctx, "archive-me")
	require.NoError(t, err)
	assert.Equal(t, "archived", ws.Status)
}

func TestAdapterDelete(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Create(ctx, "delete-me", CreateOptions{})
	require.NoError(t, err)

	err = adapter.Delete(ctx, "delete-me")
	require.NoError(t, err)

	_, err = adapter.Get(ctx, "delete-me")
	assert.Error(t, err)
}

func TestAdapterGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Get(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestAdapterCreateDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Create(ctx, "dupe", CreateOptions{})
	require.NoError(t, err)

	_, err = adapter.Create(ctx, "dupe", CreateOptions{})
	assert.Error(t, err)
}
