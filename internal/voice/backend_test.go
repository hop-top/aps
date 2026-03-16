package voice_test

import (
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
