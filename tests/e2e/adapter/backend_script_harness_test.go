package adapter_e2e

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// Harness for asserting what command a backend script builds.
//
// Backend scripts under adapters/<adapter>/backends/<backend>/ are
// pure argv assembly: they read env vars, juggle conditional flags,
// and exec the backend binary. The bugs live in that assembly —
// unquoted expansions that word-split, flags emitted when they should
// be omitted, heredoc headers in the wrong place — none of which a
// recording of the backend's *response* would reveal.
//
// So these tests capture the invocation instead. _lib.sh resolves the
// binary through the backend manifest's `bin_env_var` operator
// override (HIMALAYA_BIN for the email adapter); pointing that at a
// recorder stub is the production-supported seam, not a test-only
// backdoor. Nothing in cmd/aps or internal/core changes, and the real
// script runs unmodified — only the leaf binary is swapped.
//
// The scripts are bash and rely on BASH_SOURCE, ${!var} indirect
// expansion, and [[ =~ ]], so these tests are skipped on Windows.

// backendInvocation is one captured call to the backend binary.
type backendInvocation struct {
	// Argv holds the arguments the script passed, excluding argv[0].
	Argv []string

	// Stdin is everything the script piped to the binary — for
	// himalaya's `template send` that is the full RFC822 message.
	// Empty for calls the script does not pipe into.
	Stdin string
}

// argvString renders Argv with each element bracketed, so a
// word-splitting bug ("-a work acct" splitting into three args) is
// visible in the failure message rather than hidden by whitespace.
func (inv backendInvocation) argvString() string {
	var b strings.Builder
	for i, a := range inv.Argv {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte('[')
		b.WriteString(a)
		b.WriteByte(']')
	}
	return b.String()
}

