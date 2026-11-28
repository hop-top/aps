//go:build !windows

package fsperm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPOSIXModeAndWarning(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s")
	if err := WriteSecret(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", fi.Mode().Perm())
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	warn, err := CheckSecret(p)
	if err != nil {
		t.Fatal(err)
	}
	want := "insecure permissions (644). It should be 0600."
	if warn != want {
		t.Fatalf("warn = %q, want %q", warn, want)
	}
}

func TestWriteSecretTightensBroaderMode(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteSecret(p, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", fi.Mode().Perm())
	}
}
