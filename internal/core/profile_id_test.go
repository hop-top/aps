package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateProfileID(t *testing.T) {
	valid := []string{
		"alice",
		"0xjadb",
		"nul-bot",
		"console",
		"com10",
		"lpt0",
		"com",
		"lpt",
		"auxiliary",
		"conx.txt",
		"my.profile",
		"Mixed Case",
		"with space inside",
		"under_score",
		"café",
	}
	for _, id := range valid {
		t.Run("valid/"+id, func(t *testing.T) {
			if err := ValidateProfileID(id); err != nil {
				t.Fatalf("ValidateProfileID(%q) = %v, want nil", id, err)
			}
		})
	}

	invalid := []struct {
		id   string
		rule string // substring expected in the error message
	}{
		// empty / dot entries
		{"", "empty"},
		{".", "reserved path"},
		{"..", "reserved path"},

		// reserved device names: case-insensitive, with or without extension
		{"nul", "reserved device name"},
		{"NUL", "reserved device name"},
		{"Nul", "reserved device name"},
		{"con", "reserved device name"},
		{"prn", "reserved device name"},
		{"aux", "reserved device name"},
		{"con.txt", "reserved device name"},
		{"CON.TXT", "reserved device name"},
		{"aux.json", "reserved device name"},
		{"nul.tar.gz", "reserved device name"},
		{"nul .txt", "reserved device name"},
		{"com1", "reserved device name"},
		{"COM9", "reserved device name"},
		{"Com5.log", "reserved device name"},
		{"lpt1", "reserved device name"},
		{"LPT9", "reserved device name"},
		{"com¹", "reserved device name"},
		{"COM²", "reserved device name"},
		{"lpt³.txt", "reserved device name"},

		// characters invalid in Windows file names
		{"a<b", "invalid character"},
		{"a>b", "invalid character"},
		{"a:b", "invalid character"},
		{`a"b`, "invalid character"},
		{"a/b", "invalid character"},
		{`a\b`, "invalid character"},
		{"a|b", "invalid character"},
		{"a?b", "invalid character"},
		{"a*b", "invalid character"},
		{"../escape", "invalid character"},
		{"a\x00b", "control character"},
		{"a\tb", "control character"},
		{"a\nb", "control character"},
		{"a\x1fb", "control character"},

		// trailing dot / space
		{"alice.", "trailing"},
		{"alice ", "trailing"},
		{"alice. ", "trailing"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.id, func(t *testing.T) {
			err := ValidateProfileID(tc.id)
			if err == nil {
				t.Fatalf("ValidateProfileID(%q) = nil, want error", tc.id)
			}
			if !errors.Is(err, ErrInvalidProfileID) {
				t.Errorf("errors.Is(err, ErrInvalidProfileID) = false; err = %v", err)
			}
			if !strings.Contains(err.Error(), tc.rule) {
				t.Errorf("error %q should name rule %q", err, tc.rule)
			}
			if tc.id != "" && !strings.Contains(err.Error(), quoteID(tc.id)) {
				t.Errorf("error %q should name offending id %s", err, quoteID(tc.id))
			}
		})
	}
}

func quoteID(id string) string { return fmt.Sprintf("%q", id) }

// assertNoProfilesWritten fails when anything was created under the data
// dir's profiles/ tree.
func assertNoProfilesWritten(t *testing.T, dataDir string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dataDir, "profiles"))
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read profiles dir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected no profile dirs, found %v", names)
	}
}

func TestCreateProfile_RejectsInvalidIDBeforeTouchingFS(t *testing.T) {
	for _, id := range []string{"nul", "Con.txt", "a:b", "x.", ".."} {
		t.Run(id, func(t *testing.T) {
			dataDir := t.TempDir()
			t.Setenv("APS_DATA_PATH", dataDir)

			err := CreateProfileWithContext(context.Background(), id, Profile{DisplayName: "x"})
			if !errors.Is(err, ErrInvalidProfileID) {
				t.Fatalf("CreateProfile(%q) err = %v, want ErrInvalidProfileID", id, err)
			}
			assertNoProfilesWritten(t, dataDir)
		})
	}
}

func TestImportProfileBundle_RejectsInvalidIDBeforeTouchingFS(t *testing.T) {
	cases := []struct {
		name     string
		sourceID string // id carried in the bundle
		newID    string // --id override
		force    bool
	}{
		{"bundle id reserved", "nul", "", false},
		{"override reserved", "alice", "AUX.json", false},
		{"override invalid char", "alice", "a|b", false},
		{"dotdot with force", "alice", "..", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := t.TempDir()
			t.Setenv("APS_DATA_PATH", dataDir)

			// Sentinel proves a --force import of ".." cannot wipe the
			// data dir (profiles/.. resolves to the data dir itself).
			sentinel := filepath.Join(dataDir, "sentinel")
			if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
				t.Fatal(err)
			}

			bundlePath := filepath.Join(t.TempDir(), "b.aps-profile.yaml")
			bundle := "version: v1\nsource_id: " + tc.sourceID + "\nprofile:\n  id: " + tc.sourceID + "\n  display_name: X\n"
			if err := os.WriteFile(bundlePath, []byte(bundle), 0o600); err != nil {
				t.Fatal(err)
			}

			_, _, err := ImportProfileBundle(bundlePath, tc.newID, tc.force)
			if !errors.Is(err, ErrInvalidProfileID) {
				t.Fatalf("ImportProfileBundle err = %v, want ErrInvalidProfileID", err)
			}
			if _, err := os.Stat(sentinel); err != nil {
				t.Fatalf("data dir sentinel removed: %v", err)
			}
			assertNoProfilesWritten(t, dataDir)
		})
	}
}

// Existing profiles whose id predates validation must stay loadable
// and listable; validation applies on creation only.
func TestLegacyInvalidProfileID_StillLoadsAndLists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ids rejected by ValidateProfileID cannot exist as directories on Windows")
	}
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)

	id := "nul"
	dir := filepath.Join(dataDir, "profiles", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte("id: nul\ndisplay_name: Legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadProfile(id)
	if err != nil {
		t.Fatalf("LoadProfile(%q): %v", id, err)
	}
	if p.ID != id {
		t.Errorf("loaded id = %q, want %q", p.ID, id)
	}

	ids, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	found := false
	for _, got := range ids {
		if got == id {
			found = true
		}
	}
	if !found {
		t.Errorf("ListProfiles() = %v, want it to include %q", ids, id)
	}
}
