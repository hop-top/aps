// Cross-process round-trip tests for aps.adapter.* bus topics.
//
// Each test:
//  1. starts a kit/bus hub in the test process (httptest WS server),
//  2. subscribes to aps.adapter.* on that hub,
//  3. spawns the aps binary as a child process with APS_BUS_ADDR +
//     APS_BUS_TOKEN pointing at the hub,
//  4. asserts the expected event arrives within a deadline.
//
// Closes T-0163. Mirrors profile_roundtrip_test.go (T-0162) against the
// adapter lifecycle topics emitted by `aps adapter link/unlink`. T-0176
// drain fix (commit 4c79fac) is a prerequisite — without it, the child
// can exit before the async forwarder flushes and the event is lost.
//
//go:build bus_e2e

package bus

import (
	"strings"
	"testing"

	"hop.top/aps/internal/events"
)

// TestBusAdapterLinked_CrossProcess verifies aps.adapter.linked is
// published by `aps adapter link` (process B) and received by a
// subscriber in the test process (process A) via a real kit/bus hub.
//
// Setup creates an adapter via `aps adapter create --type protocol` so
// that link has a valid device to operate on. LinkAdapter does not
// validate that the profile exists (see internal/core/adapter/manager.go),
// so no profile creation is required for the link path itself.
func TestBusAdapterLinked_CrossProcess(t *testing.T) {
	hub := setupBusHub(t)
	waitFor := hub.subscribe(t, "aps.adapter.*")

	home := t.TempDir()

	// Setup: create an adapter to link.
	if out, err := runAPSChild(t, home, hub,
		"adapter", "create", "test-protocol",
		"--type", "protocol",
	); err != nil {
		t.Fatalf("setup adapter create failed: %v\noutput:\n%s", err, out)
	}

	// Action: link the adapter to a profile id.
	out, err := runAPSChild(t, home, hub,
		"adapter", "link", "test-protocol",
		"--profile", "noor-link",
	)
	if err != nil {
		t.Fatalf("aps adapter link failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Linked") {
		t.Fatalf("expected link confirmation; got:\n%s", out)
	}

	got := waitFor(busPropagationDeadline, 1)
	if len(got) == 0 {
		t.Fatalf("no events received within %s; child output:\n%s", busPropagationDeadline, out)
	}

	ev, ok := findEvent(got, events.TopicAdapterLinked)
	if !ok {
		t.Fatalf("aps.adapter.linked not found in %d received events: %+v", len(got), got)
	}
	if ev.Source == "" {
		t.Errorf("expected non-empty Source on event; got %+v", ev)
	}
	if id := payloadString(ev.Payload, "ProfileID"); id != "noor-link" {
		t.Errorf("payload ProfileID = %q, want %q (full payload: %#v)", id, "noor-link", ev.Payload)
	}
	if aid := payloadString(ev.Payload, "AdapterID"); aid != "test-protocol" {
		t.Errorf("payload AdapterID = %q, want %q", aid, "test-protocol")
	}
	if at := payloadString(ev.Payload, "AdapterType"); at != "protocol" {
		t.Errorf("payload AdapterType = %q, want %q", at, "protocol")
	}
}

// TestBusAdapterUnlinked_CrossProcess verifies aps.adapter.unlinked is
// published when an adapter is unlinked from a profile. The setup
// invocations (create + link) emit aps.adapter.linked on the hub; the
// subscriber pattern matches both, and we filter by topic in the
// assertion.
//
// Persistence prerequisite (T-0181): AdapterManifest now carries a
// `linked_to` field; SaveAdapter writes it, loadAdapterFromPath reads
// it. Without that, `aps adapter link` (process B) writes the manifest
// without LinkedTo, then `aps adapter unlink` (process C) reloads with
// LinkedTo=[] and fails IsLinkedToProfile before reaching publishEvent.
// In-process tests miss the bug because the same *Adapter pointer
// carries LinkedTo across calls.
func TestBusAdapterUnlinked_CrossProcess(t *testing.T) {
	hub := setupBusHub(t)
	waitFor := hub.subscribe(t, "aps.adapter.*")

	home := t.TempDir()

	// Setup: create an adapter and link it so it can be unlinked.
	if out, err := runAPSChild(t, home, hub,
		"adapter", "create", "test-protocol",
		"--type", "protocol",
	); err != nil {
		t.Fatalf("setup adapter create failed: %v\noutput:\n%s", err, out)
	}
	if out, err := runAPSChild(t, home, hub,
		"adapter", "link", "test-protocol",
		"--profile", "noor-unlink",
	); err != nil {
		t.Fatalf("setup adapter link failed: %v\noutput:\n%s", err, out)
	}

	// Action: unlink.
	out, err := runAPSChild(t, home, hub,
		"adapter", "unlink", "test-protocol",
		"--profile", "noor-unlink",
	)
	if err != nil {
		t.Fatalf("aps adapter unlink failed: %v\noutput:\n%s", err, out)
	}

	got := waitFor(busPropagationDeadline, 2)
	t.Logf("received %d event(s) on aps.adapter.*: %+v", len(got), got)
	ev, ok := findEvent(got, events.TopicAdapterUnlinked)
	if !ok {
		t.Fatalf("aps.adapter.unlinked not found in %d received events: %+v", len(got), got)
	}
	if id := payloadString(ev.Payload, "ProfileID"); id != "noor-unlink" {
		t.Errorf("payload ProfileID = %q, want %q", id, "noor-unlink")
	}
	if aid := payloadString(ev.Payload, "AdapterID"); aid != "test-protocol" {
		t.Errorf("payload AdapterID = %q, want %q", aid, "test-protocol")
	}
	if at := payloadString(ev.Payload, "AdapterType"); at != "protocol" {
		t.Errorf("payload AdapterType = %q, want %q", at, "protocol")
	}
}
