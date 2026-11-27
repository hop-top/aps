package squad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestSquad(id string, members ...string) Squad {
	return Squad{ID: id, Name: id, Type: SquadTypePlatform, Domain: "ops", Members: members}
}

// Two managers over one dir simulate two processes.
func TestFileManager_PersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	a := NewFileManager(dir)
	if err := a.Create(newTestSquad("demo", "alice", "bob")); err != nil {
		t.Fatal(err)
	}

	b := NewFileManager(dir)
	got, err := b.Get("demo")
	if err != nil {
		t.Fatalf("second instance Get: %v", err)
	}
	if got.Domain != "ops" || len(got.Members) != 2 || got.CreatedAt.IsZero() {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if n := len(b.List()); n != 1 {
		t.Errorf("List len = %d, want 1", n)
	}
}

func TestFileManager_MutationsPersist(t *testing.T) {
	dir := t.TempDir()
	a := NewFileManager(dir)
	if err := a.Create(newTestSquad("demo", "alice")); err != nil {
		t.Fatal(err)
	}
	if err := a.AddMember("demo", "bob"); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveMember("demo", "alice"); err != nil {
		t.Fatal(err)
	}
	u := newTestSquad("demo", "bob")
	u.Domain = "infra"
	if err := a.Update(u); err != nil {
		t.Fatal(err)
	}

	b := NewFileManager(dir)
	got, err := b.Get("demo")
	if err != nil {
		t.Fatal(err)
	}
	if got.Domain != "infra" || len(got.Members) != 1 || got.Members[0] != "bob" {
		t.Errorf("mutations lost: %+v", got)
	}
	if err := a.Create(newTestSquad("demo")); err == nil {
		t.Error("duplicate create across instances should fail")
	}

	if _, err := os.Stat(filepath.Join(dir, "demo", "manifest.yaml")); err != nil {
		t.Fatalf("manifest should exist before delete: %v", err)
	}
	if err := b.Delete("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get("demo"); err == nil {
		t.Error("delete not visible to other instance")
	}
	if _, err := os.Stat(filepath.Join(dir, "demo")); !os.IsNotExist(err) {
		t.Errorf("squad directory should be gone: %v", err)
	}
}

func TestFileManager_RejectsPathTraversalID(t *testing.T) {
	dir := t.TempDir()
	if err := NewFileManager(dir).Create(newTestSquad("../evil")); err == nil {
		t.Error("expected error for traversal id")
	}
}

func TestFileManager_CorruptFileSurfaced(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("ok", "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "bad"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad", "manifest.yaml"), []byte(": : :\n\t-"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ListChecked(); err == nil {
		t.Error("expected load error")
	}
	// A mutation of an unrelated squad used to be refused here; that
	// pinned abort-on-first-bad-file, not a requirement. Isolation is
	// asserted in TestFileManager_CorruptSquadIsolated.
	if err := m.Create(newTestSquad("other", "a")); err != nil {
		t.Errorf("unrelated create must not be blocked: %v", err)
	}
}

func TestDefaultManager_HonoursDataPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)
	if err := NewDefaultManager().Create(newTestSquad("demo", "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "squads", "demo", "manifest.yaml")); err != nil {
		t.Errorf("not under APS_DATA_PATH: %v", err)
	}
	if _, err := NewDefaultManager().Get("demo"); err != nil {
		t.Errorf("fresh default manager: %v", err)
	}
}

func TestFileManager_IDDirMismatchIsError(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("alpha", "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "alpha"), filepath.Join(dir, "beta")); err != nil {
		t.Fatal(err)
	}
	_, err := m.ListChecked()
	if err == nil || !strings.Contains(err.Error(), "beta") {
		t.Errorf("want load error naming dir beta, got %v", err)
	}
}

func TestFileManager_SkipsDirWithoutManifestAndPlainFiles(t *testing.T) {
	dir := t.TempDir()
	m := NewFileManager(dir)
	if err := m.Create(newTestSquad("ok", "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stray.yaml"), []byte("id: stray\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := m.ListChecked()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "ok" {
		t.Errorf("got %+v", got)
	}
}

func TestFileManager_SaveWritesManifestInSquadDir(t *testing.T) {
	dir := t.TempDir()
	if err := NewFileManager(dir).Create(newTestSquad("demo", "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "demo", "manifest.yaml")); err != nil {
		t.Errorf("manifest missing: %v", err)
	}
	ents, _ := os.ReadDir(filepath.Join(dir, "demo"))
	if len(ents) != 1 {
		t.Errorf("leftover temp files: %v", ents)
	}
}
