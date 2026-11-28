//go:build !windows

package core

import (
	"os"
	"syscall"
	"testing"
)

func saveWithUmask(t *testing.T, umask int, p *Profile) {
	t.Helper()
	old := syscall.Umask(umask)
	defer syscall.Umask(old)
	if err := SaveProfile(p); err != nil {
		t.Fatal(err)
	}
}

func TestSaveProfileKeepsExistingMode(t *testing.T) {
	for _, mode := range []os.FileMode{0o600, 0o640} {
		t.Run(mode.String(), func(t *testing.T) {
			t.Setenv("APS_DATA_PATH", t.TempDir())
			p := &Profile{ID: "modekeep", DisplayName: "M"}
			saveWithUmask(t, 0o022, p)
			path, _ := GetProfilePath("modekeep")
			if err := os.Chmod(path, mode); err != nil {
				t.Fatal(err)
			}
			p.DisplayName = "Changed"
			saveWithUmask(t, 0o022, p)
			if fi, _ := os.Stat(path); fi.Mode().Perm() != mode {
				t.Fatalf("mode = %o, want %o", fi.Mode().Perm(), mode)
			}
		})
	}
}

func TestSaveProfileNewFileModeIs0644MinusUmask(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	saveWithUmask(t, 0o027, &Profile{ID: "modenew", DisplayName: "N"})
	path, _ := GetProfilePath("modenew")
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", fi.Mode().Perm())
	}
}