// stdinLines splits captured stdin into lines with the trailing
// newline dropped, so callers can assert on header position without
// an empty final element.
func (inv backendInvocation) stdinLines() []string {
	s := strings.TrimSuffix(inv.Stdin, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// stubResponse scripts what the recorder stub prints on stdout for a
// call whose leading argv matches Match.
//
// Some scripts consume the backend's output and feed it back into a
// second call — reply.sh runs `template reply`, splits the result with
// sed, and pipes a rebuilt message into `template send`. Asserting the
// second call's argv and stdin therefore requires controlling what the
// first one returned.
type stubResponse struct {
	// Match is a leading-argv prefix, e.g. []string{"template",
	// "reply"}. The first entry whose prefix matches wins.
	Match []string

	// Stdout is printed verbatim for a matching call.
	Stdout string
}

// stubOptions configures the recorder stub.
type stubOptions struct {
	// Responses are consulted in order; the first prefix match
	// supplies the call's stdout. Calls matching nothing print
	// nothing.
	Responses []stubResponse

	// ReadsStdin lists argv prefixes for calls the script pipes
	// into. The stub only drains stdin for these, because an
	// unconditional `cat` blocks forever on a call that receives no
	// input and no EOF.
	ReadsStdin [][]string
}

// recorderStub writes an executable stub that records each call's
// argv and stdin into its own file under callDir, and returns the stub
// path.
//
// Argv is written NUL-delimited rather than space-delimited: an
// argument containing a space must stay one element, and a delimiter
// that can appear inside an argument would make the split ambiguous —
// exactly the bug class these tests exist to catch.
//
// Calls are numbered by counting existing files, so a script invoking
// the binary more than once yields one file per call in order.
//
// The stub exits 0. Failure-path coverage (a backend that exits
// non-zero) belongs with the script's error handling, not its argv
// assembly, so it is not modelled here.
func recorderStub(t *testing.T, dir, callDir string, opts stubOptions) string {
	t.Helper()

	if err := os.MkdirAll(callDir, 0o755); err != nil {
		t.Fatalf("mkdir call dir: %v", err)
	}
	stub := filepath.Join(dir, "recorder.sh")

	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -uo pipefail\n")
	// Call index: count files already written, then add one.
	b.WriteString("n=$(find \"" + callDir + "\" -name 'call*' | wc -l | tr -d ' ')\n")
	b.WriteString("n=$((n+1))\n")
	b.WriteString("out=\"" + callDir + "/call$n\"\n")
	b.WriteString("{\n")
	b.WriteString("  for a in \"$@\"; do printf '%s\\0' \"$a\"; done\n")
	b.WriteString("  printf 'ARGV_END\\0'\n")
	b.WriteString("} > \"$out\"\n")

	// Drain stdin only for calls declared to receive it.
	for _, prefix := range opts.ReadsStdin {
		b.WriteString("if " + argvPrefixTest(prefix) + "; then cat >> \"$out\"; fi\n")
	}

	// Scripted stdout, first match wins.
	for _, r := range opts.Responses {
		b.WriteString("if " + argvPrefixTest(r.Match) + "; then\n")
		b.WriteString("  printf '%s' " + shellSingleQuote(r.Stdout) + "\n")
		b.WriteString("  exit 0\n")
		b.WriteString("fi\n")
	}
	b.WriteString("exit 0\n")

	if err := os.WriteFile(stub, []byte(b.String()), 0o755); err != nil {
		t.Fatalf("write recorder stub: %v", err)
	}
	return stub
}

// argvPrefixTest renders a bash condition matching a leading-argv
// prefix. An empty prefix matches every call.
func argvPrefixTest(prefix []string) string {
	if len(prefix) == 0 {
		return "true"
	}
	var parts []string
	for i, want := range prefix {
		parts = append(parts, "[ \"${"+itoa(i+1)+":-}\" = "+shellSingleQuote(want)+" ]")
	}
	return strings.Join(parts, " && ")
}

// itoa avoids importing strconv for positional-parameter indices.
func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

// shellSingleQuote wraps s in single quotes, escaping any embedded
// single quote, so it is safe to embed in the generated stub.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// readInvocations parses every recorded call, in invocation order.
func readInvocations(t *testing.T, callDir string) []backendInvocation {
	t.Helper()

	entries, err := os.ReadDir(callDir)
	if err != nil {
		t.Fatalf("read call dir: %v", err)
	}

	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "call") {
			names = append(names, e.Name())
		}
	}
	// Numeric order: call1 .. call9 sort correctly as strings, and
	// no backend script here makes ten calls.
	sort.Strings(names)

	out := make([]backendInvocation, 0, len(names))
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(callDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		const sentinel = "ARGV_END\x00"
		idx := strings.Index(string(raw), sentinel)
		if idx < 0 {
			t.Fatalf("%s missing argv sentinel; got %q", name, string(raw))
		}

		var argv []string
		for a := range strings.SplitSeq(string(raw[:idx]), "\x00") {
			if a == "" {
				continue
			}
			argv = append(argv, a)
		}
		out = append(out, backendInvocation{
			Argv:  argv,
			Stdin: string(raw[idx+len(sentinel):]),
		})
	}
	return out
}

// backendScriptPath resolves a backend script in the repo's adapters
// tree. Scripts are run from their real location so the ../../../
// _lib.sh source path and the sibling backend.yaml both resolve the
// way they do in production.
func backendScriptPath(t *testing.T, adapter, backend, action string) string {
	t.Helper()
	path := filepath.Join(adaptersRoot(t), adapter, "backends", backend, action+".sh")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("backend script not found: %v", err)
	}
	return path
}

// adaptersRoot is the repo's adapters/ directory.
func adaptersRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return filepath.Join(root, "adapters")
}

// backendBinEnvVar reads the `bin_env_var` key from a backend's
// manifest — the operator override _lib.sh honours when resolving
// $BIN, and therefore the name the harness must set to redirect the
// script at its recorder stub.
//
// Read from backend.yaml rather than hardcoded per backend: the
// override name is part of the backend's published contract, so a
// manifest that renamed or dropped it would silently stop redirecting
// and let tests exercise the operator's real binary. Failing here
// instead makes that a test error.
func backendBinEnvVar(t *testing.T, adapter, backend string) string {
	t.Helper()

	path := filepath.Join(adaptersRoot(t), adapter, "backends", backend, "backend.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read backend.yaml: %v", err)
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "bin_env_var:")
		if !ok {
			continue
		}
		name := strings.Trim(strings.TrimSpace(rest), `"'`)
		if name == "" {
			t.Fatalf("%s declares an empty bin_env_var", path)
		}
		return name
	}
	t.Fatalf("%s declares no bin_env_var; the harness cannot redirect this backend", path)
	return ""
}

