package session

import (
	"bytes"
	"strings"
	"testing"

	coresession "hop.top/aps/internal/core/session"
	"hop.top/kit/go/console/progress"
)

// TestTerminateSession_EmitsProgress drives terminateSession against a
// fake session (no PID, no tmux socket) so the kill path is a no-op and
// the function reaches the OK emit. We assert the terminate phase + an
// ok:true completion event surface through the progress reporter.
func TestTerminateSession_EmitsProgress(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	registry := coresession.GetRegistry()
	sess := &coresession.SessionInfo{
		ID:        "fake-progress-sess",
		ProfileID: "noor",
		Status:    coresession.SessionActive,
	}
	if err := registry.Register(sess); err != nil {
		t.Fatalf("Register: %v", err)
	}

	var pbuf bytes.Buffer
	ctx := progress.WithReporter(t.Context(), progress.JSONL(&pbuf))

	if err := terminateSession(ctx, sess, false, 1); err != nil {
		t.Fatalf("terminateSession: %v", err)
	}

	got := pbuf.String()
	if !strings.Contains(got, `"phase":"terminate"`) {
		t.Errorf("expected terminate phase, got: %q", got)
	}
	if !strings.Contains(got, `"ok":true`) {
		t.Errorf("expected ok:true on completion, got: %q", got)
	}
}

// TestTerminateSession_ForceUsesKillPhase verifies --force surfaces a
// distinct kill phase, matching the human-facing semantics of the cmd.
func TestTerminateSession_ForceUsesKillPhase(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	registry := coresession.GetRegistry()
	sess := &coresession.SessionInfo{
		ID:        "fake-progress-sess-force",
		ProfileID: "noor",
		Status:    coresession.SessionActive,
	}
	if err := registry.Register(sess); err != nil {
		t.Fatalf("Register: %v", err)
	}

	var pbuf bytes.Buffer
	ctx := progress.WithReporter(t.Context(), progress.JSONL(&pbuf))

	if err := terminateSession(ctx, sess, true, 1); err != nil {
		t.Fatalf("terminateSession: %v", err)
	}

	if !strings.Contains(pbuf.String(), `"phase":"kill"`) {
		t.Errorf("expected kill phase under --force, got: %q", pbuf.String())
	}
}
