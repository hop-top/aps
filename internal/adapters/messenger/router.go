package messenger

import (
	"context"
	"fmt"
	"time"

	msgtypes "oss-aps-cli/internal/core/messenger"
)

// RouteResolver resolves a messenger channel to its linked profile and action.
// This interface decouples the router from the concrete Manager implementation,
// which lives in the core layer and may still be under construction.
type RouteResolver interface {
	// ResolveChannelRoute returns the ProfileMessengerLink and target action
	// mapping string for the given messenger name and channel ID. If no route
	// is found, it returns an error satisfying msgtypes.IsUnknownChannel.
	ResolveChannelRoute(messengerName, channelID string) (*msgtypes.ProfileMessengerLink, string, error)
}

// RoutingResult captures the outcome of routing a message to a profile action.
type RoutingResult struct {
	MessageID  string `json:"message_id"`
	ProfileID  string `json:"profile_id"`
	ActionName string `json:"action_name"`
	Route      string `json:"route"`  // "profile=action" canonical format
	Status     string `json:"status"` // "routed", "unknown_channel", "no_action", "error"
	Error      error  `json:"error,omitempty"`
}

// ActionResult captures the outcome of executing a routed action.
type ActionResult struct {
	Status        string        `json:"status"` // "success", "failed", "timeout"
	Output        string        `json:"output"`
	OutputData    any           `json:"output_data,omitempty"`
	ExecutionTime time.Duration `json:"execution_time"`
	Error         error         `json:"error,omitempty"`
}

// MessageRouter routes normalized messages to profile actions by resolving
// channel-to-profile mappings through the RouteResolver, then executing
// the target action.
type MessageRouter struct {
	resolver   RouteResolver
	normalizer *Normalizer
}

// NewMessageRouter creates a MessageRouter with the given RouteResolver and Normalizer.
func NewMessageRouter(resolver RouteResolver, normalizer *Normalizer) *MessageRouter {
	return &MessageRouter{
		resolver:   resolver,
		normalizer: normalizer,
	}
}

// Route resolves the target profile and action for a normalized message.
// It uses the message's Platform as the messenger name and Channel.ID as the
// channel identifier to look up the route through the resolver.
func (r *MessageRouter) Route(ctx context.Context, msg *msgtypes.NormalizedMessage) (*RoutingResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	result := &RoutingResult{
		MessageID: msg.ID,
	}

	link, actionMapping, err := r.resolver.ResolveChannelRoute(msg.Platform, msg.Channel.ID)
	if err != nil {
		if msgtypes.IsUnknownChannel(err) {
			result.Status = "unknown_channel"
			result.Error = err
			return result, nil
		}
		result.Status = "error"
		result.Error = msgtypes.ErrRoutingFailed(msg.ID, err)
		return result, result.Error
	}

	if actionMapping == "" {
		result.ProfileID = link.ProfileID
		result.Status = "no_action"
		result.Error = msgtypes.ErrActionNotFound(link.ProfileID, "(none)")
		return result, nil
	}

	// Parse the "profile=action" mapping string into its components.
	target, err := msgtypes.ParseTargetAction(actionMapping)
	if err != nil {
		result.Status = "error"
		result.Error = msgtypes.ErrRoutingFailed(msg.ID, err)
		return result, result.Error
	}

	result.ProfileID = target.ProfileID
	result.ActionName = target.ActionName
	result.Route = target.String()
	result.Status = "routed"

	// Stamp the message with the resolved profile ID so downstream
	// handlers know which profile context to use.
	msg.ProfileID = target.ProfileID

	return result, nil
}

// ExecuteAction invokes the routed action for a message. This is currently a
// placeholder that will integrate with the A2A Protocol Server once available.
// It records wall-clock execution time and returns a successful ActionResult.
func (r *MessageRouter) ExecuteAction(ctx context.Context, profileID, actionName string, msg *msgtypes.NormalizedMessage) (*ActionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	start := time.Now()

	// Placeholder: the real implementation will dispatch to the A2A protocol
	// server, invoking the profile's registered action with the normalized
	// message as input. For now we record what would happen.
	output := fmt.Sprintf("action %q dispatched to profile %q (message %s, platform %s, channel %s)",
		actionName, profileID, msg.ID, msg.Platform, msg.Channel.ID)

	elapsed := time.Since(start)

	return &ActionResult{
		Status:        "success",
		Output:        output,
		ExecutionTime: elapsed,
	}, nil
}

// HandleMessage is the full message processing pipeline: normalize (already
// done by caller), route the message to a profile action, execute the action,
// and return the result. If routing finds no channel mapping, it returns an
// ActionResult with status "failed" rather than an error, so the caller can
// respond to the platform appropriately.
func (r *MessageRouter) HandleMessage(ctx context.Context, msg *msgtypes.NormalizedMessage) (*ActionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	routeResult, err := r.Route(ctx, msg)
	if err != nil {
		return &ActionResult{
			Status: "failed",
			Output: fmt.Sprintf("routing failed: %v", err),
			Error:  err,
		}, err
	}

	if routeResult.Status != "routed" {
		return &ActionResult{
			Status: "failed",
			Output: fmt.Sprintf("message not routed: %s", routeResult.Status),
			Error:  routeResult.Error,
		}, nil
	}

	actionResult, err := r.ExecuteAction(ctx, routeResult.ProfileID, routeResult.ActionName, msg)
	if err != nil {
		return &ActionResult{
			Status: "failed",
			Output: fmt.Sprintf("action execution failed: %v", err),
			Error:  err,
		}, err
	}

	return actionResult, nil
}
