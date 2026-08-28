package main

import (
	"bytes"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
)

const headerRows = 2 // column header + separator

// aliasCount counts the serviceTypeAliases entries resolving to the given
// canonical type, via the exported accessor, so the pins track the live map.
func aliasCount(canonicalType string) int {
	count := 0
	for _, target := range core.ServiceTypeAliases() {
		if strings.HasPrefix(target, canonicalType+" ") {
			count++
		}
	}
	return count
}

func renderOrFail(t *testing.T, fragment string) string {
	t.Helper()
	out, err := render(fragment)
	if err != nil {
		t.Fatalf("render(%q): %v", fragment, err)
	}
	return out
}

func rowCount(out string) int {
	return len(strings.Split(strings.TrimSuffix(out, "\n"), "\n"))
}

func TestRenderRowCountsPinnedToMaps(t *testing.T) {
	cases := []struct {
		fragment string
		want     int
	}{
		// One row per known message adapter, aliased or not.
		{"message-aliases", headerRows + len(core.MessageAdapterCatalogue())},
		{"ticket-aliases", headerRows + aliasCount("ticket")},
		// Explicit --type/--adapter example row plus one row per alias.
		{"persisted-adapters", headerRows + 1 + aliasCount("message")},
		{"ticket-adapters", headerRows + aliasCount("ticket")},
	}
	for _, tc := range cases {
		out := renderOrFail(t, tc.fragment)
		if got := rowCount(out); got != tc.want {
			t.Errorf("%s: got %d table rows, want %d\n%s", tc.fragment, got, tc.want, out)
		}
	}
}

func TestRenderStableOutput(t *testing.T) {
	fragments := []string{"message-aliases", "ticket-aliases", "persisted-adapters", "ticket-adapters"}
	for _, fragment := range fragments {
		first := renderOrFail(t, fragment)
		for i := 0; i < 5; i++ {
			if again := renderOrFail(t, fragment); again != first {
				t.Fatalf("%s: unstable output\nfirst:\n%s\nagain:\n%s", fragment, first, again)
			}
		}
		if !strings.HasSuffix(first, "|\n") {
			t.Errorf("%s: output does not end with a table row", fragment)
		}
	}
}

func TestRenderRowsSorted(t *testing.T) {
	out := renderOrFail(t, "ticket-adapters")
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")[headerRows:]
	for i := 1; i < len(lines); i++ {
		if lines[i-1] >= lines[i] {
			t.Errorf("ticket-adapters rows not strictly sorted:\n%s\n%s", lines[i-1], lines[i])
		}
	}
}

func TestRunUnknownFragmentExitsNonZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"bogus"}, &stdout, &stderr); code == 0 {
		t.Fatal("run with unknown fragment returned exit code 0")
	}
	if !strings.Contains(stderr.String(), "unknown fragment") {
		t.Errorf("stderr missing unknown-fragment diagnostic: %q", stderr.String())
	}
}

func TestRunNoArgsExitsNonZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code == 0 {
		t.Fatal("run without arguments returned exit code 0")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr missing usage line: %q", stderr.String())
	}
}

func TestRunKnownFragmentExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"message-aliases"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run(message-aliases) exit code %d, stderr: %s", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("run(message-aliases) wrote nothing to stdout")
	}
}
