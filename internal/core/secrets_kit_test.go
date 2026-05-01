package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/kit/go/storage/secret"
)

// TestDotenvStoreRoundTrip exercises the dotenvStore wrapper used by the
// "file" backend: write → list → get → delete → ErrNotFound.
func TestDotenvStoreRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := newDotenvStore(filepath.Join(dir, "secrets.env"))
	ctx := context.Background()

	require.NoError(t, store.Set(ctx, "API_TOKEN", []byte("abc123")))
	require.NoError(t, store.Set(ctx, "WEBHOOK_SECRET", []byte("hush")))

	keys, err := store.List(ctx, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"API_TOKEN", "WEBHOOK_SECRET"}, keys)

	got, err := store.Get(ctx, "API_TOKEN")
	require.NoError(t, err)
	assert.Equal(t, "abc123", string(got.Value))

	exists, err := store.Exists(ctx, "API_TOKEN")
	require.NoError(t, err)
	assert.True(t, exists)

	require.NoError(t, store.Delete(ctx, "API_TOKEN"))
	_, err = store.Get(ctx, "API_TOKEN")
	assert.ErrorIs(t, err, secret.ErrNotFound)
}

// TestDotenvStoreCompatibleWithLegacyFile ensures the wrapper round-trips
// values written by the legacy godotenv path.
func TestDotenvStoreCompatibleWithLegacyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.env")
	require.NoError(t, writeFile0600(path, "FOO=bar\nBAZ=qux\n"))

	store := newDotenvStore(path)
	ctx := context.Background()

	got, err := store.Get(ctx, "FOO")
	require.NoError(t, err)
	assert.Equal(t, "bar", string(got.Value))

	keys, err := store.List(ctx, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"FOO", "BAZ"}, keys)
}

// TestSecretsConfigDefault verifies LoadConfig fills in the file backend
// default when no value is provided.
func TestSecretsConfigDefault(t *testing.T) {
	t.Parallel()

	cfg := &Config{Prefix: DefaultPrefix}
	if cfg.Secrets.Backend == "" {
		cfg.Secrets.Backend = SecretsBackendFile
	}
	assert.Equal(t, SecretsBackendFile, cfg.Secrets.Backend)
}

func writeFile0600(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0600)
}
