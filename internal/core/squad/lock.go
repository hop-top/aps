package squad

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// lockName is the store-level lock file inside the squads directory. It is a
// plain file, so reload (which only reads directories) never sees it, and it
// is never renamed over, unlike manifests.
const lockName = ".store.lock"

const (
	defaultLockTimeout = 5 * time.Second
	lockPollInterval   = 10 * time.Millisecond
)

// procLocks serialises holders inside this process, keyed by lock path. It
// is held in addition to the file lock so exclusion between Managers of one
// process does not depend on per-platform flock semantics for handles opened
// by the same process.
var procLocks sync.Map // string -> chan struct{} (capacity 1)

func procLock(path string) chan struct{} {
	c, _ := procLocks.LoadOrStore(path, make(chan struct{}, 1))
	return c.(chan struct{})
}

// acquireStoreLock takes the exclusive store lock for dir, creating dir if
// needed, polling until timeout. The returned func releases it and is safe
// to call once. Cross-process exclusion uses an advisory OS file lock, which
// the kernel drops if the holder dies.
func acquireStoreLock(dir string, timeout time.Duration) (func(), error) {
	if timeout <= 0 {
		timeout = defaultLockTimeout
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- squad data is not secret; mode matches the profile store
		return nil, fmt.Errorf("create squads dir: %w", err)
	}
	path := filepath.Join(dir, lockName)
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(lockPollInterval)
	defer tick.Stop()
	timedOut := func() error {
		return fmt.Errorf("timed out after %s waiting for squad store lock %s (held by another process)", timeout, path)
	}

	proc := procLock(path)
	select {
	case proc <- struct{}{}:
	case <-deadline.C:
		return nil, timedOut()
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644) // #nosec G302 G304 -- under squads dir; mode matches the profile store
	if err != nil {
		<-proc
		return nil, fmt.Errorf("open squad store lock %s: %w", path, err)
	}
	for {
		ok, err := tryLockFile(f)
		if err != nil {
			_ = f.Close()
			<-proc
			return nil, fmt.Errorf("lock squad store %s: %w", path, err)
		}
		if ok {
			return func() {
				unlockFile(f)
				_ = f.Close()
				<-proc
			}, nil
		}
		select {
		case <-deadline.C:
			_ = f.Close()
			<-proc
			return nil, timedOut()
		case <-tick.C:
		}
	}
}
