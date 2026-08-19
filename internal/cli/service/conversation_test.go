package service

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	msgtypes "hop.top/aps/internal/core/messenger"
)

func newConversationTestCmd() *cobra.Command {
	cmd := newTestServiceCmd()
	cmd.PersistentFlags().String("format", "", "output format")
	return cmd
}

func runConversationCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newConversationTestCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"conversation"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func seedConversationStore(t *testing.T) (alice, bob, slack *msgtypes.NormalizedMessage) {
	t.Helper()
	store, err := msgtypes.OpenDefaultConversationStore(msgtypes.ConversationStoreOptions{})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	alice = &msgtypes.NormalizedMessage{
		ID: "a1", Platform: "sms", Timestamp: time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC),
		Sender: msgtypes.Sender{ID: "+15550000001", Name: "Alice"}, Channel: msgtypes.Channel{ID: "+15550001111"},
		Text: "alice here", PlatformMetadata: map[string]any{"service_id": "support-sms"},
	}
	bob = &msgtypes.NormalizedMessage{
		ID: "b1", Platform: "sms", Timestamp: time.Date(2026, 8, 19, 9, 30, 0, 0, time.UTC),
		Sender: msgtypes.Sender{ID: "+15550000002", Name: "Bob"}, Channel: msgtypes.Channel{ID: "+15550001111"},
		Text: "bob here", PlatformMetadata: map[string]any{"service_id": "support-sms"},
	}
	slack = &msgtypes.NormalizedMessage{
		ID: "s1", Platform: "slack", Timestamp: time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC),
		Sender: msgtypes.Sender{ID: "U1"}, Channel: msgtypes.Channel{ID: "C1", Type: msgtypes.ChannelTypeGroup},
		Text: "slack root", PlatformMetadata: map[string]any{"service_id": "slack-svc"},
	}
	slackThread := &msgtypes.NormalizedMessage{
		ID: "s2", Platform: "slack", Timestamp: time.Date(2026, 8, 19, 10, 5, 0, 0, time.UTC),
		Sender: msgtypes.Sender{ID: "U2"}, Channel: msgtypes.Channel{ID: "C1", Type: msgtypes.ChannelTypeGroup},
		Text: "slack thread reply", Thread: &msgtypes.Thread{ID: "1700.1", Type: msgtypes.ThreadTypeReply},
		PlatformMetadata: map[string]any{"service_id": "slack-svc"},
	}

	_, err = store.AppendTurn(ctx, msgtypes.NewInboundTurn(alice, "support-sms", "assistant", "reply"))
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, msgtypes.NewOutboundTurn(alice, "hi alice", "support-sms", "assistant", "reply"))
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, msgtypes.NewInboundTurn(bob, "support-sms", "assistant", "reply"))
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, msgtypes.NewInboundTurn(slack, "slack-svc", "assistant", "triage"))
	require.NoError(t, err)
	_, err = store.AppendTurn(ctx, msgtypes.NewInboundTurn(slackThread, "slack-svc", "assistant", "triage"))
	require.NoError(t, err)
	return alice, bob, slack
}

func TestConversationList_EmptyWhenStoreMissing(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	out, err := runConversationCmd(t, "list", "--format", "json")
	require.NoError(t, err)
	assert.JSONEq(t, "[]", out)

	path, err := msgtypes.DefaultConversationStorePath()
	require.NoError(t, err)
	assert.NoFileExists(t, path, "listing must not create the store")
}

func TestConversationList_JSONAndFilters(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	alice, bob, slack := seedConversationStore(t)

	out, err := runConversationCmd(t, "list", "--format", "json")
	require.NoError(t, err)
	var rows []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	require.Len(t, rows, 3)
	assert.Equal(t, slack.ConversationState().ConversationID, rows[0]["conversation_id"], "most recently appended first")
	assert.Equal(t, bob.ConversationState().ConversationID, rows[1]["conversation_id"])
	assert.Equal(t, alice.ConversationState().ConversationID, rows[2]["conversation_id"])
	assert.Equal(t, "support-sms", rows[2]["service_id"])
	assert.Equal(t, "sms", rows[2]["platform"])
	assert.Equal(t, "+15550001111", rows[2]["channel_id"])
	assert.EqualValues(t, 2, rows[2]["turn_count"])
	assert.Equal(t, "outbound", rows[2]["last_direction"])
	assert.Equal(t, "hi alice", rows[2]["last_text"])
	assert.Equal(t, "2026-08-19T09:00:00Z", rows[2]["first_at"])
	assert.EqualValues(t, 2, rows[0]["turn_count"], "root and thread turns share one conversation")

	out, err = runConversationCmd(t, "list", "--format", "json", "--service", "slack-svc")
	require.NoError(t, err)
	rows = nil
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, "slack", rows[0]["platform"])

	out, err = runConversationCmd(t, "list", "--format", "json", "--platform", "sms", "--limit", "1")
	require.NoError(t, err)
	rows = nil
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, bob.ConversationState().ConversationID, rows[0]["conversation_id"])
}

func TestConversationList_TableHeaders(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	seedConversationStore(t)

	out, err := runConversationCmd(t, "list")
	require.NoError(t, err)
	assert.Contains(t, out, "CONVERSATION")
	assert.Contains(t, out, "SERVICE")
	assert.Contains(t, out, "TURNS")
	assert.Contains(t, out, "support-sms")
}

func TestConversationShow_TurnsNewestLast(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	alice, _, slack := seedConversationStore(t)

	out, err := runConversationCmd(t, "show", alice.ConversationState().ConversationID, "--format", "json")
	require.NoError(t, err)
	var rows []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	require.Len(t, rows, 2)
	assert.Equal(t, "inbound", rows[0]["direction"])
	assert.Equal(t, "alice here", rows[0]["text"])
	assert.Equal(t, "Alice", rows[0]["sender_name"])
	assert.Equal(t, "+15550000001", rows[0]["sender_id"])
	assert.Equal(t, "2026-08-19T09:00:00Z", rows[0]["timestamp"])
	assert.Equal(t, "outbound", rows[1]["direction"])
	assert.Equal(t, "hi alice", rows[1]["text"])
	assert.Equal(t, "assistant", rows[1]["profile_id"])
	assert.Equal(t, "reply", rows[1]["action_name"])
	assert.Equal(t, alice.ConversationState().SessionID, rows[1]["session_id"])

	out, err = runConversationCmd(t, "show", alice.ConversationState().ConversationID, "--format", "json", "--limit", "1")
	require.NoError(t, err)
	rows = nil
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, "hi alice", rows[0]["text"], "limit keeps the newest turn")

	threadSession := slack.ConversationState().SessionID + ":thread:1700.1"
	out, err = runConversationCmd(t, "show", slack.ConversationState().ConversationID, "--format", "json", "--session", threadSession)
	require.NoError(t, err)
	rows = nil
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, "slack thread reply", rows[0]["text"])

	out, err = runConversationCmd(t, "show", alice.ConversationState().ConversationID)
	require.NoError(t, err)
	assert.Contains(t, out, "DIRECTION")
	assert.Contains(t, out, "alice here")
}

func TestConversationShow_UnknownConversationErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	seedConversationStore(t)

	_, err := runConversationCmd(t, "show", "msgconv:v1:nope", "--format", "json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recorded turns")

	// A missing store also reports the conversation as unknown, without
	// creating the database.
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "other"))
	_, err = runConversationCmd(t, "show", "msgconv:v1:nope", "--format", "json")
	require.Error(t, err)
	path, _ := msgtypes.DefaultConversationStorePath()
	assert.NoFileExists(t, path)
}
