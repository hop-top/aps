package fsperm

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"hop.top/aps/internal/atomicfile"
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

func TestWriteSecretFaultKeepsOldAndNoTemp(t *testing.T) {
	for _, stage := range []string{atomicfile.StageWrite, atomicfile.StageSync, atomicfile.StageRename} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "secrets.env")
			if err := WriteSecret(p, []byte("A=old\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			boom := errors.New("boom")
			restore := atomicfile.InjectFault(stage, boom)
			err := WriteSecret(p, []byte("A=new-and-longer\n"), 0o600)
			restore()
			if !errors.Is(err, boom) {
				t.Fatalf("err = %v", err)
			}
			if got, _ := os.ReadFile(p); string(got) != "A=old\n" {
				t.Fatalf("target = %q", got)
			}
			ents, _ := os.ReadDir(dir)
			if len(ents) != 1 {
				t.Fatalf("stray files: %v", ents)
			}
		})
	}
}

func TestWriteSecretSymlinkAndDangling(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "dotfiles", "secrets.env")
	if err := os.MkdirAll(filepath.Dir(realPath), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "secrets.env")
	if err := os.Symlink(realPath, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// dangling first: creates the file it points at
	if err := WriteSecret(link, []byte("A=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(realPath); string(got) != "A=1\n" {
		t.Fatalf("real = %q", got)
	}
	if err := WriteSecret(link, []byte("A=2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("link replaced by file")
	}
	if got, _ := os.ReadFile(realPath); string(got) != "A=2\n" {
		t.Fatalf("real = %q", got)
	}
}

func TestWriteSecretNoSyncFaultKeepsOldAndNoTemp(t *testing.T) {
	for _, stage := range []string{atomicfile.StageWrite, atomicfile.StageRename} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "q.json")
			if err := WriteSecretNoSync(p, []byte("old\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			boom := errors.New("boom")
			restore := atomicfile.InjectFault(stage, boom)
			err := WriteSecretNoSync(p, []byte("new-and-longer\n"), 0o600)
			restore()
			if !errors.Is(err, boom) {
				t.Fatalf("err = %v", err)
			}
			if got, _ := os.ReadFile(p); string(got) != "old\n" {
				t.Fatalf("target = %q", got)
			}
			if ents, _ := os.ReadDir(dir); len(ents) != 1 {
				t.Fatalf("stray files: %v", ents)
			}
		})
	}
}
