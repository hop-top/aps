package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hop.top/aps/internal/core"
	"hop.top/kit/go/storage/kv"
	"hop.top/kit/go/storage/kv/sqlite"
)

// sessionKeyPrefix scopes every session row under a fixed namespace
// inside the shared kv table. It must NOT contain ':' or any other
// character that would clash with the kit prefix-scan semantics.
const sessionKeyPrefix = "session/"

// sessionDBFile is the on-disk filename used for the sqlite backend.
// Sits inside <dataDir>/sessions/ alongside any future per-session
// artifacts.
const sessionDBFile = "registry.db"

// legacyRegistryFile is the pre-kv json blob that prior aps releases
// wrote next to the new sqlite file. When present at startup we
// migrate its contents into kv and remove the legacy file.
const legacyRegistryFile = RegistryFile

func sessionKey(id string) string { return sessionKeyPrefix + id }

// openSessionStore opens (and migrates) the sqlite-backed kv store
// rooted at the supplied directory. The dir is created if missing.
// Callers are responsible for closing the returned store via Close.
func openSessionStore(dir string) (kv.Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}
	store, err := sqlite.New(filepath.Join(dir, sessionDBFile))
	if err != nil {
		return nil, fmt.Errorf("open sessions kv: %w", err)
	}
	return store, nil
}

// migrateLegacyJSONLocked imports any pre-existing registry.json into
// the kv store, then renames the legacy file to a timestamped backup
// (registry.json.migrated-<unix>) instead of deleting it. Safe to call
// multiple times: missing file or empty file are no-ops; rows already
// present in kv are not overwritten so a partial migration can be
// resumed; the rename preserves the original data should the operator
// need to recover from a corrupted kv store.
func (r *SessionRegistry) migrateLegacyJSONLocked(ctx context.Context, dir string) error {
	legacyPath := filepath.Join(dir, legacyRegistryFile)
	// #nosec G304 -- path is constructed from core.GetDataDir(), not user input
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read legacy registry: %w", err)
	}
	if len(data) == 0 {
		return archiveLegacy(legacyPath)
	}
	var legacy map[string]*SessionInfo
	if err := json.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("parse legacy registry: %w", err)
	}
	s, err := r.getStore()
	if err != nil {
		return err
	}
	for id, info := range legacy {
		if info == nil {
			continue
		}
		if _, ok, err := s.Get(ctx, sessionKey(id)); err != nil {
			return fmt.Errorf("kv get during legacy migration: %w", err)
		} else if ok {
			continue
		}
		buf, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("marshal legacy session %q: %w", id, err)
		}
		if err := s.Put(ctx, sessionKey(id), buf); err != nil {
			return fmt.Errorf("kv put during legacy migration: %w", err)
		}
	}
	return archiveLegacy(legacyPath)
}

// archiveLegacy renames a legacy JSON file to a timestamped backup
// (<path>.migrated-<unix>). Missing source is a no-op so concurrent
// migrations don't fight over the rename.
func archiveLegacy(path string) error {
	dest := fmt.Sprintf("%s.migrated-%d", path, time.Now().Unix())
	if err := os.Rename(path, dest); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("archive legacy file: %w", err)
	}
	return nil
}

// kvPutSessionLocked serializes the session and writes it to the
// store. Caller must hold r.mu when an enclosing compound operation
// needs atomicity; for single-key writes the sqlite backend is
// already safe.
func (r *SessionRegistry) kvPutSessionLocked(ctx context.Context, info *SessionInfo) error {
	s, err := r.getStore()
	if err != nil {
		return err
	}
	buf, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := s.Put(ctx, sessionKey(info.ID), buf); err != nil {
		return fmt.Errorf("kv put: %w", err)
	}
	return nil
}

// kvGetSession returns the session row for id, or nil if absent.
func (r *SessionRegistry) kvGetSession(ctx context.Context, id string) (*SessionInfo, error) {
	s, err := r.getStore()
	if err != nil {
		return nil, err
	}
	raw, ok, err := s.Get(ctx, sessionKey(id))
	if err != nil {
		return nil, fmt.Errorf("kv get: %w", err)
	}
	if !ok {
		return nil, nil
	}
	var info SessionInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &info, nil
}

