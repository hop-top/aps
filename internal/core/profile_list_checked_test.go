package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestListProfilesCheckedReportsCorrupt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	for id, body := range map[string]string{
		"ok":  "id: ok\n",
		"bad": "id: bad\n  : [broken\n",
	} {
		pd, err := GetProfileDir(id)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(pd, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pd, "profile.yaml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ListProfilesChecked()
	var skipped *SkippedProfilesError
	if !errors.As(err, &skipped) || len(skipped.Errs) != 1 {
		t.Fatalf("err = %v, want one SkippedProfilesError", err)
	}
	if len(got) != 1 || got[0].ID != "ok" {
		t.Fatalf("got %v, want only ok", got)
	}

	// ListProfilesFull stays silent for its other callers.
	full, err := ListProfilesFull()
	if err != nil || len(full) != 1 {
		t.Fatalf("ListProfilesFull = %v, %v; want 1 profile, nil", full, err)
	}
}
