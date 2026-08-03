package session

import (
	"context"
	"strings"
	"testing"

	"hop.top/kit/go/core/uxp/invoke"
	xrr "hop.top/xrr"

	"hop.top/aps/internal/core/cmdrun"
	coresession "hop.top/aps/internal/core/session"
)

// Coverage for the tmux command lines aps builds.
//
// These are argv-assembly tests in the same spirit as the adapter
// backend-script tests, but tmux is invoked from Go rather than from a
// bash backend with a bin_env_var override. The seam is cmdrun.Runner:
// production wires cmdrun.Exec(), these tests wire a recording runner,
// so what aps asked tmux to do is observable without a live server.
//
// Flag semantics asserted here were verified against tmux 3.7b:
//   -S <path>   server socket; MUST precede the subcommand
//   capture-pane -t <target>   target-pane
//   capture-pane -S <line>     start-line ("-" = start of history)
//   capture-pane -e            include escape sequences
// capture-pane has no timestamp flag.

// recordingRunner captures specs and returns a canned result.
type recordingRunner struct {
	specs  []invoke.CommandSpec
	result invoke.Result
}

func (r *recordingRunner) Run(_ context.Context, spec invoke.CommandSpec) (invoke.Result, error) {
	r.specs = append(r.specs, spec)
	res := r.result
	res.Command = spec
	return res, nil
}

// argvOf renders a captured spec as bracketed argv for legible
// failure messages.
func argvOf(spec invoke.CommandSpec) string {
	var b strings.Builder
	b.WriteString("[" + spec.Path + "]")
	for _, a := range spec.Args {
		b.WriteString(" [" + a + "]")
	}
	return b.String()
}

// assertSpec compares a captured command line against want.
// Every command asserted here is tmux, so the binary is implicit;
// the args are what vary and what regress.
func assertSpec(t *testing.T, spec invoke.CommandSpec, wantArgs ...string) {
	t.Helper()
	got := argvOf(spec)
	want := argvOf(invoke.CommandSpec{Path: "tmux", Args: wantArgs})
	if got != want {
		t.Errorf("command line mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestTmuxCaptureSpec_SocketPrecedesSubcommand is the core regression.
//
// -S is a server option: `tmux -S <sock> capture-pane ...`. Inside
// capture-pane, -S means start-line and -t means target-pane, so a
// socket passed after the verb is silently reinterpreted. The previous
// implementation emitted `capture-pane -p -S <sock> -t <id>`, which
// made tmux read <sock> as a start-line number.
func TestTmuxCaptureSpec_SocketPrecedesSubcommand(t *testing.T) {
	t.Parallel()

	spec := tmuxCaptureSpec("/tmp/aps.sock", "sess-1", "", false)

	if len(spec.Args) < 3 || spec.Args[0] != "-S" || spec.Args[1] != "/tmp/aps.sock" {
		t.Fatalf("socket is not the leading server option: %s", argvOf(spec))
	}
	if spec.Args[2] != "capture-pane" {
		t.Errorf("subcommand does not follow the socket: %s", argvOf(spec))
	}
}

// TestTmuxCaptureSpec_Default covers the no-tail case: visible pane
// with escape sequences.
func TestTmuxCaptureSpec_Default(t *testing.T) {
	t.Parallel()

	spec := tmuxCaptureSpec("/tmp/aps.sock", "sess-1", "", false)

	assertSpec(t, spec,
		"-S", "/tmp/aps.sock", "capture-pane", "-p", "-t", "sess-1", "-e")
}

// TestTmuxCaptureSpec_TailAll covers "all": start-of-history is -S -,
// not the socket path.
func TestTmuxCaptureSpec_TailAll(t *testing.T) {
	t.Parallel()

	spec := tmuxCaptureSpec("/tmp/aps.sock", "sess-1", "all", false)

	assertSpec(t, spec,
		"-S", "/tmp/aps.sock", "capture-pane", "-p", "-t", "sess-1", "-S", "-")
}

// TestTmuxCaptureSpec_TailN covers a numeric tail: N lines back is
// -S -N. The previous implementation used -E N (end-line), which
// bounds the wrong end of the range.
func TestTmuxCaptureSpec_TailN(t *testing.T) {
	t.Parallel()

	spec := tmuxCaptureSpec("/tmp/aps.sock", "sess-1", "50", false)

	assertSpec(t, spec,
		"-S", "/tmp/aps.sock", "capture-pane", "-p", "-t", "sess-1", "-S", "-50")
}

// TestTmuxCaptureSpec_TimestampsEmitsNoFlag pins the unsupported
// option. capture-pane has no timestamp flag; the previous code
// appended "-t", which is target-pane and consumed the following "-S"
// as its value. Emitting nothing (and warning at the caller) is the
// correct handling — emitting some other flag would repeat the bug.
func TestTmuxCaptureSpec_TimestampsEmitsNoFlag(t *testing.T) {
	t.Parallel()

	with := tmuxCaptureSpec("/tmp/aps.sock", "sess-1", "", true)
	without := tmuxCaptureSpec("/tmp/aps.sock", "sess-1", "", false)

	if argvOf(with) != argvOf(without) {
		t.Errorf("--timestamps changed the command line\n with: %s\nwithout: %s",
			argvOf(with), argvOf(without))
	}
	if tmuxSupportsTimestamps() {
		t.Error("tmuxSupportsTimestamps() reports true; capture-pane has no such flag")
	}
}

// TestTmuxCaptureSpec_TargetIsPairedWithT guards the target flag from
// drifting away from its value — the failure mode that made the
// original bug invisible.
func TestTmuxCaptureSpec_TargetIsPairedWithT(t *testing.T) {
	t.Parallel()

	for _, tail := range []string{"", "all", "50"} {
		spec := tmuxCaptureSpec("/tmp/aps.sock", "sess-1", tail, false)
		if !hasPair(spec.Args, "-t", "sess-1") {
			t.Errorf("tail=%q: target not paired with -t: %s", tail, argvOf(spec))
		}
	}
}

// TestTmuxPipePaneSpec covers follow mode.
func TestTmuxPipePaneSpec(t *testing.T) {
	t.Parallel()

	spec := tmuxPipePaneSpec("/tmp/aps.sock", "sess-1")

	assertSpec(t, spec,
		"-S", "/tmp/aps.sock", "pipe-pane", "-t", "sess-1", "cat")
}

// TestTmuxKillSpec covers the teardown command line.
func TestTmuxKillSpec(t *testing.T) {
	t.Parallel()

	spec := tmuxKillSpec("/tmp/aps.sock", "sess-1")

	assertSpec(t, spec,
		"-S", "/tmp/aps.sock", "kill-session", "-t", "sess-1")
}

// TestKillTmuxSession_BenignErrorIsNotAFailure covers the race the
// teardown path exists to tolerate: tmux exits non-zero when the
// session is already gone, which is the desired end state.
func TestKillTmuxSession_BenignErrorIsNotAFailure(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{result: invoke.Result{
		Code:   1,
		Stderr: []byte("no server running on /tmp/aps.sock"),
	}}

	if err := killTmuxSessionWith(context.Background(), runner,
		"/tmp/aps.sock", "sess-1"); err != nil {
		t.Fatalf("benign already-gone error surfaced as a failure: %v", err)
	}
	if len(runner.specs) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runner.specs))
	}
	assertSpec(t, runner.specs[0],
		"-S", "/tmp/aps.sock", "kill-session", "-t", "sess-1")
}

