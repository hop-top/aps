package squad

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"hop.top/aps/internal/core"

	"gopkg.in/yaml.v3"
)

func writeRaw(t *testing.T, dir, id, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, id), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, id, "manifest.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFileManager_UnsafeIDsRefused(t *testing.T) {
	ids := []string{
		"", ".", "..", "../evil", "a/b", `a\b`, "a\x00b", "/abs",
		"con", "CON", "nul", "Aux.txt", "com1", "LPT9", "prn",
		"a:b", "c:", "a*b", "a?b", `a"b`, "a<b", "a>b", "a|b",
		"trail.", "trail ", "ctl\x1fchar", strings.Repeat("x", 300),
	}
	for _, id := range ids {
		t.Run(strings.ReplaceAll(id, "\x00", "NUL"), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "squads")
			err := NewFileManager(dir).Create(newTestSquad(id, "a"))
			if err == nil {
				t.Fatalf("Create(%q) succeeded, want refusal", id)
			}
			if !strings.Contains(err.Error(), "invalid squad ID") && id != "" {
				t.Errorf("err = %v, want invalid squad ID", err)
			}
			// nothing may be created anywhere under or beside the store
			parent := filepath.Dir(dir)
			var found []string
			_ = filepath.Walk(parent, func(p string, _ os.FileInfo, _ error) error {
				if p != parent && p != dir {
					found = append(found, p)
				}
				return nil
			})
			if len(found) != 0 {
				t.Errorf("unsafe id left files behind: %v", found)
			}
		})
	}
}

func TestFileManager_SafeIDsAccepted(t *testing.T) {
	for _, id := range []string{"demo", "demo-squad", "a.b", "Squad_1", "v1.2.3", "console", "aux1", "ünï"} {
		dir := t.TempDir()
		if err := NewFileManager(dir).Create(newTestSquad(id, "a")); err != nil {
			t.Errorf("Create(%q): %v", id, err)
		}
	}
}

func TestFileManager_DeleteUnsafeIDTouchesNothing(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "squads")
	victim := filepath.Join(root, "victim")
	if err := os.MkdirAll(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(victim, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("ok", "a")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../victim", "..", ".", "", "a/../../victim"} {
		if err := m.Delete(id); err == nil {
			t.Errorf("Delete(%q) succeeded", id)
		}
	}
	if _, err := os.Stat(filepath.Join(victim, "keep")); err != nil {
		t.Errorf("victim touched: %v", err)
	}
	if _, err := m.Get("ok"); err != nil {
		t.Errorf("squad store damaged: %v", err)
	}
}

// One broken squad must not hide or block the others: siblings skip
// unreadable records in listings, and a lookup only fails for the id asked.
func TestFileManager_CorruptSquadIsolated(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("ok", "a")); err != nil {
		t.Fatal(err)
	}
	bad := writeRaw(t, dir, "bad", ": : :\n\t-")

	got, err := m.Get("ok")
	if err != nil || got.ID != "ok" {
		t.Fatalf("Get(ok) over a corrupt neighbour: %v", err)
	}
	if err := m.AddMember("ok", "b"); err != nil {
		t.Errorf("AddMember over a corrupt neighbour: %v", err)
	}
	if err := m.Create(newTestSquad("other", "a")); err != nil {
		t.Errorf("Create over a corrupt neighbour: %v", err)
	}

	list, err := m.ListChecked()
	if len(list) != 2 {
		t.Errorf("ListChecked returned %d squads, want the 2 healthy ones", len(list))
	}
	if err == nil || !strings.Contains(err.Error(), bad) {
		t.Errorf("ListChecked err = %v, want it to name %s", err, bad)
	}
	if n := len(m.List()); n != 2 {
		t.Errorf("List len = %d, want 2", n)
	}

	_, err = m.Get("bad")
	if err == nil || !strings.Contains(err.Error(), bad) {
		t.Errorf("Get(bad) err = %v, want it to name %s", err, bad)
	}
}

