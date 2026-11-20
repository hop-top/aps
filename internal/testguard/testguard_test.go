package testguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeM struct{ fn func() }

func (f fakeM) Run() int { f.fn(); return 0 }

func TestRunSandboxesEnv(t *testing.T) {
	var home, cfg string
	code := Run(fakeM{func() {
		home = os.Getenv("HOME")
		cfg = os.Getenv("XDG_CONFIG_HOME")
	}})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(home, "aps-testguard-") || filepath.Dir(cfg) != home {
		t.Fatalf("env not sandboxed: HOME=%q XDG_CONFIG_HOME=%q", home, cfg)
	}
}

func TestRunFailsWhenTestLeaksEnv(t *testing.T) {
	prev, had := os.LookupEnv("XDG_CONFIG_HOME")
	t.Cleanup(func() {
		if had {
			os.Setenv("XDG_CONFIG_HOME", prev)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})
	code := Run(fakeM{func() {
		// Simulates a test restoring a "real" dir with os.Setenv.
		os.Setenv("XDG_CONFIG_HOME", "/real/home/.config")
	}})
	if code == 0 {
		t.Fatal("Run returned 0 though a test left XDG_CONFIG_HOME outside the sandbox")
	}
}

func TestRunRemovesSandbox(t *testing.T) {
	var root string
	Run(fakeM{func() { root = os.Getenv("HOME") }})
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("sandbox %q not removed: %v", root, err)
	}
}
