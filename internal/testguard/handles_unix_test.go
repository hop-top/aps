//go:build !windows

package testguard

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunFailsWhenFileStaysOpen(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available")
	}
	var f *os.File
	code := Run(fakeM{func() {
		var err error
		f, err = os.Create(filepath.Join(os.Getenv("XDG_DATA_HOME"), "held.db"))
		if err != nil {
			// Data dir does not exist yet; create it.
			if mkErr := os.MkdirAll(os.Getenv("XDG_DATA_HOME"), 0o750); mkErr != nil {
				t.Errorf("mkdir: %v", mkErr)
				return
			}
			f, err = os.Create(filepath.Join(os.Getenv("XDG_DATA_HOME"), "held.db"))
			if err != nil {
				t.Errorf("create: %v", err)
			}
		}
	}})
	if f != nil {
		if err := f.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	}
	if code == 0 {
		t.Fatal("Run returned 0 though a file under the sandbox was still open")
	}
}

func TestRunPassesWhenFileClosed(t *testing.T) {
	code := Run(fakeM{func() {
		dir := os.Getenv("XDG_DATA_HOME")
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Errorf("mkdir: %v", err)
			return
		}
		f, err := os.Create(filepath.Join(dir, "closed.db"))
		if err != nil {
			t.Errorf("create: %v", err)
			return
		}
		if err := f.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	}})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
}