func TestFileManager_CreateNeverOverwritesCorruptSquad(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	bad := writeRaw(t, dir, "bad", ": : :\n\t-")
	err := m.Create(newTestSquad("bad", "a"))
	if err == nil || !strings.Contains(err.Error(), bad) {
		t.Fatalf("Create(bad) err = %v, want refusal naming %s", err, bad)
	}
	body, _ := os.ReadFile(bad)
	if string(body) != ": : :\n\t-" {
		t.Errorf("corrupt manifest was overwritten: %q", body)
	}
	for _, op := range []func() error{
		func() error { return m.AddMember("bad", "x") },
		func() error { return m.RemoveMember("bad", "x") },
		func() error { return m.Update(newTestSquad("bad")) },
	} {
		if err := op(); err == nil || !strings.Contains(err.Error(), bad) {
			t.Errorf("mutation of corrupt squad err = %v, want it to name %s", err, bad)
		}
	}
	if body, _ := os.ReadFile(bad); string(body) != ": : :\n\t-" {
		t.Errorf("corrupt manifest was modified: %q", body)
	}
}

func TestFileManager_DeleteCanRemoveCorruptSquad(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	writeRaw(t, dir, "bad", ": : :\n\t-")
	if err := m.Delete("bad"); err != nil {
		t.Fatalf("Delete(bad): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "bad")); !os.IsNotExist(err) {
		t.Errorf("corrupt squad dir should be gone: %v", err)
	}
	if _, err := m.ListChecked(); err != nil {
		t.Errorf("store should be healthy again: %v", err)
	}
}

func TestFileManager_ErrorNamesMismatchedManifest(t *testing.T) {
	dir := t.TempDir()
	p := writeRaw(t, dir, "beta", "id: alpha\nname: a\ntype: platform\ndomain: d\nmembers: []\n")
	_, err := NewFileManager(dir).ListChecked()
	if err == nil || !strings.Contains(err.Error(), p) {
		t.Errorf("err = %v, want it to name %s", err, p)
	}
}

// Replacing a manifest must swap in a new file rather than rewrite the old
// one in place, or a crash mid-write leaves a truncated record.
func TestFileManager_SaveReplacesFileAtomically(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("demo", "a")); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "demo", "manifest.yaml")
	before := statByHandle(t, p)
	if err := m.AddMember("demo", "b"); err != nil {
		t.Fatal(err)
	}
	after := statByHandle(t, p)
	if os.SameFile(before, after) {
		t.Error("manifest rewritten in place; want temp file + rename")
	}
}

// statByHandle stats p through an open handle, so the file's identity is
// read now. os.Stat on windows resolves identity lazily from the path, which
// after a rename names the replacement file. The handle is closed before
// returning: an open one would block the rename there.
func statByHandle(t *testing.T, p string) os.FileInfo {
	t.Helper()
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := f.Stat()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		t.Fatal(err)
	}
	return fi
}

// Readers racing a writer must always see a complete record.
func TestFileManager_ReaderNeverSeesPartialManifest(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Go opens files without FILE_SHARE_DELETE there, so a rename over
		// a manifest a reader has open fails by design of the platform.
		t.Skip("rename over an open file is refused on windows")
	}
	dir := t.TempDir()
	m := NewFileManager(dir)
	big := make([]string, 4000)
	for i := range big {
		big[i] = "profile-with-a-reasonably-long-id-" + strings.Repeat("x", i%7)
	}
	s := newTestSquad("demo", big...)
	if err := m.Create(s); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "demo", "manifest.yaml")

	stop := make(chan struct{})
	var wg sync.WaitGroup
	bad := make(chan string, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			var got Squad
			if err := yaml.Unmarshal(data, &got); err != nil || got.ID != "demo" || len(got.Members) < len(big) {
				select {
				case bad <- "partial manifest observed":
				default:
				}
				return
			}
		}
	}()
	for i := 0; i < 150; i++ {
		if err := m.AddMember("demo", "m"+strings.Repeat("y", i%5)+string(rune('a'+i%26))); err != nil {
			t.Fatal(err)
		}
	}
	close(stop)
	wg.Wait()
	select {
	case msg := <-bad:
		t.Fatal(msg)
	default:
	}
}

