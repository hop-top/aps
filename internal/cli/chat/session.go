package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"hop.top/aps/internal/core"
	coresession "hop.top/aps/internal/core/session"
)

const transcriptKey = "chat_transcript_json"

type chatSession struct {
	id        string
	profileID string
	messages  []Message
}

func startOrAttachSession(profile *core.Profile, attachID string) (*chatSession, error) {
	registry := coresession.GetRegistry()
	if attachID != "" {
		sess, err := registry.Get(attachID)
		if err != nil {
			return nil, fmt.Errorf("chat session not found: %w", err)
		}
		if string(sess.Type) != sessionType {
			return nil, fmt.Errorf("session %s is not a chat session", attachID)
		}
		if sess.ProfileID != profile.ID {
			return nil, fmt.Errorf("session %s belongs to profile %s, not %s", attachID, sess.ProfileID, profile.ID)
		}
		messages, err := decodeMessages(sess.Environment[transcriptKey])
		if err != nil {
			return nil, fmt.Errorf("decode chat transcript: %w", err)
		}
		return &chatSession{id: sess.ID, profileID: sess.ProfileID, messages: messages}, nil
	}

	profileDir, err := core.GetProfileDir(profile.ID)
	if err != nil {
		return nil, err
	}
	id := newSessionID()
	info := &coresession.SessionInfo{
		ID:         id,
		ProfileID:  profile.ID,
		ProfileDir: profileDir,
		Command:    "aps chat",
		PID:        os.Getpid(),
		Status:     coresession.SessionActive,
		Type:       coresession.SessionType(sessionType),
		Environment: map[string]string{
			"chat_profile_id": profile.ID,
			transcriptKey:     "[]",
		},
	}
	if profile.Workspace != nil {
		info.WorkspaceID = profile.Workspace.Name
	}
	if err := registry.Register(info); err != nil {
		return nil, err
	}
	return &chatSession{id: id, profileID: profile.ID}, nil
}

func (s *chatSession) append(role, content string) error {
	s.messages = append(s.messages, Message{Role: role, Content: content})
	return s.persist()
}

func (s *chatSession) replaceLastAssistant(content string) error {
	s.setLastAssistant(content)
	return s.persist()
}

// setLastAssistant updates the in-memory transcript without touching disk.
// Used by streaming to avoid a write per delta; pair with persist() at end.
func (s *chatSession) setLastAssistant(content string) {
	if len(s.messages) == 0 || s.messages[len(s.messages)-1].Role != roleAssistant {
		s.messages = append(s.messages, Message{Role: roleAssistant, Content: content})
		return
	}
	s.messages[len(s.messages)-1].Content = content
}

func (s *chatSession) persist() error {
	raw, err := json.Marshal(s.messages)
	if err != nil {
		return err
	}
	return coresession.GetRegistry().UpdateSessionMetadata(s.id, map[string]string{
		transcriptKey:     string(raw),
		"chat_turn_count": strconv.Itoa(countUserTurns(s.messages)),
	})
}

func decodeMessages(raw string) ([]Message, error) {
	if raw == "" {
		return nil, nil
	}
	var messages []Message
	if err := json.Unmarshal([]byte(raw), &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func countUserTurns(messages []Message) int {
	n := 0
	for _, msg := range messages {
		if msg.Role == roleUser {
			n++
		}
	}
	return n
}

func newSessionID() string {
	return "chat-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}
