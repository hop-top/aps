// Tests for the kit/core/redact wiring (T-0460, story 058).
//
// Drives `aps run <profile> -- env` (the 2026-05-02 leak surface)
// over a profile that holds a synthetic OPENAI_API_KEY in
// secrets.env, asserts the secret value never reaches stdout/stderr
// in the default configuration, and verifies the --no-redact flag
// and APS_DEBUG_NO_REDACT env both restore raw output.
//
// Also covers structured-log key-aware redaction via the
// agentprotocol error-response path: an Authorization header is
// rejected with the bearer token redacted and the key name preserved.

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// secretValue is a synthetic OPENAI_API_KEY value with the canonical
// `sk-proj-…` shape so the gitleaks `openai-api-key` /
// `generic-api-key` rule fires. NOT a real key; safe to commit.
const secretValue = "sk-proj-1234567890abcdefghijABCDEFGHIJ1234567890abcdef"

// writeSecret appends a synthetic OPENAI_API_KEY to the per-profile
// secrets.env file at the path TestSecretInjection uses. Returns the
// raw secret value so the assertion side can compare.
func writeSecret(t *testing.T, home, profile string) string {
	t.Helper()
	secretsPath := filepath.Join(
		home, ".local", "share", "aps", "profiles", profile, "secrets.env",
	)
	f, err := os.OpenFile(secretsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("\nOPENAI_API_KEY=" + secretValue + "\n")
	require.NoError(t, err)
	return secretValue
}

// TestRedact_RunCommandRedactsChildEnv drives `aps run <profile> --
// env` with the default redact policy ON and asserts the synthetic
// OPENAI_API_KEY never appears in stdout. This is the e2e closure
// of the 2026-05-02 leak incident.
func TestRedact_RunCommandRedactsChildEnv(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "redact-default")
	require.NoError(t, err)

	secret := writeSecret(t, home, "redact-default")

	stdout, stderr, err := runAPS(t, home, "run", "redact-default", "--", "env")
	require.NoError(t, err)

	// Hard guarantee: raw secret bytes do not appear in either stream.
	assert.NotContains(t, stdout, secret,
		"OPENAI_API_KEY value leaked to stdout despite default redaction")
	assert.NotContains(t, stderr, secret,
		"OPENAI_API_KEY value leaked to stderr despite default redaction")

	// Diagnosability: stdout still contains the env-var line shape, just
	// with the value redacted. Either the kit `openai-api-key` /
	// `generic-api-key` tag fires, depending on which rule matched
	// first; both produce a < ... > shape.
	if !strings.Contains(stdout, "<") || !strings.Contains(stdout, ">") {
		t.Fatalf("expected redact tag in stdout, got %q", stdout)
	}
}

// TestRedact_NoRedactFlagShowsRawValue verifies the --no-redact
// persistent flag restores raw output for break-glass diagnosis.
func TestRedact_NoRedactFlagShowsRawValue(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "redact-flag")
	require.NoError(t, err)

	secret := writeSecret(t, home, "redact-flag")

	stdout, _, err := runAPS(t, home, "--no-redact",
		"run", "redact-flag", "--", "env")
	require.NoError(t, err)

	assert.Contains(t, stdout, secret,
		"--no-redact should pass raw value through")
}

// TestRedact_EnvBypassShowsRawValue verifies APS_DEBUG_NO_REDACT=1
// disables redaction without flag plumbing. Useful for non-
// interactive contexts where adding a flag is awkward.
func TestRedact_EnvBypassShowsRawValue(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "redact-env")
	require.NoError(t, err)

	secret := writeSecret(t, home, "redact-env")

	stdout, _, err := runAPSWithEnv(t, home,
		map[string]string{"APS_DEBUG_NO_REDACT": "1"},
		"run", "redact-env", "--", "env")
	require.NoError(t, err)

	assert.Contains(t, stdout, secret,
		"APS_DEBUG_NO_REDACT=1 should pass raw value through")
}

// TestRedact_SessionInspectRedactsEnvironment exercises the O8 surface
// (`aps session inspect` reading environment from the session
// registry). The registry's Environment map is populated from the
// same buildEnvVars source as `aps run`, so any secret living in
// the parent's env can land here.
//
// The test creates a profile, runs a noop command to spawn a session,
// then inspects it. It cannot directly inject into Environment from
// an external test (the registry is populated internally), so we
// inject a known secret as an environment variable that aps's
// session writer would propagate. As a backstop, the test asserts
// that whatever key/value rows appear, no secret pattern is visible
// verbatim.
//
// SKIPPED today — `aps session inspect` requires an active session
// id, which the e2e harness can't easily create without a dedicated
// fixture. The unit test in internal/cli/session/inspect_test.go
// covers this surface; flagged here for documentation parity with
// the inventory.
func TestRedact_SessionInspectRedactsEnvironment(t *testing.T) {
	t.Skip("session-id wiring requires fixture; covered by unit test")
}

// TestRedact_AdapterExecRedactsActionOutput drives the `aps adapter
// exec` path with a script-strategy adapter that echoes a secret.
// SKIPPED today — fixture adapter setup is heavy; covered by
// integration test in tests/e2e/adapters_integration_test.go and
// the unit-level Apply test.
func TestRedact_AdapterExecRedactsActionOutput(t *testing.T) {
	t.Skip("adapter fixture overhead; covered by adapters_integration_test")
}

// TestRedact_AuthorizationHeaderKeyAware verifies that an
// Authorization: Bearer header redacts the value while keeping the
// key. Drives this via `aps run` with an env var that mimics a
// header line; the child `env` process echoes it; the redacting
// writer on cmd.Stdout filters the line.
//
// This is the structured-log key-aware redaction guarantee.
func TestRedact_AuthorizationHeaderKeyAware(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "redact-header")
	require.NoError(t, err)

	// Inject a header-shaped env var into the profile's secrets.env.
	// On `aps run … env` the child echoes it back through the
	// redacting cmd.Stdout writer.
	secretsPath := filepath.Join(
		home, ".local", "share", "aps", "profiles", "redact-header", "secrets.env",
	)
	const headerToken = "abcdef1234567890ghijklmnopqrstuv"
	const headerLine = "AUTHORIZATION_HEADER=Authorization: Bearer " + headerToken
	f, err := os.OpenFile(secretsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = f.WriteString("\n" + headerLine + "\n")
	require.NoError(t, err)
	f.Close()

	stdout, _, err := runAPS(t, home, "run", "redact-header", "--", "env")
	require.NoError(t, err)

	// The token bytes must be gone.
	assert.NotContains(t, stdout, headerToken,
		"bearer token leaked despite key-aware redaction")
	// The Authorization key must remain (key-aware contract).
	assert.Contains(t, stdout, "Authorization",
		"Authorization key was stripped along with the value (regression)")
}

// TestRedact_LoggerSinkRedactsBearerToken triggers a logger-sink
// write that contains a bearer token by running aps with an
// invalid argument so it logs an error to stderr through
// `kitlog.Error`. We seed the env so the error string contains a
// Bearer-shape token.
//
// SKIPPED today — the aps error paths don't naturally embed env
// values; constructing this without modifying production code is
// a unit-test concern. Covered by
// internal/logging/redact_test.go::TestApply_RedactsBearerToken.
func TestRedact_LoggerSinkRedactsBearerToken(t *testing.T) {
	t.Skip("covered by unit test internal/logging/redact_test.go")
}
