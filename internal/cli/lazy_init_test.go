package cli

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/aps/internal/cli/policygate"
	"hop.top/aps/internal/core/capability"
)

// isolatedEnvKeys are scrubbed from the child so the host's bus token,
// policy overrides or XDG dirs cannot leak into a lazy-init assertion.
var isolatedEnvKeys = []string{
	"APS_BUS_TOKEN", "BUS_TOKEN", "APS_BUS_ADDR", "APS_NO_BUS_WARN",
	"APS_POLICY_FILE", "KIT_POLICY_DISABLE", "APS_PROFILE", "CI",
	"XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME",
	"APS_DATA_PATH", "APS_JOBS_DISABLE",
}

// runIsolated runs `aps <args>` in a child process whose HOME and XDG
// dirs all live under home, with the bus token and warning opt-outs
// scrubbed (extra overrides them). It returns stderr and the
// exit error.
func runIsolated(t *testing.T, home string, extra map[string]string, args ...string) (stderrOut string, err error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestExecuteHelperProcess$")
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		skip := false
		for _, k := range isolatedEnvKeys {
			if key == k {
				skip = true
			}
		}
		if !skip {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env,
		execHelperEnv+"="+strings.Join(args, " "),
		"HOME="+home,
		"XDG_DATA_HOME="+filepath.Join(home, "data"),
		"XDG_STATE_HOME="+filepath.Join(home, "state"),
		"XDG_CONFIG_HOME="+filepath.Join(home, "config"),
		"XDG_CACHE_HOME="+filepath.Join(home, "cache"),
		"APS_JOBS_DISABLE=",
	)
	for k, v := range extra {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stderr.String(), err
}

// tree lists every file and dir under root, relative, for failure output.
func tree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(root, func(p string, _ fs.DirEntry, err error) error {
		if err == nil && p != root {
			rel, _ := filepath.Rel(root, p)
			out = append(out, rel)
		}
		return nil
	})
	return out
}

// TestNonUsingCommandsDoNotInitialise pins that commands which use none
// of the capability cache, bus, policy engine or update check run clean:
// no cache db, no seeded policies file, no state, no stderr, with no bus
// token set. Each piece used to initialise at package init or in the
// pre-run hook of every process.
func TestNonUsingCommandsDoNotInitialise(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"profile", "list"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			home := t.TempDir()

			stderr, err := runIsolated(t, home, nil, args...)
			require.NoError(t, err, stderr)

			files := tree(t, home)
			assert.Empty(t, stderr, "no bus warning or other stderr noise")
			assert.NoFileExists(t, filepath.Join(home, "cache", "aps", "cache.db"), "capability cache: %v", files)
			assert.NoFileExists(t, filepath.Join(home, "config", "aps", "policies.yaml"), "policies file: %v", files)
			assert.NoDirExists(t, filepath.Join(home, "state"), "state: %v", files)
			assert.NoDirExists(t, filepath.Join(home, "data", "aps", "collaboration"), "audit storage: %v", files)
		})
	}
}

// TestLazyCapabilityCache_OpensOnFirstUse proves a command that does
// resolve a capability still gets the cache: nothing is opened until
// first use, then the db appears and round-trips.
func TestLazyCapabilityCache_OpensOnFirstUse(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	dbPath := filepath.Join(cacheHome, "aps", "cache.db")

	l := &lazyCapabilityCache{}
	require.NoFileExists(t, dbPath)

	ctx := context.Background()
	_, hit, err := l.Get(ctx, "demo")
	require.NoError(t, err)
	require.False(t, hit)
	require.FileExists(t, dbPath)

	require.NoError(t, l.Put(ctx, capability.Capability{Name: "demo", Description: "d"}))
	got, hit, err := l.Get(ctx, "demo")
	require.NoError(t, err)
	require.True(t, hit)
	require.Equal(t, "demo", got.Name)
}

// TestBusDisabled_WarnsWhereUsed pins the other half of the bus
// contract: a command that publishes an event with no token still gets
// the one-line "bus disabled" warning, once.
func TestBusDisabled_WarnsWhereUsed(t *testing.T) {
	home := t.TempDir()

	stderr, err := runIsolated(t, home, nil, "profile", "create", "lazy1")
	require.NoError(t, err, stderr)

	assert.Equal(t, 1, strings.Count(stderr, "bus disabled"), stderr)
	assert.NoFileExists(t, filepath.Join(home, "config", "aps", "policies.yaml"),
		"no token: no bus, so no policy engine and no seeded file")
}

