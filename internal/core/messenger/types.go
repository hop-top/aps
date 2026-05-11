package messenger

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// NormalizedMessage is the unified message format across all messaging platforms.
type NormalizedMessage struct {
	ID               string         `json:"id" yaml:"id"`
	Timestamp        time.Time      `json:"timestamp" yaml:"timestamp"`
	Platform         string         `json:"platform" yaml:"platform"`
	WorkspaceID      string         `json:"workspace_id,omitempty" yaml:"workspace_id,omitempty"`
	ProfileID        string         `json:"profile_id,omitempty" yaml:"profile_id,omitempty"`
	Sender           Sender         `json:"sender" yaml:"sender"`
	Channel          Channel        `json:"channel" yaml:"channel"`
	Text             string         `json:"text" yaml:"text"`
	Thread           *Thread        `json:"thread,omitempty" yaml:"thread,omitempty"`
	Reactions        []Reaction     `json:"reactions,omitempty" yaml:"reactions,omitempty"`
	Attachments      []Attachment   `json:"attachments,omitempty" yaml:"attachments,omitempty"`
	PlatformMetadata map[string]any `json:"platform_metadata,omitempty" yaml:"platform_metadata,omitempty"`
}

type Sender struct {
	ID             string `json:"id" yaml:"id"`
	Name           string `json:"name" yaml:"name"`
	PlatformHandle string `json:"platform_handle,omitempty" yaml:"platform_handle,omitempty"`
	PlatformID     string `json:"platform_id,omitempty" yaml:"platform_id,omitempty"`
}

type Channel struct {
	ID         string `json:"id" yaml:"id"`
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	Type       string `json:"type,omitempty" yaml:"type,omitempty"` // "direct", "group", "broadcast", "topic"
	PlatformID string `json:"platform_id,omitempty" yaml:"platform_id,omitempty"`
}

type Thread struct {
	ID   string `json:"id" yaml:"id"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"` // "reply", "topic", "issue"
}

type Reaction struct {
	Emoji string `json:"emoji" yaml:"emoji"`
	Count int    `json:"count" yaml:"count"`
}

type Attachment struct {
	Type      string `json:"type" yaml:"type"` // "image", "file", "video", "audio"
	URL       string `json:"url" yaml:"url"`
	MimeType  string `json:"mime_type,omitempty" yaml:"mime_type,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty" yaml:"size_bytes,omitempty"`
}

const (
	ChannelTypeDirect    = "direct"
	ChannelTypeGroup     = "group"
	ChannelTypeBroadcast = "broadcast"
	ChannelTypeTopic     = "topic"

	ThreadTypeReply = "reply"
	ThreadTypeTopic = "topic"
	ThreadTypeIssue = "issue"

	ConversationScopeChannel = "channel"
	ConversationScopeDirect  = "direct"
	ConversationScopeThread  = "thread"
)

