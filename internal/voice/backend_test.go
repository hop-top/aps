package voice_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/aps/internal/voice"
)

// pidPath is a per-test PID-file path under t.TempDir() so tests are
// isolated from each other and from the developer's real
// $XDG_RUNTIME_DIR/aps/voice.pid.
func pidPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "voice.pid")
}

func TestBackendManager_ResolveType_Auto_DefaultsToCompatible(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "auto",
		Backends:       map[string]voice.BackendBinConfig{},
	}
	m := voice.NewBackendManagerWithPIDPath(cfg, pidPath(t))
	resolved := m.ResolveType()
	assert.Equal(t, "compatible", resolved)
}

func TestBackendManager_ResolveType_Explicit(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "moshi-mlx",
		Backends: map[string]voice.BackendBinConfig{
			"moshi-mlx": {Bin: "/usr/local/bin/moshi-mlx", Args: []string{"--port", "8998"}},
		},
	}
	m := voice.NewBackendManagerWithPIDPath(cfg, pidPath(t))
	assert.Equal(t, "moshi-mlx", m.ResolveType())
}

func TestBackendManager_URL_ExplicitOverride(t *testing.T) {
	cfg := voice.GlobalBackendConfig{}
	m := voice.NewBackendManagerWithPIDPath(cfg, pidPath(t))
	url := m.ResolveURL(&voice.BackendConfig{URL: "ws://remote:8998"})
	assert.Equal(t, "ws://remote:8998", url)
}

func TestBackendManager_URL_ManagedDefault(t *testing.T) {
	cfg := voice.GlobalBackendConfig{}
	m := voice.NewBackendManagerWithPIDPath(cfg, pidPath(t))
	url := m.ResolveURL(&voice.BackendConfig{})
	assert.Equal(t, "ws://localhost:8998", url)
}

func TestBackendManager_IsRunning_InitiallyFalse(t *testing.T) {
	m := voice.NewBackendManagerWithPIDPath(voice.GlobalBackendConfig{}, pidPath(t))
	assert.False(t, m.IsRunning())
}

func TestBackendManager_Stop_WhenNotRunning(t *testing.T) {
	m := voice.NewBackendManagerWithPIDPath(voice.GlobalBackendConfig{}, pidPath(t))
	err := m.Stop()
	assert.NoError(t, err)
}

func TestBackendManager_Start_ExternalURL_NoOp(t *testing.T) {
	m := voice.NewBackendManagerWithPIDPath(voice.GlobalBackendConfig{}, pidPath(t))
	err := m.Start(&voice.BackendConfig{URL: "ws://remote:8998"})
	assert.NoError(t, err)
	assert.False(t, m.IsRunning()) // no local process started
}

func TestBackendManager_Start_Compatible_ReturnsError(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "auto",
		Backends:       map[string]voice.BackendBinConfig{},
	}
	m := voice.NewBackendManagerWithPIDPath(cfg, pidPath(t))
	err := m.Start(nil)
	assert.ErrorContains(t, err, "no voice backend binary configured")
}

func TestBackendManager_Start_UnknownType_ReturnsError(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "moshi-mlx",
		Backends:       map[string]voice.BackendBinConfig{},
	}
	m := voice.NewBackendManagerWithPIDPath(cfg, pidPath(t))
	err := m.Start(nil)
	assert.ErrorContains(t, err, `no binary configured for backend type "moshi-mlx"`)
}

func TestBackendManager_Start_LaunchesProcess(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not found, skipping")
	}
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "echo-backend",
		Backends: map[string]voice.BackendBinConfig{
			"echo-backend": {Bin: sleepBin, Args: []string{"30"}},
		},
	}
	pid := pidPath(t)
	m := voice.NewBackendManagerWithPIDPath(cfg, pid)
	err = m.Start(nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Stop() })

	// PID file written.
	data, err := os.ReadFile(pid)
	require.NoError(t, err)
	parsed, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(t, err)
	require.Greater(t, parsed, 0)

	assert.True(t, m.IsRunning())

	// Status() exposes structured state.
	st := m.Status()
	assert.True(t, st.Running)
	assert.Equal(t, parsed, st.PID)
	assert.Equal(t, pid, st.PIDFile)

	// Stop terminates the process and removes the pid file.
	require.NoError(t, m.Stop())
	assert.False(t, m.IsRunning())
	_, statErr := os.Stat(pid)
	assert.True(t, os.IsNotExist(statErr), "pid file should be removed after Stop")
}

func TestBackendManager_Start_RefusesIfAlreadyRunning(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not found, skipping")
	}
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "echo-backend",
		Backends: map[string]voice.BackendBinConfig{
			"echo-backend": {Bin: sleepBin, Args: []string{"30"}},
		},
	}
	pid := pidPath(t)
	m := voice.NewBackendManagerWithPIDPath(cfg, pid)
	require.NoError(t, m.Start(nil))
	t.Cleanup(func() { _ = m.Stop() })

	// Concurrent start should refuse with a clear error.
	err = m.Start(nil)
	assert.ErrorContains(t, err, "already running")
}

func TestBackendManager_Start_SweepsZombiePIDFile(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not found, skipping")
	}
	pid := pidPath(t)

	// Pre-seed a stale pid file with a PID that is guaranteed dead.
	// Use a tiny child that exits immediately, then write its (now
	// dead) PID.
	dead := exec.Command(sleepBin, "0")
	require.NoError(t, dead.Start())
	deadPID := dead.Process.Pid
	require.NoError(t, dead.Wait())
	require.NoError(t, os.WriteFile(pid, []byte(strconv.Itoa(deadPID)+"\n"), 0o600))

	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "echo-backend",
		Backends: map[string]voice.BackendBinConfig{
			"echo-backend": {Bin: sleepBin, Args: []string{"30"}},
		},
	}
	m := voice.NewBackendManagerWithPIDPath(cfg, pid)
	require.NoError(t, m.Start(nil))
	t.Cleanup(func() { _ = m.Stop() })

	assert.True(t, m.IsRunning())
}
