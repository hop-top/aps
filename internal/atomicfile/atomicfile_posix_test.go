//go:build !windows

package atomicfile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWriteAppliesPermToReplacedFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(p, []byte("old"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o666); err != nil {
		t.Fatal(err)
	}
	if err := Write(p, []byte("new"), 0o600, nil); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", fi.Mode().Perm())
	}
}

func withUmask(t *testing.T, m int) {
	t.Helper()
	old := syscall.Umask(m)
	t.Cleanup(func() { syscall.Umask(old) })
}

func TestWriteKeepModeCarriesExistingMode(t *testing.T) {
	for _, mode := range []os.FileMode{0o600, 0o640, 0o644, 0o664} {
		t.Run(mode.String(), func(t *testing.T) {
			withUmask(t, 0o022)
			p := filepath.Join(t.TempDir(), "f")
			if err := os.WriteFile(p, []byte("old"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(p, mode); err != nil {
				t.Fatal(err)
			}
			if err := WriteKeepMode(p, []byte("new"), 0o644); err != nil {
				t.Fatal(err)
			}
			if fi, _ := os.Stat(p); fi.Mode().Perm() != mode {
				t.Fatalf("mode = %o, want %o", fi.Mode().Perm(), mode)
			}
			if got, _ := os.ReadFile(p); string(got) != "new" {
				t.Fatalf("got %q", got)
			}
		})
	}
}

func TestWriteKeepModeNewFileUsesPermMinusUmask(t *testing.T) {
	withUmask(t, 0o027)
	p := filepath.Join(t.TempDir(), "f")
	if err := WriteKeepMode(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", fi.Mode().Perm())
	}
}

func TestWriteKeepModeSymlinkCarriesRealFileMode(t *testing.T) {
	withUmask(t, 0o022)
	dir := t.TempDir()
	realPath := filepath.Join(dir, "real")
	if err := os.WriteFile(realPath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(realPath, link); err != nil {
		t.Skip(err)
	}
	if err := WriteKeepMode(link, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(realPath); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", fi.Mode().Perm())
	}
}
