package storage

import (
	"context"
	"fmt"
	"time"
)

// BackdateForTest rewrites the stored_at timestamp of key to now-age so
// TTL expiry can be exercised without sleeping. sqlstore stamps entries
// with the wall clock at second precision and exposes no clock seam.
func (c *CapabilityCache) BackdateForTest(ctx context.Context, key string, age time.Duration) error {
	res, err := c.store.DB().ExecContext(ctx,
		`update kv set stored_at = ? where key = ?`,
		time.Now().Add(-age).UTC().Format(time.RFC3339), key)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return fmt.Errorf("backdate %q: %d rows updated, want 1", key, n)
	}
	return nil
}
