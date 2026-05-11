package ticket

import (
	"context"
	"fmt"
	"time"
)

type Router struct {
	resolver RouteResolver
}

func NewRouter(resolver RouteResolver) *Router {
	return &Router{resolver: resolver}
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

	keys := []string{RouteKey(t), t.ChannelID}
	for _, key := range keys {
		target, err := r.resolver.ResolveTicketRoute(t.Adapter, key)
		if err != nil || target == "" {
			continue
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

	err := fmt.Errorf("no route for %s ticket %s", t.Adapter, t.ID)
	return &RoutingResult{TicketID: t.ID, Status: "unrouted", Error: err}, nil
}

func (r *Router) ExecuteAction(ctx context.Context, profileID, actionName string, t *NormalizedTicket) (*ActionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	start := time.Now()
	output := fmt.Sprintf("action %q dispatched to profile %q (ticket %s, adapter %s, route %s)",
		actionName, profileID, t.ID, t.Adapter, RouteKey(t))
	return &ActionResult{
		Status:        "success",
		Output:        output,
		ExecutionTime: time.Since(start),
	}, nil
}

func (r *Router) HandleTicket(ctx context.Context, t *NormalizedTicket) (*ActionResult, error) {
	route, err := r.Route(ctx, t)
	if err != nil {
		return &ActionResult{Status: "failed", Output: fmt.Sprintf("routing failed: %v", err), Error: err}, err
	}
	if route.Status != "routed" {
		return &ActionResult{Status: "failed", Output: fmt.Sprintf("ticket not routed: %s", route.Status), Error: route.Error}, nil
	}
	return r.ExecuteAction(ctx, route.ProfileID, route.ActionName, t)
}
