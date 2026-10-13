package umabridge

import (
	"context"
	"fmt"
	"strings"

	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/uma"
	"hop.top/uma/companions/bot"
	"hop.top/uma/model"
)

// turnState carries the bot turn through messenger.Runtime in the context:
// the runtime hands the context to NormalizeIngress and DeliverMessage
// unchanged, so the provider keeps no per-message state.
type turnState struct {
	turn   *bot.Turn
	client uma.Client
}

type stateKey struct{}

func stateFrom(ctx context.Context) (*turnState, error) {
	st, ok := ctx.Value(stateKey{}).(*turnState)
	if !ok || st == nil || st.turn == nil || st.turn.Message == nil {
		return nil, fmt.Errorf("uma bridge: no received message in context")
	}
	return st, nil
}

// provider adapts the bridge to messenger.Runtime: it normalizes the message
// of the turn in context and delivers the action's reply as a uma reply.
type provider struct {
	platform string
	entry    entry
	key      string
	service  string
	// receiving is the address inbound messages arrive at.
	receiving string
	// mediaDir is where inbound media is materialized; empty means none.
	mediaDir string
	// validate is aps's service policy (allowed numbers, senders, channels),
	// applied to the normalized message before anything else touches it.
	validate func(*msgtypes.NormalizedMessage) error
}

var (
	_ msgtypes.MessageProvider  = (*provider)(nil)
	_ msgtypes.ProviderDelivery = (*provider)(nil)
)

func (p *provider) Metadata() msgtypes.ProviderRuntimeMetadata {
	return msgtypes.ProviderRuntimeMetadata{
		Provider:      p.platform,
		DisplayName:   "uma " + p.key,
		IngressModes:  []msgtypes.IngressMode{msgtypes.IngressModeWebhook},
		DeliveryModes: []msgtypes.DeliveryMode{msgtypes.DeliveryModeText},
	}
}

// NormalizeIngress maps the received uma message of the turn in context onto
// an aps NormalizedMessage. The ingress itself carries no payload: the
// provider webhook was verified and recorded by uma before the turn ran.
func (p *provider) NormalizeIngress(ctx context.Context, _ msgtypes.NativeIngress) (*msgtypes.NormalizedMessage, error) {
	st, err := stateFrom(ctx)
	if err != nil {
		return nil, err
	}
	m := st.turn.Message
	sender := p.entry.address(m.Author.External.ID)
	if sender == "" {
		return nil, fmt.Errorf("uma bridge: message %s has no sender address", m.Ref.ID)
	}
	channel := p.entry.address(p.receiving)
	id := string(m.Ref.ID)
	for _, ext := range m.External {
		if ext.ID != "" {
			id = ext.ID
			break
		}
	}
	msg := &msgtypes.NormalizedMessage{
		ID:        id,
		Timestamp: m.OccurredAt.UTC(),
		Platform:  p.platform,
		Sender:    msgtypes.Sender{ID: sender, Name: sender, PlatformHandle: sender, PlatformID: sender},
		Channel: msgtypes.Channel{
			ID: channel, Name: channel, Type: msgtypes.ChannelTypeDirect, PlatformID: channel,
		},
		Text: strings.TrimSpace(bot.Text(m.Content)),
		PlatformMetadata: map[string]any{
			"messenger_name":   p.service,
			"engine":           "uma",
			"uma_adapter":      p.key,
			"uma_message":      string(m.Ref.ID),
			"uma_conversation": string(st.turn.Conversation.ID),
		},
	}
	// Policy first: a refused sender's media is never copied out of the
	// engine.
	if p.validate != nil {
		if err := p.validate(msg); err != nil {
			return nil, err
		}
	}
	if p.mediaDir != "" {
		if msg.Attachments, err = materializeMedia(ctx, st, p.mediaDir, id); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

// DeliverMessage queues the reply on the turn's conversation. The operation
// ID is the turn's, so a repeat of the same delivery (the runtime's own
// retries, or a redelivered event) replays the one reply instead of sending
// again. A uma reply is accepted when queued; the engine's worker sends it.
func (p *provider) DeliverMessage(ctx context.Context, d msgtypes.DeliveryRequest) (*msgtypes.DeliveryReceipt, error) {
	st, err := stateFrom(ctx)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(d.Text)
	if text == "" {
		return nil, fmt.Errorf("uma bridge: only text replies are delivered")
	}
	correlation := st.turn.Event.Correlation
	if correlation.Validate() != nil {
		correlation = "aps-bridge"
	}
	inReplyTo := st.turn.Message.Ref
	acc, err := st.client.Messages().Reply(ctx, model.Reply{
		Command:      model.Command{Call: model.Call{Correlation: correlation}, Operation: st.turn.OperationID("reply")},
		Conversation: st.turn.Conversation,
		InReplyTo:    &inReplyTo,
		Content:      bot.TextContent(text),
	})
	if err != nil {
		return nil, fmt.Errorf("uma reply: %w", err)
	}
	return &msgtypes.DeliveryReceipt{
		Provider:   p.platform,
		DeliveryID: string(acc.Operation.Ref.ID),
		Status:     "accepted",
		ProviderData: map[string]any{
			"engine":    "uma",
			"operation": string(acc.Operation.Ref.ID),
		},
	}, nil
}
