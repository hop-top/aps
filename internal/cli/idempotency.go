package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"hop.top/aps/internal/core"

	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/cli/idemstore"
	idemsqlite "hop.top/kit/go/console/cli/idemstore/sqlite"
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
		s, err := idemsqlite.Open(path, ttl)
		if err != nil {
			return nil, path, fmt.Errorf("idempotency: open sqlite store: %w", err)
		}
		return s, path, nil
	default:
		return nil, "", fmt.Errorf("idempotency: unknown backend %q (want %q or %q)",
			backend, core.IdempotencyBackendSQLite, core.IdempotencyBackendMemory)
	}
}

// parseIdempotencyTTL returns the parsed TTL or 0 (which sqlite.Open
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

// lazyIdempotencyStore defers opening the real backend until the first
// Lookup or Record. kit only reaches the store when a command carries
// --idempotency-key, so every other invocation (version, list, show,
// --help) never loads config, creates the state dir or opens sqlite.
// Resolving at first use also means -c/--config overrides parsed by the
// persistent pre-run hook are in effect when the backend is chosen.
type lazyIdempotencyStore struct {
	once  sync.Once
	store idemstore.Store
	err   error
}

func (l *lazyIdempotencyStore) open() (idemstore.Store, error) {
	l.once.Do(func() {
		cfg, _ := core.LoadConfig()
		l.store, _, l.err = newIdempotencyStore(cfg)
		if l.err != nil {
			idempotencyOpenErr = l.err
		}
	})
	return l.store, l.err
}

func (l *lazyIdempotencyStore) Lookup(ctx context.Context, key string) (idemstore.Result, bool, error) {
	s, err := l.open()
	if err != nil {
		return idemstore.Result{}, false, err
	}
	return s.Lookup(ctx, key)
}

func (l *lazyIdempotencyStore) Record(ctx context.Context, key string, r idemstore.Result) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	return s.Record(ctx, key, r)
}

// Close releases the backend if it was ever opened; it never opens one.
func (l *lazyIdempotencyStore) Close() error {
	if l.store == nil {
		return nil
	}
	return l.store.Close()
}

// withIdempotencyStore is the cli.New opt that installs the kit
// idempotency replay backend on Root. The backend is opened lazily (see
// lazyIdempotencyStore). A failure to open does not refuse invocations
// that do not use --idempotency-key; it is captured and surfaced by
// applyNoRedactToggle on a mutating leaf that supplies a key, leaving
// --help / completion / read-only paths usable for recovery.
func withIdempotencyStore() func(*kitcli.Root) {
	return func(r *kitcli.Root) {
		idempotencyStore = &lazyIdempotencyStore{}
		r.IdemStore = idempotencyStore
	}
}

// probeIdempotencyStore forces the lazy store open so an open failure
// is captured in idempotencyOpenErr before the command runs. Called
// only for invocations that carry an idempotency key.
func probeIdempotencyStore() {
	if l := idempotencyStore; l != nil {
		_, _ = l.open()
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

// idempotencyStore is the lazy store installed on Root. Held here (not
// read back from root.IdemStore) because the pre-run hook that probes it
// is declared inside the root initializer; referencing root there would
// be an initialization cycle.
var idempotencyStore *lazyIdempotencyStore
