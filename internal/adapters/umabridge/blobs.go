package umabridge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"hop.top/uma/model"
	"hop.top/uma/ports"
)

// fileBlobs is the engine's attachment sink on the local disk: staged media
// bytes under dir/<tenant>/<id>, owner-only. It keeps bytes and decides
// nothing; the engine authorizes, bounds and verifies them.
type fileBlobs struct{ dir string }

var _ ports.BlobStore = fileBlobs{}

func (b fileBlobs) path(key ports.BlobKey) (string, error) {
	if err := key.Validate(); err != nil {
		return "", err //nolint:wrapcheck // typed uma error; the engine classifies it by code
	}
	tenantDir := string(key.Tenant)
	if tenantDir != filepath.Base(tenantDir) || strings.HasPrefix(tenantDir, ".") {
		return "", model.NewError(model.Validation, "blob key tenant: not a plain name")
	}
	return filepath.Join(b.dir, tenantDir, string(key.ID)), nil
}

func (b fileBlobs) Stage(ctx context.Context, key ports.BlobKey, r io.Reader, size int64) (err error) {
	p, err := b.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return model.NewError(model.Unavailable, "blob store unavailable")
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".stage-*")
	if err != nil {
		return model.NewError(model.Unavailable, "blob store unavailable")
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()
	n, copyErr := io.Copy(tmp, readerWithContext{ctx, r})
	if closeErr := tmp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	switch {
	case copyErr != nil:
		return copyErr
	case size >= 0 && n != size:
		return fmt.Errorf("blob length %d differs from declared %d", n, size)
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return model.NewError(model.Unavailable, "blob store unavailable")
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return model.NewError(model.Unavailable, "blob store unavailable")
	}
	return nil
}

func (b fileBlobs) Open(_ context.Context, key ports.BlobKey) (io.ReadCloser, error) {
	p, err := b.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p) // #nosec G304 -- path built from a validated key under dir
	if errors.Is(err, os.ErrNotExist) {
		return nil, model.NewError(model.NotFound, "blob not found")
	}
	if err != nil {
		return nil, model.NewError(model.Unavailable, "blob store unavailable")
	}
	return f, nil
}

func (b fileBlobs) Delete(_ context.Context, key ports.BlobKey) error {
	p, err := b.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return model.NewError(model.Unavailable, "blob store unavailable")
	}
	return nil
}

type readerWithContext struct {
	ctx context.Context
	r   io.Reader
}

func (r readerWithContext) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err //nolint:wrapcheck // reader decorator: the context error passes through
	}
	return r.r.Read(p) //nolint:wrapcheck // reader decorator: io.EOF must stay exact
}