// TestBusEnabled_BuildsOnFirstUse proves the token path is intact for a
// command that needs it: publishing builds the bus, loads the policy
// engine (seeding its default file) and opens the audit store, with no
// disabled-bus warning.
func TestBusEnabled_BuildsOnFirstUse(t *testing.T) {
	home := t.TempDir()
	env := map[string]string{"BUS_TOKEN": "t", "APS_BUS_ADDR": "ws://127.0.0.1:1/ws/bus"}

	stderr, err := runIsolated(t, home, env, "profile", "create", "lazy2")
	require.NoError(t, err, stderr)

	files := tree(t, home)
	assert.NotContains(t, stderr, "bus disabled")
	assert.FileExists(t, filepath.Join(home, "config", "aps", "policies.yaml"), "%v", files)
	assert.DirExists(t, filepath.Join(home, "data", "aps", "collaboration"), "%v", files)
}

// TestBusEnabled_VersionDoesNotBuild is the token-set counterpart of
// the non-using test: even with a token, `version` neither connects nor
// seeds config.
func TestBusEnabled_VersionDoesNotBuild(t *testing.T) {
	home := t.TempDir()
	env := map[string]string{"BUS_TOKEN": "t", "APS_BUS_ADDR": "ws://127.0.0.1:1/ws/bus"}

	stderr, err := runIsolated(t, home, env, "version")
	require.NoError(t, err, stderr)

	assert.Empty(t, stderr)
	assert.Empty(t, tree(t, home), "token-set version must write nothing")
}

// resetBusForTest clears the lazy bus and policy state so a test can
// build the bus afresh in-process, and tears it down afterwards.
func resetBusForTest(t *testing.T) {
	t.Helper()
	reset := func() {
		closePolicy()
		drainBus()
		policyOnceReset()
		busOnce, busWarnOnce = sync.Once{}, sync.Once{}
		busErr, eventBus, netAdapter, publisher = nil, nil, nil, nil
	}
	reset()
	t.Cleanup(reset)
}

// TestPolicyVeto_StillEnforcedWithBus proves the veto-gated delete path
// builds the bus lazily and still enforces: with a token, deleting a
// session without a note is refused by the bundled policy.
func TestPolicyVeto_StillEnforcedWithBus(t *testing.T) {
	resetBusForTest(t)
	t.Setenv("BUS_TOKEN", "t")
	t.Setenv("APS_BUS_ADDR", "ws://127.0.0.1:1/ws/bus")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("APS_DATA_PATH", t.TempDir())
	t.Setenv("APS_POLICY_FILE", "")
	t.Setenv("KIT_POLICY_DISABLE", "")

	_, err := policygate.PublishDeletePrePersisted(context.Background(), "session", "s1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires --note")
}

// TestPolicyMisconfig_FailsTheGate proves a broken policies file fails
// the veto-gated delete instead of running it unenforced.
func TestPolicyMisconfig_FailsTheGate(t *testing.T) {
	resetBusForTest(t)
	bad := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(bad, []byte("policies: [ {name: x, on: nope"), 0o600))
	t.Setenv("BUS_TOKEN", "t")
	t.Setenv("APS_BUS_ADDR", "ws://127.0.0.1:1/ws/bus")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("APS_DATA_PATH", t.TempDir())
	t.Setenv("APS_POLICY_FILE", bad)
	t.Setenv("KIT_POLICY_DISABLE", "")

	_, err := policygate.PublishDeletePrePersisted(context.Background(), "session", "s1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy")
}

// TestBusState_DoesNotBuildBus pins that `aps status` reports the bus
// from the token alone.
func TestBusState_DoesNotBuildBus(t *testing.T) {
	t.Setenv("APS_BUS_TOKEN", "")
	t.Setenv("BUS_TOKEN", "")
	assert.Equal(t, "disabled", busState())
	t.Setenv("BUS_TOKEN", "t")
	assert.Equal(t, "configured", busState())
	assert.Nil(t, netAdapter, "status must not construct the network adapter")
}

