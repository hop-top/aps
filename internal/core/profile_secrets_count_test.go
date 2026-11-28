package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func secretsCountFixture(t *testing.T, id, body string, write bool) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "cfg"))
	t.Setenv("APS_DATA_PATH", filepath.Join(home, "data"))
	if err := CreateProfile(id, Profile{ID: id, DisplayName: id}); err != nil {
		t.Fatal(err)
	}
	if !write {
		return
	}
	dir, err := GetProfileDir(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secrets.env"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A new profile's secrets.env holds only a comment template: it has no
// secrets.
func TestProfileSecretCount_TemplateOnlyIsZero(t *testing.T) {
	secretsCountFixture(t, "fresh", "", false)
	n, err := ProfileSecretCount("fresh")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 || ProfileHasSecrets("fresh") {
		t.Errorf("count=%d has=%v, want 0/false", n, ProfileHasSecrets("fresh"))
	}
}

func TestProfileSecretCount_CountsKeys(t *testing.T) {
	secretsCountFixture(t, "keyed", "# c\nA=1\nB=op://V/i/f\n", true)
	n, err := ProfileSecretCount("keyed")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || !ProfileHasSecrets("keyed") {
		t.Errorf("count=%d has=%v, want 2/true", n, ProfileHasSecrets("keyed"))
	}
}

func TestProfileSecretCount_MissingFileIsZero(t *testing.T) {
	secretsCountFixture(t, "nofile", "", false)
	dir, _ := GetProfileDir("nofile")
	if err := os.Remove(filepath.Join(dir, "secrets.env")); err != nil {
		t.Fatal(err)
	}
	n, err := ProfileSecretCount("nofile")
	if err != nil || n != 0 {
		t.Errorf("count=%d err=%v, want 0/nil", n, err)
	}
}

// A malformed file is an error, never a count, and the error carries
// location only.
func TestProfileSecretCount_MalformedIsError(t *testing.T) {
	secretsCountFixture(t, "bad", "GOOD=1\nBAD!KEY=hunter2-sentinel\n", true)
	n, err := ProfileSecretCount("bad")
	if !errors.Is(err, ErrSecretsParse) {
		t.Fatalf("err=%v, want ErrSecretsParse", err)
	}
	if n != 0 || ProfileHasSecrets("bad") {
		t.Errorf("count=%d has=%v, want 0/false on error", n, ProfileHasSecrets("bad"))
	}
	if got := err.Error(); strings.Contains(got, "hunter2-sentinel") {
		t.Errorf("error leaks content: %s", got)
	}
}
