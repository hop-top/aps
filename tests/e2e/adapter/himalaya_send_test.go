package adapter_e2e

import (
	"strings"
	"testing"
)

// Coverage for adapters/email/backends/himalaya/send.sh.
//
// send.sh carries the most conditional logic of the email actions: two
// optional pieces (account flag, Cc header) that must appear when set
// and vanish entirely when not, plus a heredoc whose header order and
// blank-line separator determine whether the result is a valid RFC822
// message at all.

// sendScript resolves send.sh from the adapters tree.
func sendScript(t *testing.T) backendScript {
	t.Helper()
	return backendScript{Adapter: "email", Backend: "himalaya", Action: "send"}
}

// baseSendEnv is the minimum env send.sh requires: the four vars it
// dereferences with ${VAR:?}. Cases copy and adjust it.
func baseSendEnv() map[string]string {
	return map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"EMAIL_TO":       "dest@example.com",
		"EMAIL_SUBJECT":  "Deploy complete",
		"EMAIL_BODY":     "All services green.",
	}
}

// TestHimalayaSend_NoAccount is the baseline: with no account
// configured the -a flag must be absent entirely rather than passed
// empty, which himalaya would reject as a missing flag value.
func TestHimalayaSend_NoAccount(t *testing.T) {
	requireBash(t)
	t.Parallel()

	inv, out, err := runBackendScript(t, sendScript(t), baseSendEnv())
	if err != nil {
		t.Fatalf("send.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "template", "send")

	wantLines := []string{
		"From: ops@example.com",
		"To: dest@example.com",
		"Subject: Deploy complete",
		"",
		"All services green.",
	}
	assertStdinLines(t, inv, wantLines)
}

// TestHimalayaSend_WithAccount covers the -a flag appearing when
// APS_EMAIL_ACCOUNT is set, as two argv elements rather than one
// fused "-a work" string.
func TestHimalayaSend_WithAccount(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseSendEnv()
	env["APS_EMAIL_ACCOUNT"] = "work"

	inv, out, err := runBackendScript(t, sendScript(t), env)
	if err != nil {
		t.Fatalf("send.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "template", "send", "-a", "work")
}

// TestHimalayaSend_EmptyAccountOmitsFlag pins the distinction between
// unset and set-but-empty: an empty APS_EMAIL_ACCOUNT must be treated
// as "no account", not as an account whose name is "".
func TestHimalayaSend_EmptyAccountOmitsFlag(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseSendEnv()
	env["APS_EMAIL_ACCOUNT"] = ""

	inv, out, err := runBackendScript(t, sendScript(t), env)
	if err != nil {
		t.Fatalf("send.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "template", "send")
}

// TestHimalayaSend_WithCC covers the Cc header: it must sit between
// To: and Subject: and must not introduce a blank line that would
// terminate the header block early and push Subject: into the body.
func TestHimalayaSend_WithCC(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseSendEnv()
	env["EMAIL_CC"] = "cc@example.com"

	inv, out, err := runBackendScript(t, sendScript(t), env)
	if err != nil {
		t.Fatalf("send.sh failed: %v\noutput: %s", err, out)
	}

	wantLines := []string{
		"From: ops@example.com",
		"To: dest@example.com",
		"Cc: cc@example.com",
		"Subject: Deploy complete",
		"",
		"All services green.",
	}
	assertStdinLines(t, inv, wantLines)
}

// TestHimalayaSend_EmptyCCOmitsHeader asserts an empty EMAIL_CC emits
// no Cc line at all. A bare "Cc:" header would be malformed, and a
// stray blank line here would split the headers from Subject:.
func TestHimalayaSend_EmptyCCOmitsHeader(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseSendEnv()
	env["EMAIL_CC"] = ""

	inv, out, err := runBackendScript(t, sendScript(t), env)
	if err != nil {
		t.Fatalf("send.sh failed: %v\noutput: %s", err, out)
	}

	for _, line := range inv.stdinLines() {
		if strings.HasPrefix(line, "Cc:") {
			t.Errorf("empty EMAIL_CC produced a Cc header:\n%s", inv.Stdin)
			break
		}
	}
	assertStdinLines(t, inv, []string{
		"From: ops@example.com",
		"To: dest@example.com",
		"Subject: Deploy complete",
		"",
		"All services green.",
	})
}

// TestHimalayaSend_MultilineBody covers a body spanning several lines:
// the heredoc must preserve it verbatim, including its internal blank
// line, rather than collapsing or re-wrapping it.
func TestHimalayaSend_MultilineBody(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseSendEnv()
	env["EMAIL_BODY"] = "First paragraph.\n\nSecond paragraph."

	inv, out, err := runBackendScript(t, sendScript(t), env)
	if err != nil {
		t.Fatalf("send.sh failed: %v\noutput: %s", err, out)
	}

	assertStdinLines(t, inv, []string{
		"From: ops@example.com",
		"To: dest@example.com",
		"Subject: Deploy complete",
		"",
		"First paragraph.",
		"",
		"Second paragraph.",
	})
}

// TestHimalayaSend_SubjectWithSpecialChars covers a subject carrying
// characters the shell would act on if the heredoc were unquoted:
// $VAR expansion, backticks, and quotes must all survive literally.
//
// The heredoc delimiter in send.sh is unquoted (<<EOF), so parameter
// and command substitution DO run inside the body. This test pins the
// resulting behaviour rather than the ideal: it documents that a
// literal $ or backtick in a subject is not delivered as typed.
func TestHimalayaSend_SubjectWithSpecialChars(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseSendEnv()
	env["EMAIL_SUBJECT"] = `Quote " and 'single' and \backslash`

	inv, out, err := runBackendScript(t, sendScript(t), env)
	if err != nil {
		t.Fatalf("send.sh failed: %v\noutput: %s", err, out)
	}

	want := `Subject: Quote " and 'single' and \backslash`
	if !containsLine(inv, want) {
		t.Errorf("subject not preserved verbatim\nwant line: %s\ngot:\n%s",
			want, inv.Stdin)
	}
}

// TestHimalayaSend_MissingRequiredVar covers the ${VAR:?} guards: with
// a required var absent the script must fail before invoking the
// backend, so a partial message is never handed to himalaya.
func TestHimalayaSend_MissingRequiredVar(t *testing.T) {
	requireBash(t)
	t.Parallel()

	for _, missing := range []string{
		"APS_EMAIL_FROM",
		"EMAIL_TO",
		"EMAIL_SUBJECT",
		"EMAIL_BODY",
	} {
		t.Run(missing, func(t *testing.T) {
			t.Parallel()

			env := baseSendEnv()
			delete(env, missing)

			inv, out, err := runBackendScript(t, sendScript(t), env)
			if err == nil {
				t.Fatalf("expected failure with %s unset, got success\nargv: %s",
					missing, inv.argvString())
			}
			if len(inv.Argv) != 0 {
				t.Errorf("backend was invoked despite missing %s: %s",
					missing, inv.argvString())
			}
			if !strings.Contains(out, missing) {
				t.Errorf("diagnostic does not name %s:\n%s", missing, out)
			}
		})
	}
}

// TestHimalayaSend_AccountWithSpace is the word-splitting case.
//
// send.sh builds ACCOUNT_FLAG="-a $ACCOUNT" as a single string and
// expands it unquoted, so an account name containing a space splits
// into separate argv elements and himalaya receives a truncated
// account plus a stray positional argument.
//
// Account names with spaces are legal in himalaya's config, so this is
// reachable. The test asserts the correct two-element form; it fails
// against the current script by design, marking the bug rather than
// freezing it.
func TestHimalayaSend_AccountWithSpace(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseSendEnv()
	env["APS_EMAIL_ACCOUNT"] = "work account"

	inv, out, err := runBackendScript(t, sendScript(t), env)
	if err != nil {
		t.Fatalf("send.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "template", "send", "-a", "work account")
}

// containsLine reports whether the captured stdin has want as a full
// line.
func containsLine(inv backendInvocation, want string) bool {
	for _, line := range inv.stdinLines() {
		if line == want {
			return true
		}
	}
	return false
}

// assertStdinLines compares captured stdin line-for-line, reporting
// the first divergence plus both full texts. Blank lines are
// significant: the header/body separator is one.
func assertStdinLines(t *testing.T, inv backendInvocation, want []string) {
	t.Helper()

	got := inv.stdinLines()
	if len(got) != len(want) {
		t.Errorf("stdin line count = %d, want %d\ngot:\n%s\nwant:\n%s",
			len(got), len(want), strings.Join(got, "\n"), strings.Join(want, "\n"))
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stdin line %d = %q, want %q\ngot:\n%s\nwant:\n%s",
				i+1, got[i], want[i],
				strings.Join(got, "\n"), strings.Join(want, "\n"))
			return
		}
	}
}
