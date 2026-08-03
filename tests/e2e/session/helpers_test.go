// Package session_e2e exercises `aps session list` rich row + filter
// flags end-to-end (T-0429). Tests seed registry.json fixtures and
// run the compiled aps binary against an isolated HOME.
package session_e2e

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
	binName := "aps-session-e2e"
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

// writeRegistry writes a registry.json under the test home so the
// running binary's session.GetRegistry() loads fixture sessions on
// first call. Body is the raw JSON (caller composes the map).
func writeRegistry(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "aps", "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir sessions dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write registry.json: %v", err)
	}
}
