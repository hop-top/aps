package policy

// Golden-style test for the T-0456 migration: the policy list table
// must emit headers + per-setting rows via listing.RenderList, with
// no ANSI/box-drawing leakage on the non-TTY path. The test is
// deliberately structural (substring + ANSI absence) because the
// kit/output column projection is responsive to terminal width.

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/kit/go/console/output"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestPolicyRow_NonTTYPlainOutput(t *testing.T) {
	rows := []policyRow{
		{Setting: "Mode", Value: "allow-all"},
		{Setting: "Allowed Devices", Value: "device-1"},
		{Setting: "", Value: "device-2"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"SETTING", "VALUE", "Mode", "allow-all", "Allowed Devices", "device-1", "device-2"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in policy table output, got: %q", want, out)
		}
	}
	if ansiRe.MatchString(out) {
		t.Errorf("non-TTY policy output leaked ANSI escapes: %q", out)
	}
	for _, r := range []rune{'┌', '┐', '└', '┘', '│', '─'} {
		if strings.ContainsRune(out, r) {
			t.Errorf("non-TTY policy output leaked box-drawing rune %q: %q", r, out)
		}
	}
}
