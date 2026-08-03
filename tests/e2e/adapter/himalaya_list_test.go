package adapter_e2e

import (
	"strings"
	"testing"
)

// Coverage for adapters/email/backends/himalaya/list.sh.
//
// list.sh differs from the other email actions in two ways that matter
// here: it applies its own defaults in bash (EMAIL_FOLDER, EMAIL_LIMIT)
// in addition to the manifest's, and it post-processes the backend's
// output through a pipeline rather than passing it straight through.

// listScript resolves list.sh from the adapters tree.
func listScript(t *testing.T) backendScript {
	t.Helper()
	return backendScript{Adapter: "email", Backend: "himalaya", Action: "list"}
}

// listStubOptions makes the stub emit stdout for the envelope-list
// call, so tests can assert what survives the `head` pipeline.
func listStubOptions(stdout string) stubOptions {
	return stubOptions{
		Responses: []stubResponse{
			{Match: []string{"envelope", "list"}, Stdout: stdout},
		},
	}
}

// TestHimalayaList_Defaults covers the bash-level fallbacks: with
// neither input set the folder must default to INBOX and the JSON
// output flag must always be present, since the caller parses JSON.
func TestHimalayaList_Defaults(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, listScript(t),
		map[string]string{}, listStubOptions("[]\n"))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 1 {
		t.Fatalf("expected 1 invocation, got %d:\n%s", len(invs), invocationsString(invs))
	}

	assertArgv(t, invs[0], "envelope", "list", "-f", "INBOX", "-o", "json")
}

// TestHimalayaList_ExplicitFolder covers EMAIL_FOLDER overriding the
// default.
func TestHimalayaList_ExplicitFolder(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, listScript(t), map[string]string{
		"EMAIL_FOLDER": "Archive",
	}, listStubOptions("[]\n"))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "envelope", "list", "-f", "Archive", "-o", "json")
}

// TestHimalayaList_FolderWithSpace guards the folder argument. IMAP
// folder names with spaces are common ("[Gmail]/All Mail", "Sent
// Items"), so this is a realistic input, not a synthetic edge case.
func TestHimalayaList_FolderWithSpace(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, listScript(t), map[string]string{
		"EMAIL_FOLDER": "Sent Items",
	}, listStubOptions("[]\n"))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "envelope", "list", "-f", "Sent Items", "-o", "json")
}

// TestHimalayaList_WithAccount covers the account flag position,
// between the folder and the output-format flag.
func TestHimalayaList_WithAccount(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, listScript(t), map[string]string{
		"APS_EMAIL_ACCOUNT": "work",
	}, listStubOptions("[]\n"))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "envelope", "list", "-f", "INBOX", "-a", "work", "-o", "json")
}

// TestHimalayaList_AccountWithSpace is the word-splitting case for
// list.sh.
func TestHimalayaList_AccountWithSpace(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, listScript(t), map[string]string{
		"APS_EMAIL_ACCOUNT": "work account",
	}, listStubOptions("[]\n"))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "envelope", "list", "-f", "INBOX", "-a", "work account", "-o", "json")
}

// TestHimalayaList_LimitTruncates covers the EMAIL_LIMIT pipeline:
// the limit is applied by `head` on the backend's output, not by a
// flag, so it must actually truncate.
func TestHimalayaList_LimitTruncates(t *testing.T) {
	requireBash(t)
	t.Parallel()

	backendOut := "line1\nline2\nline3\nline4\nline5\n"

	invs, out, err := runBackendScriptMulti(t, listScript(t), map[string]string{
		"EMAIL_LIMIT": "2",
	}, listStubOptions(backendOut))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}

	// The limit is a pipeline stage, so it must not appear in argv.
	assertArgv(t, invs[0], "envelope", "list", "-f", "INBOX", "-o", "json")

	gotLines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(gotLines) != 2 {
		t.Errorf("EMAIL_LIMIT=2 yielded %d lines, want 2:\n%s", len(gotLines), out)
	}
}

// TestHimalayaList_DefaultLimit pins the built-in limit of 10 applied
// when EMAIL_LIMIT is unset.
func TestHimalayaList_DefaultLimit(t *testing.T) {
	requireBash(t)
	t.Parallel()

	var b strings.Builder
	for i := range 15 {
		b.WriteString("line")
		b.WriteString(itoa(i + 1))
		b.WriteByte('\n')
	}

	_, out, err := runBackendScriptMulti(t, listScript(t),
		map[string]string{}, listStubOptions(b.String()))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}

	gotLines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(gotLines) != 10 {
		t.Errorf("default limit yielded %d lines, want 10:\n%s", len(gotLines), out)
	}
}

// TestHimalayaList_LargeOutputUnderLimit is the SIGPIPE case.
//
// list.sh runs under `set -o pipefail` and pipes the backend into
// `head -n N`. When the backend produces more than N lines, head exits
// after N and the writer receives SIGPIPE; pipefail then propagates
// that as a non-zero status for the whole pipeline. Whether this
// surfaces depends on output exceeding the pipe buffer, so the backend
// here emits well past 64KiB.
//
// A truncating list is the normal case — an inbox almost always has
// more messages than the limit — so a non-zero exit here would make
// the common path look like a failure to the caller.
func TestHimalayaList_LargeOutputUnderLimit(t *testing.T) {
	requireBash(t)
	t.Parallel()

	var b strings.Builder
	// ~100 bytes/line * 2000 lines = ~200KiB, comfortably past the
	// 64KiB pipe buffer on both Linux and macOS.
	filler := strings.Repeat("x", 90)
	for i := range 2000 {
		b.WriteString("line")
		b.WriteString(itoa(i%10 + 1))
		b.WriteString(filler)
		b.WriteByte('\n')
	}

	_, out, err := runBackendScriptMulti(t, listScript(t), map[string]string{
		"EMAIL_LIMIT": "5",
	}, listStubOptions(b.String()))
	if err != nil {
		t.Fatalf("list.sh exited non-zero truncating a large listing "+
			"(SIGPIPE under set -o pipefail): %v\noutput: %s", err, out)
	}

	gotLines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(gotLines) != 5 {
		t.Errorf("EMAIL_LIMIT=5 yielded %d lines, want 5", len(gotLines))
	}
}
