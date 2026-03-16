package voice

import "hop.top/aps/internal/core"

// Re-export core voice config types so callers can use either package.
type (
	BackendConfig          = core.VoiceBackendConfig
	TelegramChannelConfig  = core.VoiceTelegramChannelConfig
	TwilioChannelConfig    = core.VoiceTwilioChannelConfig
	ChannelsConfig         = core.VoiceChannelsConfig
	Config                 = core.VoiceConfig
)
