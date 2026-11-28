package squad

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	helperEnv     = "SQUAD_LOCK_HELPER"
	helperDirEnv  = "SQUAD_LOCK_DIR"
	helperArgEnv  = "SQUAD_LOCK_ARG"
	helperWaitEnv = "SQUAD_LOCK_TIMEOUT"
)

// TestLockHelperProcess is not a test: it is the child entry point for the
// cross-process tests. It runs one store operation and exits 0 on success,
// 1 on error (message on stderr).
func TestLockHelperProcess(t *testing.T) {
	op := os.Getenv(helperEnv)
	if op == "" {
		t.Skip("helper process only")
	}
	m := NewFileManager(os.Getenv(helperDirEnv))
	if d, err := time.ParseDuration(os.Getenv(helperWaitEnv)); err == nil {
		m.lockTimeout = d
	}
	arg := os.Getenv(helperArgEnv)
	var err error
	switch op {
	case "create":
		err = m.Create(newTestSquad(arg))
	case "addmember":
		err = m.AddMember("shared", arg)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func startHelper(t *testing.T, dir, op, arg, timeout string) (*exec.Cmd, *strings.Builder) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockHelperProcess$")
	cmd.Env = append(os.Environ(),
		helperEnv+"="+op, helperDirEnv+"="+dir, helperArgEnv+"="+arg, helperWaitEnv+"="+timeout)
	var out strings.Builder
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd, &out
}

func TestStore_ConcurrentCreateSameID(t *testing.T) {
	dir := t.TempDir()
	const n = 12
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := NewFileManager(dir) // separate Store values, one dir
			<-start
			errs[i] = m.Create(newTestSquad("dup"))
		}()
	}
	close(start)
	wg.Wait()
	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !strings.Contains(err.Error(), "squad already exists"):
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d creates succeeded, want exactly 1", ok)
	}
}

func TestStore_ConcurrentAddMemberNoLostUpdate(t *testing.T) {
	dir := t.TempDir()
	if err := NewFileManager(dir).Create(newTestSquad("shared")); err != nil {
		t.Fatal(err)
	}
	const n = 16
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := NewFileManager(dir)
			// Generous bound: this test is about lost updates, not speed, and
			// sixteen queued writers can exceed the default wait on a slow disk.
			m.lockTimeout = 30 * time.Second
			<-start
			if err := m.AddMember("shared", fmt.Sprintf("p%d", i)); err != nil {
				t.Errorf("add p%d: %v", i, err)
			}
		}()
	}
	close(start)
	wg.Wait()
	got, err := NewFileManager(dir).Get("shared")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Members) != n {
		t.Fatalf("members = %d (%v), want %d: updates lost", len(got.Members), got.Members, n)
	}
}

// Real processes: the parent holds the store lock, a child must block and
// then fail with the timeout error; once released a child succeeds.
func TestStore_CrossProcessExclusion(t *testing.T) {
	dir := t.TempDir()
	release, err := acquireStoreLock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cmd, out := startHelper(t, dir, "create", "blocked", "300ms")
	werr := cmd.Wait()
	if werr == nil {
		release()
		t.Fatal("child wrote while parent held the lock")
	}
	if !strings.Contains(out.String(), filepath.Join(dir, lockName)) || !strings.Contains(out.String(), "timed out") {
		t.Errorf("child stderr = %q, want timeout error naming the lock path", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "blocked")); err == nil {
		t.Error("blocked child left a squad behind")
	}
	release()
	cmd, out = startHelper(t, dir, "create", "free", "5s")
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child after release: %v: %s", err, out)
	}
}

// Several child processes mutate one squad at once: none may be lost.
func TestStore_CrossProcessNoLostUpdate(t *testing.T) {
	dir := t.TempDir()
	if err := NewFileManager(dir).Create(newTestSquad("shared")); err != nil {
		t.Fatal(err)
	}
	const n = 8
	cmds := make([]*exec.Cmd, n)
	outs := make([]*strings.Builder, n)
	for i := range cmds {
		cmds[i], outs[i] = startHelper(t, dir, "addmember", fmt.Sprintf("p%d", i), "30s")
	}
	for i, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Errorf("child %d: %v: %s", i, err, outs[i])
		}
	}
	got, err := NewFileManager(dir).Get("shared")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Members) != n {
		t.Fatalf("members = %d (%v), want %d: updates lost", len(got.Members), got.Members, n)
	}
}

func TestStore_LockTimeoutError(t *testing.T) {
	dir := t.TempDir()
	release, err := acquireStoreLock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	m := NewFileManager(dir)
	m.lockTimeout = 50 * time.Millisecond
	err = m.Create(newTestSquad("x"))
	if err == nil || !strings.Contains(err.Error(), filepath.Join(dir, lockName)) {
		t.Fatalf("err = %v, want timeout naming lock path", err)
	}
}

func TestStore_LockReleasedAfterError(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	m.lockTimeout = 200 * time.Millisecond
	if err := m.Create(newTestSquad("a")); err != nil {
		t.Fatal(err)
	}
	// error paths inside the critical section
	if err := m.Create(newTestSquad("a")); err == nil {
		t.Fatal("duplicate create succeeded")
	}
	if err := m.AddMember("missing", "p"); err == nil {
		t.Fatal("add to missing squad succeeded")
	}
	if err := m.RemoveMember("a", "nobody"); err == nil {
		t.Fatal("remove of non-member succeeded")
	}
	// lock must be free: a second manager acquires within the short bound
	other := NewFileManager(dir)
	other.lockTimeout = 200 * time.Millisecond
	if err := other.AddMember("a", "p"); err != nil {
		t.Fatalf("lock leaked after errors: %v", err)
	}
}

func TestStore_LockFileIgnoredByListing(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("a")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, lockName)); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
	got, err := m.ListChecked()
	if err != nil || len(got) != 1 {
		t.Fatalf("ListChecked = %v, %v; want 1 squad, nil", got, err)
	}
}
