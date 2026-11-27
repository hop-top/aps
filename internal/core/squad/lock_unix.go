//go:build unix

package squad

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// tryLockFile takes an exclusive non-blocking flock; false when held.
func tryLockFile(f *os.File) (bool, error) {
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) // #nosec G115 -- fd fits int
		switch {
		case err == nil:
			return true, nil
		case errors.Is(err, unix.EINTR):
			continue
		case errors.Is(err, unix.EWOULDBLOCK):
			return false, nil
		}
		return false, fmt.Errorf("flock: %w", err)
	}
}

func unlockFile(f *os.File) { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN) } // #nosec G115 -- fd fits int
