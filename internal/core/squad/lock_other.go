//go:build !unix && !windows

package squad

import (
	"errors"
	"os"
)

func tryLockFile(*os.File) (bool, error) {
	return false, errors.New("file locks unsupported on this platform")
}

func unlockFile(*os.File) {}
