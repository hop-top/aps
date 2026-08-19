package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
	"hop.top/aps/internal/logging"
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
// the target action. When a ConversationStore is attached, the router
// records each routed inbound message as a conversation turn and attaches
// the prior turns of the same session to the action payload.
type MessageRouter struct {
	resolver       RouteResolver
	normalizer     *Normalizer
	executor       ActionExecutor
	history        msgtypes.ConversationStore
	priorTurnLimit int
}

var _ msgtypes.MessageRouter = (*MessageRouter)(nil)

// RouterOption configures a MessageRouter at construction time.
type RouterOption func(*MessageRouter)

// WithConversationStore attaches the store used to persist turns and to
// resolve prior turns for action payloads.
func WithConversationStore(store msgtypes.ConversationStore) RouterOption {
	return func(r *MessageRouter) { r.history = store }
}

// WithPriorTurnLimit sets the default number of prior turns attached to
// each action run. Values <= 0 keep msgtypes.DefaultPriorTurnLimit.
func WithPriorTurnLimit(limit int) RouterOption {
	return func(r *MessageRouter) {
		if limit > 0 {
			r.priorTurnLimit = limit
		}
	}
}

// ExecuteOption tunes a single ExecuteAction / HandleMessage call.
type ExecuteOption func(*executeOptions)

type executeOptions struct {
	priorTurnLimit int
}

// PriorTurnLimit overrides the router default prior-turn bound for one
// call (for example from a per-service option). Values <= 0 are ignored.
func PriorTurnLimit(limit int) ExecuteOption {
	return func(o *executeOptions) {
		if limit > 0 {
			o.priorTurnLimit = limit
		}
	}
}

// ActionPayload is the JSON object written to a routed action's stdin: the
// normalized message fields at the top level, plus the policy conversation
// identity and the bounded prior turns of the same session (oldest first,
// newest last; empty when history is disabled or nothing was recorded).
type ActionPayload struct {
	*msgtypes.NormalizedMessage
	Conversation msgtypes.ConversationState  `json:"conversation"`
	PriorTurns   []msgtypes.ConversationTurn `json:"prior_turns"`
}

// NewMessageRouter creates a MessageRouter with the given RouteResolver and Normalizer.
func NewMessageRouter(resolver RouteResolver, normalizer *Normalizer) *MessageRouter {
	core, err := protocol.NewAPSAdapter()
	var executor ActionExecutor
	if err == nil {
		executor = core
	}
	return NewMessageRouterWithExecutor(resolver, normalizer, executor)
}

// NewMessageRouterWithExecutor creates a MessageRouter with an explicit
// ActionExecutor and optional RouterOptions (history store, prior-turn
// limit).
func NewMessageRouterWithExecutor(resolver RouteResolver, normalizer *Normalizer, executor ActionExecutor, opts ...RouterOption) *MessageRouter {
	router := &MessageRouter{
		resolver:       resolver,
		normalizer:     normalizer,
		executor:       executor,
		priorTurnLimit: msgtypes.DefaultPriorTurnLimit,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(router)
		}
	}
	return router
}

