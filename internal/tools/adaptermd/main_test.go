package main

import (
	"bytes"
	"strings"
	"testing"

	"hop.top/aps/internal/core/adapter"
)

// render runs one fragment and fails the test on a non-zero exit.
func render(t *testing.T, fragment string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := run([]string{fragment}, &stdout, &stderr); code != 0 {
		t.Fatalf("run(%q) exited %d, stderr: %s", fragment, code, stderr.String())
	}
	return stdout.String()
}

// countPrefixed returns how many lines of s start with prefix.
func countPrefixed(s, prefix string) int {
	n := 0
	for line := range strings.Lines(s) {
		if strings.HasPrefix(line, prefix) {
			n++
		}
	}
	return n
}

func TestKindsTableRowPerKind(t *testing.T) {
	out := render(t, "kinds-table")
	if got, want := countPrefixed(out, "| **"), len(adapter.AdapterTypes); got != want {
		t.Errorf("kinds-table has %d rows, want %d (one per AdapterTypes entry)", got, want)
	}
	for k := range adapter.AdapterTypes {
		if !strings.Contains(out, "`"+string(k)+"`") {
			t.Errorf("kinds-table missing key %q", k)
		}
	}
}

func TestStrategiesTableRowPerStrategy(t *testing.T) {
	out := render(t, "strategies-table")
	if got, want := countPrefixed(out, "| **"), len(adapter.LoadingStrategies); got != want {
		t.Errorf("strategies-table has %d rows, want %d (one per LoadingStrategies entry)", got, want)
	}
	for _, meta := range adapter.LoadingStrategies {
		if !strings.Contains(out, "| **"+meta.Display+"** |") {
			t.Errorf("strategies-table missing strategy %q", meta.Display)
		}
	}
}

func TestKindsInlineEntryPerKind(t *testing.T) {
	out := strings.TrimSpace(render(t, "kinds-inline"))
	if !strings.HasPrefix(out, "(") || !strings.HasSuffix(out, ")") {
		t.Fatalf("kinds-inline not parenthesized: %q", out)
	}
	entries := strings.Split(strings.Trim(out, "()"), " / ")
	if got, want := len(entries), len(adapter.AdapterTypes); got != want {
		t.Errorf("kinds-inline has %d entries, want %d (one per AdapterTypes entry)", got, want)
	}
	for k := range adapter.AdapterTypes {
		found := false
		for _, e := range entries {
			if e == string(k) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("kinds-inline missing kind %q", k)
		}
	}
}

func TestKindsListEntryPerKind(t *testing.T) {
	out := render(t, "kinds-list")
	if got, want := countPrefixed(out, "- **"), len(adapter.AdapterTypes); got != want {
		t.Errorf("kinds-list has %d bullets, want %d (one per AdapterTypes entry)", got, want)
	}
}

func TestFragmentsStableAcrossRuns(t *testing.T) {
	for _, fragment := range []string{"kinds-table", "strategies-table", "kinds-inline", "kinds-list"} {
		first := render(t, fragment)
		for range 5 {
			if again := render(t, fragment); again != first {
				t.Errorf("%s output unstable across runs:\n%s\nvs\n%s", fragment, first, again)
				break
			}
		}
	}
}

func TestUnknownFragmentFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"bogus"}, &stdout, &stderr); code == 0 {
		t.Error("run with unknown fragment exited 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("unknown fragment stderr missing usage, got: %s", stderr.String())
	}
}

func TestMissingArgFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code == 0 {
		t.Error("run with no args exited 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("no-arg stderr missing usage, got: %s", stderr.String())
	}
}
