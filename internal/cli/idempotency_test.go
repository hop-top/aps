package cli

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/core"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/cli/idemstore"
)

// TestIdempotencyStore_DefaultsToSQLite asserts the default backend
// resolves to a sqlite path under the XDG state dir when no
// idempotency.backend is set in config.
func TestIdempotencyStore_DefaultsToSQLite(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	store, path, err := newIdempotencyStore(&core.Config{})
	if err != nil {
		t.Fatalf("newIdempotencyStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	want := filepath.Join(dir, "aps", idemstoreFileName)
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if store == nil {
		t.Error("expected non-nil idemstore.Store")
	}
}

// TestIdempotencyStore_MemoryBackend asserts the memory backend
// switch returns an in-process store with no on-disk artifact.
func TestIdempotencyStore_MemoryBackend(t *testing.T) {
	cfg := &core.Config{
		Idempotency: core.IdempotencyConfig{
			Backend: core.IdempotencyBackendMemory,
		},
	}
	store, path, err := newIdempotencyStore(cfg)
	if err != nil {
		t.Fatalf("newIdempotencyStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if path != "" {
		t.Errorf("path = %q, want \"\" for memory backend", path)
	}

	ctx := context.Background()
	if err := store.Record(ctx, "k", idemstore.Result{Output: []byte("payload")}); err != nil {
		t.Fatalf("record: %v", err)
	}
	got, hit, err := store.Lookup(ctx, "k")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !hit || string(got.Output) != "payload" {
		t.Errorf("Lookup(k) = %+v hit=%v, want payload hit=true", got, hit)
	}
}

// TestIdempotencyStore_UnknownBackend asserts an unknown backend
// surfaces a typed error rather than silently falling through.
func TestIdempotencyStore_UnknownBackend(t *testing.T) {
	cfg := &core.Config{
		Idempotency: core.IdempotencyConfig{Backend: "redis"},
	}
	if _, _, err := newIdempotencyStore(cfg); err == nil {
		t.Fatal("expected error for unknown backend, got nil")
	}
}

// TestIdempotencyStore_CustomPath asserts an explicit Path overrides
// the default XDG location.
func TestIdempotencyStore_CustomPath(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom-idemstore.db")
	cfg := &core.Config{
		Idempotency: core.IdempotencyConfig{Path: custom},
	}
	store, path, err := newIdempotencyStore(cfg)
	if err != nil {
		t.Fatalf("newIdempotencyStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if path != custom {
		t.Errorf("path = %q, want %q", path, custom)
	}
}

// TestIdempotencyStore_CustomTTL asserts a parseable TTL string in
// config flows through to the underlying store.
func TestIdempotencyStore_CustomTTL(t *testing.T) {
	cfg := &core.Config{
		Idempotency: core.IdempotencyConfig{TTL: "30m"},
	}
	d := parseIdempotencyTTL(cfg)
	if d.String() != "30m0s" {
		t.Errorf("parseIdempotencyTTL = %s, want 30m0s", d)
	}

	cfg.Idempotency.TTL = "not-a-duration"
	if d := parseIdempotencyTTL(cfg); d != 0 {
		t.Errorf("parseIdempotencyTTL(bad) = %s, want 0", d)
	}
}

// TestRoot_IdempotencyStoreAttached asserts cli.New installed the
// kit-managed Store on the live root. The store is what kit's
// wrapIdempotencyRunE consumes when --idempotency-key fires.
func TestRoot_IdempotencyStoreAttached(t *testing.T) {
	if root == nil {
		t.Fatal("aps root is nil")
	}
	if root.IdemStore == nil {
		if err := idempotencyHealthErr(); err != nil {
			t.Fatalf("idempotency store unavailable: %v", err)
		}
		t.Fatal("root.IdemStore is nil and no health error was captured")
	}
}

// TestRoot_IdempotencyKeyFlag_OnConditionalCmds asserts kit's
// installIdempotencyKeyFlag auto-registered --idempotency-key on every
// leaf whose annotations are conditional + write-shared|destructive.
// kit walks the tree inside Root.Execute → WrapRunE, so we trigger
// the same walk here to make the assertion deterministic at test
// time (not contingent on a separate Execute() round).
func TestRoot_IdempotencyKeyFlag_OnConditionalCmds(t *testing.T) {
	if root == nil || root.Cmd == nil {
		t.Fatal("aps root is nil")
	}
	root.WrapRunE()

	const flagName = "idempotency-key"

	mutatingNet := map[string]bool{
		"write-shared":       true,
		"destructive":        true,
		"destructive-local":  true,
		"destructive-shared": true,
	}

	type miss struct {
		path    string
		sideEff string
		idempo  string
	}
	var misses []miss
	var visit func(c *cobra.Command)
	visit = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			visit(sub)
		}
		if c.HasSubCommands() || !c.Runnable() || c.Hidden {
			return
		}
		se := c.Annotations["kit/side-effect"]
		idempo := c.Annotations["kit/idempotent"]
		if idempo != "conditional" || !mutatingNet[se] {
			return
		}
		if c.Flags().Lookup(flagName) == nil {
			misses = append(misses, miss{c.CommandPath(), se, idempo})
		}
	}
	visit(root.Cmd)
	for _, m := range misses {
		t.Errorf("cmd %q (side-effect=%s idempotent=%s) missing --%s",
			m.path, m.sideEff, m.idempo, flagName)
	}
}

// TestIdempotency_RecordsAndReplays exercises the round-trip directly
// against the memory backend so the kit middleware contract is
// verified without depending on a live network target. Tests for
// real aps commands (e.g. a2a tasks send) need fixture servers and
// belong in tests/e2e.
func TestIdempotency_RecordsAndReplays(t *testing.T) {
	store := idemstore.Memory()
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	key := "scenario-1"
	want := []byte(`{"status":"ok","artifact":"a2a-msg-001"}`)

	// Pre-condition: lookup miss.
	if _, hit, err := store.Lookup(ctx, key); err != nil || hit {
		t.Fatalf("pre-condition lookup: hit=%v err=%v", hit, err)
	}

	// Adopter records the envelope after a successful invocation.
	if err := store.Record(ctx, key, idemstore.Result{
		Key:      key,
		ExitCode: 0,
		Output:   want,
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	// Replay: second invocation with same key returns recorded payload.
	got, hit, err := store.Lookup(ctx, key)
	if err != nil {
		t.Fatalf("replay lookup: %v", err)
	}
	if !hit {
		t.Fatal("replay miss; want hit for already-recorded key")
	}
	if string(got.Output) != string(want) {
		t.Errorf("replay Output = %q, want %q", got.Output, want)
	}
}

// TestApplyNoRedactToggle_SurfacesIdempotencyHealthOnMutatingLeaves
// proves that a captured store-open failure surfaces on mutating
// commands and stays silent on read-only ones. Without this wiring the
// sentinel `idempotencyOpenErr` is set but never reaches the user;
// kit's wrapIdempotencyRunE returns the original RunE when
// Root.IdemStore is nil, so `--idempotency-key` silently degrades to
// no replay protection.
func TestApplyNoRedactToggle_SurfacesIdempotencyHealthOnMutatingLeaves(t *testing.T) {
	t.Setenv("KIT_POLICY_DISABLE", "1")

	sentinel := errors.New("test-store-open-failure")
	t.Cleanup(func() { idempotencyOpenErr = nil })
	idempotencyOpenErr = sentinel

	readCmd := &cobra.Command{Use: "show"}
	kitcli.SetSideEffect(readCmd, kitcli.SideEffectRead)

	writeCmd := &cobra.Command{Use: "delete"}
	kitcli.SetSideEffect(writeCmd, kitcli.SideEffectDestructive)

	parent := &cobra.Command{Use: "aps"}
	parent.AddCommand(readCmd, writeCmd)

	if err := applyNoRedactToggle(readCmd, nil); err != nil {
		t.Fatalf("read-only cmd surfaced idempotency error: %v", err)
	}

	err := applyNoRedactToggle(writeCmd, nil)
	if err == nil {
		t.Fatal("mutating cmd did not surface idempotency error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error chain missing sentinel: got %v", err)
	}
	if !strings.Contains(err.Error(), "replay store unavailable") {
		t.Errorf("error message missing context: got %q", err.Error())
	}
}