// Separate managers (processes) writing at once never corrupt the store.
// There is no cross-process lock: two simultaneous read-modify-write
// cycles can drop one of the members (last writer wins), so only integrity
// is asserted, not that every member survived.
func TestFileManager_ConcurrentProcessesNeverCorrupt(t *testing.T) {
	dir := t.TempDir()
	if err := NewFileManager(dir).Create(newTestSquad("demo", "seed")); err != nil {
		t.Fatal(err)
	}
	const n = 16
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = NewFileManager(dir).AddMember("demo", "m"+string(rune('a'+i)))
		}(i)
	}
	wg.Wait()

	got, err := NewFileManager(dir).Get("demo")
	if err != nil {
		t.Fatalf("store corrupted by concurrent writers: %v", err)
	}
	if got.Members[0] != "seed" || len(got.Members) < 2 {
		t.Errorf("members = %v", got.Members)
	}
	ents, _ := os.ReadDir(filepath.Join(dir, "demo"))
	if len(ents) != 1 {
		t.Errorf("leftover files after concurrent writes: %v", ents)
	}
}

// One manager shared by goroutines serializes through its mutex: every
// member survives.
func TestFileManager_SharedManagerKeepsAllMembers(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("demo", "seed")); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := m.AddMember("demo", "m"+string(rune('a'+i))); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	got, err := NewFileManager(dir).Get("demo")
	if err != nil || len(got.Members) != 17 {
		t.Errorf("members = %v, err = %v", got, err)
	}
}

// Permissions follow the profile store, the closest sibling.
func TestFileManager_PermissionsMatchProfileStore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix permission bits")
	}
	data := t.TempDir()
	t.Setenv("APS_DATA_PATH", data)
	if err := core.SaveProfile(&core.Profile{ID: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := NewDefaultManager().Create(newTestSquad("demo", "a")); err != nil {
		t.Fatal(err)
	}
	perm := func(p string) os.FileMode {
		t.Helper()
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		return fi.Mode().Perm()
	}
	if got, want := perm(filepath.Join(data, "squads", "demo", "manifest.yaml")), perm(filepath.Join(data, "profiles", "p", "profile.yaml")); got != want {
		t.Errorf("manifest perm = %o, profile.yaml perm = %o", got, want)
	}
	if got, want := perm(filepath.Join(data, "squads", "demo")), perm(filepath.Join(data, "profiles", "p")); got != want {
		t.Errorf("squad dir perm = %o, profile dir perm = %o", got, want)
	}
}

// The in-memory manager never touches disk and does not apply storage
// naming rules.
func TestInMemoryManager_NoDiskNoStorageRules(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	m := NewManager()
	if err := m.Create(newTestSquad("a/b", "x")); err != nil {
		t.Fatalf("in-memory create: %v", err)
	}
	if _, err := m.Get("a/b"); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete("a/b"); err != nil {
		t.Fatal(err)
	}
	if ents, _ := os.ReadDir(os.Getenv("APS_DATA_PATH")); len(ents) != 0 {
		t.Errorf("in-memory manager wrote to disk: %v", ents)
	}
}

func TestFileManager_DuplicateCreateIsError(t *testing.T) {
	m := NewFileManager(t.TempDir())
	if err := m.Create(newTestSquad("demo", "a")); err != nil {
		t.Fatal(err)
	}
	err := NewFileManager(m.dir).Create(newTestSquad("demo", "b"))
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("err = %v", err)
	}
	got, _ := m.Get("demo")
	if len(got.Members) != 1 || got.Members[0] != "a" {
		t.Errorf("duplicate create altered the record: %+v", got)
	}
}

// Members are stored verbatim: not trimmed, deduplicated or checked
// against the profile store.
func TestFileManager_MembersStoredVerbatim(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("demo", "no-such-profile", "Mixed Case", "")); err != nil {
		t.Fatal(err)
	}
	got, err := NewFileManager(dir).Get("demo")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Members, "|") != "no-such-profile|Mixed Case|" {
		t.Errorf("members = %q", got.Members)
	}
}
