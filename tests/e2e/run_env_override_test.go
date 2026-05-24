package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunEnv_InlineOverride(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-inline")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "run", "env-inline",
		"--env", "FOO=bar",
		"--env", "BAZ=qux",
		"--", "env")
	require.NoError(t, err)
	assert.Contains(t, stdout, "FOO=bar")
	assert.Contains(t, stdout, "BAZ=qux")
}

func TestRunEnv_FileOverride(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-file")
	require.NoError(t, err)

	envPath := filepath.Join(t.TempDir(), "vars.env")
	require.NoError(t, os.WriteFile(envPath,
		[]byte("ALPHA=one\nBETA=\"two words\"\n# comment\nGAMMA=three\n"), 0o600))

	stdout, _, err := runAPS(t, home, "run", "env-file",
		"--env-file", envPath,
		"--", "env")
	require.NoError(t, err)
	assert.Contains(t, stdout, "ALPHA=one")
	assert.Contains(t, stdout, "BETA=two words")
	assert.Contains(t, stdout, "GAMMA=three")
}

func TestRunEnv_PrecedenceInlineBeatsFile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-prec")
	require.NoError(t, err)

	envPath := filepath.Join(t.TempDir(), "vars.env")
	require.NoError(t, os.WriteFile(envPath, []byte("MODE=file\n"), 0o600))

	stdout, _, err := runAPS(t, home, "run", "env-prec",
		"--env-file", envPath,
		"--env", "MODE=flag",
		"--", "env")
	require.NoError(t, err)
	assert.Contains(t, stdout, "MODE=flag")
	assert.NotContains(t, stdout, "MODE=file")
}

func TestRunEnv_PrecedenceLastInlineWins(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-last")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "run", "env-last",
		"--env", "TARGET=first",
		"--env", "TARGET=second",
		"--", "env")
	require.NoError(t, err)
	assert.Contains(t, stdout, "TARGET=second")
	assert.NotContains(t, stdout, "TARGET=first\n")
}

func TestRunEnv_OverrideBeatsProfileInjected(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-aps")
	require.NoError(t, err)

	// APS_PROFILE_ID is normally injected by buildEnvVars. --env should win.
	stdout, _, err := runAPS(t, home, "run", "env-aps",
		"--env", "APS_PROFILE_ID=clobbered",
		"--", "env")
	require.NoError(t, err)
	assert.Contains(t, stdout, "APS_PROFILE_ID=clobbered")
	assert.NotContains(t, stdout, "APS_PROFILE_ID=env-aps\n")
}

func TestRunEnv_InlineValidationMissingEquals(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-bad-inline")
	require.NoError(t, err)

	_, stderr, err := runAPS(t, home, "run", "env-bad-inline",
		"--env", "FOO",
		"--", "env")
	require.Error(t, err)
	assert.Contains(t, stderr, "expected KEY=VALUE")
}

func TestRunEnv_FileMissingFatal(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-missing-file")
	require.NoError(t, err)

	missing := filepath.Join(t.TempDir(), "nope.env")
	_, stderr, err := runAPS(t, home, "run", "env-missing-file",
		"--env-file", missing,
		"--", "env")
	require.Error(t, err)
	assert.Contains(t, stderr, "nope.env")
}

func TestRunEnv_FileMalformedReportsLine(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-bad-file")
	require.NoError(t, err)

	envPath := filepath.Join(t.TempDir(), "bad.env")
	require.NoError(t, os.WriteFile(envPath,
		[]byte("GOOD=ok\nthis-line-has-no-equals\n"), 0o600))

	_, stderr, err := runAPS(t, home, "run", "env-bad-file",
		"--env-file", envPath,
		"--", "env")
	require.Error(t, err)
	assert.Contains(t, stderr, "line 2")
}

func TestRunEnv_OverrideValueRedactedInChildEcho(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-redact")
	require.NoError(t, err)

	// OpenAI-shape token triggers gitleaks `openai-api-key` /
	// `generic-api-key` rules. The child `env` process echoes it back
	// through the redacting cmd.Stdout writer; the raw bytes must be
	// gone but the env-var line shape must survive.
	const secret = "sk-proj-1234567890abcdefghijABCDEFGHIJ1234567890abcdef"
	stdout, stderr, err := runAPS(t, home, "run", "env-redact",
		"--env", "OPENAI_API_KEY="+secret,
		"--", "env")
	require.NoError(t, err)
	assert.NotContains(t, stdout, secret,
		"override value leaked to stdout despite default redaction")
	assert.NotContains(t, stderr, secret,
		"override value leaked to stderr despite default redaction")
	if !strings.Contains(stdout, "<") || !strings.Contains(stdout, ">") {
		t.Fatalf("expected redact tag in stdout, got %q", stdout)
	}
}

func TestRunEnv_OverrideValuePassesToChildWhenNoRedact(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "env-noredact")
	require.NoError(t, err)

	const secret = "sk-proj-1234567890abcdefghijABCDEFGHIJ1234567890abcdef"
	stdout, _, err := runAPS(t, home, "--no-redact",
		"run", "env-noredact",
		"--env", "OPENAI_API_KEY="+secret,
		"--", "env")
	require.NoError(t, err)
	assert.Contains(t, stdout, "OPENAI_API_KEY="+secret,
		"--no-redact must pass override value through")
}
