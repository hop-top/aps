package e2e

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E tests for cli-conventions-with-kit.md §6.5 progress mandate.
//
// `aps run` is the wired site we test against because it produces a
// deterministic two-event progress envelope (`exec` start → `exit`
// end) over a known subprocess (`true`/`false`). Network-bound sites
// (a2a tasks send/cancel/subscribe, a2a card fetch) are exercised in
// their own e2e suites with peers; this file covers the contract.

const ansiEscape = "\x1b["

// ansiCursorRE matches CSI cursor-control escape sequences. The kit
// Human reporter writes plain text, so any CSI sequence on stderr
// would be a regression.
var ansiCursorRE = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

// TestProgress_RunEmitsExecExit asserts that `aps run` emits at
// least the canonical two-phase envelope on stderr (Human format,
// the default), and that the underlying subprocess output is
// undisturbed on stdout.
func TestProgress_RunEmitsExecExit(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "progress-run")
	require.NoError(t, err)

	stdout, stderr, err := runAPS(t, home, "run", "progress-run", "--", "true")
	require.NoError(t, err)

	// Human reporter output: "[exec] true" + "[exit] true ok"
	assert.Contains(t, stderr, "[exec]", "expected exec phase line")
	assert.Contains(t, stderr, "[exit]", "expected exit phase line")
	assert.Contains(t, stderr, "ok", "expected exit ok marker")

	// Subprocess (`true`) emits no stdout. aps must not pollute it.
	assert.Empty(t, strings.TrimSpace(stdout),
		"stdout should stay empty when the child writes nothing")

	// At least N=2 progress updates.
	progressLines := countPrefix(stderr, "[")
	assert.GreaterOrEqual(t, progressLines, 2,
		"expected >=2 Human progress lines on stderr, got %d", progressLines)
}

// TestProgress_RunQuietSilent asserts that --quiet drops every
// progress event (Reporter resolves to Discard).
func TestProgress_RunQuietSilent(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "progress-quiet")
	require.NoError(t, err)

	_, stderr, err := runAPS(t, home, "--quiet", "run", "progress-quiet", "--", "true")
	require.NoError(t, err)

	assert.NotContains(t, stderr, "[exec]")
	assert.NotContains(t, stderr, "[exit]")
}

// TestProgress_RunJSONLOnStderr asserts --progress-format json emits
// valid JSONL with required fields (phase, at).
func TestProgress_RunJSONLOnStderr(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "progress-json")
	require.NoError(t, err)

	_, stderr, err := runAPS(t, home,
		"--progress-format", "json",
		"run", "progress-json", "--", "true")
	require.NoError(t, err)

	require.NotEmpty(t, stderr, "expected JSONL events on stderr")

	var phases []string
	var sawExitOK bool
	for _, line := range strings.Split(strings.TrimRight(stderr, "\n"), "\n") {
		if line == "" {
			continue
		}
		var ev struct {
			Phase string `json:"phase"`
			At    string `json:"at"`
			OK    *bool  `json:"ok,omitempty"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &ev),
			"each progress line must be valid JSON: %q", line)
		assert.NotEmpty(t, ev.Phase, "phase required")
		assert.NotEmpty(t, ev.At, "at (RFC3339) required")
		phases = append(phases, ev.Phase)
		if ev.Phase == "exit" && ev.OK != nil && *ev.OK {
			sawExitOK = true
		}
	}
	assert.Contains(t, phases, "exec")
	assert.Contains(t, phases, "exit")
	assert.True(t, sawExitOK, "exit event must carry OK:true on success")
}

// TestProgress_RunNonTTYNoANSI asserts the Human reporter emits no
// CSI escape sequences over the pipe (which is what runAPS uses).
// runAPS captures stderr via a pipe (`bytes.Buffer`) so isatty
// reports false — this matches how progress lands in CI.
func TestProgress_RunNonTTYNoANSI(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "progress-notty")
	require.NoError(t, err)

	_, stderr, err := runAPS(t, home, "run", "progress-notty", "--", "true")
	require.NoError(t, err)

	if ansiCursorRE.MatchString(stderr) {
		t.Errorf("non-TTY stderr must not contain ANSI cursor controls; got %q", stderr)
	}
	assert.NotContains(t, stderr, ansiEscape,
		"plain Human progress output must not include ESC[ sequences")
}

// TestProgress_RunFailureEmitsOKFalse asserts that a non-zero child
// exit produces a final exit event with OK:false in JSON mode.
func TestProgress_RunFailureEmitsOKFalse(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "progress-fail")
	require.NoError(t, err)

	_, stderr, runErr := runAPS(t, home,
		"--progress-format", "json",
		"run", "progress-fail", "--", "false")
	// `false` exits non-zero; aps reports the error.
	require.Error(t, runErr, "child false must surface as a non-zero exit")

	var sawFail bool
	for _, line := range strings.Split(strings.TrimRight(stderr, "\n"), "\n") {
		if line == "" {
			continue
		}
		var ev struct {
			Phase string `json:"phase"`
			OK    *bool  `json:"ok,omitempty"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			// Non-JSON lines (e.g. error envelope) are tolerated; we
			// only assert that at least one JSONL event marked failure.
			continue
		}
		if ev.Phase == "exit" && ev.OK != nil && !*ev.OK {
			sawFail = true
		}
	}
	assert.True(t, sawFail,
		"expected at least one exit event with OK:false on child failure; stderr=%q", stderr)
}

// countPrefix counts lines that start with the given prefix.
func countPrefix(s, prefix string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, prefix) {
			n++
		}
	}
	return n
}
