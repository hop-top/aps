package migrate

// Golden-style test for the T-0456 migration: the dry-run preview
// table must surface every messenger row via listing.RenderList with
// no ANSI/box-drawing leakage on the non-TTY path.

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/kit/go/console/output"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestMigrateDryRunRow_NonTTYPlainOutput(t *testing.T) {
	rows := []migrateDryRunRow{
		{Messenger: "telegram", Type: "messenger", Scope: "global", Action: "migrate"},
		{Messenger: "discord", Type: "messenger", Scope: "profile (alpha)", Action: "migrate"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"MESSENGER", "TYPE", "SCOPE", "ACTION",
		"telegram", "discord", "global", "profile (alpha)", "migrate",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in migrate dry-run output, got: %q", want, out)
		}
	}
	if ansiRe.MatchString(out) {
		t.Errorf("non-TTY migrate output leaked ANSI escapes: %q", out)
	}
	for _, r := range []rune{'┌', '┐', '└', '┘', '│', '─'} {
		if strings.ContainsRune(out, r) {
			t.Errorf("non-TTY migrate output leaked box-drawing rune %q: %q", r, out)
		}
	}
}
