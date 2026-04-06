package bundle

import (
	"bytes"
	"os"
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
// causes a validation failure.
func TestValidateCmd_InvalidFile_EmptyName(t *testing.T) {
	invalidYAML := `name: ""
description: Empty name bundle
`
	path := writeTemp(t, invalidYAML)
	err := runValidate(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
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
