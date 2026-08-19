package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/cli/listing"
	msgtypes "hop.top/aps/internal/core/messenger"
)

// lastTextPreviewRunes bounds the LAST TEXT column in `conversation list`;
// `conversation show` prints full turns.
const lastTextPreviewRunes = 60

// conversationRow is the table/json/yaml row for `aps service conversation
// list`. Higher-priority columns survive narrow terminals.
type conversationRow struct {
	ConversationID string `table:"CONVERSATION,priority=10" json:"conversation_id" yaml:"conversation_id"`
	ServiceID      string `table:"SERVICE,priority=9"       json:"service_id"      yaml:"service_id"`
	Platform       string `table:"PLATFORM,priority=8"      json:"platform"        yaml:"platform"`
	ChannelID      string `table:"CHANNEL,priority=7"       json:"channel_id"      yaml:"channel_id"`
	TurnCount      int    `table:"TURNS,priority=6"         json:"turn_count"      yaml:"turn_count"`
	LastAt         string `table:"LAST,priority=5"          json:"last_at"         yaml:"last_at"`
	LastDirection  string `table:"DIR,priority=4"           json:"last_direction"  yaml:"last_direction"`
	LastText       string `table:"LAST TEXT,priority=3"     json:"last_text"       yaml:"last_text"`
	FirstAt        string `json:"first_at" yaml:"first_at"`
}

// turnRow is the table/json/yaml row for `aps service conversation show`.
// Untagged-for-table fields still serialize in json/yaml.
type turnRow struct {
	Seq            int64                 `table:"SEQ,priority=10"       json:"seq"             yaml:"seq"`
	Timestamp      string                `table:"AT,priority=9"         json:"timestamp"       yaml:"timestamp"`
	Direction      string                `table:"DIRECTION,priority=8"  json:"direction"       yaml:"direction"`
	Sender         string                `table:"SENDER,priority=7"     json:"sender"          yaml:"sender"`
	Text           string                `table:"TEXT,priority=6"       json:"text"            yaml:"text"`
	MessageID      string                `table:"MESSAGE,priority=2"    json:"message_id"      yaml:"message_id"`
	SenderID       string                `json:"sender_id"       yaml:"sender_id"`
	SenderName     string                `json:"sender_name"     yaml:"sender_name"`
	ProfileID      string                `json:"profile_id"      yaml:"profile_id"`
	ActionName     string                `json:"action_name"     yaml:"action_name"`
	ServiceID      string                `json:"service_id"      yaml:"service_id"`
	Platform       string                `json:"platform"        yaml:"platform"`
	ChannelID      string                `json:"channel_id"      yaml:"channel_id"`
	ConversationID string                `json:"conversation_id" yaml:"conversation_id"`
	SessionID      string                `json:"session_id"      yaml:"session_id"`
	Attachments    []msgtypes.Attachment `json:"attachments,omitempty" yaml:"attachments,omitempty"`
}

func newConversationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "conversation",
		Aliases: []string{"conversations"},
		Short:   "Inspect recorded message-service conversations",
		Long: `Query the message thread history recorded by message services.

Every routed inbound message and every delivered reply is persisted as a
conversation turn keyed by the conversation policy identity
(msgconv:v1:... / msgsess:v1:...). The same turns are attached to routed
action runs as prior_turns.`,
	}
	cmd.AddCommand(newConversationListCmd())
	cmd.AddCommand(newConversationShowCmd())
	return cmd
}