// ConversationStore returns the attached history store, or nil.
func (r *MessageRouter) ConversationStore() msgtypes.ConversationStore {
	return r.history
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

// ExecuteAction invokes the routed profile action with an ActionPayload
// (normalized message JSON plus conversation identity and prior turns) as
// stdin and returns captured stdout for platform replies. RunInput.ThreadID
// carries the policy session key so run state can be correlated to the
// multi-turn thread. The inbound message is recorded as a conversation turn
// before the action runs; history failures are logged and never block the
// action.
func (r *MessageRouter) ExecuteAction(ctx context.Context, profileID, actionName string, msg *msgtypes.NormalizedMessage, opts ...ExecuteOption) (*ActionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.executor == nil {
		return nil, fmt.Errorf("message action executor is not configured")
	}
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	start := time.Now()
	state := msg.ConversationState()
	prior := r.priorTurns(ctx, state, r.resolveExecuteOptions(opts))
	r.recordInboundTurn(ctx, msg, profileID, actionName)

	payload, err := json.Marshal(ActionPayload{
		NormalizedMessage: msg,
		Conversation:      state,
		PriorTurns:        prior,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode message payload: %w", err)
	}
	runState, err := r.executor.ExecuteRun(ctx, protocol.RunInput{
		ProfileID: profileID,
		ActionID:  actionName,
		Payload:   payload,
		ThreadID:  state.SessionID,
	}, nil)
	if err != nil {
		return nil, err
	}
	if runState == nil {
		return nil, fmt.Errorf("action executor returned nil run state")
	}

	elapsed := time.Since(start)
	status := "success"
	output := runState.Output
	if runState.Status != protocol.RunStatusCompleted {
		status = "failed"
		if output == "" {
			output = runState.Error
		}
	}

	return &ActionResult{
		Status:        status,
		Output:        output,
		OutputData:    runState,
		ExecutionTime: elapsed,
	}, nil
}

func (r *MessageRouter) resolveExecuteOptions(opts []ExecuteOption) executeOptions {
	resolved := executeOptions{priorTurnLimit: r.priorTurnLimit}
	if resolved.priorTurnLimit <= 0 {
		resolved.priorTurnLimit = msgtypes.DefaultPriorTurnLimit
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&resolved)
		}
	}
	return resolved
}

// priorTurns resolves the bounded, session-scoped prior turns for the
// payload. It always returns a non-nil slice so prior_turns serializes as
// an array.
func (r *MessageRouter) priorTurns(ctx context.Context, state msgtypes.ConversationState, opts executeOptions) []msgtypes.ConversationTurn {
	if r.history == nil {
		return []msgtypes.ConversationTurn{}
	}
	turns, err := r.history.RecentTurns(ctx, msgtypes.ConversationQuery{
		ConversationID: state.ConversationID,
		SessionID:      state.SessionID,
		Limit:          opts.priorTurnLimit,
	})
	if err != nil {
		logging.GetLogger().Error("messenger history: failed to load prior turns", err,
			"conversation_id", state.ConversationID,
			"session_id", state.SessionID,
		)
		return []msgtypes.ConversationTurn{}
	}
	if turns == nil {
		turns = []msgtypes.ConversationTurn{}
	}
	return turns
}

// recordInboundTurn persists the routed inbound message as a turn. Store
// failures are logged; message handling continues.
func (r *MessageRouter) recordInboundTurn(ctx context.Context, msg *msgtypes.NormalizedMessage, profileID, actionName string) {
	if r.history == nil || msg == nil {
		return
	}
	turn := msgtypes.NewInboundTurn(msg, "", profileID, actionName)
	if _, err := r.history.AppendTurn(ctx, turn); err != nil {
		logging.GetLogger().Error("messenger history: failed to record inbound turn", err,
			"conversation_id", turn.ConversationID,
			"message_id", msg.ID,
		)
	}
}

// recordOutboundTurn persists a delivered reply as a turn of the
// conversation msg belongs to. Blank replies are not turns.
func (r *MessageRouter) recordOutboundTurn(ctx context.Context, msg *msgtypes.NormalizedMessage, text, profileID, actionName string) {
	if r.history == nil || msg == nil || strings.TrimSpace(text) == "" {
		return
	}
	turn := msgtypes.NewOutboundTurn(msg, text, "", profileID, actionName)
	if _, err := r.history.AppendTurn(ctx, turn); err != nil {
		logging.GetLogger().Error("messenger history: failed to record outbound turn", err,
			"conversation_id", turn.ConversationID,
			"message_id", msg.ID,
		)
	}
}

// HandleMessage is the full message processing pipeline: normalize (already
// done by caller), route the message to a profile action, execute the action,
// and return the result. If routing finds no channel mapping, it returns an
// ActionResult with status "failed" rather than an error, so the caller can
// respond to the platform appropriately.
func (r *MessageRouter) HandleMessage(ctx context.Context, msg *msgtypes.NormalizedMessage, opts ...ExecuteOption) (*ActionResult, error) {
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

	actionResult, err := r.ExecuteAction(ctx, routeResult.ProfileID, routeResult.ActionName, msg, opts...)
	if err != nil {
		return &ActionResult{
			Status: "failed",
			Output: fmt.Sprintf("action execution failed: %v", err),
			Error:  err,
		}, err
	}

	return actionResult, nil
}
