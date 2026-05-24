package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOverrideEnv_InlineOnly(t *testing.T) {
	out, err := BuildOverrideEnv(nil, []string{"FOO=bar", "BAZ=qux"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"FOO=bar", "BAZ=qux"}
	if len(out) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(out), len(want), out)
	}
	for i, w := range want {
		if out[i] != w {
			t.Errorf("out[%d] = %q, want %q", i, out[i], w)
		}
	}
}

func TestBuildOverrideEnv_EmptyValueAllowed(t *testing.T) {
	out, err := BuildOverrideEnv(nil, []string{"FOO="})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(out) != 1 || out[0] != "FOO=" {
		t.Errorf("got %v, want [FOO=]", out)
	}
}

func TestBuildOverrideEnv_MissingEqualsRejected(t *testing.T) {
	_, err := BuildOverrideEnv(nil, []string{"FOO"})
	if err == nil {
		t.Fatalf("expected error for KEY without =")
	}
	if !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Errorf("error %q lacks guidance", err.Error())
	}
}

func TestBuildOverrideEnv_EmptyKeyRejected(t *testing.T) {
	_, err := BuildOverrideEnv(nil, []string{"=bar"})
	if err == nil {
		t.Fatalf("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "empty key") {
		t.Errorf("error %q does not name the empty-key violation", err.Error())
	}
}

func TestBuildOverrideEnv_InvalidKeyShape(t *testing.T) {
	_, err := BuildOverrideEnv(nil, []string{"1FOO=bar"})
	if err == nil {
		t.Fatalf("expected error for key starting with digit")
	}
}

func TestBuildOverrideEnv_FileEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vars.env")
	content := "# comment\n\nFOO=bar\nexport BAZ=qux\nQUOTED=\"hello world\"\nSQUOTED='one two'\nWITH_HASH=val # trailing\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := BuildOverrideEnv([]string{path}, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	want := []string{
		"FOO=bar",
		"BAZ=qux",
		"QUOTED=hello world",
		"SQUOTED=one two",
		"WITH_HASH=val",
	}
	if len(out) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(out), len(want), out)
	}
	for i, w := range want {
		if out[i] != w {
			t.Errorf("out[%d] = %q, want %q", i, out[i], w)
		}
	}
}

func TestBuildOverrideEnv_FileMissing(t *testing.T) {
	_, err := BuildOverrideEnv([]string{"/does/not/exist.env"}, nil)
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "/does/not/exist.env") {
		t.Errorf("error %q lacks path context", err.Error())
	}
}

func TestBuildOverrideEnv_FileMalformedReportsLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.env")
	content := "GOOD=ok\nbroken-line\nALSO=fine\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := BuildOverrideEnv([]string{path}, nil)
	if err == nil {
		t.Fatalf("expected error for malformed line")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error %q must include line number", err.Error())
	}
}

func TestBuildOverrideEnv_OrderFilesThenInline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.env")
	if err := os.WriteFile(path, []byte("FOO=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := BuildOverrideEnv([]string{path}, []string{"FOO=from-flag"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 entries, got %v", out)
	}
	if out[0] != "FOO=from-file" || out[1] != "FOO=from-flag" {
		t.Errorf("order wrong: %v", out)
	}
}

func TestDedupEnvLastWins(t *testing.T) {
	in := []string{
		"FOO=1",
		"BAR=a",
		"FOO=2",
		"BAZ=z",
		"FOO=3",
		"BAR=b",
	}
	got := dedupEnvLastWins(in)
	want := []string{"BAZ=z", "FOO=3", "BAR=b"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestDedupEnvLastWins_PreservesNonKV(t *testing.T) {
	in := []string{"FOO=1", "garbage", "FOO=2"}
	got := dedupEnvLastWins(in)
	want := []string{"garbage", "FOO=2"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}
