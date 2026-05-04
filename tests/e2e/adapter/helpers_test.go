// Package adapter_e2e exercises `aps adapter list` rich row + filter
// flags end-to-end (T-0435). Each test compiles the aps binary into a
// temp dir, creates fixture adapters in an isolated HOME, and asserts
// stdout against the listing helper's expected output.
package adapter_e2e

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
	binName := "aps-adapter-e2e"
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

// runAPS executes the compiled aps binary in an isolated HOME and
// returns stdout/stderr/err.
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

// writeGlobalAdapter writes a manifest.yaml directly under the global
// devices dir. Bypasses `aps adapter create` so tests can populate
// adapters with specific config (platform, linked_to) without going
// through interactive prompts.
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

// writeProfileAdapter writes a manifest.yaml under a profile-scoped
// devices dir. The profile id is treated as the "workspace" by
// adapterSummaryRow when scope=profile.
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
