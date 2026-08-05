package profile_e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// destructiveToken mirrors kit's destructiveTokenSha (kit
// go/console/cli/policy_runE.go): the first 12 hex chars of
// sha256(cmd.CommandPath()). Non-TTY invocations of destructive
// leaves must pass it via --confirm-token=<sha>.
func destructiveToken(commandPath string) string {
	h := sha256.Sum256([]byte(commandPath))
	return hex.EncodeToString(h[:6])
}

// readProfileYAML returns the raw profile.yaml for id in home, failing
// the test when the file cannot be read.
func readProfileYAML(t *testing.T, home, id string) string {
	t.Helper()
	path := filepath.Join(home, ".local", "share", "aps", "profiles", id, "profile.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read profile.yaml for %s: %v", id, err)
	}
	return string(data)
}

// profileYAMLExists reports whether a profile.yaml is present for id.
func profileYAMLExists(t *testing.T, home, id string) bool {
	t.Helper()
	path := filepath.Join(home, ".local", "share", "aps", "profiles", id, "profile.yaml")
	_, err := os.Stat(path)
	return err == nil
}

func TestProfileCreate_ReportsTo(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeProfile(t, home, "boss", "id: boss\ndisplay_name: Boss\n")

	_, stderr, err := runAPS(t, home, "profile", "create", "worker", "--reports-to", "boss")
	if err != nil {
		t.Fatalf("profile create --reports-to: %v\nstderr: %s", err, stderr)
	}
	if got := readProfileYAML(t, home, "worker"); !strings.Contains(got, "reports_to: boss") {
		t.Errorf("expected reports_to: boss in profile.yaml, got:\n%s", got)
	}
}

func TestProfileEdit_SetAndClearReportsTo(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeProfile(t, home, "boss", "id: boss\ndisplay_name: Boss\n")
	writeProfile(t, home, "worker", "id: worker\ndisplay_name: Worker\n")

	stdout, stderr, err := runAPS(t, home, "profile", "edit", "worker", "--reports-to", "boss")
	if err != nil {
		t.Fatalf("profile edit --reports-to: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "reports_to") {
		t.Errorf("expected reports_to in edit confirmation, got: %s", stdout)
	}
	if got := readProfileYAML(t, home, "worker"); !strings.Contains(got, "reports_to: boss") {
		t.Errorf("expected reports_to: boss in profile.yaml, got:\n%s", got)
	}

	// Empty string clears, per the edit clearing convention.
	_, stderr, err = runAPS(t, home, "profile", "edit", "worker", "--reports-to", "")
	if err != nil {
		t.Fatalf("profile edit --reports-to '': %v\nstderr: %s", err, stderr)
	}
	if got := readProfileYAML(t, home, "worker"); strings.Contains(got, "reports_to") {
		t.Errorf("expected reports_to cleared from profile.yaml, got:\n%s", got)
	}
}

func TestProfileCreate_ReportsToSelfRejected(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "profile", "create", "solo", "--reports-to", "solo")
	if err == nil {
		t.Fatal("expected self-referencing --reports-to to fail")
	}
	if !strings.Contains(stderr, "cannot report to itself") {
		t.Errorf("expected self-reference error, got: %s", stderr)
	}
	if profileYAMLExists(t, home, "solo") {
		t.Error("rejected create must not leave a profile behind")
	}
}

func TestProfileEdit_ReportsToCycleRejected(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeProfile(t, home, "aa", "id: aa\ndisplay_name: A\n")
	writeProfile(t, home, "bb", "id: bb\ndisplay_name: B\nreports_to: aa\n")

	_, stderr, err := runAPS(t, home, "profile", "edit", "aa", "--reports-to", "bb")
	if err == nil {
		t.Fatal("expected cycle-introducing --reports-to to fail")
	}
	if !strings.Contains(stderr, "reporting cycle") {
		t.Errorf("expected reporting cycle error, got: %s", stderr)
	}
	if got := readProfileYAML(t, home, "aa"); strings.Contains(got, "reports_to") {
		t.Errorf("rejected edit must not persist reports_to, got:\n%s", got)
	}
}

func TestProfileCreate_ReportsToDanglingRejected(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "profile", "create", "orphan", "--reports-to", "ghost")
	if err == nil {
		t.Fatal("expected dangling --reports-to to fail")
	}
	if !strings.Contains(stderr, "does not match an existing profile") {
		t.Errorf("expected dangling-target error, got: %s", stderr)
	}
	if profileYAMLExists(t, home, "orphan") {
		t.Error("rejected create must not leave a profile behind")
	}
}

func TestProfileDelete_InboundReportsBlockedWithoutForce(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeProfile(t, home, "boss", "id: boss\ndisplay_name: Boss\n")
	writeProfile(t, home, "worker", "id: worker\ndisplay_name: Worker\nreports_to: boss\n")

	token := destructiveToken("aps profile delete")
	_, stderr, err := runAPS(t, home, "profile", "delete", "boss", "--yes", "--confirm-token", token)
	if err == nil {
		t.Fatal("expected delete of a reported-to profile to be blocked")
	}
	if !strings.Contains(stderr, "worker") {
		t.Errorf("expected inbound reporter listed in error, got: %s", stderr)
	}
	if !profileYAMLExists(t, home, "boss") {
		t.Error("blocked delete must not remove the profile")
	}

	// --force proceeds, warning about the dangling reporters.
	_, stderr, err = runAPS(t, home, "profile", "delete", "boss", "--yes", "--force", "--confirm-token", token)
	if err != nil {
		t.Fatalf("profile delete --force: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "worker") {
		t.Errorf("expected dangling-reporter warning on stderr, got: %s", stderr)
	}
	if profileYAMLExists(t, home, "boss") {
		t.Error("forced delete must remove the profile")
	}
}

func TestProfileCreate_TypeHumanAccepted(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "profile", "create", "jad", "--type", "human")
	if err != nil {
		t.Fatalf("profile create --type human: %v\nstderr: %s", err, stderr)
	}
	if got := readProfileYAML(t, home, "jad"); !strings.Contains(got, "type: human") {
		t.Errorf("expected type: human in profile.yaml, got:\n%s", got)
	}
}

func TestProfileCreate_TypeInvalidRejected(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "profile", "create", "fruit", "--type", "banana")
	if err == nil {
		t.Fatal("expected invalid --type to fail")
	}
	if !strings.Contains(stderr, "invalid profile type") {
		t.Errorf("expected invalid-type error, got: %s", stderr)
	}
	if profileYAMLExists(t, home, "fruit") {
		t.Error("rejected create must not leave a profile behind")
	}
}

func TestProfileEdit_TypeSetAndInvalidRejected(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeProfile(t, home, "shape", "id: shape\ndisplay_name: Shape\n")

	_, stderr, err := runAPS(t, home, "profile", "edit", "shape", "--type", "human")
	if err != nil {
		t.Fatalf("profile edit --type human: %v\nstderr: %s", err, stderr)
	}
	if got := readProfileYAML(t, home, "shape"); !strings.Contains(got, "type: human") {
		t.Errorf("expected type: human in profile.yaml, got:\n%s", got)
	}

	_, stderr, err = runAPS(t, home, "profile", "edit", "shape", "--type", "banana")
	if err == nil {
		t.Fatal("expected invalid --type edit to fail")
	}
	if !strings.Contains(stderr, "invalid profile type") {
		t.Errorf("expected invalid-type error, got: %s", stderr)
	}
	if got := readProfileYAML(t, home, "shape"); !strings.Contains(got, "type: human") {
		t.Errorf("rejected edit must not change persisted type, got:\n%s", got)
	}
}