func newConversationListCmd() *cobra.Command {
	var (
		serviceID string
		platform  string
		limit     int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recorded conversations",
		Long: `List conversations recorded by message services, most recently
active first. Each row reports the conversation ID, owning service,
platform, channel, turn count, last activity time and direction, and a
preview of the last turn's text.

Filters: --service, --platform, --limit. Output respects the global
--format flag (table|json|yaml). Read-only: the store is never created
by this command; a missing store lists nothing. Idempotent.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rows := []conversationRow{}
			err := withConversationStore(func(store msgtypes.ConversationStore) error {
				summaries, err := store.ListConversations(commandContext(cmd), msgtypes.ConversationFilter{
					ServiceID: serviceID,
					Platform:  platform,
					Limit:     limit,
				})
				if err != nil {
					return fmt.Errorf("list conversations: %w", err)
				}
				for _, summary := range summaries {
					rows = append(rows, conversationToRow(summary))
				}
				return nil
			})
			if err != nil {
				return err
			}
			format, _ := cmd.Flags().GetString("format")
			return listing.RenderList(cmd.OutOrStdout(), format, rows)
		},
	}
	cmd.Flags().StringVar(&serviceID, "service", "", "Only conversations recorded by this service ID")
	cmd.Flags().StringVar(&platform, "platform", "", "Only conversations on this platform (sms, whatsapp, telegram, slack, discord)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum conversations to list (0 = all)")
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	_ = kitcli.SetExamples(cmd, []kitcli.Example{
		{Title: "List every recorded conversation", Command: "aps service conversation list"},
		{Title: "List SMS conversations for one service as JSON", Command: "aps service conversation list --service support-sms --format json"},
	})
	return cmd
}

func newConversationShowCmd() *cobra.Command {
	var (
		sessionID string
		limit     int
	)
	cmd := &cobra.Command{
		Use:   "show <conversation-id>",
		Short: "Show the recorded turns of a conversation",
		Long: `Print the turns recorded for one conversation, oldest first so the
newest turn is last — the same order and shape actions receive in
prior_turns. Each turn carries its direction (inbound/outbound), sender,
text, message ID, profile/action, and the session (thread) key.

--session narrows the conversation to one policy session key (a platform
thread); --limit keeps only the newest N turns. Output respects the
global --format flag (table|json|yaml). Read-only. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conversationID := args[0]
			rows := []turnRow{}
			err := withConversationStore(func(store msgtypes.ConversationStore) error {
				turns, err := store.RecentTurns(commandContext(cmd), msgtypes.ConversationQuery{
					ConversationID: conversationID,
					SessionID:      sessionID,
					Limit:          limit,
				})
				if err != nil {
					return fmt.Errorf("show conversation: %w", err)
				}
				for _, turn := range turns {
					rows = append(rows, turnToRow(turn))
				}
				return nil
			})
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				return fmt.Errorf("conversation %q has no recorded turns", conversationID)
			}
			format, _ := cmd.Flags().GetString("format")
			return listing.RenderList(cmd.OutOrStdout(), format, rows)
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "Only turns in this session (thread) key")
	cmd.Flags().IntVar(&limit, "limit", msgtypes.DefaultConversationRetention, "Maximum turns to show, newest kept")
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	_ = kitcli.SetExamples(cmd, []kitcli.Example{
		{Title: "Show a conversation's turns", Command: "aps service conversation show 'msgconv:v1:service:support-sms:platform:sms:channel:%2B15550001111:sender:%2B15559990000'"},
		{Title: "Show the newest 5 turns as JSON", Command: "aps service conversation show <conversation-id> --limit 5 --format json"},
	})
	return cmd
}

// withConversationStore opens the default store read-only for fn. A store
// that was never created is treated as empty and is not created here.
func withConversationStore(fn func(store msgtypes.ConversationStore) error) error {
	path, err := msgtypes.DefaultConversationStorePath()
	if err != nil {
		return fmt.Errorf("resolve conversation store: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat conversation store: %w", err)
	}
	store, err := msgtypes.OpenConversationStore(path, msgtypes.ConversationStoreOptions{})
	if err != nil {
		return fmt.Errorf("open conversation store: %w", err)
	}
	defer func() { _ = store.Close() }()
	return fn(store)
}

func conversationToRow(summary msgtypes.ConversationSummary) conversationRow {
	return conversationRow{
		ConversationID: summary.ConversationID,
		ServiceID:      summary.ServiceID,
		Platform:       summary.Platform,
		ChannelID:      summary.ChannelID,
		TurnCount:      summary.TurnCount,
		FirstAt:        formatTurnTime(summary.FirstAt),
		LastAt:         formatTurnTime(summary.LastAt),
		LastDirection:  summary.LastDirection,
		LastText:       previewText(summary.LastText, lastTextPreviewRunes),
	}
}

func turnToRow(turn msgtypes.ConversationTurn) turnRow {
	sender := turn.SenderName
	if sender == "" {
		sender = turn.SenderID
	}
	return turnRow{
		Seq:            turn.Seq,
		Timestamp:      formatTurnTime(turn.Timestamp),
		Direction:      turn.Direction,
		Sender:         sender,
		Text:           turn.Text,
		MessageID:      turn.MessageID,
		SenderID:       turn.SenderID,
		SenderName:     turn.SenderName,
		ProfileID:      turn.ProfileID,
		ActionName:     turn.ActionName,
		ServiceID:      turn.ServiceID,
		Platform:       turn.Platform,
		ChannelID:      turn.ChannelID,
		ConversationID: turn.ConversationID,
		SessionID:      turn.SessionID,
		Attachments:    turn.Attachments,
	}
}

func formatTurnTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func previewText(text string, maxRunes int) string {
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxRunes]) + "..."
}

func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}