// ConversationState describes the stable APS conversation and session identity
// for a normalized platform message.
type ConversationState struct {
	ServiceID      string `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	Platform       string `json:"platform" yaml:"platform"`
	WorkspaceID    string `json:"workspace_id,omitempty" yaml:"workspace_id,omitempty"`
	ChannelID      string `json:"channel_id" yaml:"channel_id"`
	ChannelType    string `json:"channel_type,omitempty" yaml:"channel_type,omitempty"`
	SenderID       string `json:"sender_id,omitempty" yaml:"sender_id,omitempty"`
	ThreadID       string `json:"thread_id,omitempty" yaml:"thread_id,omitempty"`
	ThreadType     string `json:"thread_type,omitempty" yaml:"thread_type,omitempty"`
	Scope          string `json:"scope" yaml:"scope"`
	ConversationID string `json:"conversation_id" yaml:"conversation_id"`
	SessionID      string `json:"session_id" yaml:"session_id"`
}

// Validate checks that the message has required fields.
func (m *NormalizedMessage) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("message ID is required")
	}
	if m.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	if m.Sender.ID == "" {
		return fmt.Errorf("sender ID is required")
	}
	if m.Channel.ID == "" {
		return fmt.Errorf("channel ID is required")
	}
	return nil
}

// TextPreview returns a truncated version of the message text.
func (m *NormalizedMessage) TextPreview(maxLen int) string {
	if len(m.Text) <= maxLen {
		return m.Text
	}
	return m.Text[:maxLen] + "..."
}

// ConversationState returns the message service policy identity for this
// message. ConversationID identifies the outer channel or direct-message pair;
// SessionID narrows to a platform thread when one exists.
func (m *NormalizedMessage) ConversationState() ConversationState {
	if m == nil {
		return ConversationState{}
	}

	serviceID := m.serviceIdentity()
	scope := ConversationScopeChannel
	includeSender := m.isDirectConversation()
	if includeSender {
		scope = ConversationScopeDirect
	}

	conversationParts := []string{
		"service", firstNonEmptyString(serviceID, m.Platform),
		"platform", m.Platform,
	}
	if m.WorkspaceID != "" {
		conversationParts = append(conversationParts, "workspace", m.WorkspaceID)
	}
	conversationParts = append(conversationParts, "channel", m.Channel.ID)
	if includeSender && m.Sender.ID != "" {
		conversationParts = append(conversationParts, "sender", m.Sender.ID)
	}

	sessionParts := append([]string{}, conversationParts...)
	threadID := ""
	threadType := ""
	if m.Thread != nil && m.Thread.ID != "" {
		threadID = m.Thread.ID
		threadType = m.Thread.Type
		scope = ConversationScopeThread
		sessionParts = append(sessionParts, "thread", threadID)
	}

	return ConversationState{
		ServiceID:      serviceID,
		Platform:       m.Platform,
		WorkspaceID:    m.WorkspaceID,
		ChannelID:      m.Channel.ID,
		ChannelType:    m.Channel.Type,
		SenderID:       m.Sender.ID,
		ThreadID:       threadID,
		ThreadType:     threadType,
		Scope:          scope,
		ConversationID: stableMessageKey("msgconv:v1", conversationParts...),
		SessionID:      stableMessageKey("msgsess:v1", sessionParts...),
	}
}

// SessionThreadID returns the deterministic APS thread/session key for this
// normalized message.
func (m *NormalizedMessage) SessionThreadID() string {
	return m.ConversationState().SessionID
}

func (m *NormalizedMessage) serviceIdentity() string {
	if m.PlatformMetadata == nil {
		return ""
	}
	for _, key := range []string{"service_id", "messenger_name"} {
		if value, ok := m.PlatformMetadata[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func (m *NormalizedMessage) isDirectConversation() bool {
	if m.Channel.Type == ChannelTypeDirect {
		return true
	}
	switch MessengerPlatform(m.Platform) {
	case PlatformSMS, PlatformWhatsApp:
		return true
	default:
		return false
	}
}

func stableMessageKey(prefix string, parts ...string) string {
	encoded := make([]string, 0, len(parts)+1)
	encoded = append(encoded, prefix)
	for _, part := range parts {
		if part == "" {
			encoded = append(encoded, "_")
			continue
		}
		encoded = append(encoded, url.QueryEscape(part))
	}
	return strings.Join(encoded, ":")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// TargetAction represents a parsed "profile-id:action-name" mapping target.
type TargetAction struct {
	ProfileID  string `json:"profile_id" yaml:"profile_id"`
	ActionName string `json:"action_name" yaml:"action_name"`
}

// ParseTargetAction parses a "profile=action" or "profile:action" mapping value.
// The canonical format uses "=" as separator per UX review.
func ParseTargetAction(mapping string) (TargetAction, error) {
	// Support both = and : separators for flexibility
	sep := "="
	if !strings.Contains(mapping, "=") && strings.Contains(mapping, ":") {
		sep = ":"
	}

	parts := strings.SplitN(mapping, sep, 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return TargetAction{}, fmt.Errorf("invalid mapping format '%s': expected 'profile%saction'", mapping, sep)
	}

	return TargetAction{
		ProfileID:  parts[0],
		ActionName: parts[1],
	}, nil
}

// String returns the canonical mapping representation.
func (t TargetAction) String() string {
	return t.ProfileID + "=" + t.ActionName
}

// ProfileMessengerLink represents a link between a profile and a messenger adapter.
type ProfileMessengerLink struct {
	ProfileID      string            `json:"profile_id" yaml:"profile_id"`
	MessengerName  string            `json:"messenger_name" yaml:"messenger_name"`
	MessengerScope string            `json:"messenger_scope" yaml:"messenger_scope"` // "global" or "profile"
	Enabled        bool              `json:"enabled" yaml:"enabled"`
	Mappings       map[string]string `json:"mappings,omitempty" yaml:"mappings,omitempty"` // channel_id -> "profile=action"
	DefaultAction  string            `json:"default_action,omitempty" yaml:"default_action,omitempty"`
	CreatedAt      time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" yaml:"updated_at"`
}

// Validate checks that the link has required fields.
func (l *ProfileMessengerLink) Validate() error {
	if l.ProfileID == "" {
		return fmt.Errorf("profile ID is required")
	}
	if l.MessengerName == "" {
		return fmt.Errorf("messenger name is required")
	}
	return nil
}

// GetActionForChannel returns the target action for a channel, falling back to default.
func (l *ProfileMessengerLink) GetActionForChannel(channelID string) (string, bool) {
	if action, ok := l.Mappings[channelID]; ok {
		return action, true
	}
	if l.DefaultAction != "" {
		return l.DefaultAction, true
	}
	return "", false
}

// MessengerPlatform defines known messenger platform types.
type MessengerPlatform string

const (
	PlatformTelegram MessengerPlatform = "telegram"
	PlatformSlack    MessengerPlatform = "slack"
	PlatformDiscord  MessengerPlatform = "discord"
	PlatformGitHub   MessengerPlatform = "github"
	PlatformEmail    MessengerPlatform = "email"
	PlatformSMS      MessengerPlatform = "sms"
	PlatformWhatsApp MessengerPlatform = "whatsapp"
)

// ChannelIDFormat documents the expected channel ID format per platform.
var ChannelIDFormat = map[MessengerPlatform]string{
	PlatformTelegram: "Numeric chat ID (e.g., -1001234567890)",
	PlatformSlack:    "Alphanumeric channel ID (e.g., C01ABC2DEF)",
	PlatformDiscord:  "Numeric channel ID (e.g., 1234567890123456789)",
	PlatformGitHub:   "org/repo (e.g., myorg/myrepo)",
	PlatformEmail:    "Mailbox name or email address (e.g., inbox, work@co.com)",
	PlatformSMS:      "Phone number receiving SMS (e.g., +15551234567)",
	PlatformWhatsApp: "WhatsApp phone number ID or receiving number (e.g., 123456789012345)",
}
