// Package links_e2e exercises `aps adapter link list` rich row + filter
// flags end-to-end (T-0437). Each test compiles the aps binary into a
// temp dir, seeds messenger fixtures + link store JSON, and asserts
// stdout against the listing helper's expected output.
package links_e2e

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
	binName := "aps-links-e2e"
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

// writeProfile writes the minimal profile.yaml core.ListProfiles needs
// to discover a profile during a `link list` scan.
func writeProfile(t *testing.T, home, id string) {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "aps", "profiles", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	body := fmt.Sprintf("id: %s\ndisplay_name: %s\n", id, id)
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write profile.yaml: %v", err)
	}
}

// writeMessengerLinks writes a JSON link manifest under the profile
// directory at the path LinkStore expects (~/.local/share/aps/profiles/
// <id>/messenger-links.json).
func writeMessengerLinks(t *testing.T, home, profileID, json string) {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	path := filepath.Join(dir, "messenger-links.json")
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatalf("write messenger-links.json: %v", err)
	}
}