// backendScript identifies a script by adapter, backend, and action,
// so the harness can resolve both its path and the bin_env_var that
// redirects it.
type backendScript struct {
	Adapter string
	Backend string
	Action  string
}

// path resolves the script's location on disk.
func (s backendScript) path(t *testing.T) string {
	t.Helper()
	return backendScriptPath(t, s.Adapter, s.Backend, s.Action)
}

// runBackendScriptMulti executes a backend script and returns every
// captured invocation plus the script's own combined output.
//
// env is applied over a minimal base (PATH, HOME) rather than the
// ambient environment, so a var the operator happens to have set
// cannot silently satisfy an assertion.
//
// The script's stdin is /dev/null: these scripts read none, and an
// inherited terminal would let a stray `cat` block the test run.
func runBackendScriptMulti(
	t *testing.T,
	script backendScript,
	env map[string]string,
	opts stubOptions,
) ([]backendInvocation, string, error) {
	t.Helper()

	scriptPath := script.path(t)
	dir := t.TempDir()
	callDir := filepath.Join(dir, "calls")
	stub := recorderStub(t, dir, callDir, opts)

	full := map[string]string{
		"PATH": os.Getenv("PATH"),
		"HOME": dir,
	}
	// _lib.sh resolves $BIN through the backend manifest's declared
	// operator override; setting that name points the script at the
	// recorder without touching the script itself.
	full[backendBinEnvVar(t, script.Adapter, script.Backend)] = stub
	maps.Copy(full, env)

	envSlice := make([]string, 0, len(full))
	for k, v := range full {
		envSlice = append(envSlice, k+"="+v)
	}

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer func() { _ = devNull.Close() }()

	cmd := exec.Command("bash", scriptPath)
	cmd.Env = envSlice
	cmd.Stdin = devNull
	out, runErr := cmd.CombinedOutput()

	return readInvocations(t, callDir), string(out), runErr
}

// runBackendScript is the single-invocation form: it asserts exactly
// one call reached the backend and returns it. A script that exits
// before invoking the binary yields a zero-value invocation, so
// guard-clause tests can assert on the empty argv.
func runBackendScript(
	t *testing.T,
	script backendScript,
	env map[string]string,
) (backendInvocation, string, error) {
	t.Helper()

	invs, out, err := runBackendScriptMulti(t, script, env, stubOptions{
		// send.sh is the only single-call script that pipes a
		// message in; an empty prefix drains stdin for any call,
		// which is safe because the script's own stdin is /dev/null
		// and a non-piping call sees immediate EOF.
		ReadsStdin: [][]string{{"template", "send"}},
	})
	if len(invs) == 0 {
		return backendInvocation{}, out, err
	}
	if len(invs) > 1 {
		t.Fatalf("expected 1 backend invocation, got %d:\n%s",
			len(invs), invocationsString(invs))
	}
	return invs[0], out, err
}

// invocationsString renders a call list for failure messages.
func invocationsString(invs []backendInvocation) string {
	var b strings.Builder
	for i, inv := range invs {
		b.WriteString("  call ")
		b.WriteString(itoa(i + 1))
		b.WriteString(": ")
		b.WriteString(inv.argvString())
		b.WriteByte('\n')
	}
	return b.String()
}

// requireBash skips on platforms where the bash-specific constructs in
// _lib.sh and the backend scripts do not apply.
func requireBash(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("backend scripts require bash")
	}
}

// assertArgv fails when the captured argv differs from want, printing
// both bracketed so whitespace differences are legible.
func assertArgv(t *testing.T, inv backendInvocation, want ...string) {
	t.Helper()
	wantInv := backendInvocation{Argv: want}
	if inv.argvString() != wantInv.argvString() {
		t.Errorf("argv mismatch\n got: %s\nwant: %s",
			inv.argvString(), wantInv.argvString())
	}
}
