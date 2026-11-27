// Package squad_e2e exercises the `aps squad` commands end-to-end
// against the compiled binary. Squads persist under the data dir, so
// the tests drive separate processes over one isolated HOME. This file
// covers the empty-store list path; cross-process persistence and
// structured output live in squad_persist_test.go. Filter/positive
// matching lives in internal/cli/squad/list_test.go.
package squad_e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	apsBinary string
	// binDir is a per-run temp dir holding the compiled binary, so
	// concurrent runs of this package never share a path.
	binDir string
)

func TestMain(m *testing.M) {
	if err := compileBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "compile aps binary: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(binDir)
	os.Exit(code)
}

func compileBinary() error {
	binName := "aps-squad-e2e"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	var (
		err     error
		rootDir string
	)
	binDir, err = os.MkdirTemp("", "aps-e2e-*")
	if err != nil {
		return err
	}
	apsBinary = filepath.Join(binDir, binName)

	rootDir, err = filepath.Abs("../../..")
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", apsBinary, "./cmd/aps")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runAPS(t *testing.T, home string, args ...string) (string, string, error) {
	t.Helper()
	return runAPSEnv(t, home, nil, args...)
}

// runAPSEnv is runAPS with extra KEY=VALUE entries appended after the
// isolation variables (e.g. APS_DATA_PATH).
func runAPSEnv(t *testing.T, home string, extra []string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)

	override := map[string]bool{
		"HOME":            true,
		"USERPROFILE":     true,
		"XDG_DATA_HOME":   true,
		"XDG_CONFIG_HOME": true,
		"XDG_STATE_HOME":  true,
		"XDG_CACHE_HOME":  true,
		"APS_DATA_PATH":   true,
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"XDG_STATE_HOME=" + filepath.Join(home, ".local", "state"),
		"XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
	}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if override[key] {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = append(env, extra...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestSquadList_Empty: a fresh data dir has no squads. The list
// command must exit 0 and not panic.
func TestSquadList_Empty(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "squad", "list")
	if err != nil {
		t.Fatalf("squad list on empty store: %v\nstderr: %s", err, stderr)
	}
}

// TestSquadList_FilterFlagsAccepted: the new filter flags must be
// recognised (no "unknown flag" error) even when the store is
// empty.
func TestSquadList_FilterFlagsAccepted(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	for _, args := range [][]string{
		{"squad", "list", "--member", "alice"},
		{"squad", "list", "--role", "stream-aligned"},
		{"squad", "list", "--member", "alice", "--role", "platform"},
	} {
		_, stderr, err := runAPS(t, home, args...)
		if err != nil {
			t.Errorf("squad list %v: %v\nstderr: %s", args, err, stderr)
		}
	}
}
