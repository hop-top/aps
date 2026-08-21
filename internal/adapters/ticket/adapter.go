// Package ticket mounts ticket service webhooks (email, Jira, Linear, GitLab)
// into aps serve: per-service routes, shared service auth, payload
// normalization, sender-aware routing, and profile action execution.
package ticket

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/protocol"
	"hop.top/kit/go/ai/ext"
)

// Adapter mounts the per-service ticket webhook route into aps serve. Every
// ticket service (type: ticket) is reachable at core.ServiceWebhookPath, i.e.
// POST /services/<id>/ticket/<adapter>; there is no service-less catch-all.
type Adapter struct {
	status string
	mu     sync.RWMutex
}

var (
	_ protocol.ProtocolServer      = (*Adapter)(nil)
	_ protocol.HTTPProtocolAdapter = (*Adapter)(nil)
	_ ext.Extension                = (*Adapter)(nil)
)

// NewAdapter returns a stopped ticket adapter.
func NewAdapter() *Adapter {
	return &Adapter{status: "stopped"}
}

// Meta implements ext.Extension.
func (a *Adapter) Meta() ext.Metadata {
	return ext.Metadata{
		Name:        ServiceType,
		Version:     "v1",
		Description: "APS ticket service webhook adapter",
	}
}

// Capabilities implements ext.Extension.
func (a *Adapter) Capabilities() ext.Capability {
	return ext.CapRegistry
}

// Init implements ext.Extension.
func (a *Adapter) Init(ctx context.Context) error {
	return a.Start(ctx, nil)
}

// Close implements ext.Extension.
func (a *Adapter) Close() error {
	return a.Stop()
}

// Name implements protocol.ProtocolServer.
func (a *Adapter) Name() string {
	return ServiceType
}

// Start implements protocol.ProtocolServer.
func (a *Adapter) Start(_ context.Context, _ interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = "running"
	return nil
}

// Stop implements protocol.ProtocolServer.
func (a *Adapter) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = "stopped"
	return nil
}

// Status implements protocol.ProtocolServer.
func (a *Adapter) Status() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// RegisterRoutes mounts POST /services/{service}/ticket/{adapter}. The
// pattern must stay in step with core.ServiceWebhookPath.
func (a *Adapter) RegisterRoutes(mux *http.ServeMux, apsCore protocol.APSCore) error {
	normalizer := NewNormalizer()
	var executor ActionExecutor
	if apsCore != nil {
		executor = apsCore
	}
	router := NewRouterWithExecutor(&serviceRouteResolver{}, executor)
	mux.Handle(ServiceWebhookPattern, NewHandler(router, normalizer, nil))
	return nil
}

// ServiceWebhookPattern is the ServeMux pattern for ticket service webhooks.
const ServiceWebhookPattern = "POST /services/{service}/ticket/{adapter}"

// serviceRouteResolver dispatches a ticket through the service that received
// it: the service route table (author-keyed) when declared, else option
// default_action. Tickets without a service are unrouted.
type serviceRouteResolver struct{}

var (
	_ RouteResolver       = (*serviceRouteResolver)(nil)
	_ SenderRouteResolver = (*serviceRouteResolver)(nil)
)

// ResolveTicketRoute keeps the route-key contract for callers without a
// ticket in hand. Service dispatch needs the ticket (service ID and author),
// so this always reports unrouted.
func (r *serviceRouteResolver) ResolveTicketRoute(adapter, routeKey string) (string, error) {
	return "", fmt.Errorf("%w: %s route %s needs the ticket to resolve a service", ErrUnrouted, adapter, routeKey)
}

func (r *serviceRouteResolver) ResolveRouteForTicket(_ context.Context, t *NormalizedTicket) (string, error) {
	if t == nil {
		return "", fmt.Errorf("ticket is nil")
	}
	serviceID := ticketServiceID(t)
	if serviceID == "" {
		return "", fmt.Errorf("%w: ticket %s carries no service", ErrUnrouted, t.ID)
	}
	service, err := core.LoadService(serviceID)
	if err != nil {
		return "", fmt.Errorf("load service %s: %w", serviceID, err)
	}
	if service.Type != ServiceType {
		return "", fmt.Errorf("service %s has type %q, not ticket", serviceID, service.Type)
	}
	if core.HasRouteTable(service) {
		return routeTableRoute(service, t)
	}
	return defaultActionRoute(service, t)
}

// routeTableRoute compiles the service route table, resolves the author, and
// stamps the decision on the ticket. Load failures fail closed.
func routeTableRoute(service *core.ServiceConfig, t *NormalizedTicket) (string, error) {
	table, err := core.LoadServiceRouteTable(service)
	if err != nil {
		return "", fmt.Errorf("service %s: %w", service.ID, err)
	}
	decision := table.Resolve(ticketSenderKey(t))
	mapping := decision.Mapping()
	if mapping == "" {
		return "", fmt.Errorf("%w: service %s route table produced no target for ticket %s", ErrUnrouted, service.ID, t.ID)
	}
	if t.Metadata == nil {
		t.Metadata = map[string]any{}
	}
	t.Metadata[MetadataRouting] = decision.Metadata()
	return mapping, nil
}

// defaultActionRoute expands option default_action into a profile=action
// mapping, defaulting the profile to the service profile.
func defaultActionRoute(service *core.ServiceConfig, t *NormalizedTicket) (string, error) {
	defaultAction := strings.TrimSpace(service.Options[core.OptionDefaultAction])
	if defaultAction == "" {
		return "", fmt.Errorf("%w: service %s has no %s for ticket %s", ErrUnrouted, service.ID, core.OptionDefaultAction, t.ID)
	}
	if strings.Contains(defaultAction, "=") || strings.Contains(defaultAction, ":") {
		return defaultAction, nil
	}
	if strings.TrimSpace(service.Profile) == "" {
		return "", fmt.Errorf("service %s has no profile to run action %q", service.ID, defaultAction)
	}
	return service.Profile + "=" + defaultAction, nil
}

// ticketSenderKey is the author identity a route table matches on: the email
// address when known, else the platform ID, else the handle.
func ticketSenderKey(t *NormalizedTicket) string {
	for _, candidate := range []string{t.Author.Email, t.Author.ID, t.Author.Handle} {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}
