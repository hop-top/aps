// Package fsperm writes and audits secret-bearing files.
//
// On POSIX the owner-only guarantee is the 0600 mode bit. On Windows Go only
// toggles the read-only attribute, so files inherit whatever ACL the parent
// directory carries; this package instead applies an explicit protected DACL
// granting the current user, SYSTEM and Administrators, and audits the DACL
// rather than mode bits.
package fsperm

import (
	"os"
)

// OpenSecret opens path like os.OpenFile and, on Windows, restricts its ACL
// to the owner-only DACL before the caller can write any data. Callers pass
// 0o600 as perm; it applies on POSIX only (subject to umask, as os.OpenFile).
func OpenSecret(path string, flag int, perm os.FileMode) (*os.File, error) {
	return openSecret(path, flag, perm)
}

// WriteSecret is os.WriteFile for secret material: create or truncate path,
// then write data, with owner-only access enforced on Windows.
func WriteSecret(path string, data []byte, perm os.FileMode) error {
	f, err := OpenSecret(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// CheckSecret audits path. It returns a non-empty warning phrase when access
// is broader than owner-only, "" when the file is acceptable. The phrase reads
// "insecure permissions (...)..." so callers can format
// "Secrets file X has <warning>". A missing file yields an error satisfying
// errors.Is(err, os.ErrNotExist).
func CheckSecret(path string) (string, error) {
	return checkSecret(path)
}
