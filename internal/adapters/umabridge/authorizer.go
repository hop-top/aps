package umabridge

import (
	"context"

	"hop.top/uma/model"
	"hop.top/uma/policy"
)

// The bridge acts as two actors in one tenant. The service actor provisions
// the channel and inbox and runs the receiver and delivery worker; the bot
// actor reads the subscription and replies.
const (
	tenant       model.TenantID = "aps"
	actorService model.ActorID  = "aps-bridge"
	actorBot     model.ActorID  = "aps-bridge-bot"
)

// serviceActions and botActions are everything the bridge's actors may do.
// Anything else, and any other actor or tenant, is denied.
var (
	serviceActions = actionSet(
		model.ActionChannelConfigure, model.ActionChannelRead, model.ActionChannelReceive,
		model.ActionInboxConfigure, model.ActionInboxRead,
		model.ActionAttachmentPrepare, model.ActionAttachmentUpload,
	)
	botActions = actionSet(
		model.ActionSubscriptionOpen, model.ActionSubscriptionPoll, model.ActionSubscriptionAck, model.ActionSubscriptionClose,
		model.ActionMessageRead, model.ActionMessageReply,
		model.ActionConversationRead, model.ActionInboxRead,
		model.ActionOperationRead,
		model.ActionAttachmentRead,
	)
)

func actionSet(actions ...string) map[string]bool {
	out := make(map[string]bool, len(actions))
	for _, a := range actions {
		out[a] = true
	}
	return out
}

// authorizer admits the bridge's own actors for the actions above and
// nothing else; it carries no per-resource rules because the engine holds
// one tenant with one channel.
type authorizer struct{}

func (authorizer) Authorize(_ context.Context, a policy.Authorization) (policy.Decision, error) {
	allow := func(reason string) (policy.Decision, error) {
		return policy.Decision{Allow: true, ReasonCode: reason}, nil
	}
	deny := func(reason string) (policy.Decision, error) {
		return policy.Decision{Allow: false, ReasonCode: reason}, nil
	}
	if a.Principal.Tenant != tenant {
		return deny("aps.tenant")
	}
	switch a.Principal.Actor {
	case actorService:
		if serviceActions[a.Action] {
			return allow("aps.service")
		}
	case actorBot:
		if botActions[a.Action] {
			return allow("aps.bot")
		}
	}
	return deny("aps.action")
}