// TestKillTmuxSession_RealErrorPropagates covers the other side: a
// non-benign failure must not be swallowed, or a session that is still
// running would be reported as terminated.
func TestKillTmuxSession_RealErrorPropagates(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{result: invoke.Result{
		Code:   1,
		Stderr: []byte("permission denied"),
	}}

	err := killTmuxSessionWith(context.Background(), runner, "/tmp/aps.sock", "sess-1")
	if err == nil {
		t.Fatal("a real teardown failure was swallowed")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error does not carry tmux's diagnostic: %v", err)
	}
}

// TestCaptureTmuxLogs_RecordsToCassette exercises the full path
// through cmdrun with an xrr-backed runner, proving the command line
// aps issues is what lands in the cassette.
func TestCaptureTmuxLogs_RecordsToCassette(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sess := &coresession.SessionInfo{ID: "sess-1", TmuxSocket: "/tmp/aps.sock"}

	// Record against a stub runner so no tmux server is needed; the
	// cassette still keys on the real argv.
	inner := &recordingRunner{result: invoke.Result{Code: 0, Stdout: []byte("pane text\n")}}
	rec := cmdrun.NewRecorder(xrr.NewSession(xrr.ModeRecord, xrr.NewFileCassette(dir)), inner)

	if err := captureTmuxLogs(context.Background(), rec, sess, false, "all", false); err != nil {
		t.Fatalf("capture: %v", err)
	}

	if len(inner.specs) != 1 {
		t.Fatalf("expected 1 tmux call, got %d", len(inner.specs))
	}
	assertSpec(t, inner.specs[0],
		"-S", "/tmp/aps.sock", "capture-pane", "-p", "-t", "sess-1", "-S", "-")

	// Replay with no inner runner: a hit proves the cassette carried
	// the result, and would miss if the argv had changed.
	replay := cmdrun.NewRecorder(xrr.NewSession(xrr.ModeReplay, xrr.NewFileCassette(dir)), nil)
	if err := captureTmuxLogs(context.Background(), replay, sess, false, "all", false); err != nil {
		t.Fatalf("replay: %v", err)
	}
}

// TestCaptureTmuxLogs_FollowIssuesPipePane covers follow mode issuing
// the second command after the capture.
func TestCaptureTmuxLogs_FollowIssuesPipePane(t *testing.T) {
	t.Parallel()

	sess := &coresession.SessionInfo{ID: "sess-1", TmuxSocket: "/tmp/aps.sock"}
	runner := &recordingRunner{result: invoke.Result{Code: 0}}

	if err := captureTmuxLogs(context.Background(), runner, sess, true, "", false); err != nil {
		t.Fatalf("capture with follow: %v", err)
	}

	if len(runner.specs) != 2 {
		t.Fatalf("expected capture + pipe-pane, got %d", len(runner.specs))
	}
	assertSpec(t, runner.specs[1],
		"-S", "/tmp/aps.sock", "pipe-pane", "-t", "sess-1", "cat")
}

// TestCaptureTmuxLogs_NonZeroExitIsReported covers a failing capture
// surfacing rather than yielding a silent empty log.
func TestCaptureTmuxLogs_NonZeroExitIsReported(t *testing.T) {
	t.Parallel()

	sess := &coresession.SessionInfo{ID: "sess-1", TmuxSocket: "/tmp/aps.sock"}
	runner := &recordingRunner{result: invoke.Result{
		Code:   1,
		Stderr: []byte("can't find pane: sess-1"),
	}}

	err := captureTmuxLogs(context.Background(), runner, sess, false, "", false)
	if err == nil {
		t.Fatal("a failing capture was reported as success")
	}
	if !strings.Contains(err.Error(), "can't find pane") {
		t.Errorf("error does not carry tmux's diagnostic: %v", err)
	}
}

// hasPair reports whether flag appears immediately followed by value.
func hasPair(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