// noticeTree builds root -> top -> leaf with the persistent flags the
// gate reads, so flag cases never mutate the real command tree.
func noticeTree(top, leaf string) *cobra.Command {
	r := &cobra.Command{Use: "aps"}
	r.PersistentFlags().String(formatFlagName, "table", "")
	r.PersistentFlags().Bool("quiet", false, "")
	t := &cobra.Command{Use: top}
	l := &cobra.Command{Use: leaf, Run: func(*cobra.Command, []string) {}}
	t.AddCommand(l)
	r.AddCommand(t)
	return l
}

func TestWantUpgradeNotice(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("APS_NO_UPDATE_NOTIFIER", "")

	require.True(t, wantUpgradeNotice(noticeTree("profile", "list"), true), "interactive management command notifies")

	for _, top := range []string{"version", "upgrade", "serve", "listen", "run", "completion", "help", "docs", "env", "toolspec", "__complete"} {
		assert.False(t, wantUpgradeNotice(noticeTree(top, "x"), true), top)
	}

	assert.False(t, wantUpgradeNotice(noticeTree("profile", "list"), false), "stderr not a terminal")
	assert.False(t, wantUpgradeNotice(&cobra.Command{Use: "aps"}, true), "bare root (TUI / dispatch)")

	for _, v := range []string{"json", "yaml"} {
		l := noticeTree("profile", "list")
		f := l.Flag(formatFlagName)
		require.NoError(t, f.Value.Set(v))
		f.Changed = true
		assert.False(t, wantUpgradeNotice(l, true), "--format "+v)
	}
	q := noticeTree("profile", "list")
	require.NoError(t, q.Root().PersistentFlags().Set("quiet", "true"))
	assert.False(t, wantUpgradeNotice(q, true), "--quiet")

	t.Setenv("CI", "1")
	assert.False(t, wantUpgradeNotice(noticeTree("profile", "list"), true), "CI")
	t.Setenv("CI", "")
	t.Setenv("APS_NO_UPDATE_NOTIFIER", "1")
	assert.False(t, wantUpgradeNotice(noticeTree("profile", "list"), true), "opt-out env")
}

// TestUpgradeNotice_WiredIntoPreRun pins the hook: the real pre-run
// chain prints the notice for an interactive command and not for
// version. The notice used to hang off rootCmd.PersistentPreRun, which
// cobra never runs once kit installs PersistentPreRunE.
func TestUpgradeNotice_WiredIntoPreRun(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("APS_NO_UPDATE_NOTIFIER", "")
	prevTTY, prevN := stderrIsTerminal, upgradeNotifier
	t.Cleanup(func() { stderrIsTerminal, upgradeNotifier = prevTTY, prevN })
	stderrIsTerminal = func() bool { return true }
	calls := 0
	upgradeNotifier = func(context.Context, io.Writer) { calls++ }

	find := func(args ...string) *cobra.Command {
		c, _, err := rootCmd.Find(args)
		require.NoError(t, err)
		return c
	}

	require.NoError(t, applyNoRedactToggle(find("version"), nil))
	assert.Equal(t, 0, calls, "version never notifies")

	require.NoError(t, applyNoRedactToggle(find("profile", "list"), nil))
	assert.Equal(t, 1, calls, "interactive command notifies once")
}

// TestServers_BuildBusAtStartup pins the per-command wiring: the
// long-running servers build the bus in the pre-run hook, bus-free
// commands do not.
func TestServers_BuildBusAtStartup(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("BUS_TOKEN", "t")
	t.Setenv("APS_BUS_ADDR", "ws://127.0.0.1:1/ws/bus")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("APS_DATA_PATH", t.TempDir())
	t.Setenv("APS_POLICY_FILE", "")
	t.Setenv("KIT_POLICY_DISABLE", "")

	find := func(args ...string) *cobra.Command {
		c, _, err := rootCmd.Find(args)
		require.NoError(t, err)
		return c
	}

	resetBusForTest(t)
	require.NoError(t, applyNoRedactToggle(find("version"), nil))
	assert.Nil(t, eventBus, "version must not build the bus")

	for _, args := range [][]string{{"serve"}, {"webhook", "server"}} {
		resetBusForTest(t)
		require.NoError(t, applyNoRedactToggle(find(args...), nil))
		assert.NotNil(t, eventBus, "%v must build the bus at startup", args)
	}
}
