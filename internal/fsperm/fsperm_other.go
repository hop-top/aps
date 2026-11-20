//go:build !windows

package fsperm

import (
	"fmt"
	"os"
)

func openSecret(path string, flag int, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, flag, perm) // #nosec G304 -- caller-controlled secret path
	return f, err                           //nolint:wrapcheck // PathError passthrough keeps os.IsNotExist semantics
}

func checkSecret(path string) (string, error) {
	info, err := os.Stat(path) // #nosec G304 -- caller-controlled secret path
	if err != nil {
		return "", err //nolint:wrapcheck // PathError passthrough
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		return fmt.Sprintf("insecure permissions (%o). It should be 0600.", mode), nil
	}
	return "", nil
}
