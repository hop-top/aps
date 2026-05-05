package session

// Golden-style test for the T-0456 migration: `aps session inspect`
// renders both the property block and the optional environment block
// via listing.RenderList. Non-TTY callers see plain tabwriter output.

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/kit/go/console/output"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestSessionPropertyRow_NonTTYPlainOutput(t *testing.T) {
	rows := []sessionPropertyRow{
		{Property: "ID", Value: "sess-001"},
		{Property: "Profile ID", Value: "alpha"},
		{Property: "PID", Value: "12345"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"PROPERTY", "VALUE",
		"ID", "sess-001", "Profile ID", "alpha", "PID", "12345",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in session property output, got: %q", want, out)
		}
	}
	if ansiRe.MatchString(out) {
		t.Errorf("non-TTY session property output leaked ANSI escapes: %q", out)
	}
}

func TestSessionEnvRow_NonTTYPlainOutput(t *testing.T) {
	rows := []sessionEnvRow{
		{Key: "PATH", Value: "/usr/bin"},
		{Key: "EDITOR", Value: "vim"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"ENVIRONMENT", "VALUE", "PATH", "/usr/bin", "EDITOR", "vim"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in session env output, got: %q", want, out)
		}
	}
	if ansiRe.MatchString(out) {
		t.Errorf("non-TTY session env output leaked ANSI escapes: %q", out)
	}
}
