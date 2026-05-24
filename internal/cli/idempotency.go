package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"hop.top/aps/internal/core"

	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/cli/idemstore"
)

// idemstoreFileName is the on-disk filename for the sqlite-backed
// idempotency store. Kept distinct from any other aps sqlite db so
// $XDG_STATE_HOME/aps/idemstore.db is unambiguously the replay log.
const idemstoreFileName = "idemstore.db"

// newIdempotencyStore opens the kit-managed idempotency Store for aps
// per the profile/global config in cfg. The selection rules are:
//
//   - cfg.Idempotency.Backend == "memory" → in-process store.
//   - cfg.Idempotency.Backend == "sqlite" (default; empty also maps
//     here) → sqlite at cfg.Idempotency.Path, or
//     $XDG_STATE_HOME/aps/idemstore.db when Path is empty.
//
// TTL parses cfg.Idempotency.TTL as a time.Duration. Empty / zero /
// unparseable values fall back to idemstore.DefaultTTL.
//
// Returns a Store the caller must Close, plus the resolved path for
// diagnostic logging (empty string for the memory backend).
func newIdempotencyStore(cfg *core.Config) (idemstore.Store, string, error) {
	backend := core.IdempotencyBackendSQLite
	if cfg != nil && cfg.Idempotency.Backend != "" {
		backend = cfg.Idempotency.Backend
	}

	switch backend {
	case core.IdempotencyBackendMemory:
		return idemstore.Memory(), "", nil
	case core.IdempotencyBackendSQLite, "":
		path := ""
		if cfg != nil {
			path = cfg.Idempotency.Path
		}
		if path == "" {
			stateDir, err := core.GetStateDir()
			if err != nil {
				return nil, "", fmt.Errorf("idempotency: resolve state dir: %w", err)
			}
			if err := core.EnsureDir(stateDir); err != nil {
				return nil, "", fmt.Errorf("idempotency: ensure state dir: %w", err)
			}
			path = filepath.Join(stateDir, idemstoreFileName)
		}
		ttl := parseIdempotencyTTL(cfg)
		s, err := idemstore.OpenSQLite(path, ttl)
		if err != nil {
			return nil, path, fmt.Errorf("idempotency: open sqlite store: %w", err)
		}
		return s, path, nil
	default:
		return nil, "", fmt.Errorf("idempotency: unknown backend %q (want %q or %q)",
			backend, core.IdempotencyBackendSQLite, core.IdempotencyBackendMemory)
	}
}

// parseIdempotencyTTL returns the parsed TTL or 0 (which OpenSQLite
// interprets as idemstore.DefaultTTL).
func parseIdempotencyTTL(cfg *core.Config) time.Duration {
	if cfg == nil || cfg.Idempotency.TTL == "" {
		return 0
	}
	d, err := time.ParseDuration(cfg.Idempotency.TTL)
	if err != nil {
		return 0
	}
	return d
}

// withIdempotencyStore is the cli.New opt that installs the kit
// idempotency replay backend on Root. Failure to open the store does
// not panic at construction time — that would refuse all invocations
// when the on-disk db is malformed. Instead the error is captured and
// surfaced once via PrePersistentRunE, leaving --help / completion /
// other read-only paths usable for recovery.
func withIdempotencyStore() func(*kitcli.Root) {
	return func(r *kitcli.Root) {
		cfg, _ := core.LoadConfig()
		store, _, err := newIdempotencyStore(cfg)
		if err != nil {
			idempotencyOpenErr = err
			return
		}
		r.IdemStore = store
	}
}

// idempotencyOpenErr captures any error encountered while opening the
// kit-managed idempotency store at root construction. Read by
// applyIdempotencyHealth (the PrePersistentRunE hook) so the failure
// surfaces on the first mutating invocation rather than at boot.
var idempotencyOpenErr error

// idempotencyHealthErr surfaces the captured open error to the caller.
// Tests use this to assert that an open failure was observed.
func idempotencyHealthErr() error {
	if idempotencyOpenErr == nil {
		return nil
	}
	return errors.Join(errors.New("idempotency: replay store unavailable"), idempotencyOpenErr)
}
