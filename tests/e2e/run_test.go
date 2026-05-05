package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionInjection(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", "exec-agent")
	require.NoError(t, err)

	// Run env
	stdout, _, err := runAPS(t, home, "run", "exec-agent", "--", "env")
	require.NoError(t, err)

	// Verify standard injections
	assert.Contains(t, stdout, "APS_PROFILE_ID=exec-agent")
	assert.Contains(t, stdout, fmt.Sprintf("APS_PROFILE_DIR=%s", filepath.Join(home, ".local", "share", "aps", "profiles", "exec-agent")))
}

func TestSecretInjection(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", "secret-agent")
	require.NoError(t, err)

	// Modify secrets.env
	secretsPath := filepath.Join(home, ".local", "share", "aps", "profiles", "secret-agent", "secrets.env")
	// Append a secret
	f, err := os.OpenFile(secretsPath, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("\nMY_SUPER_SECRET=TopSecretValue123\n")
	require.NoError(t, err)

	// Run env. Use --no-redact: this test asserts the secret IS
	// injected into the child env — exactly the surface T-0460
	// redacts by default. The new tests in redact_test.go cover
	// the redacting-by-default path; here we verify injection
	// itself with redaction explicitly bypassed.
	stdout, _, err := runAPS(t, home, "--no-redact", "run", "secret-agent", "--", "env")
	require.NoError(t, err)

	// Verify secret
	assert.Contains(t, stdout, "MY_SUPER_SECRET=TopSecretValue123")
}

func TestShorthandExecution(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", "short-agent")
	require.NoError(t, err)

	// Run command using shorthand: aps <profile> <cmd>
	stdout, _, err := runAPS(t, home, "short-agent", "env")
	require.NoError(t, err)

	assert.Contains(t, stdout, "APS_PROFILE_ID=short-agent")
}
