// Package profile_e2e exercises `aps profile list` rich row + filter
// flags end-to-end (T-0428). Each test compiles the aps binary into a
// temp dir, creates fixture profiles in an isolated HOME, and asserts
// stdout against the listing helper's expected output.
package profile_e2e

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

// apsBinary is the absolute path to the compiled aps binary used by
// all child-process invocations. Built once in TestMain.
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
	binName := "aps-profile-e2e"
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

// writeProfile writes a profile.yaml directly into the test home,
// bypassing `aps profile create` so tests can populate fields the
// CLI doesn't expose (squads, roles, identity, persona.tone, …).
func writeProfile(t *testing.T, home, id, yamlBody string) {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "aps", "profiles", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	path := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(path, []byte(yamlBody), 0o644); err != nil {
		t.Fatalf("write profile.yaml: %v", err)
	}
}

// writeSecrets writes a secrets.env into the profile directory; used
// to exercise the --has-secrets filter.
func writeSecrets(t *testing.T, home, id, body string) {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "aps", "profiles", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	path := filepath.Join(dir, "secrets.env")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write secrets.env: %v", err)
	}
}
