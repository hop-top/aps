// Package messenger_e2e exercises `aps adapter messenger list` rich row
// + filter flags end-to-end (T-0436). Each test compiles the aps binary
// into a temp dir, seeds messenger fixtures via raw manifests, and
// asserts stdout against the listing helper's expected output.
package messenger_e2e

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
	binName := "aps-messenger-e2e"
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

func writeGlobalAdapter(t *testing.T, home, name, manifest string) {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "aps", "devices", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir adapter dir: %v", err)
	}
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest.yaml: %v", err)
	}
}

func writeProfileAdapter(t *testing.T, home, profileID, name, manifest string) {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "devices", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir profile adapter dir: %v", err)
	}
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest.yaml: %v", err)
	}
}
