package squad

// Golden-style test for T-0477: `aps squad check` renders the
// topology validation report via listing.RenderList. Non-TTY callers
// see plain tabwriter output with no ANSI / box-drawing leakage.
// JSON/YAML round-trip via the typed `checkRow` preserves the field
// set.

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/kit/go/console/output"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestCheckRow_NonTTYPlainOutput(t *testing.T) {
	rows := []checkRow{
		{Check: "8.1 squad-bounded-context", Status: "PASS", Detail: "-"},
		{Check: "8.2 contracts-defined", Status: "FAIL", Detail: "missing contracts"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"CHECK", "STATUS", "DETAIL",
		"8.1 squad-bounded-context", "PASS",
		"8.2 contracts-defined", "FAIL", "missing contracts",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in check output, got: %q", want, out)
		}
	}
	if ansiRe.MatchString(out) {
		t.Errorf("non-TTY check output leaked ANSI escapes: %q", out)
	}
	for _, r := range []rune{'┌', '┐', '└', '┘', '│', '─'} {
		if strings.ContainsRune(out, r) {
			t.Errorf("non-TTY check output leaked box-drawing rune %q: %q", r, out)
		}
	}
}

func TestCheckRow_JSONRoundTrip(t *testing.T) {
	rows := []checkRow{
		{Check: "8.1 squad-bounded-context", Status: "PASS", Detail: "-"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.JSON, rows); err != nil {
		t.Fatalf("RenderList JSON: %v", err)
	}
	var got []checkRow
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, buf.String())
	}
	if len(got) != 1 || got[0] != rows[0] {
		t.Errorf("check JSON round-trip: got %+v, want %+v", got, rows)
	}
}
