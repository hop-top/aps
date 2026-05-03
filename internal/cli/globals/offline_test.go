package globals_test

// T-0411 — table-driven enumeration of network-touching commands that
// honor --offline via the globals.IsOffline() accessor.
//
// Strategy: static source-grep. For each (subpackage, file) listed
// below, the test asserts the file contains both an import of
// "hop.top/aps/internal/cli/globals" and a call to globals.IsOffline().
// This catches future regressions where someone adds a new RunE to one
// of the listed files without wiring the gate, or refactors away the
// accessor without replacing it.
//
// Adding a new gated command? Append it to the table; CI will then
// require the gate to land in the same change.

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"hop.top/aps/internal/cli/globals"
)

func TestCommandsHonorOffline(t *testing.T) {
	repoRoot := repoRootFromTestFile(t)

	cases := []struct {
		name string
		path string // relative to repo root
	}{
		// directory — AGNTCY Directory RPCs.
		{"directory.register", "internal/cli/directory/register.go"},
		{"directory.discover", "internal/cli/directory/discover.go"},
		{"directory.show", "internal/cli/directory/show.go"},
		{"directory.delete", "internal/cli/directory/delete.go"},

		// a2a — agent-to-agent client calls.
		{"a2a.send", "internal/cli/a2a/send_task.go"},
		{"a2a.cancel", "internal/cli/a2a/cancel_task.go"},
		{"a2a.subscribe", "internal/cli/a2a/subscribe_task.go"},
		{"a2a.fetch_card", "internal/cli/a2a/fetch_card.go"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			abs := filepath.Join(repoRoot, tc.path)
			data, err := os.ReadFile(abs)
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			src := string(data)
			if !strings.Contains(src, `"hop.top/aps/internal/cli/globals"`) {
				t.Errorf("%s missing globals import; --offline gate not wired", tc.path)
			}
			if !strings.Contains(src, "globals.IsOffline()") {
				t.Errorf("%s missing globals.IsOffline() check; add a short-circuit at the top of RunE", tc.path)
			}
		})
	}
}

// TestIsOffline_NilSafe asserts the accessor returns false before
// SetViper has wired the underlying viper. This protects unit tests
// in subpackages that exercise commands without booting the root CLI.
func TestIsOffline_NilSafe(t *testing.T) {
	// Reset to nil-state. Subsequent tests in this package may re-call
	// SetViper if they need a wired instance.
	globals.SetViper(nil)
	if globals.IsOffline() {
		t.Fatal("IsOffline() = true with nil viper; want false")
	}
}

// TestIsOffline_ReadsViper covers the wired path: SetViper writes the
// underlying viper, IsOffline reflects the "offline" key.
func TestIsOffline_ReadsViper(t *testing.T) {
	v := viper.New()
	globals.SetViper(v)
	t.Cleanup(func() { globals.SetViper(nil) })

	if globals.IsOffline() {
		t.Fatal("IsOffline() = true with unset offline; want false")
	}
	v.Set("offline", true)
	if !globals.IsOffline() {
		t.Fatal("IsOffline() = false after viper.Set(offline, true); want true")
	}
}

// TestErrOffline_Wrapped asserts ErrOffline survives fmt.Errorf %w
// wrapping so callers can match via errors.Is.
func TestErrOffline_Wrapped(t *testing.T) {
	wrapped := errors.New("not offline")
	if errors.Is(wrapped, globals.ErrOffline) {
		t.Fatal("unrelated error matched ErrOffline")
	}
	wrappedReal := wrapErr(globals.ErrOffline)
	if !errors.Is(wrappedReal, globals.ErrOffline) {
		t.Fatalf("wrapped %v did not match ErrOffline", wrappedReal)
	}
}

// wrapErr exists so the test exercises the same %w pattern callers use.
func wrapErr(err error) error {
	return &wrappedErr{msg: "directory register", err: err}
}

type wrappedErr struct {
	msg string
	err error
}

func (w *wrappedErr) Error() string { return w.msg + ": " + w.err.Error() }
func (w *wrappedErr) Unwrap() error { return w.err }

// repoRootFromTestFile walks up from the test file to find the module
// root (directory containing go.mod). Avoids hard-coding paths.
func repoRootFromTestFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from test file")
		}
		dir = parent
	}
}
