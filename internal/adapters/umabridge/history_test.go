package umabridge

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

// historyFile is the one SQLite file a service's conversation history lives
// in. Every bridge in a test opens it by path, as successive `aps serve`
// processes do.
func historyFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "conversations.db")
}

func openHistory(t *testing.T, path string) *msgtypes.SQLiteConversationStore {
	t.Helper()
	store, err := msgtypes.OpenConversationStore(path, msgtypes.ConversationStoreOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func withHistory(store msgtypes.ConversationStore) func(*Config) {
	return func(c *Config) { c.History = store }
}

// nativeSMS is the message the native SMS provider normalizes from the same
// Twilio event: the identity continuity is measured against.
func nativeSMS(t *testing.T, service, sid, body string) *msgtypes.NormalizedMessage {
	t.Helper()
	msg, err := msgtypes.NormalizeSMSPayload(map[string]any{
		"MessageSid": sid, "From": testSender, "To": testNumber, "Body": body,
	}, time.Now())
	require.NoError(t, err)
	msg.PlatformMetadata["messenger_name"] = service
	return msg
}

func turnsOf(t *testing.T, store msgtypes.ConversationStore, msg *msgtypes.NormalizedMessage) []msgtypes.ConversationTurn {
	t.Helper()
	state := msg.ConversationState()
	turns, err := store.RecentTurns(t.Context(), msgtypes.ConversationQuery{
		ConversationID: state.ConversationID, SessionID: state.SessionID, Limit: 100,
	})
	require.NoError(t, err)
	return turns
}

func outboundTurns(turns []msgtypes.ConversationTurn) []msgtypes.ConversationTurn {
	var out []msgtypes.ConversationTurn
	for _, turn := range turns {
		if turn.Direction == msgtypes.TurnDirectionOutbound {
			out = append(out, turn)
		}
	}
	return out
}

func TestReplyIsRecordedUnderTheKeysTheNativeProviderUses(t *testing.T) {
	tests := map[string]struct {
		adapter string
		native  func(t *testing.T) *msgtypes.NormalizedMessage
		form    func(url.Values)
	}{
		"twilio.sms": {
			adapter: "twilio.sms",
			native: func(t *testing.T) *msgtypes.NormalizedMessage {
				return nativeSMS(t, "sms-uma", "SM10000000000000000000000000000031", "keys")
			},
			form: func(url.Values) {},
		},
		"twilio.whatsapp": {
			adapter: "twilio.whatsapp",
			native: func(t *testing.T) *msgtypes.NormalizedMessage {
				msg, err := msgtypes.NormalizeWhatsAppPayload(map[string]any{
					"MessageSid": "SM10000000000000000000000000000031",
					"From":       "whatsapp:" + testSender, "To": "whatsapp:" + testNumber, "Body": "keys",
				}, time.Now())
				require.NoError(t, err)
				msg.PlatformMetadata["messenger_name"] = "sms-uma"
				return msg
			},
			form: func(f url.Values) {
				f.Set("From", "whatsapp:"+testSender)
				f.Set("To", "whatsapp:"+testNumber)
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			svc := smsService()
			svc.Options[core.OptionUmaAdapter] = tc.adapter
			if tc.adapter == "twilio.whatsapp" {
				svc.Options["allowed_numbers"] = "whatsapp:" + testSender
			}
			store := openHistory(t, historyFile(t))
			h := newHarnessWith(t, svc, replyWith("on it"), withHistory(store))

			form := incomingSMS("SM10000000000000000000000000000031", "keys")
			tc.form(form)
			require.Equal(t, http.StatusOK, h.postSigned(t, form).StatusCode)
			waitFor(t, "reply recorded", func() bool { return len(outboundTurns(turnsOf(t, store, tc.native(t)))) == 1 })

			// Same conversation and session as the native provider's, so
			// a service converted to uma, under the same id, continues.
			native := tc.native(t)
			state := native.ConversationState()
			turn := outboundTurns(turnsOf(t, store, native))[0]
			assert.Equal(t, state.ConversationID, turn.ConversationID)
			assert.Equal(t, state.SessionID, turn.SessionID)
			assert.Equal(t, "sms-uma", turn.ServiceID)
			assert.Equal(t, native.Platform, turn.Platform)
			assert.Equal(t, "assistant", turn.ProfileID)
			assert.Equal(t, "triage", turn.ActionName)
			assert.Equal(t, "on it", turn.Text)
			assert.Equal(t, "SM10000000000000000000000000000031", turn.MessageID, "outbound turns carry the answered provider id, never a uma id")
			assert.Equal(t, native.Channel.ID, turn.ChannelID)
		})
	}
}

// The turn means "the reply was accepted for delivery", as the receipt
// status says. uma sends from its own worker, so a provider refusal settles
// after the turn is written; nothing in aps history is revised then.
func TestReplyTheProviderRefusesIsStillRecordedAsQueued(t *testing.T) {
	store := openHistory(t, historyFile(t))
	h := newHarnessWith(t, smsService(), replyWith("on it"), withHistory(store))
	h.twilio.rejectWith(http.StatusBadRequest)

	require.Equal(t, http.StatusOK, h.postSigned(t, incomingSMS("SM10000000000000000000000000000032", "refused")).StatusCode)

	waitFor(t, "result", func() bool { return len(h.outcomes()) == 1 })
	out := h.outcomes()[0]
	require.NoError(t, out.err)
	assert.Equal(t, "accepted", out.result.Delivery.Status)
	turns := turnsOf(t, store, nativeSMS(t, "sms-uma", "SM10000000000000000000000000000032", "refused"))
	require.Len(t, outboundTurns(turns), 1)
	assert.Equal(t, "on it", outboundTurns(turns)[0].Text)
}

func TestActionWithoutReplyRecordsNoOutboundTurn(t *testing.T) {
	store := openHistory(t, historyFile(t))
	h := newHarnessWith(t, smsService(), func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
		return &msgtypes.ExecutionResult{Status: "success", Output: "done"}, nil
	}, withHistory(store))

	h.postSigned(t, incomingSMS("SM10000000000000000000000000000033", "quiet"))

	waitFor(t, "result", func() bool { return len(h.outcomes()) == 1 })
	assert.Empty(t, turnsOf(t, store, nativeSMS(t, "sms-uma", "SM10000000000000000000000000000033", "quiet")))
}

func TestAnsweredEventRedeliveredAfterARestartIsNotRunAgain(t *testing.T) {
	path := historyFile(t)
	twilio := newFakeTwilio(t)
	form := incomingSMS("SM10000000000000000000000000000034", "once only")
	native := nativeSMS(t, "sms-uma", "SM10000000000000000000000000000034", "once only")

	reader := openHistory(t, path)
	first := newHarnessOn(t, twilio, smsService(), replyWith("first"), withHistory(openHistory(t, path)))
	require.Equal(t, http.StatusOK, first.postSigned(t, form).StatusCode)
	waitFor(t, "first reply", func() bool { return len(twilio.sent()) == 1 })
	waitFor(t, "first reply recorded", func() bool { return len(outboundTurns(turnsOf(t, reader, native))) == 1 })
	first.stop()

	// A new process: uma's memory, including its record of the event, is
	// gone. The provider delivers the event again.
	store := openHistory(t, path)
	second := newHarnessOn(t, twilio, smsService(), replyWith("second"), withHistory(store))
	require.Equal(t, http.StatusOK, second.postSigned(t, form).StatusCode)
	waitFor(t, "second result", func() bool { return len(second.outcomes()) == 1 })

	assert.Empty(t, second.exec.calls(), "the action does not run for an event whose reply is already queued")
	assert.Len(t, twilio.sent(), 1, "no second reply")
	require.NoError(t, second.outcomes()[0].err)
	assert.Equal(t, StatusDuplicate, second.outcomes()[0].result.Result.Status)
	assert.Nil(t, second.outcomes()[0].result.Delivery)
	assert.Len(t, outboundTurns(turnsOf(t, store, native)), 1, "history records the reply once")
}

func TestNewEventsAndUnansweredEventsStillRunAfterARestart(t *testing.T) {
	path := historyFile(t)
	twilio := newFakeTwilio(t)
	answered := incomingSMS("SM10000000000000000000000000000035", "answered")
	unanswered := incomingSMS("SM10000000000000000000000000000036", "no reply yet")
	reader := openHistory(t, path)

	first := newHarnessOn(t, twilio, smsService(), func(h msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
		// The router records the inbound turn before the action runs, so a
		// failed action leaves an inbound turn and no reply.
		_, _ = reader.AppendTurn(context.Background(), msgtypes.NewInboundTurn(h.Message, "", h.ProfileID, h.ActionName))
		if h.Message.ID == "SM10000000000000000000000000000036" {
			return nil, errors.New("action crashed")
		}
		return &msgtypes.ExecutionResult{Status: "success", Reply: &msgtypes.DeliveryRequest{Text: "first"}}, nil
	}, withHistory(openHistory(t, path)))
	first.postSigned(t, answered)
	first.postSigned(t, unanswered)
	waitFor(t, "both handled", func() bool { return len(first.outcomes()) == 2 })
	waitFor(t, "reply sent", func() bool { return len(twilio.sent()) == 1 })
	first.stop()

	second := newHarnessOn(t, twilio, smsService(), replyWith("again"), withHistory(openHistory(t, path)))
	second.postSigned(t, unanswered)
	second.postSigned(t, incomingSMS("SM10000000000000000000000000000037", "brand new"))
	waitFor(t, "both run", func() bool { return len(second.exec.calls()) == 2 })

	ids := []string{second.exec.calls()[0].Message.ID, second.exec.calls()[1].Message.ID}
	assert.ElementsMatch(t, []string{"SM10000000000000000000000000000036", "SM10000000000000000000000000000037"}, ids,
		"an event whose action never produced a reply is not mistaken for an answered one")
}

// A history that cannot be read must not swallow messages: the router treats
// it the same way (logged, routing continues).
func TestUnreadableHistoryDoesNotBlockTheAction(t *testing.T) {
	h := newHarnessWith(t, smsService(), replyWith("on it"), withHistory(brokenReads{}))

	require.Equal(t, http.StatusOK, h.postSigned(t, incomingSMS("SM10000000000000000000000000000038", "hi")).StatusCode)

	waitFor(t, "reply", func() bool { return len(h.twilio.sent()) == 1 })
	assert.Len(t, h.exec.calls(), 1)
}

type brokenReads struct{}

func (brokenReads) AppendTurn(_ context.Context, turn msgtypes.ConversationTurn) (msgtypes.ConversationTurn, error) {
	return turn, nil
}

func (brokenReads) RecentTurns(context.Context, msgtypes.ConversationQuery) ([]msgtypes.ConversationTurn, error) {
	return nil, errors.New("history unreadable")
}

func (brokenReads) ListConversations(context.Context, msgtypes.ConversationFilter) ([]msgtypes.ConversationSummary, error) {
	return nil, errors.New("history unreadable")
}

func (brokenReads) Close() error { return nil }
