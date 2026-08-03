package adapter_e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// End-to-end coverage of `aps adapter exec` failure paths and output
// handling. Complements exec_fixture_test.go, which covers the happy
// path: here every case asserts a non-nil error plus a specific
// fragment of the message the runtime actually emits.
//
// Two properties of the runtime shape these assertions:
//
//   - ExecAction runs the script with CombinedOutput, so a stub's
//     stderr is folded into the returned error string rather than
//     reaching the aps process's own stderr independently. Assertions
//     therefore read stdout+stderr combined.
//   - The CLI error renderer wraps long lines inside a boxed ERROR
//     block, so a long absolute path can be split mid-token. Tests
//     assert on short fragments that survive wrapping, or on path
//     segments rather than whole paths.

// execFixtureScriptPath returns the on-disk location of a fixture
// action's stub script, given the adapter name written by
// writeExecFixtureAdapter with default options.
func execFixtureScriptPath(home, name, action string) string {
	return filepath.Join(
		home, ".local", "share", "aps", "devices", name,
		"backends", execFixtureBackend, action+".sh",
	)
}

// execFixtureAdapterDir returns the adapter's manifest directory —
// the value ExecAction assigns to cmd.Dir.
func execFixtureAdapterDir(home, name string) string {
	return filepath.Join(
		home, ".local", "share", "aps", "devices", name,
	)
}

// TestExecFailure_NonZeroExitWrapsActionAndOutput asserts that a stub
// exiting non-zero produces an error naming the action, reporting the
// underlying exit status, and carrying the script's combined output
// (both the stdout markers and the stderr diagnostic) under an
// "output:" section.
func TestExecFailure_NonZeroExitWrapsActionAndOutput(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name:           "exitfail",
		FailingActions: map[string]int{"list": 7},
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "list",
		"--from", "ops@example.com",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit to fail, got success\nstdout: %s", stdout)
	}

	combined := stdout + stderr
	for _, want := range []string{
		// Action name, from the "action %q failed" wrapper.
		`action "list" failed`,
		// Underlying *exec.ExitError, carrying the stub's exit code.
		"exit status 7",
		// The wrapper's own output section header.
		"output:",
		// Stub stdout, merged in by CombinedOutput.
		"ACTION list",
		// Stub stderr, merged into the same stream.
		"failed deliberately",
	} {
		if !strings.Contains(combined, want) {
			t.Errorf("error output missing %q\nstdout: %s\nstderr: %s",
				want, stdout, stderr)
		}
	}

	// CombinedOutput folds the stub's stderr into the error string, so
	// the action's own stdout never reaches the caller's stdout on the
	// failure path — exec.go returns before logging.Print.
	if strings.Contains(stdout, "ACTION list") {
		t.Errorf("action output leaked to stdout on failure:\n%s", stdout)
	}
}

// TestExecFailure_ScriptDeclaredButMissing asserts that an action
// declared in the manifest whose script is absent from disk fails at
// resolution time, before any process is spawned. resolveActionScript
// stats the path and reports "not found".
func TestExecFailure_ScriptDeclaredButMissing(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "missingscript",
	})
	script := execFixtureScriptPath(home, name, "list")
	if err := os.Remove(script); err != nil {
		t.Fatalf("remove fixture script: %v", err)
	}

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "list",
		"--from", "ops@example.com",
	)
	if err == nil {
		t.Fatalf("expected missing script to fail, got success\nstdout: %s", stdout)
	}

	combined := stdout + stderr
	for _, want := range []string{
		"not found",
		"no such file or directory",
		// Final path segment survives the renderer's line wrapping.
		"list.sh",
	} {
		if !strings.Contains(combined, want) {
			t.Errorf("error output missing %q\nstdout: %s\nstderr: %s",
				want, stdout, stderr)
		}
	}

	// Resolution fails before exec, so nothing from the action wrapper
	// appears.
	if strings.Contains(combined, `action "list" failed`) {
		t.Errorf("expected resolution-stage failure, got exec-stage wrapper:\n%s",
			combined)
	}
}

// TestExecFailure_ScriptNotExecutable asserts that a script present on
// disk but lacking the execute bit fails at spawn time with the
// action wrapper and a fork/exec permission error.
func TestExecFailure_ScriptNotExecutable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "noexecbit",
	})
	script := execFixtureScriptPath(home, name, "list")
	if err := os.Chmod(script, 0o644); err != nil {
		t.Fatalf("chmod fixture script: %v", err)
	}

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "list",
		"--from", "ops@example.com",
	)
	if err == nil {
		t.Fatalf("expected non-executable script to fail, got success\nstdout: %s",
			stdout)
	}

	combined := stdout + stderr
	for _, want := range []string{
		// Stat succeeded, so this reaches the exec wrapper.
		`action "list" failed`,
		"fork/exec",
		"permission denied",
	} {
		if !strings.Contains(combined, want) {
			t.Errorf("error output missing %q\nstdout: %s\nstderr: %s",
				want, stdout, stderr)
		}
	}
}

