package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- end to end through the env builder, with a real `op` stand-in ---

func skipNoShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell stand-in for op")
	}
}

// installFakeOp puts an `op` script first on PATH. body is the shell body.
func installFakeOp(t *testing.T, body string) {
	t.Helper()
	skipNoShell(t)
	dir := t.TempDir()
	script := "#!/bin/sh\n" + body + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "op"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func newRefProfile(t *testing.T, id, secretsBody string) *Profile {
	t.Helper()
	t.Setenv("APS_DATA_PATH", t.TempDir())
	require.NoError(t, CreateProfile(id, Profile{ID: id, DisplayName: "T"}))
	dir, err := GetProfileDir(id)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.env"), []byte(secretsBody), 0o600))
	p, err := LoadProfile(id)
	require.NoError(t, err)
	return p
}

func TestInjectEnvironmentResolvesOnePasswordReference(t *testing.T) {
	installFakeOp(t, `printf '%s' "tok-from-op"`)
	p := newRefProfile(t, "ref-ok", "BOS_AGENT_TOKEN=op://V/item/token\nPLAIN=kept\n")

	cmd := exec.Command("env")
	require.NoError(t, InjectEnvironment(cmd, p))

	env := strings.Join(cmd.Env, "\n")
	assert.Contains(t, env, "BOS_AGENT_TOKEN=tok-from-op")
	assert.Contains(t, env, "PLAIN=kept")
	assert.NotContains(t, env, "op://V/item/token", "the literal reference must never be injected")
}

func TestInjectEnvironmentUnresolvableReferenceFailsClosed(t *testing.T) {
	installFakeOp(t, `echo "[ERROR] no such item" >&2; exit 1`)
	p := newRefProfile(t, "ref-bad", "BOS_AGENT_TOKEN=op://V/item/token\nPLAIN=plain-value-77\n")

	cmd := exec.Command("env")
	err := InjectEnvironment(cmd, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOS_AGENT_TOKEN")
	assert.NotContains(t, err.Error(), "plain-value-77")
	assert.Nil(t, cmd.Env, "no environment may be handed to a child on failure")
}

func TestInjectEnvironmentMissingOpBinaryFailsClosed(t *testing.T) {
	skipNoShell(t)
	t.Setenv("PATH", t.TempDir())
	p := newRefProfile(t, "ref-noop", "BOS_AGENT_TOKEN=op://V/item/token\n")

	err := InjectEnvironment(exec.Command("env"), p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOS_AGENT_TOKEN")
}

func TestInjectEnvironmentPlainValuesNeverReachVault(t *testing.T) {
	skipNoShell(t)
	bin := t.TempDir()
	marker := filepath.Join(bin, "called")
	installFakeOp(t, "touch "+marker+`; printf '%s' leaked`)
	p := newRefProfile(t, "ref-plain", "A=plain\nURL=postgres://localhost/db\n")

	cmd := exec.Command("env")
	require.NoError(t, InjectEnvironment(cmd, p))
	env := strings.Join(cmd.Env, "\n")
	assert.Contains(t, env, "A=plain")
	assert.Contains(t, env, "URL=postgres://localhost/db")
	_, err := os.Stat(marker)
	assert.Error(t, err)
}

func TestInjectEnvironmentAccountQueryPassedToOp(t *testing.T) {
	dir := t.TempDir()
	argFile := filepath.Join(dir, "args")
	installFakeOp(t, `printf '%s\n' "$@" > `+argFile+`; printf '%s' value`)
	p := newRefProfile(t, "ref-acct", "K=op://V/i/f?account=ideacrafters\n")

	cmd := exec.Command("env")
	require.NoError(t, InjectEnvironment(cmd, p))
	assert.Contains(t, strings.Join(cmd.Env, "\n"), "K=value")
	raw, err := os.ReadFile(argFile)
	require.NoError(t, err)
	assert.Equal(t, []string{"read", "--no-newline", "--account", "ideacrafters", "op://V/i/f"},
		strings.Split(strings.TrimSpace(string(raw)), "\n"))
}

func TestInjectEnvironmentSecretNameFromEnv(t *testing.T) {
	t.Setenv("APS_TEST_REF_SRC", "env-value")
	p := newRefProfile(t, "ref-name", "K=secret:APS_TEST_REF_SRC\n")
	cmd := exec.Command("env")
	require.NoError(t, InjectEnvironment(cmd, p))
	assert.Contains(t, strings.Join(cmd.Env, "\n"), "K=env-value")
}

func TestInjectEnvironmentUnknownSecretNameFailsClosed(t *testing.T) {
	p := newRefProfile(t, "ref-miss", "K=secret:APS_TEST_NOPE_XYZ\nOTHER=sibling-val-31\n")
	cmd := exec.Command("env")
	err := InjectEnvironment(cmd, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "K")
	assert.NotContains(t, err.Error(), "sibling-val-31")
	assert.Nil(t, cmd.Env)
}

func TestSecretRefOpenerStoreBeatsEnv(t *testing.T) {
	o := secretRefOpener(
		func(string) (string, bool) { return "from-store", true },
		func(string) (string, bool) { return "from-env", true })
	v, found, err := o.Lookup(context.Background(), "X")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "from-store", v)
}
