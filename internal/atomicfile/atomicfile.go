// Package atomicfile replaces files atomically: a reader, or a process that
// crashes at any point, sees the complete old or the complete new content,
// never a partial file.
package atomicfile

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// OpenFunc creates the temp file. It must honour flag and perm like
// os.OpenFile; secret callers pass one that applies owner-only access before
// any data is written.
type OpenFunc func(path string, flag int, perm os.FileMode) (*os.File, error)

// Write replaces path with data: hidden exclusive temp file in the same
// directory (created via open, os.OpenFile when nil), write, sync, close,
// rename over path, then a best-effort directory fsync. Any failure before
// the rename removes the temp file and leaves path untouched.
//
// A symlink at path is resolved and the write lands beside its target, so the
// link survives; a dangling link creates the file it points at. The result is
// always a new file created with perm (subject to umask on POSIX), so an
// existing file's broader mode is tightened to perm; this is the secret-file
// behaviour. Hard links to the old file no longer follow writes. A process
// killed mid-write can leave ".<name>.tmp-<random>" behind; it is not swept
// (unsafe with concurrent writers).
func Write(path string, data []byte, perm os.FileMode, open OpenFunc) error {
	return write(path, data, perm, open, false)
}

// WriteKeepMode is Write for non-secret files: when the resolved target
// already exists as a regular file, the replacement takes its permission bits
// exactly (set with Chmod, so umask does not narrow them); otherwise the new
// file gets perm minus umask. A Chmod failure is a pre-rename failure. On
// Windows Chmod only toggles the read-only attribute.
func WriteKeepMode(path string, data []byte, perm os.FileMode) error {
	return write(path, data, perm, nil, true)
}

func write(path string, data []byte, perm os.FileMode, open OpenFunc, keepMode bool) (err error) {
	if open == nil {
		open = os.OpenFile
	}
	path, err = resolveLink(path)
	if err != nil {
		return err
	}
	carry := false
	var mode os.FileMode
	if keepMode {
		if fi, serr := os.Stat(path); serr == nil && fi.Mode().IsRegular() {
			carry, mode = true, fi.Mode().Perm()
		}
	}
	dir := filepath.Dir(path)
	f, tmp, err := createTemp(open, dir, filepath.Base(path), perm)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if err != nil {
			if !closed {
				_ = f.Close()
			}
			_ = os.Remove(tmp)
		}
	}()
	if err = hooks.write(f, data); err != nil {
		return err
	}
	if carry {
		if err = f.Chmod(mode); err != nil {
			return err //nolint:wrapcheck // PathError passthrough
		}
	}
	if err = hooks.sync(f); err != nil {
		return err
	}
	closed = true
	if err = f.Close(); err != nil {
		return err //nolint:wrapcheck // PathError passthrough
	}
	if err = hooks.rename(tmp, path); err != nil {
		return err
	}
	_ = syncDir(dir) // best effort: not every filesystem can fsync a directory
	return nil
}

func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil // directory handles cannot be fsynced on windows
	}
	d, err := os.Open(dir) // #nosec G304 -- directory of a caller-controlled path
	if err != nil {
		return fmt.Errorf("atomicfile: open dir: %w", err)
	}
	err = d.Sync()
	if cerr := d.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("atomicfile: sync dir: %w", err)
	}
	return nil
}

func createTemp(open OpenFunc, dir, base string, perm os.FileMode) (*os.File, string, error) {
	for i := 0; i < 100; i++ {
		var rnd [6]byte
		if _, err := rand.Read(rnd[:]); err != nil {
			return nil, "", fmt.Errorf("atomicfile: temp name: %w", err)
		}
		name := filepath.Join(dir, "."+base+".tmp-"+hex.EncodeToString(rnd[:]))
		f, err := open(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		return f, name, nil
	}
	return nil, "", fmt.Errorf("atomicfile: no free temp name in %s", dir)
}

// resolveLink follows symlinks at path (a dangling one resolves to the file it
// points at) so the replacement lands beside the real file.
func resolveLink(path string) (string, error) {
	for i := 0; i < 40; i++ {
		fi, err := os.Lstat(path)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			return path, nil //nolint:nilerr // missing or regular file: write here
		}
		dest, err := os.Readlink(path)
		if err != nil {
			return "", err //nolint:wrapcheck // PathError passthrough
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(filepath.Dir(path), dest)
		}
		path = dest
	}
	return "", fmt.Errorf("atomicfile: too many symlinks resolving %s", path)
}
