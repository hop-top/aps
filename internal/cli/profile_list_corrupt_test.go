package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
)

func runProfileList(t *testing.T, format string) (stdout, stderr string) {
	t.Helper()
	prev := root.Viper.GetString("format")
	root.Viper.Set("format", format)
	t.Cleanup(func() {
		root.Viper.Set("format", prev)
		profileListCmd.SetOut(nil)
		profileListCmd.SetErr(nil)
	})
	var out, errb bytes.Buffer
	profileListCmd.SetOut(&out)
	profileListCmd.SetErr(&errb)
	if err := profileListCmd.RunE(profileListCmd, nil); err != nil {
		t.Fatalf("profile list: %v", err)
	}
	return out.String(), errb.String()
}

func writeProfileYAML(t *testing.T, id, content string) {
	t.Helper()
	dir, err := core.GetProfileDir(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestProfileListNamesCorruptProfileOnStderr(t *testing.T) {
	isolateAPSData(t)
	writeProfileYAML(t, "good-1", "id: good-1\ndisplay_name: Good\n")
	writeProfileYAML(t, "good-2", "id: good-2\ndisplay_name: Good Two\n")

	for _, format := range []string{"table", "json"} {
		cleanOut, cleanErr := runProfileList(t, format)
		if cleanErr != "" {
			t.Fatalf("%s: no corrupt profile, stderr = %q", format, cleanErr)
		}

		writeProfileYAML(t, "bad-1", "id: bad-1\n  : [broken\n")
		out, errOut := runProfileList(t, format)
		if err := os.RemoveAll(mustProfileDir(t, "bad-1")); err != nil {
			t.Fatal(err)
		}

		if out != cleanOut {
			t.Errorf("%s: stdout changed by corrupt profile\nclean: %q\ngot:   %q", format, cleanOut, out)
		}
		if strings.Contains(out, "bad-1") || strings.Contains(out, "warning") {
			t.Errorf("%s: stdout carries corrupt profile or warning: %q", format, out)
		}
		for _, want := range []string{"warning", "bad-1", "parse"} {
			if !strings.Contains(errOut, want) {
				t.Errorf("%s: stderr %q missing %q", format, errOut, want)
			}
		}
		if strings.Contains(errOut, "good-1") {
			t.Errorf("%s: stderr names a valid profile: %q", format, errOut)
		}
		if format == "json" {
			var rows []map[string]any
			if err := json.Unmarshal([]byte(out), &rows); err != nil {
				t.Fatalf("json stdout invalid: %v\n%s", err, out)
			}
			if len(rows) != 2 {
				t.Errorf("json rows = %d, want 2", len(rows))
			}
		}
	}
}

// A directory without profile.yaml is not a profile; it stays out of
// the listing without a warning.
func TestProfileListIgnoresDirWithoutManifest(t *testing.T) {
	isolateAPSData(t)
	writeProfileYAML(t, "good-1", "id: good-1\n")
	if err := os.MkdirAll(mustProfileDir(t, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, errOut := runProfileList(t, "table")
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

func mustProfileDir(t *testing.T, id string) string {
	t.Helper()
	d, err := core.GetProfileDir(id)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
