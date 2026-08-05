// Package org_e2e exercises the `aps org` command group end-to-end:
// check (hierarchy consistency findings), show (management chain +
// reports), and snapshot (point-in-time organigram). Each test run
// compiles the aps binary once into a temp dir, seeds fixture
// profile.yaml files in an isolated HOME, and asserts stdout/exit
// codes of child-process invocations.
package org_e2e

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
	binName := "aps-org-e2e"
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
// bypassing `aps profile create` so tests can populate fields the CLI
// doesn't expose (reports_to, type, squads, …).
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

// seedTree seeds the canonical five-profile fixture used across the
// show/snapshot tests:
//
//	ceo (human)
//	└── vp (email channel; reports_to ceo)
//	    ├── eng1 (squad core-team) ── intern
//	    └── eng2 (squad core-team)
func seedTree(t *testing.T, home string) {
	t.Helper()
	writeProfile(t, home, "ceo", "id: ceo\ndisplay_name: Chief\ntype: human\n")
	writeProfile(t, home, "vp", "id: vp\ndisplay_name: VP Eng\nreports_to: ceo\nemail: vp@example.com\n")
	writeProfile(t, home, "eng1", "id: eng1\ndisplay_name: Eng One\nreports_to: vp\nsquads:\n  - core-team\n")
	writeProfile(t, home, "eng2", "id: eng2\ndisplay_name: Eng Two\nreports_to: vp\nsquads:\n  - core-team\n")
	writeProfile(t, home, "intern", "id: intern\ndisplay_name: Intern\nreports_to: eng1\n")
}

// seedCycle seeds two profiles reporting to each other.
func seedCycle(t *testing.T, home string) {
	t.Helper()
	writeProfile(t, home, "loopa", "id: loopa\ndisplay_name: Loop A\nreports_to: loopb\n")
	writeProfile(t, home, "loopb", "id: loopb\ndisplay_name: Loop B\nreports_to: loopa\n")
}