// TestExec_RunsInManifestDirectory asserts ExecAction sets cmd.Dir to
// the manifest's directory — the adapter root, not the backends/
// subdirectory the script itself lives in. The stub echoes its $PWD.
func TestExec_RunsInManifestDirectory(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "cwdcheck",
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "list",
		"--from", "ops@example.com",
	)
	if err != nil {
		t.Fatalf("exec list: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var got string
	for line := range strings.SplitSeq(stdout, "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "PWD "); ok {
			got = after
			break
		}
	}
	if got == "" {
		t.Fatalf("stub did not report PWD:\n%s", stdout)
	}

	want := execFixtureAdapterDir(home, name)
	// t.TempDir may hand back a path through a symlink (/var vs
	// /private/var on darwin); compare resolved forms.
	wantResolved, err := filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatalf("resolve want dir %s: %v", want, err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("resolve got dir %s: %v", got, err)
	}
	if gotResolved != wantResolved {
		t.Errorf("script cwd = %q, want manifest dir %q", gotResolved, wantResolved)
	}

	// The script lives one level deeper; cmd.Dir must not be the
	// script's own directory.
	if filepath.Base(gotResolved) == execFixtureBackend {
		t.Errorf("cwd is the script dir, want the manifest dir: %q", gotResolved)
	}
}

// execRedactSecrets are the secret-shaped tokens the redaction stub
// emits, paired with the tag the redactor is expected to substitute.
// Shapes chosen from internal/logging/redact.go's aps-domain rules and
// the gitleaks corpus loaded by redact.Default().
var execRedactSecrets = []struct {
	name  string
	line  string
	token string
	tag   string
}{
	{
		name:  "authorization_header",
		line:  "Authorization: Bearer abcdef1234567890ghijklmnopqrstuv",
		token: "abcdef1234567890ghijklmnopqrstuv",
		// Key-aware rule: keeps "Authorization: Bearer ", tags the value.
		tag: "<aps-bearer-header>",
	},
	{
		name:  "bare_bearer",
		line:  "saw token Bearer zyxwvu9876543210abcdefghijklmnop",
		token: "zyxwvu9876543210abcdefghijklmnop",
		tag:   "<aps-generic-bearer>",
	},
	{
		name:  "aws_access_key",
		line:  "AWS_ACCESS_KEY_ID=AKIAZZ4RTHISISNOTREAL",
		token: "AKIAZZ4RTHISISNOTREAL",
		tag:   "<generic-api-key>",
	},
}

// writeExecRedactScript overwrites a fixture action's stub so it emits
// the secret-shaped lines above instead of its env dump.
func writeExecRedactScript(t *testing.T, home, name, action string) {
	t.Helper()

	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -u\n")
	for _, s := range execRedactSecrets {
		b.WriteString("echo '" + s.line + "'\n")
	}
	b.WriteString("exit 0\n")

	path := execFixtureScriptPath(home, name, action)
	if err := os.WriteFile(path, []byte(b.String()), 0o755); err != nil {
		t.Fatalf("write redaction stub: %v", err)
	}
}

// TestExec_RedactsActionOutput asserts that action stdout — external
// content by definition — passes through logging.Print's redactor
// before reaching the terminal, so no raw secret token appears in the
// command's output.
//
// This is the guarantee the skipped placeholder in
// tests/e2e/redact_test.go was meant to make.
func TestExec_RedactsActionOutput(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "redactout",
	})
	writeExecRedactScript(t, home, name, "list")

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "list",
		"--from", "ops@example.com",
	)
	if err != nil {
		t.Fatalf("exec list: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	combined := stdout + stderr
	for _, s := range execRedactSecrets {
		if strings.Contains(combined, s.token) {
			t.Errorf("%s: raw secret %q leaked into output:\n%s",
				s.name, s.token, combined)
		}
		if !strings.Contains(stdout, s.tag) {
			t.Errorf("%s: expected redaction tag %q in stdout:\n%s",
				s.name, s.tag, stdout)
		}
	}

	// Key-aware redaction keeps the header name readable.
	if !strings.Contains(stdout, "Authorization: Bearer <aps-bearer-header>") {
		t.Errorf("expected key-preserving header redaction in stdout:\n%s", stdout)
	}
}

// TestExec_NoRedactBypassEmitsRawSecrets is the negative control for
// TestExec_RedactsActionOutput: with the explicit --no-redact bypass
// the very same stub output reaches stdout verbatim. Without this the
// redaction assertion could pass for the wrong reason (e.g. the stub
// never ran, or its output never reached stdout at all).
func TestExec_NoRedactBypassEmitsRawSecrets(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "redactbypass",
	})
	writeExecRedactScript(t, home, name, "list")

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "list",
		"--from", "ops@example.com",
		"--no-redact",
	)
	if err != nil {
		t.Fatalf("exec list --no-redact: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}

	for _, s := range execRedactSecrets {
		if !strings.Contains(stdout, s.token) {
			t.Errorf("%s: --no-redact should emit raw secret %q, stdout:\n%s",
				s.name, s.token, stdout)
		}
		if strings.Contains(stdout, s.tag) {
			t.Errorf("%s: --no-redact should not tag output, found %q:\n%s",
				s.name, s.tag, stdout)
		}
	}
}
