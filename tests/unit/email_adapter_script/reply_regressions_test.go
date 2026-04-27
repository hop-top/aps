// Package email_adapter_script tests the bash backend scripts that
// the email adapter dispatches to. The package intentionally has no
// imports from internal/* — the scripts are a separate artefact and
// must remain testable even when other internal packages fail to
// compile.
package email_adapter_script_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEmailReply_LargeHTMLNoSIGPIPE is a regression for T-0332.
//
// Repro: replying to a message whose template carries a multi-MB
// HTML alternative used to fail with status 141 (SIGPIPE). The
// script split the himalaya template via `echo "$T" | sed '/^$/q'`
// under `set -o pipefail`. sed quits after the first blank line,
// closing its stdin while echo is still writing the rest of the
// body — echo gets SIGPIPE, pipefail propagates 141, and `set -e`
// aborts the script before himalaya is ever invoked.
//
// Fix verified by this test: reply.sh must split the template
// without piping the full body through an early-exiting consumer.
func TestEmailReply_LargeHTMLNoSIGPIPE(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash adapter scripts are POSIX-only")
	}

	repoRoot := findRepoRoot(t)
	script := filepath.Join(
		repoRoot,
		"adapters", "email", "backends", "himalaya", "reply.sh",
	)
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("reply.sh missing: %v", err)
	}

	// Stub himalaya on PATH:
	//   `himalaya template reply ...`  -> emits a >50KB template
	//                                     (headers + blank + huge HTML)
	//   `himalaya template send ...`   -> echoes captured stdin so the
	//                                     test can assert the body
	//                                     reached the sender.
	stubDir := t.TempDir()
	stub := filepath.Join(stubDir, "himalaya")
	stubBody := `#!/usr/bin/env bash
case "$1 $2" in
  "template reply")
    printf 'From: stub@example.com\nTo: peer@example.com\nSubject: Re: big\n\n'
    # ~120KB of HTML — well over the 50KB acceptance threshold.
    awk 'BEGIN { for (i=0;i<2000;i++)
      printf "<p>line %d filler text to grow the body</p>\n", i }'
    ;;
  "template send")
    cat
    ;;
esac
`
	if err := os.WriteFile(stub, []byte(stubBody), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(),
		"PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"APS_EMAIL_FROM=stub@example.com",
		"EMAIL_ID=42",
		"EMAIL_BODY=Thanks for the report.",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("reply.sh failed: %v\nstderr: %s",
			err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Thanks for the report.") {
		t.Errorf("reply body missing from sent message; stdout=%q",
			out)
	}
	if !strings.Contains(out, "<p>line 1999") {
		t.Errorf("quoted HTML body truncated; stdout tail=%q",
			tail(out, 200))
	}
}

// TestEmailReply_PlainTextStillWorks guards the happy path the
// SIGPIPE fix could regress: a small plain-text-only template must
// still produce a well-formed message with header, body, and
// quoted original separated by blank lines.
func TestEmailReply_PlainTextStillWorks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash adapter scripts are POSIX-only")
	}

	repoRoot := findRepoRoot(t)
	script := filepath.Join(
		repoRoot,
		"adapters", "email", "backends", "himalaya", "reply.sh",
	)

	stubDir := t.TempDir()
	stub := filepath.Join(stubDir, "himalaya")
	stubBody := `#!/usr/bin/env bash
case "$1 $2" in
  "template reply")
    printf 'From: stub@example.com\nTo: peer@example.com\nSubject: Re: hi\n\n> original line one\n> original line two\n'
    ;;
  "template send")
    cat
    ;;
esac
`
	if err := os.WriteFile(stub, []byte(stubBody), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(),
		"PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"APS_EMAIL_FROM=stub@example.com",
		"EMAIL_ID=7",
		"EMAIL_BODY=Reply body here.",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("reply.sh failed: %v\nstderr: %s",
			err, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"Subject: Re: hi",
		"Reply body here.",
		"> original line one",
		"> original line two",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for d := dir; d != "/" && d != "."; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
	}
	t.Fatalf("repo root (go.mod) not found from %s", dir)
	return ""
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
