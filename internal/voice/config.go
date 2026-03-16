package voice

// BackendConfig selects and locates the speech-to-speech backend.
// Type values: auto | personaplex-cuda | personaplex-mlx | moshi | moshi-mlx | compatible
// URL empty means APS manages the backend process; set to delegate to an external instance.
type BackendConfig struct {
	URL  string `yaml:"url,omitempty"`
	Type string `yaml:"type,omitempty"` // default: "auto"
}

// TelegramChannelConfig holds Telegram bot credentials for a profile's voice channel.
type TelegramChannelConfig struct {
	Enabled        bool   `yaml:"enabled,omitempty"`
	BotTokenSecret string `yaml:"bot_token_secret,omitempty"`
}

// TwilioChannelConfig holds Twilio credentials and phone number for inbound call routing.
type TwilioChannelConfig struct {
	Enabled          bool   `yaml:"enabled,omitempty"`
	PhoneNumber      string `yaml:"phone_number,omitempty"`
	AccountSIDSecret string `yaml:"account_sid_secret,omitempty"`
	AuthTokenSecret  string `yaml:"auth_token_secret,omitempty"`
}

// ChannelsConfig declares which channels this profile's voice is active on.
type ChannelsConfig struct {
	Web      bool                   `yaml:"web,omitempty"`
	TUI      bool                   `yaml:"tui,omitempty"`
	Telegram *TelegramChannelConfig `yaml:"telegram,omitempty"`
	Twilio   *TwilioChannelConfig   `yaml:"twilio,omitempty"`
}

// Config is the voice block inside a Profile.
// All fields are optional; APS provides sensible defaults.
type Config struct {
	Enabled        bool           `yaml:"enabled,omitempty"`
	Backend        BackendConfig  `yaml:"backend,omitempty"`
	VoiceID        string         `yaml:"voice_id,omitempty"` // e.g. "NATF0"
	PromptTemplate string         `yaml:"prompt_template,omitempty"`
	Channels       ChannelsConfig `yaml:"channels,omitempty"`
}
