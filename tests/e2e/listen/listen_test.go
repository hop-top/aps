// Cross-process e2e tests for `aps listen` (story 051, T-0097).
//
//go:build listen_e2e

package listen

import (
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestListen_SubscribesAndPrints — happy path:
// publish one event matching the default --topics; assert the listener
// child prints exactly one JSONL record carrying that topic.
func TestListen_SubscribesAndPrints(t *testing.T) {
	hub := setupBusHub(t)
	child := startListen(t, hub, "noor-listen")

	hub.publish(t, "aps.profile.created", map[string]any{
		"ProfileID":   "alice",
		"DisplayName": "Alice",
	})

	lines := child.WaitForLines(1, propagationDeadline)
	if len(lines) == 0 {
		t.Fatalf("no JSONL lines printed within %s\nstderr:\n%s",
			propagationDeadline, child.stderr())
	}
	if !hasTopic(lines, "aps.profile.created") {
		t.Fatalf("expected aps.profile.created in lines; got %+v", lines)
	}
	if lines[0].Profile != "noor-listen" {
		t.Errorf("Profile field = %q, want noor-listen", lines[0].Profile)
	}
}

// TestListen_GracefulShutdown — SIGTERM/SIGINT exits cleanly within the
// drain budget. Acceptance scenario 5 from story 051.
func TestListen_GracefulShutdown(t *testing.T) {
	hub := setupBusHub(t)
	child := startListen(t, hub, "noor-shutdown")

	// Send SIGTERM; expect exit within drainTimeout (3s) + slack.
	if err := child.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("SIGTERM: %v", err)
	}
	exited, err := child.WaitExit(8 * time.Second)
	if !exited {
		t.Fatalf("aps listen did not exit within 8s of SIGTERM\nstderr:\n%s",
			child.stderr())
	}
	if err != nil {
		// signal.NotifyContext + clean RunE return → exit 0 from cobra.
		t.Errorf("expected clean exit, got: %v\nstderr:\n%s", err, child.stderr())
	}
	if !strings.Contains(child.stderr(), "shutdown signal received") {
		t.Errorf("expected 'shutdown signal received' in stderr; got:\n%s",
			child.stderr())
	}
}

// TestListen_ExitAfterEvents — --exit-after-events N causes clean exit
// after N events without an external signal.
func TestListen_ExitAfterEvents(t *testing.T) {
	hub := setupBusHub(t)
	child := startListen(t, hub, "noor-exit", "--exit-after-events", "1")

	hub.publish(t, "aps.profile.created", map[string]any{"ProfileID": "bob"})

	exited, err := child.WaitExit(propagationDeadline)
	if !exited {
		t.Fatalf("aps listen did not exit after 1 event within %s\nstderr:\n%s",
			propagationDeadline, child.stderr())
	}
	if err != nil {
		t.Errorf("expected clean exit, got: %v", err)
	}
	if !strings.Contains(child.stderr(), "exit-after-events=1 reached") {
		t.Errorf("expected exit-after-events trigger in stderr; got:\n%s",
			child.stderr())
	}

	lines := child.Lines()
	if len(lines) < 1 {
		t.Fatalf("expected ≥1 JSONL line, got %d", len(lines))
	}
}

// TestListen_MultipleTopics — subscribing to two patterns delivers
// events from both namespaces.
func TestListen_MultipleTopics(t *testing.T) {
	hub := setupBusHub(t)
	child := startListen(t, hub, "noor-multi", "--topics", "aps.#,tlc.#")

	hub.publish(t, "aps.profile.created", map[string]any{"ProfileID": "carol"})
	hub.publish(t, "tlc.task.assigned", map[string]any{"TaskID": "T-0001"})

	lines := child.WaitForLines(2, propagationDeadline)
	if len(lines) < 2 {
		t.Fatalf("expected ≥2 lines, got %d\nlines:%+v\nstderr:\n%s",
			len(lines), lines, child.stderr())
	}
	if !hasTopic(lines, "aps.profile.created") {
		t.Errorf("missing aps.profile.created in %+v", lines)
	}
	if !hasTopic(lines, "tlc.task.assigned") {
		t.Errorf("missing tlc.task.assigned in %+v", lines)
	}
}

// TestListen_FailSoft — acceptance scenario 4 (handler error must not
// crash the daemon). SKIPPED until T-0152 lands a real handler/dispatch
// surface; today the listener only prints, so there is no handler to
// fail. Also blocked by T-0182 (kit/bus star-relay gap) for any
// cross-process variant that publishes via a third process.
func TestListen_FailSoft(t *testing.T) {
	t.Skip("blocked: needs T-0152 handler dispatch + T-0182 cross-process relay; " +
		"listener currently only prints, no handler error path to assert")
}
