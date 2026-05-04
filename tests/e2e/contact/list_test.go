// Package contact holds e2e tests for `aps contact list` filter flags.
//
// Full happy-path requires a cardamum binary + CardDAV server; out of
// scope for the test rig. These tests assert the command surface
// (flags + help) and the no-adapter sad path. Filter/parsing logic is
// covered by internal/cli/contact_test.go (unit).
package contact

import (
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
	binName := "aps-contact-e2e"
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

// TestContactList_HelpHasNewFlags asserts the new filters appear in
// `aps contact list --help`. Cheaper + more deterministic than
// driving a full cardamum/CardDAV stack from a unit test.
func TestContactList_HelpHasNewFlags(t *testing.T) {
	cmd := exec.Command(apsBinary, "contact", "list", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help: %v\n%s", err, out)
	}
	body := string(out)
	for _, want := range []string{
		"--addressbook",
		"--org",
		"--has-email",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %s in --help; got:\n%s", want, body)
		}
	}
}

// TestContactList_NoAdapter exits non-zero with a helpful message when
// the contacts adapter has not been installed in the active HOME.
func TestContactList_NoAdapter(t *testing.T) {
	home := t.TempDir()
	cmd := exec.Command(apsBinary, "contact", "list")
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit; stdout=%s", out)
	}
	// Message wording is owned by the adapter package; assert any
	// non-empty stderr surface.
	if len(out) == 0 {
		t.Fatalf("expected diagnostic on stderr/stdout; got nothing")
	}
}
