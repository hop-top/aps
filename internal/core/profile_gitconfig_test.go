package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestCreateProfile_GitConfig pins the gitconfig seeded by CreateProfile:
// the [user] section must carry the profile's own name/email (never a
// placeholder address) and every value must be quoted/escaped per
// git-config syntax so that no input can break out into new keys or
// sections.
func TestCreateProfile_GitConfig(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		displayName string
		email       string
		wantFile    string
		// wantGit is the decoded key/value set git reads back; nil
		// entries mean the key must be absent.
		wantGit map[string]string
	}{
		{
			name:        "email set",
			id:          "t",
			displayName: "T",
			email:       "t@x.io",
			wantFile:    "[user]\n\tname = \"T\"\n\temail = \"t@x.io\"\n",
			wantGit:     map[string]string{"user.name": "T", "user.email": "t@x.io"},
		},
		{
			name:        "email empty omits email line",
			id:          "noemail",
			displayName: "No Email",
			email:       "",
			wantFile:    "[user]\n\tname = \"No Email\"\n",
			wantGit:     map[string]string{"user.name": "No Email"},
		},
		{
			name:        "display name empty falls back to id",
			id:          "fallback",
			displayName: "",
			email:       "f@x.io",
			wantFile:    "[user]\n\tname = \"fallback\"\n\temail = \"f@x.io\"\n",
			wantGit:     map[string]string{"user.name": "fallback", "user.email": "f@x.io"},
		},
		{
			name:        "quote and backslash escaped",
			id:          "quoted",
			displayName: `Jo "JB" B\tar`,
			email:       `q"b\@x.io`,
			wantFile:    "[user]\n\tname = \"Jo \\\"JB\\\" B\\\\tar\"\n\temail = \"q\\\"b\\\\@x.io\"\n",
			wantGit:     map[string]string{"user.name": `Jo "JB" B\tar`, "user.email": `q"b\@x.io`},
		},
		{
			name:        "comment chars and surrounding space preserved",
			id:          "comment",
			displayName: " A ; B # C ",
			email:       "c@x.io",
			wantFile:    "[user]\n\tname = \" A ; B # C \"\n\temail = \"c@x.io\"\n",
			wantGit:     map[string]string{"user.name": " A ; B # C ", "user.email": "c@x.io"},
		},
		{
			name:        "newline and tab cannot inject keys",
			id:          "inject",
			displayName: "Evil\tName\n[core]\n\tsshCommand = touch /tmp/pwned",
			email:       "e@x.io\n\tname = spoofed",
			wantFile: "[user]\n" +
				"\tname = \"Evil\\tName\\n[core]\\n\\tsshCommand = touch /tmp/pwned\"\n" +
				"\temail = \"e@x.io\\n\\tname = spoofed\"\n",
			wantGit: map[string]string{
				"user.name":  "Evil\tName\n[core]\n\tsshCommand = touch /tmp/pwned",
				"user.email": "e@x.io\n\tname = spoofed",
			},
		},
		{
			name:        "NUL dropped",
			id:          "nul",
			displayName: "a\x00b",
			email:       "n@x.io",
			wantFile:    "[user]\n\tname = \"ab\"\n\temail = \"n@x.io\"\n",
			wantGit:     map[string]string{"user.name": "ab", "user.email": "n@x.io"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APS_DATA_PATH", t.TempDir())

			err := CreateProfile(tc.id, Profile{
				DisplayName: tc.displayName,
				Email:       tc.email,
				Git:         GitConfig{Enabled: true},
			})
			if err != nil {
				t.Fatalf("CreateProfile: %v", err)
			}

			dir, err := GetProfileDir(tc.id)
			if err != nil {
				t.Fatalf("GetProfileDir: %v", err)
			}
			path := filepath.Join(dir, "gitconfig")
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read gitconfig: %v", err)
			}
			if string(got) != tc.wantFile {
				t.Errorf("gitconfig mismatch\n got: %q\nwant: %q", got, tc.wantFile)
			}
			if strings.Contains(string(got), "agent@example.com") {
				t.Errorf("gitconfig contains placeholder email: %q", got)
			}

			assertGitReadsBack(t, path, tc.wantGit)
		})
	}
}

// TestCreateProfile_GitDisabledWritesNoGitconfig guards the opt-in: no
// gitconfig file unless Git.Enabled.
func TestCreateProfile_GitDisabledWritesNoGitconfig(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	if err := CreateProfile("nogit", Profile{DisplayName: "N", Email: "n@x.io"}); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	dir, err := GetProfileDir("nogit")
	if err != nil {
		t.Fatalf("GetProfileDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "gitconfig")); !os.IsNotExist(err) {
		t.Fatalf("expected no gitconfig, stat err = %v", err)
	}
}

// assertGitReadsBack parses the file with the real git binary (skipped
// when git is unavailable) and asserts the exact set of keys and decoded
// values, proving the escaping round-trips and nothing leaks into
// additional keys or sections.
func assertGitReadsBack(t *testing.T, path string, want map[string]string) {
	t.Helper()
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Log("git not on PATH; skipping round-trip check")
		return
	}

	cmd := exec.Command(gitBin, "config", "--file", path, "--list", "-z")
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git config --list: %v", err)
	}

	got := map[string]string{}
	for _, entry := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if entry == "" {
			continue
		}
		k, v, _ := strings.Cut(entry, "\n")
		if _, dup := got[k]; dup {
			t.Errorf("git read duplicate key %q", k)
		}
		got[k] = v
	}

	if len(got) != len(want) {
		t.Errorf("git keys = %v, want %v", sortedKeys(got), sortedKeys(want))
	}
	for k, v := range want {
		if g, ok := got[k]; !ok {
			t.Errorf("git missing key %q", k)
		} else if g != v {
			t.Errorf("git %s = %q, want %q", k, g, v)
		}
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
