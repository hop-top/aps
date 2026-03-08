package bundle

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTemp writes content to a temp file and returns the path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestValidateCmd_ValidFile(t *testing.T) {
	validYAML := `name: test-valid
description: A valid test bundle
version: "1.0"
`
	path := writeTemp(t, validYAML)

	cmd := newValidateCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{path})

	err := cmd.Execute()
	assert.NoError(t, err)
}

// TestValidateCmd_InvalidFile_EmptyName tests that a bundle with an empty name
// causes a validation failure. Because the validate command calls os.Exit(1) on
// failure, we run the test in a subprocess via VALIDATE_TEST_SUBPROCESS so the
// parent test process is not killed.
func TestValidateCmd_InvalidFile_EmptyName(t *testing.T) {
	if os.Getenv("VALIDATE_TEST_SUBPROCESS") == "1" {
		// Running inside the subprocess: execute runValidate directly.
		invalidYAML := `name: ""
description: Empty name bundle
`
		path := writeTemp(t, invalidYAML)
		// runValidate calls os.Exit(1) on validation failure.
		_ = runValidate(path)
		return
	}

	// Parent: re-run only this test in a subprocess and expect exit code 1.
	exe, err := os.Executable()
	require.NoError(t, err)

	cmd := exec.Command(exe, "-test.run=TestValidateCmd_InvalidFile_EmptyName", "-test.v")
	cmd.Env = append(os.Environ(), "VALIDATE_TEST_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()

	// The subprocess should exit with a non-zero status due to os.Exit(1).
	var exitErr *exec.ExitError
	if assert.ErrorAs(t, err, &exitErr) {
		assert.Equal(t, 1, exitErr.ExitCode())
	}
	_ = out // output is available for debugging if needed
}

// TestValidateCmd_MissingFile tests that a non-existent file path is rejected.
func TestValidateCmd_MissingFile(t *testing.T) {
	cmd := newValidateCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"/nonexistent/path/bundle.yaml"})

	err := cmd.Execute()
	// RunE returns the error; cobra surfaces it.
	assert.Error(t, err)
}
