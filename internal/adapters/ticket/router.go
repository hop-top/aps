package ticket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"hop.top/aps/internal/core/protocol"
)

// ActionExecutor runs a profile action. protocol.APSCore satisfies it; tests
// substitute a recorder.
type ActionExecutor interface {
	ExecuteRun(ctx context.Context, input protocol.RunInput, stream protocol.StreamWriter) (*protocol.RunState, error)
}

// ErrUnrouted is wrapped into RoutingResult.Error / ActionResult.Error when no
// resolver entry, route table row, or default_action claims the ticket.
var ErrUnrouted = errors.New("ticket not routed")

// IsUnrouted reports whether err means the ticket had no route.
func IsUnrouted(err error) bool {
	return errors.Is(err, ErrUnrouted)
}

// ActionPayload is the JSON document written to the routed action's stdin:
// the normalized ticket fields at the top level plus the service that received
// it and the route key the resolver matched on.
type ActionPayload struct {
	*NormalizedTicket
	ServiceID string `json:"service_id,omitempty"`
	RouteKey  string `json:"route_key"`
}

// SenderRouteResolver is an optional RouteResolver extension that sees the
// whole normalized ticket, so a resolver can branch on the receiving service
// and the author (route tables, default_action) instead of the route key
// alone. Router prefers it when the configured resolver implements it.
type SenderRouteResolver interface {
	// ResolveRouteForTicket returns the profile=action mapping for t, or an
	// error satisfying IsUnrouted when nothing claims the ticket.
	ResolveRouteForTicket(ctx context.Context, t *NormalizedTicket) (string, error)
}

type Router struct {
	resolver RouteResolver
	executor ActionExecutor
}

// NewRouter creates a Router backed by the default APS action executor. When
// the executor cannot be constructed the router still routes but
// ExecuteAction fails closed.
func NewRouter(resolver RouteResolver) *Router {
	var executor ActionExecutor
	if apsCore, err := protocol.NewAPSAdapter(); err == nil {
		executor = apsCore
	}
	return NewRouterWithExecutor(resolver, executor)
}

// NewRouterWithExecutor creates a Router with an explicit ActionExecutor.
func NewRouterWithExecutor(resolver RouteResolver, executor ActionExecutor) *Router {
	return &Router{resolver: resolver, executor: executor}
}

func RouteKey(t *NormalizedTicket) string {
	if t == nil {
		return ""
	}
	if t.ThreadID != "" {
		return t.ChannelID + "#" + t.ThreadID
	}
	return t.ChannelID
}

func (r *Router) Route(ctx context.Context, t *NormalizedTicket) (*RoutingResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("ticket is nil")
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}

	target, err := r.resolveTarget(ctx, t)
	if err != nil {
		if IsUnrouted(err) {
			return &RoutingResult{TicketID: t.ID, Status: "unrouted", Error: err}, nil
		}
		return &RoutingResult{TicketID: t.ID, Status: "error", Error: err}, err
	}
	parsed, err := ParseTargetAction(target)
	if err != nil {
		return &RoutingResult{TicketID: t.ID, Status: "error", Error: err}, err
	}
	return &RoutingResult{
		TicketID:   t.ID,
		ProfileID:  parsed.ProfileID,
		ActionName: parsed.ActionName,
		Route:      parsed.String(),
		Status:     "routed",
	}, nil
}

// resolveTarget prefers a ticket-aware resolver and falls back to the route
// key contract (thread key first, then channel).
func (r *Router) resolveTarget(ctx context.Context, t *NormalizedTicket) (string, error) {
	if aware, ok := r.resolver.(SenderRouteResolver); ok {
		target, err := aware.ResolveRouteForTicket(ctx, t)
		if err != nil {
			return "", fmt.Errorf("resolve route for %s ticket %s: %w", t.Adapter, t.ID, err)
		}
		if target == "" {
			return "", fmt.Errorf("%w: no route for %s ticket %s", ErrUnrouted, t.Adapter, t.ID)
		}
		return target, nil
	}
	for _, key := range []string{RouteKey(t), t.ChannelID} {
		target, err := r.resolver.ResolveTicketRoute(t.Adapter, key)
		if err != nil || target == "" {
			continue
		}
		return target, nil
	}
	return "", fmt.Errorf("%w: no route for %s ticket %s", ErrUnrouted, t.Adapter, t.ID)
}

// ExecuteAction runs the routed profile action with the ActionPayload on
// stdin and returns the captured stdout. RunInput.ThreadID carries the route
// key so run state correlates to the ticket thread.
func (r *Router) ExecuteAction(ctx context.Context, profileID, actionName string, t *NormalizedTicket) (*ActionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.executor == nil {
		return nil, fmt.Errorf("ticket action executor is not configured")
	}
	if t == nil {
		return nil, fmt.Errorf("ticket is nil")
	}

	start := time.Now()
	payload, err := json.Marshal(ActionPayload{
		NormalizedTicket: t,
		ServiceID:        ticketServiceID(t),
		RouteKey:         RouteKey(t),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode ticket payload: %w", err)
	}
	runState, err := r.executor.ExecuteRun(ctx, protocol.RunInput{
		ProfileID: profileID,
		ActionID:  actionName,
		Payload:   payload,
		ThreadID:  RouteKey(t),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("execute %s/%s: %w", profileID, actionName, err)
	}
	if runState == nil {
		return nil, fmt.Errorf("action executor returned nil run state")
	}

	status := StatusSuccess
	output := runState.Output
	if runState.Status != protocol.RunStatusCompleted {
		status = StatusFailed
		if output == "" {
			output = runState.Error
		}
	}
	return &ActionResult{
		Status:        status,
		Output:        output,
		OutputData:    runState,
		ExecutionTime: time.Since(start),
	}, nil
}

// HandleTicket routes then executes. Routing errors and executor errors are
// returned alongside a failed ActionResult; an unrouted ticket yields a failed
// result whose Error satisfies IsUnrouted and no error.
func (r *Router) HandleTicket(ctx context.Context, t *NormalizedTicket) (*ActionResult, error) {
	route, err := r.Route(ctx, t)
	if err != nil {
		return &ActionResult{Status: StatusFailed, Output: fmt.Sprintf("routing failed: %v", err), Error: err}, err
	}
	if route.Status != "routed" {
		return &ActionResult{Status: StatusFailed, Output: fmt.Sprintf("ticket not routed: %s", route.Status), Error: route.Error}, nil
	}
	result, err := r.ExecuteAction(ctx, route.ProfileID, route.ActionName, t)
	if err != nil {
		return &ActionResult{Status: StatusFailed, Output: fmt.Sprintf("action execution failed: %v", err), Error: err}, err
	}
	return result, nil
}

// ticketServiceID reads the service stamped on the ticket by the HTTP
// handler; empty for tickets routed outside a service.
func ticketServiceID(t *NormalizedTicket) string {
	if t == nil || t.Metadata == nil {
		return ""
	}
	id, _ := t.Metadata[MetadataServiceID].(string)
	return id
}
