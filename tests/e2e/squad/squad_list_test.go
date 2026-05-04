// Package squad_e2e exercises `aps squad list` end-to-end (T-0431).
// Squads are stored in-memory only by the running CLI process, so e2e
// tests can only assert the empty-registry path. Filter/positive
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

var apsBinary string

func TestMain(m *testing.M) {
	if err := compileBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "compile aps binary: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.Remove(apsBinary)
	os.Exit(code)
}

func compileBinary() error {
	binName := "aps-squad-e2e"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	apsBinary = filepath.Join(os.TempDir(), binName)

	rootDir, err := filepath.Abs("../../..")
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
	cmd := exec.Command(apsBinary, args...)

	override := map[string]bool{
		"HOME":          true,
		"USERPROFILE":   true,
		"XDG_DATA_HOME": true,
		"APS_DATA_PATH": true,
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
	}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if override[key] {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestSquadList_Empty: a fresh process has an empty squad registry
// (in-memory; not yet persisted). The list command must exit 0 and
// not panic.
func TestSquadList_Empty(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "squad", "list")
	if err != nil {
		t.Fatalf("squad list on empty registry: %v\nstderr: %s", err, stderr)
	}
}

// TestSquadList_FilterFlagsAccepted: the new filter flags must be
// recognised (no "unknown flag" error) even when the registry is
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
