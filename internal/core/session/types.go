package session

// SessionType identifies the runtime flavour of a session entry in the
// SessionRegistry. The zero value is SessionTypeStandard so existing
// persisted registry entries (which predate this field) deserialize as
// standard sessions and require no migration.
type SessionType string

const (
	// SessionTypeStandard is the default session flavour: a tmux/profile
	// shell session created via `aps session run` and friends.
	SessionTypeStandard SessionType = ""

	// SessionTypeVoice is a voice session: registered when the voice
	// stack starts a conversation. Voice-specific runtime (mic/speaker
	// handles, audio pipeline) lives in internal/voice/ and keys off the
	// unified session ID stored here.
	SessionTypeVoice SessionType = "voice"

	// SessionTypeChat is a native APS chat session. Chat-specific turns are
	// persisted by internal/core/chat while lifecycle remains in the unified
	// session registry.
	SessionTypeChat SessionType = "chat"
)
