package workspace

// Golden-style test for T-0473: `aps workspace activity` renders
// the event table via listing.RenderList. Non-TTY callers see plain
// tabwriter output with no ANSI / box-drawing leakage. JSON/YAML
// round-trip via the typed `activityRow` preserves the field set.

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/kit/go/console/output"
)

func TestActivityRow_NonTTYPlainOutput(t *testing.T) {
	rows := []activityRow{
		{Timestamp: "12:00:00", Event: "profile.created", Device: "dev-1", Detail: "alpha"},
		{Timestamp: "12:01:00", Event: "action.executed", Device: "--", Detail: "deploy"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"TIMESTAMP", "EVENT", "DEVICE", "DETAIL",
		"12:00:00", "profile.created", "dev-1", "alpha",
		"12:01:00", "action.executed", "deploy",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in activity output, got: %q", want, out)
		}
	}
	if ansiRe.MatchString(out) {
		t.Errorf("non-TTY activity output leaked ANSI escapes: %q", out)
	}
	for _, r := range []rune{'┌', '┐', '└', '┘', '│', '─'} {
		if strings.ContainsRune(out, r) {
			t.Errorf("non-TTY activity output leaked box-drawing rune %q: %q", r, out)
		}
	}
}

func TestActivityRow_JSONRoundTrip(t *testing.T) {
	rows := []activityRow{
		{Timestamp: "12:00:00", Event: "profile.created", Device: "dev-1", Detail: "alpha"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.JSON, rows); err != nil {
		t.Fatalf("RenderList JSON: %v", err)
	}
	var got []activityRow
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, buf.String())
	}
	if len(got) != 1 || got[0] != rows[0] {
		t.Errorf("activity row JSON round-trip: got %+v, want %+v", got, rows)
	}
}
