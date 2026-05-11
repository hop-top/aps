package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
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

type ActionExecutor interface {
	ExecuteRun(ctx context.Context, input protocol.RunInput, stream protocol.StreamWriter) (*protocol.RunState, error)
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
	executor   ActionExecutor
}

var _ msgtypes.MessageRouter = (*MessageRouter)(nil)

// NewMessageRouter creates a MessageRouter with the given RouteResolver and Normalizer.
func NewMessageRouter(resolver RouteResolver, normalizer *Normalizer) *MessageRouter {
	core, err := protocol.NewAPSAdapter()
	var executor ActionExecutor
	if err == nil {
		executor = core
	}
	return NewMessageRouterWithExecutor(resolver, normalizer, executor)
}

func NewMessageRouterWithExecutor(resolver RouteResolver, normalizer *Normalizer, executor ActionExecutor) *MessageRouter {
	return &MessageRouter{
		resolver:   resolver,
		normalizer: normalizer,
		executor:   executor,
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

	messengerName := msg.Platform
	if msg.PlatformMetadata != nil {
		if configuredName, ok := msg.PlatformMetadata["messenger_name"].(string); ok && configuredName != "" {
			messengerName = configuredName
		}
	}

	link, actionMapping, err := r.resolver.ResolveChannelRoute(messengerName, msg.Channel.ID)
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

func (r *MessageRouter) ResolveMessageRoute(ctx context.Context, msg *msgtypes.NormalizedMessage) (msgtypes.ExecutionRoute, error) {
	result, err := r.Route(ctx, msg)
	if err != nil {
		return msgtypes.ExecutionRoute{}, err
	}
	if result == nil {
		return msgtypes.ExecutionRoute{}, fmt.Errorf("message route result is nil")
	}
	if result.Status != "routed" {
		if result.Error != nil {
			return msgtypes.ExecutionRoute{}, result.Error
		}
		return msgtypes.ExecutionRoute{}, fmt.Errorf("message not routed: %s", result.Status)
	}
	return msgtypes.ExecutionRoute{
		ProfileID:  result.ProfileID,
		ActionName: result.ActionName,
		Mapping:    result.Route,
	}, nil
}

// ExecuteAction invokes the routed profile action with the normalized message
// JSON as stdin and returns captured stdout for platform replies.
func (r *MessageRouter) ExecuteAction(ctx context.Context, profileID, actionName string, msg *msgtypes.NormalizedMessage) (*ActionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.executor == nil {
		return nil, fmt.Errorf("message action executor is not configured")
	}

	start := time.Now()

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message payload: %w", err)
	}
	state, err := r.executor.ExecuteRun(ctx, protocol.RunInput{
		ProfileID: profileID,
		ActionID:  actionName,
		Payload:   payload,
	}, nil)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("action executor returned nil run state")
	}

	elapsed := time.Since(start)
	status := "success"
	output := state.Output
	if state.Status != protocol.RunStatusCompleted {
		status = "failed"
		if output == "" {
			output = state.Error
		}
	}

	return &ActionResult{
		Status:        status,
		Output:        output,
		OutputData:    state,
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
