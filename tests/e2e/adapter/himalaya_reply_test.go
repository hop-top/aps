package adapter_e2e

import (
	"strings"
	"testing"
)

// Coverage for adapters/email/backends/himalaya/reply.sh.
//
// reply.sh is the only email action that invokes the backend twice:
// `template reply` produces a reply skeleton, the script splits that
// with sed into header block and quoted original, then pipes a
// reassembled message into `template send`. Both calls must carry the
// account flag, and the reassembly must place the operator's body
// between the headers and the quoted original.

// replyScript resolves reply.sh from the adapters tree.
func replyScript(t *testing.T) backendScript {
	t.Helper()
	return backendScript{Adapter: "email", Backend: "himalaya", Action: "reply"}
}

// replyTemplate is a representative `himalaya template reply` result:
// a header block, a blank separator, then the quoted original.
const replyTemplate = "From: ops@example.com\n" +
	"To: sender@example.com\n" +
	"Subject: Re: Deploy status\n" +
	"\n" +
	"> original line one\n" +
	"> original line two\n"

// replyStubOptions scripts the first call's output and declares that
// the second call receives stdin.
func replyStubOptions(template string) stubOptions {
	return stubOptions{
		Responses: []stubResponse{
			{Match: []string{"template", "reply"}, Stdout: template},
		},
		ReadsStdin: [][]string{{"template", "send"}},
	}
}

// baseReplyEnv is the minimum env reply.sh requires.
func baseReplyEnv() map[string]string {
	return map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"EMAIL_ID":       "7131",
		"EMAIL_BODY":     "Confirmed, shipping now.",
	}
}

// TestHimalayaReply_TwoCallFlow is the baseline: both invocations, in
// order, with the From header threaded into the template call and the
// reassembled message piped into the send call.
func TestHimalayaReply_TwoCallFlow(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, replyScript(t),
		baseReplyEnv(), replyStubOptions(replyTemplate))
	if err != nil {
		t.Fatalf("reply.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d:\n%s", len(invs), invocationsString(invs))
	}

	assertArgv(t, invs[0], "template", "reply", "7131", "-H", "From:ops@example.com")
	assertArgv(t, invs[1], "template", "send")

	assertStdinLines(t, invs[1], []string{
		"From: ops@example.com",
		"To: sender@example.com",
		"Subject: Re: Deploy status",
		"",
		"Confirmed, shipping now.",
		"",
		"> original line one",
		"> original line two",
	})
}

// TestHimalayaReply_WithAccount asserts the account flag reaches BOTH
// calls. Omitting it from either one silently targets the wrong
// account: the template would be built from one and sent from another.
func TestHimalayaReply_WithAccount(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseReplyEnv()
	env["APS_EMAIL_ACCOUNT"] = "work"

	invs, out, err := runBackendScriptMulti(t, replyScript(t),
		env, replyStubOptions(replyTemplate))
	if err != nil {
		t.Fatalf("reply.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d:\n%s", len(invs), invocationsString(invs))
	}

	assertArgv(t, invs[0], "template", "reply", "7131", "-H", "From:ops@example.com", "-a", "work")
	assertArgv(t, invs[1], "template", "send", "-a", "work")
}

// TestHimalayaReply_AccountWithSpace is the word-splitting case, and
// it applies twice over: reply.sh expands $ACCOUNT_FLAG unquoted in
// both invocations.
func TestHimalayaReply_AccountWithSpace(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseReplyEnv()
	env["APS_EMAIL_ACCOUNT"] = "work account"

	invs, out, err := runBackendScriptMulti(t, replyScript(t),
		env, replyStubOptions(replyTemplate))
	if err != nil {
		t.Fatalf("reply.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d:\n%s", len(invs), invocationsString(invs))
	}

	assertArgv(t, invs[0], "template", "reply", "7131", "-H", "From:ops@example.com", "-a", "work account")
	assertArgv(t, invs[1], "template", "send", "-a", "work account")
}

// TestHimalayaReply_MultilineBody covers a body spanning several
// lines: it must land intact between the header block and the quoted
// original, without disturbing either boundary.
func TestHimalayaReply_MultilineBody(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseReplyEnv()
	env["EMAIL_BODY"] = "First point.\n\nSecond point."

	invs, out, err := runBackendScriptMulti(t, replyScript(t),
		env, replyStubOptions(replyTemplate))
	if err != nil {
		t.Fatalf("reply.sh failed: %v\noutput: %s", err, out)
	}

	assertStdinLines(t, invs[1], []string{
		"From: ops@example.com",
		"To: sender@example.com",
		"Subject: Re: Deploy status",
		"",
		"First point.",
		"",
		"Second point.",
		"",
		"> original line one",
		"> original line two",
	})
}

// TestHimalayaReply_MissingRequiredVar covers the ${VAR:?} guards.
// The backend must not be invoked at all, so a missing body cannot
// produce an empty reply that still gets sent.
func TestHimalayaReply_MissingRequiredVar(t *testing.T) {
	requireBash(t)
	t.Parallel()

	for _, missing := range []string{
		"APS_EMAIL_FROM",
		"EMAIL_ID",
		"EMAIL_BODY",
	} {
		t.Run(missing, func(t *testing.T) {
			t.Parallel()

			env := baseReplyEnv()
			delete(env, missing)

			invs, out, err := runBackendScriptMulti(t, replyScript(t),
				env, replyStubOptions(replyTemplate))
			if err == nil {
				t.Fatalf("expected failure with %s unset, got success:\n%s",
					missing, invocationsString(invs))
			}
			if len(invs) != 0 {
				t.Errorf("backend invoked despite missing %s:\n%s",
					missing, invocationsString(invs))
			}
			if !strings.Contains(out, missing) {
				t.Errorf("diagnostic does not name %s:\n%s", missing, out)
			}
		})
	}
}

// TestHimalayaReply_QuotedOriginalWithBlankLines guards the sed split.
//
// HEADER takes everything through the first blank line and QUOTED
// takes everything after it. A quoted original containing its own
// blank lines (a reply to a multi-paragraph message — entirely
// ordinary) must not confuse that split: only the FIRST blank line
// separates headers from body.
func TestHimalayaReply_QuotedOriginalWithBlankLines(t *testing.T) {
	requireBash(t)
	t.Parallel()

	template := "From: ops@example.com\n" +
		"To: sender@example.com\n" +
		"Subject: Re: Multi-paragraph\n" +
		"\n" +
		"> first paragraph\n" +
		">\n" +
		"> second paragraph\n"

	invs, out, err := runBackendScriptMulti(t, replyScript(t),
		baseReplyEnv(), replyStubOptions(template))
	if err != nil {
		t.Fatalf("reply.sh failed: %v\noutput: %s", err, out)
	}

	assertStdinLines(t, invs[1], []string{
		"From: ops@example.com",
		"To: sender@example.com",
		"Subject: Re: Multi-paragraph",
		"",
		"Confirmed, shipping now.",
		"",
		"> first paragraph",
		">",
		"> second paragraph",
	})
}
