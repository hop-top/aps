package voice

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"hop.top/aps/internal/core/session"
)

// ChannelMetaKey is the SessionInfo.Environment key under which the
// voice channel type (web | tui | telegram | twilio) is stored when a
// voice session is registered with the core SessionRegistry.
const ChannelMetaKey = "voice_channel"

// RegisterSession registers a new voice session in the core
// SessionRegistry and returns the persisted SessionInfo.
//
// This replaces the legacy in-memory voice.SessionManager: voice
// sessions are now first-class entries in the unified registry. The
// voice-specific channel type is preserved via Environment so existing
// voice runtime code (mic/speaker handles, audio pipeline) can key off
// the unified session ID.
func RegisterSession(profileID, channelType string) (*session.SessionInfo, error) {
	if profileID == "" {
		return nil, fmt.Errorf("voice session: profile ID required")
	}
	info := &session.SessionInfo{
		ID:        uuid.New().String(),
		ProfileID: profileID,
		PID:       os.Getpid(),
		Status:    session.SessionActive,
		Type:      session.SessionTypeVoice,
		Environment: map[string]string{
			ChannelMetaKey: channelType,
		},
	}
	if err := session.GetRegistry().Register(info); err != nil {
		return nil, fmt.Errorf("voice session: register: %w", err)
	}
	return info, nil
}

// CloseSession marks a voice session inactive in the core registry.
// Mirrors the previous SessionManager.Close behaviour: the entry stays
// in the registry (so `aps session list` shows the recently-closed
// state) until the reaper expires it.
func CloseSession(sessionID string) error {
	return session.GetRegistry().UpdateStatus(sessionID, session.SessionInactive)
}
