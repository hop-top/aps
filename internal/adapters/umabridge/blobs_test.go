package umabridge

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/uma/model"
	"hop.top/uma/ports"
)

const blobID model.ID = "00000000-0000-7000-8000-0000000000a1"

func TestFileBlobsStageOpenDelete(t *testing.T) {
	b := fileBlobs{dir: t.TempDir()}
	key := ports.BlobKey{Tenant: tenant, ID: blobID}

	require.NoError(t, b.Stage(t.Context(), key, strings.NewReader("bytes"), 5))
	require.NoError(t, b.Stage(t.Context(), key, strings.NewReader("bytes"), -1), "staging again is idempotent")

	p, err := b.path(key)
	require.NoError(t, err)
	info, err := os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	r, err := b.Open(t.Context(), key)
	require.NoError(t, err)
	got, _ := io.ReadAll(r)
	_ = r.Close()
	assert.Equal(t, "bytes", string(got))

	require.NoError(t, b.Delete(t.Context(), key))
	require.NoError(t, b.Delete(t.Context(), key), "deleting a missing key succeeds")
	_, err = b.Open(t.Context(), key)
	assert.Equal(t, model.NotFound, model.CodeOf(err))
}

func TestFileBlobsRefuseWrongLengthAndLeaveNothing(t *testing.T) {
	dir := t.TempDir()
	b := fileBlobs{dir: dir}
	key := ports.BlobKey{Tenant: tenant, ID: blobID}

	require.Error(t, b.Stage(t.Context(), key, strings.NewReader("bytes"), 9))
	_, err := b.Open(t.Context(), key)
	assert.Equal(t, model.NotFound, model.CodeOf(err))
	entries, err := os.ReadDir(dir + "/" + string(tenant))
	require.NoError(t, err)
	assert.Empty(t, entries, "no temp file is left behind")
}

func TestFileBlobsRefuseKeysThatEscapeTheDirectory(t *testing.T) {
	b := fileBlobs{dir: t.TempDir()}
	for _, tn := range []model.TenantID{"../x", ".hidden", "a/b"} {
		err := b.Stage(t.Context(), ports.BlobKey{Tenant: tn, ID: blobID}, strings.NewReader("x"), 1)
		require.Error(t, err, string(tn))
	}
	require.Error(t, b.Stage(t.Context(), ports.BlobKey{Tenant: tenant, ID: "../../etc"}, strings.NewReader("x"), 1))
}
