// Cross-process round-trip tests for aps.profile.* bus topics.
//
// Each test:
//  1. starts a kit/bus hub in the test process (httptest WS server),
//  2. subscribes to aps.profile.* on that hub,
//  3. spawns the aps binary as a child process with APS_BUS_ADDR +
//     APS_BUS_TOKEN pointing at the hub,
//  4. asserts the expected event arrives within a deadline.
//
// Closes T-0162. Mirrors the kit/bus network e2e harness
// (network_e2e_test.go) and aps webhook e2e (tests/e2e/webhook_test.go).
//
//go:build bus_e2e

package bus

import (
	"strings"
	"testing"
	"time"

	"hop.top/aps/internal/events"
)

// busPropagationDeadline is the wall-clock budget for a child process
// to start, dial the hub, publish, and let the network adapter forward
// the event back to the test subscriber. Be generous: builds, dialer
// retries, and macOS sandbox latency all eat into this.
const busPropagationDeadline = 5 * time.Second

// TestBusProfileCreated_CrossProcess verifies aps.profile.created is
// published by `aps profile create` (process B) and received by a
// subscriber in the test process (process A) via a real kit/bus hub.
//
// Today this exercises the full path: child boots → reads APS_BUS_ADDR
// → dials WS hub → auths with APS_BUS_TOKEN → publishes locally →
// NetworkAdapter forwards over WS → test process readLoop publishes
// onto local bus → subscriber fires.
//
// If the test fails, the most likely failure modes are: (a) child
// disconnect before async forward completes (publisher exits too
// fast), (b) auth token mismatch, (c) topic-pattern mismatch on the
// subscriber. See tests/e2e/bus/README in the runbook for triage.
func TestBusProfileCreated_CrossProcess(t *testing.T) {
	hub := setupBusHub(t)
	waitFor := hub.subscribe(t, "aps.profile.*")

	home := t.TempDir()
	out, err := runAPSChild(t, home, hub,
		"profile", "create", "noor-test",
		"--display-name", "Noor Test",
		"--email", "noor-test@example.com",
	)
	if err != nil {
		t.Fatalf("aps profile create failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "created successfully") {
		t.Fatalf("expected create confirmation; got:\n%s", out)
	}

	got := waitFor(busPropagationDeadline, 1)
	if len(got) == 0 {
		t.Fatalf("no events received within %s; child output:\n%s", busPropagationDeadline, out)
	}

	ev, ok := findEvent(got, events.TopicProfileCreated)
	if !ok {
		t.Fatalf("aps.profile.created not found in %d received events: %+v", len(got), got)
	}
	if ev.Source == "" {
		t.Errorf("expected non-empty Source on event; got %+v", ev)
	}
	if id := payloadString(ev.Payload, "ProfileID"); id != "noor-test" {
		t.Errorf("payload ProfileID = %q, want %q (full payload: %#v)", id, "noor-test", ev.Payload)
	}
	if email := payloadString(ev.Payload, "Email"); email != "noor-test@example.com" {
		t.Errorf("payload Email = %q, want %q", email, "noor-test@example.com")
	}
	if name := payloadString(ev.Payload, "DisplayName"); name != "Noor Test" {
		t.Errorf("payload DisplayName = %q, want %q", name, "Noor Test")
	}
}

// TestBusProfileUpdated_CrossProcess verifies aps.profile.updated is
// published when a profile is mutated. The mutation path used here is
// `aps profile workspace set`, which calls publishEvent directly with
// Fields=["workspace"] (see internal/cli/profile_workspace.go).
//
// We pre-create the profile in a separate child invocation. That call
// will also emit aps.profile.created on the hub; the subscriber
// pattern matches both, and we filter by topic in the assertion.
func TestBusProfileUpdated_CrossProcess(t *testing.T) {
	hub := setupBusHub(t)
	waitFor := hub.subscribe(t, "aps.profile.*")

	home := t.TempDir()

	// Setup: create the profile we'll update.
	if out, err := runAPSChild(t, home, hub,
		"profile", "create", "noor-update",
		"--email", "noor-update@example.com",
	); err != nil {
		t.Fatalf("setup create failed: %v\n%s", err, out)
	}

	// Action: mutate via workspace set.
	out, err := runAPSChild(t, home, hub,
		"profile", "workspace", "set", "noor-update", "ops-workspace",
	)
	if err != nil {
		t.Fatalf("aps profile workspace set failed: %v\noutput:\n%s", err, out)
	}

	got := waitFor(busPropagationDeadline, 2)
	t.Logf("received %d event(s) on aps.profile.*: %+v", len(got), got)
	ev, ok := findEvent(got, events.TopicProfileUpdated)
	if !ok {
		t.Fatalf("aps.profile.updated not found in %d received events: %+v", len(got), got)
	}
	if id := payloadString(ev.Payload, "ProfileID"); id != "noor-update" {
		t.Errorf("payload ProfileID = %q, want %q", id, "noor-update")
	}
	fields := payloadStringSlice(ev.Payload, "Fields")
	foundWorkspace := false
	for _, f := range fields {
		if f == "workspace" {
			foundWorkspace = true
		}
	}
	if !foundWorkspace {
		t.Errorf("payload Fields = %v, want to contain %q", fields, "workspace")
	}
}

// TestBusProfileDeleted_CrossProcess verifies aps.profile.deleted
// publishes across process boundaries. Uses --yes to bypass the
// interactive confirmation (stdin in the child is not a tty here).
func TestBusProfileDeleted_CrossProcess(t *testing.T) {
	hub := setupBusHub(t)
	waitFor := hub.subscribe(t, "aps.profile.*")

	home := t.TempDir()

	if out, err := runAPSChild(t, home, hub,
		"profile", "create", "noor-delete",
		"--email", "noor-delete@example.com",
	); err != nil {
		t.Fatalf("setup create failed: %v\n%s", err, out)
	}

	out, err := runAPSChild(t, home, hub,
		"profile", "delete", "noor-delete", "--yes",
	)
	if err != nil {
		t.Fatalf("aps profile delete failed: %v\noutput:\n%s", err, out)
	}

	got := waitFor(busPropagationDeadline, 2)
	ev, ok := findEvent(got, events.TopicProfileDeleted)
	if !ok {
		t.Fatalf("aps.profile.deleted not found in %d received events: %+v", len(got), got)
	}
	if id := payloadString(ev.Payload, "ProfileID"); id != "noor-delete" {
		t.Errorf("payload ProfileID = %q, want %q", id, "noor-delete")
	}
}

// TestBusReconnect_AfterHubRestart verifies that a long-lived
// subscriber resumes receiving events after the hub is killed and a
// new hub comes up at the same address.
//
// SKIPPED today: the aps publisher is a short-lived child process —
// each `aps profile <verb>` invocation dials the hub, publishes, and
// exits. There is no long-running aps process to "reconnect".
//
// To make this test meaningful we need either:
//  1. A long-lived aps subscriber daemon (story 051 listener-daemon,
//     not yet implemented), OR
//  2. A way to keep the publisher alive across the hub restart (e.g.
//     `aps daemon` mode), OR
//  3. Move the assertion to the test-process side: keep the test
//     subscriber alive, restart the hub on the same port, fire a new
//     child after the restart, and assert the subscriber still receives.
//     This is testable today but it tests the *test harness's* network
//     adapter reconnect, not aps's — so it does not close the gap
//     T-0162 is filed for.
//
// File a follow-up gap task in tools-showcase-scenarios for option 3
// once 051 lands; until then we record the contract via t.Skip so the
// scaffolding exists and grep-finds when 051 is in flight.
func TestBusReconnect_AfterHubRestart(t *testing.T) {
	t.Skip("requires long-lived aps subscriber (story 051 listener-daemon, not yet implemented); see test doc-comment for the three implementation options and follow-up gap task")
}
