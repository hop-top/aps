package e2e

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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

func TestProfileScopedExternalLLMCLIs(t *testing.T) {
	t.Parallel()

	for _, cliName := range []string{"claude", "codex", "gemini", "opencode"} {
		cliName := cliName
		t.Run(cliName, func(t *testing.T) {
			t.Parallel()

			home := t.TempDir()
			binDir := t.TempDir()
			writeLLMStub(t, binDir, cliName, 0)

			_, _, err := runAPS(t, home, "profile", "create", "llm-agent")
			require.NoError(t, err)

			secretsPath := filepath.Join(home, ".local", "share", "aps", "profiles", "llm-agent", "secrets.env")
			f, err := os.OpenFile(secretsPath, os.O_APPEND|os.O_WRONLY, 0600)
			require.NoError(t, err)
			_, err = f.WriteString("\nANTHROPIC_API_KEY=test-anthropic-key\nOPENAI_API_KEY=test-openai-key\n")
			require.NoError(t, err)
			require.NoError(t, f.Close())

			env := map[string]string{
				"PATH": binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			}

			stdout, stderr, err := runAPSWithEnv(t, home, env, "--no-redact", "run", "llm-agent", "--", cliName, "hello", "--model", "stub")
			require.NoError(t, err)
			assert.Contains(t, stdout, "CLI="+cliName)
			assert.Contains(t, stdout, "PROFILE=llm-agent")
			assert.Contains(t, stdout, "ARGS=hello --model stub")
			assert.Contains(t, stdout, "ANTHROPIC=test-anthropic-key")
			assert.Contains(t, stdout, "OPENAI=test-openai-key")
			assert.Contains(t, stderr, "STDERR="+cliName)

			stdout, _, err = runAPSWithEnv(t, home, env, "--no-redact", "llm-agent", cliName, "short")
			require.NoError(t, err)
			assert.Contains(t, stdout, "CLI="+cliName)
			assert.Contains(t, stdout, "PROFILE=llm-agent")
			assert.Contains(t, stdout, "ARGS=short")
		})
	}
}

func TestProfileScopedExternalLLMCLIExitCode(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	binDir := t.TempDir()
	writeLLMStub(t, binDir, "claude", 42)

	_, _, err := runAPS(t, home, "profile", "create", "llm-exit-agent")
	require.NoError(t, err)

	env := map[string]string{
		"PATH": binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
	}

	stdout, stderr, err := runAPSWithEnv(t, home, env, "run", "llm-exit-agent", "--", "claude", "fail")
	require.Error(t, err)
	assert.Contains(t, stdout, "CLI=claude")
	assert.Contains(t, stderr, "STDERR=claude")

	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 42, exitErr.ExitCode())
	assert.Contains(t, stderr, "exit status 42")
}

func writeLLMStub(t *testing.T, dir, name string, exitCode int) {
	t.Helper()

	path := filepath.Join(dir, name)
	body := fmt.Sprintf(`#!/bin/sh
echo "CLI=%s"
echo "PROFILE=${APS_PROFILE_ID}"
echo "ARGS=$*"
echo "ANTHROPIC=${ANTHROPIC_API_KEY}"
echo "OPENAI=${OPENAI_API_KEY}"
echo "STDERR=%s" >&2
exit %d
`, name, name, exitCode)
	require.NoError(t, os.WriteFile(path, []byte(body), 0755))
}
