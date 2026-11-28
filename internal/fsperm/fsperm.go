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

	"hop.top/aps/internal/atomicfile"
)

// OpenSecret opens path like os.OpenFile and, on Windows, restricts its ACL
// to the owner-only DACL before the caller can write any data. Callers pass
// 0o600 as perm; it applies on POSIX only (subject to umask, as os.OpenFile).
func OpenSecret(path string, flag int, perm os.FileMode) (*os.File, error) {
	return openSecret(path, flag, perm)
}

// WriteSecret is os.WriteFile for secret material, made atomic: data goes to
// a hidden exclusive temp file beside path (created through OpenSecret, so
// owner-only access holds before any byte is written), is synced, then
// renamed over path; see atomicfile.Write for the guarantees (old or new
// content, temp removed on failure, symlinks preserved, replaced file ends
// with perm rather than its old, possibly broader, mode).
func WriteSecret(path string, data []byte, perm os.FileMode) error {
	return atomicfile.Write(path, data, perm, OpenSecret) //nolint:wrapcheck // error passthrough, same contract as os.WriteFile
}

// WriteSecretNoSync is WriteSecret without the file and directory sync: same
// owner-only temp file and rename, so readers never see a partial file and a
// killed writer cannot truncate path, but the content is not forced to disk
// and may be lost on power failure. For high-frequency writes (queues,
// caches) where per-write fsync costs more than durability is worth.
func WriteSecretNoSync(path string, data []byte, perm os.FileMode) error {
	return atomicfile.WriteNoSync(path, data, perm, OpenSecret) //nolint:wrapcheck // error passthrough, same contract as os.WriteFile
}

// CheckSecret audits path. It returns a non-empty warning phrase when access
// is broader than owner-only, "" when the file is acceptable. The phrase reads
// "insecure permissions (...)..." so callers can format
// "Secrets file X has <warning>". A missing file yields an error satisfying
// errors.Is(err, os.ErrNotExist).
func CheckSecret(path string) (string, error) {
	return checkSecret(path)
}
