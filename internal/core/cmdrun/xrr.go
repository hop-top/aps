package cmdrun

import (
	"context"
	"fmt"
	"io"

	xrr "hop.top/xrr"
	xexec "hop.top/xrr/adapters/exec"

	"hop.top/kit/go/core/uxp/invoke"
)

// Recorder wraps a Runner in an xrr session so every command it runs
// is written to (or replayed from) a cassette.
//
// The cassette key is xrr's exec fingerprint — sha256 over argv, stdin
// and cwd — so the recorded artifact IS the command line. That makes
// it the right tool here: a test asserts what aps asked tmux to do by
// reading a YAML cassette, and a replay run fails on a fingerprint
// miss the moment an argv changes.
//
// Cwd is deliberately left empty on the request unless the spec sets
// Dir. Per xrr's exec adapter docs, a non-empty Cwd is a Go-only
// extension to the v1 cassette format and cassettes carrying it will
// not replay in the other language ports.
type Recorder struct {
	session *xrr.FileSession
	inner   Runner
}

// NewRecorder wraps inner in session. In replay mode inner is never
// called, so it may be nil.
func NewRecorder(session *xrr.FileSession, inner Runner) *Recorder {
	return &Recorder{session: session, inner: inner}
}

// Run implements Runner.
func (r *Recorder) Run(ctx context.Context, spec invoke.CommandSpec) (invoke.Result, error) {
	req := &xexec.Request{
		Argv: append([]string{spec.Path}, spec.Args...),
		Cwd:  spec.Dir,
	}

	resp, err := r.session.Record(ctx, xexec.NewAdapter(), req, func() (xrr.Response, error) {
		if r.inner == nil {
			return nil, fmt.Errorf("cmdrun: recorder has no inner runner (replay-only)")
		}
		res, runErr := r.inner.Run(ctx, spec)
		if runErr != nil {
			return nil, fmt.Errorf("cmdrun: inner runner: %w", runErr)
		}
		return &xexec.Response{
			Stdout:   string(res.Stdout),
			Stderr:   string(res.Stderr),
			ExitCode: res.Code,
		}, nil
	})
	if err != nil {
		return invoke.Result{Command: spec}, fmt.Errorf("cmdrun: xrr session: %w", err)
	}

	execResp, err := execResponse(resp)
	if err != nil {
		return invoke.Result{Command: spec}, err
	}

	return invoke.Result{
		Command: spec,
		Code:    execResp.ExitCode,
		Stdout:  []byte(execResp.Stdout),
		Stderr:  []byte(execResp.Stderr),
	}, nil
}

// execResponse normalises what Session.Record hands back.
//
// On record the live *exec.Response comes through unchanged; on replay
// xrr returns a *xrr.RawResponse carrying the decoded cassette payload
// as a map, because the session has no way to know the concrete
// adapter type. Both shapes have to be handled — a type assertion on
// *exec.Response alone compiles fine and then fails only in replay.
func execResponse(resp xrr.Response) (*xexec.Response, error) {
	switch typed := resp.(type) {
	case *xexec.Response:
		return typed, nil
	case *xrr.RawResponse:
		return &xexec.Response{
			Stdout:   stringField(typed.Payload, "stdout"),
			Stderr:   stringField(typed.Payload, "stderr"),
			ExitCode: intField(typed.Payload, "exit_code"),
		}, nil
	default:
		return nil, fmt.Errorf("cmdrun: unexpected response type %T from cassette", resp)
	}
}

// stringField reads a string from a replayed payload, tolerating an
// absent key (an omitempty field that was empty when recorded).
func stringField(payload map[string]any, key string) string {
	v, ok := payload[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// intField reads an int from a replayed payload. YAML decoding may
// yield int or float64 depending on the value, so both are accepted.
func intField(payload map[string]any, key string) int {
	switch v := payload[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// Stream is deliberately not implemented as a passthrough: an
// interactive command has no recordable response, and silently
// falling back to the real binary during a replay run would spawn a
// terminal the test never asked for. Callers get ErrNotStreamable.
func (r *Recorder) Stream(
	_ context.Context,
	spec invoke.CommandSpec,
	_ io.Reader,
	_, _ io.Writer,
) error {
	return fmt.Errorf("%w: %s", ErrNotStreamable, spec.Path)
}
