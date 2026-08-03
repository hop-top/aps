// Package cmdrun is the seam between aps and the external binaries it
// shells out to (tmux today; gh and the skills CLI next).
//
// Call sites build a kit invoke.CommandSpec and hand it to a Runner
// instead of calling exec.Command directly. Production wires Exec();
// tests wire a recording or replaying Runner, so what a command line
// actually looked like is observable without spawning the real binary.
//
// The types are kit's (invoke.CommandSpec, invoke.Result) rather than
// local ones: they already describe "how to spawn a binary" and "what
// came back", and they map 1:1 onto xrr's exec.Request/exec.Response,
// so an xrr-backed Runner is a thin adapter rather than a translation
// layer. kit's own invoke.Runner is not reused — its Invocation type
// is AI-CLI-shaped (Prompt, Model, Agent, Sandbox) and does not
// describe `tmux -S <sock> kill-session -t <name>`.
package cmdrun

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"hop.top/kit/go/core/uxp/invoke"
)

// Runner executes a CommandSpec and reports what happened.
//
// Implementations must not treat a non-zero exit as an error: Result
// carries Code, and callers distinguish "ran and failed" from "could
// not run" themselves. Several tmux paths depend on that distinction —
// kill-session returns non-zero when the session is already gone,
// which is a benign race, not a failure.
//
// A returned error means the command could not be run to completion at
// all (binary missing, context cancelled, I/O failure).
type Runner interface {
	Run(ctx context.Context, spec invoke.CommandSpec) (invoke.Result, error)
}

// Streamer is the optional interface for commands that must inherit
// the caller's terminal rather than have their output captured.
//
// `tmux attach` is interactive: it needs the real stdin/stdout/stderr,
// and its output is a live terminal session, not a value to record.
// Runners that cannot provide a terminal (any recording or replaying
// Runner) should not implement this, so an attempt to record an
// interactive command fails loudly instead of hanging on a stdin that
// will never arrive.
type Streamer interface {
	Stream(ctx context.Context, spec invoke.CommandSpec, stdin io.Reader, stdout, stderr io.Writer) error
}

// ErrNotStreamable is returned when an interactive command is issued
// to a Runner that cannot attach a terminal.
var ErrNotStreamable = errors.New("cmdrun: runner cannot stream an interactive command")

// execRunner runs commands as real subprocesses.
type execRunner struct{}

// Exec returns the production Runner.
func Exec() Runner { return execRunner{} }

// Run implements Runner.
func (execRunner) Run(ctx context.Context, spec invoke.CommandSpec) (invoke.Result, error) {
	// #nosec G204 -- specs are built from registry state and internal
	// constants, never from unvalidated user input.
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	if len(spec.Env) > 0 {
		cmd.Env = spec.Env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := invoke.Result{
		Command: spec,
		Code:    ExitCode(err),
		Stdout:  stdout.Bytes(),
		Stderr:  stderr.Bytes(),
	}

	// A clean non-zero exit is a Result, not an error — see Runner.
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return res, fmt.Errorf("cmdrun: run %s: %w", spec.Path, err)
	}
	return res, nil
}

// Stream implements Streamer.
func (execRunner) Stream(
	ctx context.Context,
	spec invoke.CommandSpec,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	// #nosec G204 -- see Run.
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	if len(spec.Env) > 0 {
		cmd.Env = spec.Env
	}
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmdrun: stream %s: %w", spec.Path, err)
	}
	return nil
}

// ExitCode extracts a process exit code from err: 0 when err is nil,
// the process code when err is (or wraps) *exec.ExitError, and -1 when
// the command never produced a clean exit at all.
//
// Mirrors xrr's adapters/exec.ExitCodeFromError so a recorded Result
// and a live one agree on what "-1" means.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// Spec is a convenience constructor for the common case.
func Spec(path string, args ...string) invoke.CommandSpec {
	return invoke.CommandSpec{Path: path, Args: args}
}
