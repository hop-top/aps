package session

import (
	"context"
	"fmt"
	"time"
)

// setLastSeenForTest backdates a session's LastSeenAt in the store so
// the reaper can be exercised without sleeping. Lives in a _test.go
// file so it is compiled only under `go test` and never ships in a
// production binary.
func (r *SessionRegistry) setLastSeenForTest(id string, ts time.Time) error {
	if err := r.ensureStore(); err != nil {
		return err
	}
	ctx := context.Background()
	r.mu.Lock()
	defer r.mu.Unlock()
	info, err := r.kvGetSession(ctx, id)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("session %s not found", id)
	}
	info.LastSeenAt = ts
	return r.kvPutSessionLocked(ctx, info)
}
