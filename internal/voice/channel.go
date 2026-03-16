package voice

// SessionMeta carries metadata about an incoming channel connection.
type SessionMeta struct {
	ProfileID   string // hint from channel (may be empty for intent-routed sessions)
	ChannelType string // "web" | "tui" | "telegram" | "twilio"
	CallerID    string // platform user ID, phone number, etc.
}

// ChannelSession is the uniform interface all channel adapters present to the orchestrator.
// AudioIn delivers raw PCM frames from the caller.
// AudioOut accepts raw PCM frames to send to the caller.
// TextOut accepts text responses for text-only channels (messenger).
type ChannelSession interface {
	AudioIn() <-chan []byte
	AudioOut() chan<- []byte
	TextOut() chan<- string
	Meta() SessionMeta
	Close() error
}

// ChannelAdapter listens on a channel and emits ChannelSessions.
type ChannelAdapter interface {
	Accept() (<-chan ChannelSession, error)
	Close() error
}
