package messenger

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConversationStore(t *testing.T, opts ConversationStoreOptions) *SQLiteConversationStore {
	t.Helper()
	store, err := OpenConversationStore(filepath.Join(t.TempDir(), "conversations.db"), opts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func smsMessage(id, from, text string) *NormalizedMessage {
	return &NormalizedMessage{
		ID:        id,
		Timestamp: time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC),
		Platform:  string(PlatformSMS),
		Sender:    Sender{ID: from, Name: "Caller"},
		Channel:   Channel{ID: "+15550001111"},
		Text:      text,
		PlatformMetadata: map[string]any{
			"service_id": "support-sms",
		},
	}
}

func TestNewInboundTurn_UsesConversationIdentity(t *testing.T) {
	msg := smsMessage("m1", "+15559990000", "hello")
	state := msg.ConversationState()

	turn := NewInboundTurn(msg, "support-sms", "assistant", "reply")

	assert.Equal(t, TurnDirectionInbound, turn.Direction)
	assert.Equal(t, state.ConversationID, turn.ConversationID)
	assert.Equal(t, state.SessionID, turn.SessionID)
	assert.Equal(t, "support-sms", turn.ServiceID)
	assert.Equal(t, "sms", turn.Platform)
	assert.Equal(t, "assistant", turn.ProfileID)
	assert.Equal(t, "reply", turn.ActionName)
	assert.Equal(t, "m1", turn.MessageID)
	assert.Equal(t, "+15550001111", turn.ChannelID)
	assert.Equal(t, "+15559990000", turn.SenderID)
	assert.Equal(t, "Caller", turn.SenderName)
	assert.Equal(t, "hello", turn.Text)
	assert.Equal(t, msg.Timestamp, turn.Timestamp)
	require.NoError(t, turn.Validate())
}

func TestNewOutboundTurn_KeysOnInboundIdentity(t *testing.T) {
	msg := smsMessage("m1", "+15559990000", "hello")
	state := msg.ConversationState()

	turn := NewOutboundTurn(msg, "Hi there", "support-sms", "assistant", "reply")

	assert.Equal(t, TurnDirectionOutbound, turn.Direction)
	assert.Equal(t, state.ConversationID, turn.ConversationID)
	assert.Equal(t, state.SessionID, turn.SessionID)
	assert.Equal(t, "assistant", turn.SenderID)
	assert.Equal(t, "Hi there", turn.Text)
	assert.Equal(t, "m1", turn.MessageID, "outbound turn keeps the inbound message id it replies to")
	assert.False(t, turn.Timestamp.IsZero())
	require.NoError(t, turn.Validate())
}

func TestConversationTurn_Validate(t *testing.T) {
	valid := ConversationTurn{ConversationID: "c", SessionID: "s", Direction: TurnDirectionInbound}
	require.NoError(t, valid.Validate())

	tests := []struct {
		name string
		turn ConversationTurn
		want string
	}{
		{"missing conversation", ConversationTurn{SessionID: "s", Direction: TurnDirectionInbound}, "conversation ID is required"},
		{"missing session", ConversationTurn{ConversationID: "c", Direction: TurnDirectionInbound}, "session ID is required"},
		{"bad direction", ConversationTurn{ConversationID: "c", SessionID: "s", Direction: "sideways"}, "direction"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.turn.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestSQLiteConversationStore_AppendAndRecentTurns(t *testing.T) {
	store := newTestConversationStore(t, ConversationStoreOptions{})
	ctx := context.Background()

	first := smsMessage("m1", "+15559990000", "hello")
	state := first.ConversationState()

	in1, err := store.AppendTurn(ctx, NewInboundTurn(first, "support-sms", "assistant", "reply"))
	require.NoError(t, err)
	out1, err := store.AppendTurn(ctx, NewOutboundTurn(first, "Hi! How can I help?", "support-sms", "assistant", "reply"))
	require.NoError(t, err)
	second := smsMessage("m2", "+15559990000", "my order is late")
	in2, err := store.AppendTurn(ctx, NewInboundTurn(second, "support-sms", "assistant", "reply"))
	require.NoError(t, err)

	assert.Less(t, in1.Seq, out1.Seq)
	assert.Less(t, out1.Seq, in2.Seq)

	turns, err := store.RecentTurns(ctx, ConversationQuery{ConversationID: state.ConversationID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, turns, 3)
	assert.Equal(t, []string{"hello", "Hi! How can I help?", "my order is late"}, turnTexts(turns), "newest-last")
	assert.Equal(t, TurnDirectionInbound, turns[0].Direction)
	assert.Equal(t, TurnDirectionOutbound, turns[1].Direction)
	assert.Equal(t, "assistant", turns[1].ProfileID)
	assert.Equal(t, "m1", turns[0].MessageID)
	assert.Equal(t, "Caller", turns[0].SenderName)
	assert.True(t, turns[0].Timestamp.Equal(first.Timestamp))
}

func TestSQLiteConversationStore_RecentTurns_BoundedNewestLast(t *testing.T) {
	store := newTestConversationStore(t, ConversationStoreOptions{})
	ctx := context.Background()

	msg := smsMessage("m", "+15559990000", "")
	state := msg.ConversationState()
	for i := 0; i < 5; i++ {
		turn := NewInboundTurn(smsMessage("m", "+15559990000", string(rune('a'+i))), "support-sms", "p", "a")
		_, err := store.AppendTurn(ctx, turn)
		require.NoError(t, err)
	}

	turns, err := store.RecentTurns(ctx, ConversationQuery{ConversationID: state.ConversationID, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, []string{"d", "e"}, turnTexts(turns), "limit keeps the newest turns, ordered oldest to newest")

	turns, err = store.RecentTurns(ctx, ConversationQuery{ConversationID: state.ConversationID, Limit: 0})
	require.NoError(t, err)
	assert.Len(t, turns, 5, "zero limit falls back to the default prior-turn limit (all 5 fit)")
}

func TestSQLiteConversationStore_RecentTurns_SessionScope(t *testing.T) {
	store := newTestConversationStore(t, ConversationStoreOptions{})
	ctx := context.Background()

	root := &NormalizedMessage{
		ID: "root", Platform: "slack", Sender: Sender{ID: "U1"}, Channel: Channel{ID: "C1", Type: ChannelTypeGroup}, Text: "root message",
	}
	reply := &NormalizedMessage{
		ID: "reply", Platform: "slack", Sender: Sender{ID: "U2"}, Channel: Channel{ID: "C1", Type: ChannelTypeGroup}, Text: "thread reply",
		Thread: &Thread{ID: "1700.1", Type: ThreadTypeReply},
	}
	require.Equal(t, root.ConversationState().ConversationID, reply.ConversationState().ConversationID)
	require.NotEqual(t, root.ConversationState().SessionID, reply.ConversationState().SessionID)

	_, err := store.AppendTurn(ctx, NewInboundTurn(root, "slack-svc", "p", "a"))
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, NewInboundTurn(reply, "slack-svc", "p", "a"))
	require.NoError(t, err)

	all, err := store.RecentTurns(ctx, ConversationQuery{ConversationID: root.ConversationState().ConversationID, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, []string{"root message", "thread reply"}, turnTexts(all))

	threadOnly, err := store.RecentTurns(ctx, ConversationQuery{
		ConversationID: root.ConversationState().ConversationID,
		SessionID:      reply.ConversationState().SessionID,
		Limit:          10,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"thread reply"}, turnTexts(threadOnly))
}

func TestSQLiteConversationStore_IsolatesSendersOnSameNumber(t *testing.T) {
	store := newTestConversationStore(t, ConversationStoreOptions{})
	ctx := context.Background()

	alice := smsMessage("a1", "+15550000001", "alice here")
	bob := smsMessage("b1", "+15550000002", "bob here")
	_, err := store.AppendTurn(ctx, NewInboundTurn(alice, "support-sms", "p", "a"))
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, NewInboundTurn(bob, "support-sms", "p", "a"))
	require.NoError(t, err)

	turns, err := store.RecentTurns(ctx, ConversationQuery{ConversationID: alice.ConversationState().ConversationID, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, []string{"alice here"}, turnTexts(turns))
}

func TestSQLiteConversationStore_RetentionCapsTurnsPerConversation(t *testing.T) {
	store := newTestConversationStore(t, ConversationStoreOptions{MaxTurnsPerConversation: 3})
	ctx := context.Background()

	msg := smsMessage("m", "+15559990000", "")
	state := msg.ConversationState()
	for i := 0; i < 5; i++ {
		_, err := store.AppendTurn(ctx, NewInboundTurn(smsMessage("m", "+15559990000", string(rune('a'+i))), "svc", "p", "a"))
		require.NoError(t, err)
	}
	other := smsMessage("o", "+15550000009", "untouched")
	_, err := store.AppendTurn(ctx, NewInboundTurn(other, "svc", "p", "a"))
	require.NoError(t, err)

	turns, err := store.RecentTurns(ctx, ConversationQuery{ConversationID: state.ConversationID, Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "d", "e"}, turnTexts(turns), "oldest turns beyond the cap are pruned")

	otherTurns, err := store.RecentTurns(ctx, ConversationQuery{ConversationID: other.ConversationState().ConversationID, Limit: 100})
	require.NoError(t, err)
	assert.Len(t, otherTurns, 1, "retention is per conversation")
}

func TestSQLiteConversationStore_ListConversations(t *testing.T) {
	store := newTestConversationStore(t, ConversationStoreOptions{})
	ctx := context.Background()

	alice := smsMessage("a1", "+15550000001", "alice here")
	alice.Timestamp = time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	bob := smsMessage("b1", "+15550000002", "bob here")
	bob.Timestamp = time.Date(2026, 8, 19, 11, 0, 0, 0, time.UTC)
	slack := &NormalizedMessage{
		ID: "s1", Platform: "slack", Timestamp: time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC),
		Sender: Sender{ID: "U1"}, Channel: Channel{ID: "C1", Type: ChannelTypeGroup}, Text: "slack msg",
		PlatformMetadata: map[string]any{"service_id": "slack-svc"},
	}

	_, err := store.AppendTurn(ctx, NewInboundTurn(alice, "support-sms", "p", "a"))
	require.NoError(t, err)
	reply := NewOutboundTurn(alice, "hi alice", "support-sms", "p", "a")
	reply.Timestamp = time.Date(2026, 8, 19, 9, 0, 5, 0, time.UTC)
	_, err = store.AppendTurn(ctx, reply)
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, NewInboundTurn(bob, "support-sms", "p", "a"))
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, NewInboundTurn(slack, "slack-svc", "p", "a"))
	require.NoError(t, err)

	summaries, err := store.ListConversations(ctx, ConversationFilter{})
	require.NoError(t, err)
	require.Len(t, summaries, 3)
	assert.Equal(t, slack.ConversationState().ConversationID, summaries[0].ConversationID, "most recently appended first")
	assert.Equal(t, bob.ConversationState().ConversationID, summaries[1].ConversationID)
	aliceSummary := summaries[2]
	assert.Equal(t, alice.ConversationState().ConversationID, aliceSummary.ConversationID)
	assert.Equal(t, "support-sms", aliceSummary.ServiceID)
	assert.Equal(t, "sms", aliceSummary.Platform)
	assert.Equal(t, "+15550001111", aliceSummary.ChannelID)
	assert.Equal(t, 2, aliceSummary.TurnCount)
	assert.True(t, aliceSummary.FirstAt.Equal(alice.Timestamp))
	assert.True(t, aliceSummary.LastAt.Equal(reply.Timestamp))
	assert.Equal(t, TurnDirectionOutbound, aliceSummary.LastDirection)
	assert.Equal(t, "hi alice", aliceSummary.LastText)

	filtered, err := store.ListConversations(ctx, ConversationFilter{ServiceID: "slack-svc"})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, "slack", filtered[0].Platform)

	filtered, err = store.ListConversations(ctx, ConversationFilter{Platform: "sms", Limit: 1})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, bob.ConversationState().ConversationID, filtered[0].ConversationID)
}

func TestSQLiteConversationStore_RejectsInvalidTurn(t *testing.T) {
	store := newTestConversationStore(t, ConversationStoreOptions{})
	_, err := store.AppendTurn(context.Background(), ConversationTurn{Direction: TurnDirectionInbound})
	require.Error(t, err)
}

func TestSQLiteConversationStore_PersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conversations.db")
	store, err := OpenConversationStore(path, ConversationStoreOptions{})
	require.NoError(t, err)
	msg := smsMessage("m1", "+15559990000", "persist me")
	_, err = store.AppendTurn(context.Background(), NewInboundTurn(msg, "svc", "p", "a"))
	require.NoError(t, err)
	require.NoError(t, store.Close())

	reopened, err := OpenConversationStore(path, ConversationStoreOptions{})
	require.NoError(t, err)
	defer func() { _ = reopened.Close() }()
	turns, err := reopened.RecentTurns(context.Background(), ConversationQuery{ConversationID: msg.ConversationState().ConversationID, Limit: 5})
	require.NoError(t, err)
	assert.Equal(t, []string{"persist me"}, turnTexts(turns))
}

func turnTexts(turns []ConversationTurn) []string {
	out := make([]string, 0, len(turns))
	for _, turn := range turns {
		out = append(out, turn.Text)
	}
	return out
}
