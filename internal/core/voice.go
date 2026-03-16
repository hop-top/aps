package core

// VoiceBackendConfig selects and locates the speech-to-speech backend.
// Type values: auto | personaplex-cuda | personaplex-mlx | moshi | moshi-mlx | compatible
// URL empty means APS manages the backend process; set to delegate to an external instance.
type VoiceBackendConfig struct {
	URL  string `yaml:"url,omitempty"`
	Type string `yaml:"type,omitempty"` // default: "auto"
}

// VoiceTelegramChannelConfig holds Telegram bot credentials for a profile's voice channel.
type VoiceTelegramChannelConfig struct {
	Enabled        bool   `yaml:"enabled,omitempty"`
	BotTokenSecret string `yaml:"bot_token_secret,omitempty"`
}

// VoiceTwilioChannelConfig holds Twilio credentials and phone number for inbound call routing.
type VoiceTwilioChannelConfig struct {
	Enabled          bool   `yaml:"enabled,omitempty"`
	PhoneNumber      string `yaml:"phone_number,omitempty"`
	AccountSIDSecret string `yaml:"account_sid_secret,omitempty"`
	AuthTokenSecret  string `yaml:"auth_token_secret,omitempty"`
}

// VoiceChannelsConfig declares which channels this profile's voice is active on.
type VoiceChannelsConfig struct {
	Web      bool                        `yaml:"web,omitempty"`
	TUI      bool                        `yaml:"tui,omitempty"`
	Telegram *VoiceTelegramChannelConfig `yaml:"telegram,omitempty"`
	Twilio   *VoiceTwilioChannelConfig   `yaml:"twilio,omitempty"`
}

// VoiceConfig is the voice block inside a Profile.
// All fields are optional; APS provides sensible defaults.
type VoiceConfig struct {
	Enabled        bool                `yaml:"enabled,omitempty"`
	Backend        VoiceBackendConfig  `yaml:"backend,omitempty"`
	VoiceID        string              `yaml:"voice_id,omitempty"` // e.g. "NATF0"
	PromptTemplate string              `yaml:"prompt_template,omitempty"`
	Channels       VoiceChannelsConfig `yaml:"channels,omitempty"`
}
