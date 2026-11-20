package fsperm

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSecretContract(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secrets.env")

	if err := WriteSecret(p, []byte("A=1\nB=2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil || string(got) != "A=1\nB=2\n" {
		t.Fatalf("read back = %q, %v", got, err)
	}

	// Overwrite truncates.
	if err := WriteSecret(p, []byte("C=3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(p)
	if string(got) != "C=3\n" {
		t.Fatalf("after overwrite = %q", got)
	}

	warn, err := CheckSecret(p)
	if err != nil {
		t.Fatal(err)
	}
	if warn != "" {
		t.Fatalf("freshly written secret flagged: %q", warn)
	}
}

func TestWriteSecretMissingParent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nope", "secrets.env")
	if err := WriteSecret(p, []byte("x"), 0o600); err == nil {
		t.Fatal("expected error for missing parent dir")
	}
}

func TestOpenSecretAppend(t *testing.T) {
	p := filepath.Join(t.TempDir(), "metrics.jsonl")
	for _, s := range []string{"a\n", "b\n"} {
		f, err := OpenSecret(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(s); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	got, _ := os.ReadFile(p)
	if string(got) != "a\nb\n" {
		t.Fatalf("append = %q", got)
	}
	if warn, err := CheckSecret(p); err != nil || warn != "" {
		t.Fatalf("CheckSecret = %q, %v", warn, err)
	}
}

func TestCheckSecretMissing(t *testing.T) {
	_, err := CheckSecret(filepath.Join(t.TempDir(), "absent"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want ErrNotExist", err)
	}
}