// kvDeleteSession removes a session row.
func (r *SessionRegistry) kvDeleteSession(ctx context.Context, id string) error {
	s, err := r.getStore()
	if err != nil {
		return err
	}
	if err := s.Delete(ctx, sessionKey(id)); err != nil {
		return fmt.Errorf("kv delete: %w", err)
	}
	return nil
}

// kvListSessions returns every session row in the store. Used by
// List/ListByProfile/ListByStatus/ListByTier/ListByType and by the
// reaper's CleanupInactive sweep.
func (r *SessionRegistry) kvListSessions(ctx context.Context) ([]*SessionInfo, error) {
	s, err := r.getStore()
	if err != nil {
		return nil, err
	}
	keys, err := s.List(ctx, sessionKeyPrefix)
	if err != nil {
		return nil, fmt.Errorf("kv list: %w", err)
	}
	out := make([]*SessionInfo, 0, len(keys))
	for _, k := range keys {
		raw, ok, err := s.Get(ctx, k)
		if err != nil {
			return nil, fmt.Errorf("kv get during list: %w", err)
		}
		if !ok {
			continue
		}
		var info SessionInfo
		if err := json.Unmarshal(raw, &info); err != nil {
			return nil, fmt.Errorf("unmarshal session %q: %w", k, err)
		}
		out = append(out, &info)
	}
	return out, nil
}

// ErrRegistryClosed is returned by registry operations invoked after
// Close. A closed registry never reopens its store.
var ErrRegistryClosed = errors.New("session registry closed")

// getStore returns the open kv store, or ErrRegistryClosed / the open
// error. Every store access goes through it so a concurrent Close
// can never surface as a nil dereference.
func (r *SessionRegistry) getStore() (kv.Store, error) {
	r.smu.RLock()
	defer r.smu.RUnlock()
	if r.closed {
		return nil, ErrRegistryClosed
	}
	if r.store == nil {
		if r.storeErr != nil {
			return nil, r.storeErr
		}
		return nil, errors.New("session registry store not open")
	}
	return r.store, nil
}

// ensureStore lazily attaches a kv store to the registry. Called from
// the constructor and on the first mutation when callers built the
// registry via the zero-value path (existing tests do so via
// &SessionRegistry{}). Honours APS_DATA_PATH via core.GetDataDir().
// Returns ErrRegistryClosed after Close.
func (r *SessionRegistry) ensureStore() error {
	r.smu.RLock()
	closed := r.closed
	r.smu.RUnlock()
	if closed {
		return ErrRegistryClosed
	}
	r.storeOnce.Do(func() {
		dataDir, err := core.GetDataDir()
		if err != nil {
			r.setStoreErr(fmt.Errorf("data dir: %w", err))
			return
		}
		dir := filepath.Join(dataDir, SessionsDir)
		store, err := openSessionStore(dir)
		if err != nil {
			r.setStoreErr(err)
			return
		}
		r.smu.Lock()
		if r.closed {
			r.smu.Unlock()
			_ = store.Close()
			return
		}
		r.store = store
		r.smu.Unlock()
		if err := r.migrateLegacyJSONLocked(context.Background(), dir); err != nil {
			r.setStoreErr(err)
		}
	})
	r.smu.RLock()
	defer r.smu.RUnlock()
	if r.closed {
		return ErrRegistryClosed
	}
	return r.storeErr
}

func (r *SessionRegistry) setStoreErr(err error) {
	r.smu.Lock()
	r.storeErr = err
	r.smu.Unlock()
}

// Close stops every reaper started on the registry, waits for it to
// exit, then closes the kv store. Idempotent: repeat calls return nil.
// After Close all operations return ErrRegistryClosed (reads return
// nil/empty). Must not be called from a reaper tick.
func (r *SessionRegistry) Close() error {
	r.smu.Lock()
	if r.closed {
		r.smu.Unlock()
		return nil
	}
	r.closed = true
	stops := r.reaperStops
	r.reaperStops = nil
	r.smu.Unlock()

	for _, stop := range stops {
		stop()
	}
	r.reaperWG.Wait()

	// Drain in-flight compound operations before closing the store.
	r.mu.Lock()
	defer r.mu.Unlock()
	r.smu.Lock()
	store := r.store
	r.store = nil
	r.smu.Unlock()
	if store == nil {
		return nil
	}
	if err := store.Close(); err != nil {
		return fmt.Errorf("kv close: %w", err)
	}
	return nil
}
