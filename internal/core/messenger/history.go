package messenger

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"hop.top/aps/internal/core"
)

// Turn directions recorded in the conversation store.
const (
	TurnDirectionInbound  = "inbound"
	TurnDirectionOutbound = "outbound"
)

const (
	// DefaultPriorTurnLimit bounds how many prior turns are attached to a
	// routed action run when no per-service override is configured.
	DefaultPriorTurnLimit = 20

	// DefaultConversationRetention caps how many turns the store keeps per
	// conversation before the oldest are pruned on append.
	DefaultConversationRetention = 500

	// ConversationsDir is the data-dir subdirectory holding the store.
	ConversationsDir = "messages"

	// ConversationStoreFile is the on-disk sqlite filename.
	ConversationStoreFile = "conversations.db"
)

// ConversationTurn is one persisted message-service turn. Turns are keyed by
// the policy identity from ConversationState: ConversationID is the outer
// channel or direct-message pair, SessionID narrows to a platform thread.
type ConversationTurn struct {
	Seq            int64        `json:"seq" yaml:"seq"`
	ConversationID string       `json:"conversation_id" yaml:"conversation_id"`
	SessionID      string       `json:"session_id" yaml:"session_id"`
	ServiceID      string       `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	Platform       string       `json:"platform" yaml:"platform"`
	ProfileID      string       `json:"profile_id,omitempty" yaml:"profile_id,omitempty"`
	ActionName     string       `json:"action_name,omitempty" yaml:"action_name,omitempty"`
	Direction      string       `json:"direction" yaml:"direction"`
	MessageID      string       `json:"message_id,omitempty" yaml:"message_id,omitempty"`
	ChannelID      string       `json:"channel_id,omitempty" yaml:"channel_id,omitempty"`
	SenderID       string       `json:"sender_id,omitempty" yaml:"sender_id,omitempty"`
	SenderName     string       `json:"sender_name,omitempty" yaml:"sender_name,omitempty"`
	Text           string       `json:"text" yaml:"text"`
	Attachments    []Attachment `json:"attachments,omitempty" yaml:"attachments,omitempty"`
	Timestamp      time.Time    `json:"timestamp" yaml:"timestamp"`
}

// Validate checks the identity fields required to persist a turn.
func (t ConversationTurn) Validate() error {
	if t.ConversationID == "" {
		return fmt.Errorf("conversation ID is required")
	}
	if t.SessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	switch t.Direction {
	case TurnDirectionInbound, TurnDirectionOutbound:
		return nil
	default:
		return fmt.Errorf("direction must be %q or %q, got %q", TurnDirectionInbound, TurnDirectionOutbound, t.Direction)
	}
}

// ConversationSummary is one conversation as seen by the query surface.
type ConversationSummary struct {
	ConversationID string    `json:"conversation_id" yaml:"conversation_id"`
	ServiceID      string    `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	Platform       string    `json:"platform" yaml:"platform"`
	ChannelID      string    `json:"channel_id,omitempty" yaml:"channel_id,omitempty"`
	TurnCount      int       `json:"turn_count" yaml:"turn_count"`
	FirstAt        time.Time `json:"first_at" yaml:"first_at"`
	LastAt         time.Time `json:"last_at" yaml:"last_at"`
	LastDirection  string    `json:"last_direction" yaml:"last_direction"`
	LastText       string    `json:"last_text" yaml:"last_text"`
}

// ConversationQuery selects prior turns. SessionID is optional and narrows
// the conversation to one platform thread. Limit <= 0 falls back to
// DefaultPriorTurnLimit.
type ConversationQuery struct {
	ConversationID string
	SessionID      string
	Limit          int
}

// ConversationFilter narrows ListConversations. Empty fields match all;
// Limit <= 0 returns every conversation.
type ConversationFilter struct {
	ServiceID string
	Platform  string
	Limit     int
}

// ConversationStore persists message-service turns keyed by the policy
// conversation identity.
type ConversationStore interface {
	// AppendTurn persists a turn and returns it with Seq assigned.
	AppendTurn(ctx context.Context, turn ConversationTurn) (ConversationTurn, error)
	// RecentTurns returns up to Limit turns for the conversation (optionally
	// narrowed to one session), oldest first so the newest turn is last.
	RecentTurns(ctx context.Context, query ConversationQuery) ([]ConversationTurn, error)
	// ListConversations returns conversation summaries, most recently
	// active first.
	ListConversations(ctx context.Context, filter ConversationFilter) ([]ConversationSummary, error)
	Close() error
}

// NewInboundTurn builds the persisted turn for a routed inbound message.
func NewInboundTurn(msg *NormalizedMessage, serviceID, profileID, actionName string) ConversationTurn {
	turn := baseTurn(msg, serviceID, profileID, actionName)
	turn.Direction = TurnDirectionInbound
	turn.SenderID = msg.Sender.ID
	turn.SenderName = msg.Sender.Name
	turn.Text = msg.Text
	turn.Attachments = append([]Attachment(nil), msg.Attachments...)
	turn.Timestamp = msg.Timestamp
	if turn.Timestamp.IsZero() {
		turn.Timestamp = time.Now().UTC()
	}
	return turn
}

// NewOutboundTurn builds the persisted turn for a reply delivered in response
// to msg. The reply is attributed to the profile that produced it and keeps
// the inbound message ID so the pair can be correlated.
func NewOutboundTurn(msg *NormalizedMessage, text, serviceID, profileID, actionName string) ConversationTurn {
	turn := baseTurn(msg, serviceID, profileID, actionName)
	turn.Direction = TurnDirectionOutbound
	turn.SenderID = profileID
	turn.Text = text
	turn.Timestamp = time.Now().UTC()
	return turn
}

func baseTurn(msg *NormalizedMessage, serviceID, profileID, actionName string) ConversationTurn {
	state := msg.ConversationState()
	if serviceID == "" {
		serviceID = state.ServiceID
	}
	return ConversationTurn{
		ConversationID: state.ConversationID,
		SessionID:      state.SessionID,
		ServiceID:      serviceID,
		Platform:       msg.Platform,
		ProfileID:      profileID,
		ActionName:     actionName,
		MessageID:      msg.ID,
		ChannelID:      msg.Channel.ID,
	}
}

// DefaultConversationStorePath returns the sqlite path under the APS data
// directory (honours APS_DATA_PATH via core.GetDataDir).
func DefaultConversationStorePath() (string, error) {
	dataDir, err := core.GetDataDir()
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	return filepath.Join(dataDir, ConversationsDir, ConversationStoreFile), nil
}
