package voice_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestBackendManager_ResolveType_Auto_DefaultsToCompatible(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "auto",
		Backends:       map[string]voice.BackendBinConfig{},
	}
	m := voice.NewBackendManager(cfg)
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
	m := voice.NewBackendManager(cfg)
	assert.Equal(t, "moshi-mlx", m.ResolveType())
}

func TestBackendManager_URL_ExplicitOverride(t *testing.T) {
	cfg := voice.GlobalBackendConfig{}
	m := voice.NewBackendManager(cfg)
	url := m.ResolveURL(&voice.BackendConfig{URL: "ws://remote:8998"})
	assert.Equal(t, "ws://remote:8998", url)
}

func TestBackendManager_URL_ManagedDefault(t *testing.T) {
	cfg := voice.GlobalBackendConfig{}
	m := voice.NewBackendManager(cfg)
	url := m.ResolveURL(&voice.BackendConfig{})
	assert.Equal(t, "ws://localhost:8998", url)
}

func TestBackendManager_IsRunning_InitiallyFalse(t *testing.T) {
	m := voice.NewBackendManager(voice.GlobalBackendConfig{})
	assert.False(t, m.IsRunning())
}

func TestBackendManager_Stop_WhenNotRunning(t *testing.T) {
	m := voice.NewBackendManager(voice.GlobalBackendConfig{})
	err := m.Stop()
	assert.NoError(t, err)
}

func TestBackendManager_Start_ExternalURL_NoOp(t *testing.T) {
	m := voice.NewBackendManager(voice.GlobalBackendConfig{})
	err := m.Start(&voice.BackendConfig{URL: "ws://remote:8998"})
	assert.NoError(t, err)
	assert.False(t, m.IsRunning()) // no local process started
}

func TestBackendManager_Start_Compatible_ReturnsError(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "auto",
		Backends:       map[string]voice.BackendBinConfig{},
	}
	m := voice.NewBackendManager(cfg)
	err := m.Start(nil)
	assert.ErrorContains(t, err, "no voice backend binary configured")
}

func TestBackendManager_Start_UnknownType_ReturnsError(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "moshi-mlx",
		Backends:       map[string]voice.BackendBinConfig{},
	}
	m := voice.NewBackendManager(cfg)
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
			"echo-backend": {Bin: sleepBin, Args: []string{"10"}},
		},
	}
	m := voice.NewBackendManager(cfg)
	err = m.Start(nil)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = m.Stop() })
	assert.True(t, m.IsRunning())
	assert.NoError(t, m.Stop())
	assert.False(t, m.IsRunning())
}
