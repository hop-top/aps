package voice_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestBackendConfig_Defaults(t *testing.T) {
	cfg := voice.BackendConfig{}
	assert.Equal(t, "", cfg.URL)
	assert.Equal(t, "", cfg.Type)
}

func TestVoiceConfig_IsEnabled(t *testing.T) {
	cfg := voice.Config{Enabled: true}
	assert.True(t, cfg.Enabled)
}

func TestChannelsConfig_Fields(t *testing.T) {
	cfg := voice.ChannelsConfig{
		Web: true,
		TUI: true,
	}
	assert.True(t, cfg.Web)
	assert.True(t, cfg.TUI)
}
