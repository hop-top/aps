// Cross-process e2e tests for `aps skill list` (T-0440).
//
// Each test:
//  1. seeds a profile with a couple of SKILL.md fixtures under the
//     profile-scoped skills/ dir,
//  2. spawns the aps binary with --profile + filter flags,
//  3. asserts the rendered table contains (or excludes) the expected
//     skill names per filter.
//
// Build tag skill_e2e — excluded from default `go test`. Run with:
//
//	go test -tags skill_e2e -count=1 ./tests/e2e/skill/...
//
//go:build skill_e2e

package skill

import (
	"bytes"
	"encoding/json"
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
	binName := "aps-skill-e2e"
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

func seedProfile(t *testing.T, dataDir, profileID string) {
	t.Helper()
	dir := filepath.Join(dataDir, "profiles", profileID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir profile: %v", err)
	}
	yaml := fmt.Sprintf("id: %s\ndisplay_name: %s\n", profileID, profileID)
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write profile.yaml: %v", err)
	}
}

// seedSkill writes a SKILL.md fixture under <dataDir>/profiles/<profile>/skills/<name>/
// — the profile-scoped path internal/skills/paths.go discovers first.
func seedSkill(t *testing.T, dataDir, profileID, name, description string, scripts ...string) {
	t.Helper()
	dir := filepath.Join(dataDir, "profiles", profileID, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	body := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\nBody.\n", name, description)
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if len(scripts) > 0 {
		scriptsDir := filepath.Join(dir, "scripts")
		if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
			t.Fatalf("mkdir scripts: %v", err)
		}
		for _, s := range scripts {
			if err := os.WriteFile(filepath.Join(scriptsDir, s), []byte("#!/bin/sh\n"), 0o755); err != nil {
				t.Fatalf("write script %s: %v", s, err)
			}
		}
	}
}

func runAPS(t *testing.T, dataDir string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)
	cmd.Env = append(os.Environ(), "APS_DATA_PATH="+dataDir, "HOME="+t.TempDir())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestSkillList_AllSkills — no filter, both seeded skills render.
func TestSkillList_AllSkills(t *testing.T) {
	dataDir := t.TempDir()
	seedProfile(t, dataDir, "noor")
	seedSkill(t, dataDir, "noor", "deploy", "Deploys the app", "run.sh")
	seedSkill(t, dataDir, "noor", "review", "Reviews PRs")

	out, errOut, err := runAPS(t, dataDir, "--profile", "noor", "skill", "list")
	if err != nil {
		t.Fatalf("skill list: %v\nstdout:%s\nstderr:%s", err, out, errOut)
	}
	for _, want := range []string{"deploy", "review"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output:\n%s", want, out)
		}
	}
}

// TestSkillList_SourceFilter — --source Profile keeps only seeded
// skills (which all live under the profile path).
func TestSkillList_SourceFilter(t *testing.T) {
	dataDir := t.TempDir()
	seedProfile(t, dataDir, "noor")
	seedSkill(t, dataDir, "noor", "deploy", "Deploys")
	seedSkill(t, dataDir, "noor", "lint", "Lints")

	out, errOut, err := runAPS(t, dataDir,
		"--profile", "noor", "skill", "list", "--source", "Profile")
	if err != nil {
		t.Fatalf("skill list --source Profile: %v\nstdout:%s\nstderr:%s",
			err, out, errOut)
	}
	for _, want := range []string{"deploy", "lint"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in --source=Profile output:\n%s", want, out)
		}
	}

	// --source NoSuchSource returns the empty set + the install hint.
	out, _, err = runAPS(t, dataDir,
		"--profile", "noor", "skill", "list", "--source", "Nonexistent")
	if err != nil {
		t.Fatalf("skill list --source Nonexistent: %v", err)
	}
	if !strings.Contains(out, "No skills found") {
		t.Errorf("expected empty-state message, got:\n%s", out)
	}
}

// TestSkillList_JSONFormat — --format json emits a JSON array carrying
// the canonical row shape.
func TestSkillList_JSONFormat(t *testing.T) {
	dataDir := t.TempDir()
	seedProfile(t, dataDir, "noor")
	seedSkill(t, dataDir, "noor", "deploy", "Deploys the app", "run.sh", "rollback.sh")

	out, errOut, err := runAPS(t, dataDir,
		"--profile", "noor", "--format", "json", "skill", "list")
	if err != nil {
		t.Fatalf("skill list --format json: %v\nstdout:%s\nstderr:%s",
			err, out, errOut)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\nout:%s", err, out)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1 (%v)", len(rows), rows)
	}
	if got := rows[0]["name"]; got != "deploy" {
		t.Errorf("name=%v, want deploy", got)
	}
	if got, _ := rows[0]["scripts"].(float64); int(got) != 2 {
		t.Errorf("scripts=%v, want 2", rows[0]["scripts"])
	}
}
