package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func noTemps(t *testing.T, dir string, want ...string) {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != len(want) {
		got := make([]string, 0, len(ents))
		for _, e := range ents {
			got = append(got, e.Name())
		}
		t.Fatalf("dir entries = %v, want %v", got, want)
	}
}

func TestWriteFaultsLeaveOldContent(t *testing.T) {
	boom := errors.New("boom")
	for _, stage := range []string{StageWrite, StageSync, StageRename} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "f")
			if err := os.WriteFile(p, []byte("old-content\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			restore := InjectFault(stage, boom)
			err := Write(p, []byte("new-content-longer\n"), 0o600, nil)
			restore()
			if !errors.Is(err, boom) {
				t.Fatalf("err = %v, want boom", err)
			}
			got, _ := os.ReadFile(p)
			if string(got) != "old-content\n" {
				t.Fatalf("target = %q, want old content", got)
			}
			noTemps(t, dir, "f")
		})
	}
}

func TestWriteFaultOnNewFileLeavesNothing(t *testing.T) {
	dir := t.TempDir()
	restore := InjectFault(StageRename, errors.New("boom"))
	defer restore()
	if err := Write(filepath.Join(dir, "f"), []byte("x"), 0o600, nil); err == nil {
		t.Fatal("want error")
	}
	noTemps(t, dir)
}

func TestWriteReplaces(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	for _, s := range []string{"a-long-first-value\n", "b\n"} {
		if err := Write(p, []byte(s), 0o600, nil); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(p); string(got) != s {
			t.Fatalf("got %q want %q", got, s)
		}
	}
	noTemps(t, dir, "f")
}

func TestWriteMissingParent(t *testing.T) {
	if err := Write(filepath.Join(t.TempDir(), "no", "f"), []byte("x"), 0o600, nil); err == nil {
		t.Fatal("want error")
	}
}

func TestWriteSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "real")
	if err := os.Mkdir(realPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realPath, "f"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Join("real", "f"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := Write(link, []byte("new"), 0o600, nil); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link not preserved: %v %v", fi, err)
	}
	if got, _ := os.ReadFile(filepath.Join(realPath, "f")); string(got) != "new" {
		t.Fatalf("real file = %q", got)
	}
	noTemps(t, realPath, "f")
	noTemps(t, dir, "link", "real")
}

func TestWriteDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	if err := os.Symlink("target", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := Write(link, []byte("new"), 0o600, nil); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link not preserved: %v %v", fi, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "target")); string(got) != "new" {
		t.Fatalf("target = %q", got)
	}
	noTemps(t, dir, "link", "target")
}

func TestWriteNoSyncFaultsLeaveOldContent(t *testing.T) {
	boom := errors.New("boom")
	for _, stage := range []string{StageWrite, StageRename} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "f")
			if err := os.WriteFile(p, []byte("old-content\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			restore := InjectFault(stage, boom)
			err := WriteNoSync(p, []byte("new-content-longer\n"), 0o600, nil)
			restore()
			if !errors.Is(err, boom) {
				t.Fatalf("err = %v, want boom", err)
			}
			if got, _ := os.ReadFile(p); string(got) != "old-content\n" {
				t.Fatalf("target = %q, want old content", got)
			}
			noTemps(t, dir, "f")
		})
	}
}

func TestWriteNoSyncSkipsSyncAndReplaces(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	// a sync fault must not fire: the no-sync path never syncs
	restore := InjectFault(StageSync, errors.New("sync must not run"))
	defer restore()
	for _, s := range []string{"first-longer-value\n", "b\n"} {
		if err := WriteNoSync(p, []byte(s), 0o600, nil); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(p); string(got) != s {
			t.Fatalf("got %q want %q", got, s)
		}
	}
	noTemps(t, dir, "f")
}
