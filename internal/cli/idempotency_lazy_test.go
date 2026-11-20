package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"hop.top/kit/go/console/cli/idemstore"
)

// TestIdempotencyStore_NotOpenedByVersion pins that building the root
// and running a command that does not use --idempotency-key creates no
// idempotency db and no state dir. The store used to open at package
// init, so every process (and every test binary) touched the real state
// db.
func TestIdempotencyStore_NotOpenedByVersion(t *testing.T) {
	dataHome := t.TempDir()

	runAPS(t, dataHome, "version")

	stateDir := filepath.Join(dataHome, "state", "aps")
	require.NoFileExists(t, filepath.Join(stateDir, idemstoreFileName))
	require.NoDirExists(t, stateDir)
}

// TestLazyIdempotencyStore_OpensOnFirstUse proves the lazy wrapper
// still delivers replay: the backend appears on first use and a
// recorded key is found afterwards. Close before first use opens
// nothing.
func TestLazyIdempotencyStore_OpensOnFirstUse(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	dbPath := filepath.Join(state, "aps", idemstoreFileName)

	idle := &lazyIdempotencyStore{}
	require.NoError(t, idle.Close())
	require.NoFileExists(t, dbPath)

	l := &lazyIdempotencyStore{}
	t.Cleanup(func() { _ = l.Close() })
	ctx := context.Background()

	_, hit, err := l.Lookup(ctx, "k")
	require.NoError(t, err)
	require.False(t, hit)
	require.FileExists(t, dbPath)

	require.NoError(t, l.Record(ctx, "k", idemstore.Result{Output: []byte("out")}))
	got, hit, err := l.Lookup(ctx, "k")
	require.NoError(t, err)
	require.True(t, hit)
	require.Equal(t, "out", string(got.Output))
}
