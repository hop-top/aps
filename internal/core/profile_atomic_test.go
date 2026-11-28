package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/atomicfile"
)

func TestSaveProfileFailedWriteKeepsFile(t *testing.T) {
	for _, stage := range []string{atomicfile.StageWrite, atomicfile.StageSync, atomicfile.StageRename} {
		t.Run(stage, func(t *testing.T) {
			t.Setenv("APS_DATA_PATH", t.TempDir())
			p := &Profile{ID: "atomic", DisplayName: "Atomic"}
			if err := SaveProfile(p); err != nil {
				t.Fatal(err)
			}
			path, err := GetProfilePath("atomic")
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			p.DisplayName = "A much longer display name than before"
			boom := errors.New("boom")
			restore := atomicfile.InjectFault(stage, boom)
			err = SaveProfile(p)
			restore()
			if !errors.Is(err, boom) {
				t.Fatalf("err = %v, want boom", err)
			}
			after, _ := os.ReadFile(path)
			if string(after) != string(before) {
				t.Fatalf("profile.yaml changed:\n%s\nwant:\n%s", after, before)
			}
			ents, _ := os.ReadDir(filepath.Dir(path))
			for _, e := range ents {
				if e.Name() != "profile.yaml" && e.Name() != "actions" {
					t.Fatalf("stray entry %q", e.Name())
				}
			}
		})
	}
}

func TestSaveProfileKeepsSymlink(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	p := &Profile{ID: "linked", DisplayName: "L"}
	if err := SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	path, _ := GetProfilePath("linked")
	realPath := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.Rename(path, realPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPath, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	p.DisplayName = "Changed"
	if err := SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(path); fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("link replaced")
	}
	if got, _ := os.ReadFile(realPath); !strings.Contains(string(got), "Changed") {
		t.Fatalf("realPath not updated: %s", got)
	}
}
