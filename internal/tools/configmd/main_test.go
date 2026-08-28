package main

import (
	"bytes"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
)

func render(t *testing.T, fragment string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := run([]string{fragment}, &buf); err != nil {
		t.Fatalf("run(%q): %v", fragment, err)
	}
	return buf.String()
}

func tableLines(out string) []string {
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

// TestSecretBackendsRowCount pins the table to the backend list: one
// row per backend plus the two header lines, no stale rows possible.
func TestSecretBackendsRowCount(t *testing.T) {
	lines := tableLines(render(t, "secret-backends"))
	if got, want := len(lines), len(core.SecretsBackends)+2; got != want {
		t.Fatalf("secret-backends rendered %d lines, want %d (header + separator + %d backends)",
			got, want, len(core.SecretsBackends))
	}
	for i, name := range core.SecretsBackends {
		if row := lines[i+2]; !strings.HasPrefix(row, "| `"+name+"`") {
			t.Errorf("row %d = %q, want backend %q", i, row, name)
		}
	}
}

// TestSecretBackendsMarksDefaultAndOptIn asserts the default suffix and
// the build-tag dagger render regardless of this binary's build tags.
func TestSecretBackendsMarksDefaultAndOptIn(t *testing.T) {
	out := render(t, "secret-backends")
	for _, want := range []string{
		"| `secrets.backend` | Where secrets live | Required config |",
		"| `file` (default) |",
		"| `openbao` † |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("secret-backends output missing %q:\n%s", want, out)
		}
	}
}

// TestConfigFieldsRowCount pins the table to ConfigFieldDocs: one row
// per documented field plus the two header lines.
func TestConfigFieldsRowCount(t *testing.T) {
	docs, err := core.ConfigFieldDocs()
	if err != nil {
		t.Fatalf("ConfigFieldDocs: %v", err)
	}
	lines := tableLines(render(t, "config-fields"))
	if got, want := len(lines), len(docs)+2; got != want {
		t.Fatalf("config-fields rendered %d lines, want %d (header + separator + %d fields)",
			got, want, len(docs))
	}
	for i, d := range docs {
		if row := lines[i+2]; !strings.HasPrefix(row, "| `"+d.Path+"`") {
			t.Errorf("row %d = %q, want field %q", i, row, d.Path)
		}
	}
}

// TestStableOutput asserts fragments render identically across calls,
// so docs-check never flaps.
func TestStableOutput(t *testing.T) {
	for _, fragment := range []string{"secret-backends", "config-fields"} {
		if first, second := render(t, fragment), render(t, fragment); first != second {
			t.Errorf("fragment %q output is unstable:\n%s\nvs:\n%s", fragment, first, second)
		}
	}
}

// TestBadInvocations asserts unknown fragments and missing args error,
// which main maps to a non-zero exit.
func TestBadInvocations(t *testing.T) {
	var buf bytes.Buffer
	if err := run([]string{"bogus"}, &buf); err == nil {
		t.Error("run with unknown fragment: want error, got nil")
	}
	if err := run(nil, &buf); err == nil {
		t.Error("run with no args: want error, got nil")
	}
	if err := run([]string{"secret-backends", "extra"}, &buf); err == nil {
		t.Error("run with extra args: want error, got nil")
	}
}
