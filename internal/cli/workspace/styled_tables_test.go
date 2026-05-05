package workspace

// Golden-style tests for T-0456: the four workspace tables that
// previously used the shared newTabWriter() factory now route through
// listing.RenderList. Non-TTY callers see plain tabwriter output.
//
// The factory itself is gone — these tests guard the row contracts
// (struct tags + cell text) so a future schema change has to update
// the row type and the test together.

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/kit/go/console/output"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func assertPlainTable(t *testing.T, label, out string, want []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("%s: expected %q in output, got: %q", label, w, out)
		}
	}
	if ansiRe.MatchString(out) {
		t.Errorf("%s: non-TTY output leaked ANSI escapes: %q", label, out)
	}
	for _, r := range []rune{'┌', '┐', '└', '┘', '│', '─'} {
		if strings.ContainsRune(out, r) {
			t.Errorf("%s: non-TTY output leaked box-drawing rune %q: %q", label, r, out)
		}
	}
}

func TestMemberRow_NonTTYPlainOutput(t *testing.T) {
	rows := []memberRow{
		{Agent: "alpha", Role: "owner", Status: "active", LastSeen: "2026-05-04 12:00:00"},
		{Agent: "beta", Role: "editor", Status: "idle", LastSeen: "2026-05-04 11:30:00"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	assertPlainTable(t, "members", buf.String(), []string{
		"AGENT", "ROLE", "STATUS", "LAST SEEN",
		"alpha", "owner", "active", "beta", "editor", "idle",
	})
}

func TestTaskRow_NonTTYPlainOutput(t *testing.T) {
	rows := []taskRow{
		{ID: "abc12345", Action: "review", From: "alpha", To: "beta", Status: "WORKING", Age: "5m"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	assertPlainTable(t, "tasks", buf.String(), []string{
		"ID", "ACTION", "FROM", "TO", "STATUS", "AGE",
		"abc12345", "review", "alpha", "beta", "WORKING", "5m",
	})
}

func TestAgentMatchRow_NonTTYPlainOutput(t *testing.T) {
	rows := []agentMatchRow{
		{Agent: "alpha", Score: "85%", Match: "webhooks"},
		{Agent: "beta", Score: "70%", Match: "github"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	assertPlainTable(t, "agents", buf.String(), []string{
		"AGENT", "SCORE", "MATCH",
		"alpha", "85%", "webhooks", "beta", "70%", "github",
	})
}

func TestAuditRow_NonTTYPlainOutput(t *testing.T) {
	rows := []auditRow{
		{Time: "12:00:00", Actor: "alpha", Event: "task.created", Resource: "task-1"},
		{Time: "12:01:00", Actor: "beta", Event: "ctx.set", Resource: "key-1"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	assertPlainTable(t, "audit", buf.String(), []string{
		"TIME", "ACTOR", "EVENT", "RESOURCE",
		"12:00:00", "alpha", "task.created", "task-1",
		"12:01:00", "beta", "ctx.set", "key-1",
	})
}

func TestCtxHistoryRow_NonTTYPlainOutput(t *testing.T) {
	rows := []ctxHistoryRow{
		{Version: "v1", Agent: "alpha", Old: "", New: "value-1", Time: "12:00:00"},
		{Version: "v2", Agent: "beta", Old: "value-1", New: "value-2", Time: "12:01:00"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	assertPlainTable(t, "ctx history", buf.String(), []string{
		"VERSION", "AGENT", "OLD", "NEW", "TIME",
		"v1", "alpha", "value-1", "v2", "beta", "value-2",
	})
}
