package cli

// Golden-style tests for T-0456: `aps profile trust` renders the
// score and history tables via listing.RenderList. Non-TTY callers
// see plain tabwriter output with no ANSI/box-drawing leakage.

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/kit/go/console/output"
)

var trustAnsiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestTrustScoreRow_NonTTYPlainOutput(t *testing.T) {
	rows := []trustScoreRow{
		{Domain: "code", Score: 0.85},
		{Domain: "review", Score: 0.92},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"DOMAIN", "SCORE", "code", "review", "0.85", "0.92"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in trust score output, got: %q", want, out)
		}
	}
	if trustAnsiRe.MatchString(out) {
		t.Errorf("non-TTY trust score output leaked ANSI escapes: %q", out)
	}
}

func TestTrustHistoryRow_NonTTYPlainOutput(t *testing.T) {
	rows := []trustHistoryRow{
		{Timestamp: "2026-05-04 12:34", Domain: "code", Delta: 0.10, TaskRef: "T-100", Difficulty: "L1"},
		{Timestamp: "2026-05-04 13:00", Domain: "review", Delta: -0.05, TaskRef: "T-101", Difficulty: "L2"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"TIMESTAMP", "DOMAIN", "DELTA", "TASK", "DIFFICULTY",
		"2026-05-04 12:34", "T-100", "L1", "T-101", "L2",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in trust history output, got: %q", want, out)
		}
	}
	if trustAnsiRe.MatchString(out) {
		t.Errorf("non-TTY trust history output leaked ANSI escapes: %q", out)
	}
}
