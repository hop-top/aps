// Package testguard keeps test binaries away from the developer's real
// HOME and XDG directories. A package's TestMain calls Run instead of
// m.Run; tests that touch default config, data, state or cache dirs then
// land in a throwaway sandbox, and a test that leaves the process env
// pointing elsewhere fails the run.
package testguard

import (
	"fmt"
	"os"
	"path/filepath"
)

// runner is satisfied by *testing.M.
type runner interface{ Run() int }

// sandboxed lists env vars redirected into the sandbox.
func sandboxed(root string) map[string]string {
	return map[string]string{
		"HOME":            root,
		"USERPROFILE":     root,
		"AppData":         filepath.Join(root, "AppData", "Roaming"),
		"LocalAppData":    filepath.Join(root, "AppData", "Local"),
		"XDG_CONFIG_HOME": filepath.Join(root, "config"),
		"XDG_DATA_HOME":   filepath.Join(root, "data"),
		"XDG_STATE_HOME":  filepath.Join(root, "state"),
		"XDG_CACHE_HOME":  filepath.Join(root, "cache"),
	}
}

// Run redirects HOME and the XDG/AppData dirs into a temp dir, runs the
// tests, and returns the exit code (non-zero when a test left the
// sandboxed env altered). Use as: os.Exit(testguard.Run(m)).
func Run(m runner) (code int) {
	root, err := os.MkdirTemp("", "aps-testguard-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "testguard: %v\n", err)
		return 1
	}
	defer func() {
		if err := os.RemoveAll(root); err != nil {
			fmt.Fprintf(os.Stderr, "testguard: remove sandbox %s: %v\n", root, err)
			code = 1
		}
	}()
	// Resolve symlinks (macOS /var -> /private/var) so comparisons are stable.
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}

	want := sandboxed(root)
	for k, v := range want {
		if err := os.Setenv(k, v); err != nil {
			fmt.Fprintf(os.Stderr, "testguard: setenv %s: %v\n", k, err)
			return 1
		}
	}
	// Tests must not inherit an explicit data-path override either.
	if err := os.Unsetenv("APS_DATA_PATH"); err != nil {
		fmt.Fprintf(os.Stderr, "testguard: unsetenv APS_DATA_PATH: %v\n", err)
		return 1
	}

	code = m.Run()

	for k, v := range want {
		if got := os.Getenv(k); got != v {
			fmt.Fprintf(os.Stderr,
				"testguard: %s left as %q, want sandbox %q: a test mutated process env without restoring it (use t.Setenv)\n",
				k, got, v)
			code = 1
		}
	}
	return code
}
