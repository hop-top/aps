package cmdrun_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xrr "hop.top/xrr"

	"hop.top/aps/internal/core/cmdrun"
)

// Exercises the Runner seam itself: that a real subprocess result is
// reported faithfully, that a non-zero exit is a Result rather than an
// error, and that an xrr-wrapped Runner records a cassette whose key
// IS the command line and replays it without spawning anything.

func TestExec_CapturesStdoutAndExitZero(t *testing.T) {
	t.Parallel()

	res, err := cmdrun.Exec().Run(context.Background(),
		cmdrun.Spec("echo", "hello"))
	if err != nil {
		t.Fatalf("run echo: %v", err)
	}
	if res.Code != 0 {
		t.Errorf("exit code = %d, want 0", res.Code)
	}
	if got := strings.TrimSpace(string(res.Stdout)); got != "hello" {
		t.Errorf("stdout = %q, want %q", got, "hello")
	}
}

// TestExec_NonZeroExitIsNotAnError pins the contract the tmux paths
// depend on: kill-session returns non-zero when the session is already
// gone, and callers must be able to inspect stderr to decide whether
// that was benign. An error return would collapse that distinction.
func TestExec_NonZeroExitIsNotAnError(t *testing.T) {
	t.Parallel()

	res, err := cmdrun.Exec().Run(context.Background(), cmdrun.Spec("false"))
	if err != nil {
		t.Fatalf("non-zero exit surfaced as an error: %v", err)
	}
	if res.Code != 1 {
		t.Errorf("exit code = %d, want 1", res.Code)
	}
}

// TestExec_MissingBinaryIsAnError covers the other side: a command
// that never ran at all is an error, with code -1 rather than a
// plausible-looking 0.
func TestExec_MissingBinaryIsAnError(t *testing.T) {
	t.Parallel()

	res, err := cmdrun.Exec().Run(context.Background(),
		cmdrun.Spec("definitely-not-a-real-binary-aps-test"))
	if err == nil {
		t.Fatalf("missing binary did not error; got code %d", res.Code)
	}
	if res.Code != -1 {
		t.Errorf("exit code = %d, want -1 for a command that never ran", res.Code)
	}
}

// TestExec_CapturesStderr covers stderr landing in Result rather than
// leaking to the parent's terminal — the tmux teardown paths parse it
// to classify benign errors.
func TestExec_CapturesStderr(t *testing.T) {
	t.Parallel()

	res, err := cmdrun.Exec().Run(context.Background(),
		cmdrun.Spec("sh", "-c", "echo oops >&2; exit 3"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Code != 3 {
		t.Errorf("exit code = %d, want 3", res.Code)
	}
	if got := strings.TrimSpace(string(res.Stderr)); got != "oops" {
		t.Errorf("stderr = %q, want %q", got, "oops")
	}
}

// TestRecorder_RoundTrip is the core of the xrr adoption: record a
// command, then replay it with NO inner runner at all. A replay that
// succeeds proves the cassette carried the full result and that
// nothing was spawned the second time.
func TestRecorder_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	spec := cmdrun.Spec("echo", "recorded-output")

	recSession := xrr.NewSession(xrr.ModeRecord, xrr.NewFileCassette(dir))
	recorded, err := cmdrun.NewRecorder(recSession, cmdrun.Exec()).
		Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if got := strings.TrimSpace(string(recorded.Stdout)); got != "recorded-output" {
		t.Fatalf("recorded stdout = %q", got)
	}

	// nil inner runner: any attempt to actually execute fails loudly.
	replaySession := xrr.NewSession(xrr.ModeReplay, xrr.NewFileCassette(dir))
	replayed, err := cmdrun.NewRecorder(replaySession, nil).
		Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	if string(replayed.Stdout) != string(recorded.Stdout) {
		t.Errorf("replayed stdout = %q, want %q",
			replayed.Stdout, recorded.Stdout)
	}
	if replayed.Code != recorded.Code {
		t.Errorf("replayed exit code = %d, want %d", replayed.Code, recorded.Code)
	}
}

// TestRecorder_CassetteRecordsArgv is the property that makes xrr the
// right tool for this job rather than hand-rolled argv assertions: the
// persisted artifact contains the command line verbatim, so reviewing
// a cassette diff is reviewing what aps asked the binary to do.
func TestRecorder_CassetteRecordsArgv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	spec := cmdrun.Spec("echo", "-S", "/tmp/sock", "kill-session")

	session := xrr.NewSession(xrr.ModeRecord, xrr.NewFileCassette(dir))
	if _, err := cmdrun.NewRecorder(session, cmdrun.Exec()).
		Run(context.Background(), spec); err != nil {
		t.Fatalf("record: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read cassette dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no cassette was written to %s", dir)
	}

	raw, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read cassette: %v", err)
	}

	for _, want := range []string{"-S", "/tmp/sock", "kill-session"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("cassette does not record argv element %q:\n%s", want, raw)
		}
	}
}

// TestRecorder_ArgvChangeMissesCassette is the regression property:
// replaying a command whose argv differs from what was recorded must
// fail rather than silently returning the old result. This is what
// turns a cassette into a lock on the command line.
func TestRecorder_ArgvChangeMissesCassette(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	recSession := xrr.NewSession(xrr.ModeRecord, xrr.NewFileCassette(dir))
	if _, err := cmdrun.NewRecorder(recSession, cmdrun.Exec()).
		Run(context.Background(), cmdrun.Spec("echo", "original")); err != nil {
		t.Fatalf("record: %v", err)
	}

	replaySession := xrr.NewSession(xrr.ModeReplay, xrr.NewFileCassette(dir))
	_, err := cmdrun.NewRecorder(replaySession, nil).
		Run(context.Background(), cmdrun.Spec("echo", "changed"))
	if err == nil {
		t.Fatal("replaying a changed argv hit the cassette; " +
			"an argv change must miss")
	}
}

// TestRecorder_StreamRefused pins that an interactive command cannot
// be recorded. Falling through to the real binary would spawn a
// terminal mid-test; failing is the safe outcome.
func TestRecorder_StreamRefused(t *testing.T) {
	t.Parallel()

	session := xrr.NewSession(xrr.ModeRecord, xrr.NewFileCassette(t.TempDir()))
	rec := cmdrun.NewRecorder(session, cmdrun.Exec())

	err := rec.Stream(context.Background(), cmdrun.Spec("tmux", "attach"),
		nil, os.Stdout, os.Stderr)
	if err == nil {
		t.Fatal("recorder streamed an interactive command instead of refusing")
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	if got := cmdrun.ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", got)
	}

	_, err := cmdrun.Exec().Run(context.Background(), cmdrun.Spec("false"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
