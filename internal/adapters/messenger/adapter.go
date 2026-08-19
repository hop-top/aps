package messenger

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
	"hop.top/kit/go/ai/ext"
)

type Adapter struct {
	status  string
	mu      sync.RWMutex
	history *coremessenger.LazyConversationStore
}

var _ protocol.ProtocolServer = (*Adapter)(nil)
var _ protocol.HTTPProtocolAdapter = (*Adapter)(nil)
var _ ext.Extension = (*Adapter)(nil)

func NewAdapter() *Adapter {
	return &Adapter{status: "stopped"}
}

func (a *Adapter) Meta() ext.Metadata {
	return ext.Metadata{
		Name:        "messenger",
		Version:     "v1",
		Description: "APS message service webhook adapter",
	}
}

func (a *Adapter) Capabilities() ext.Capability {
	return ext.CapRegistry
}

func (a *Adapter) Init(ctx context.Context) error {
	return a.Start(ctx, nil)
}

func (a *Adapter) Close() error {
	return a.Stop()
}

func (a *Adapter) Name() string {
	return "messenger"
}

func (a *Adapter) Start(ctx context.Context, config interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = "running"
	return nil
}

func (a *Adapter) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = "stopped"
	if a.history != nil {
		if err := a.history.Close(); err != nil {
			return fmt.Errorf("close messenger conversation store: %w", err)
		}
	}
	return nil
}

func (a *Adapter) Status() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *Adapter) RegisterRoutes(mux *http.ServeMux, apsCore protocol.APSCore) error {
	normalizer := NewNormalizer()
	router := NewMessageRouterWithExecutor(&serviceRouteResolver{base: coremessenger.NewManager()}, normalizer, apsCore,
		WithConversationStore(a.conversationStore()))
	handler := NewHandler(router, normalizer, nil)

	mux.Handle("POST /messengers/{platform}/webhook", handler)
	mux.HandleFunc("POST /services/{service}/webhook", func(w http.ResponseWriter, r *http.Request) {
		serviceID := r.PathValue("service")
		service, err := core.LoadService(serviceID)
		if err != nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("service %q not found", serviceID))
			return
		}
		if service.Type != "message" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q has type %q, not message", serviceID, service.Type))
			return
		}
		if strings.TrimSpace(service.Adapter) == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q has no message adapter", serviceID))
			return
		}
		handler.ServeServiceWebhook(w, r, service.ID, service.Adapter)
	})
	return nil
}

// conversationStore returns the adapter-owned thread history store. It opens
// lazily under the APS data dir on the first recorded turn, so registering
// routes never touches disk; open failures are logged per call by the
// router and never block message handling.
func (a *Adapter) conversationStore() coremessenger.ConversationStore {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.history == nil {
		a.history = coremessenger.NewLazyConversationStore(func() (coremessenger.ConversationStore, error) {
			return coremessenger.OpenDefaultConversationStore(coremessenger.ConversationStoreOptions{})
		})
	}
	return a.history
}

// serviceRouteResolver layers message-service dispatch on top of the legacy
// link store: explicit channel mappings win, then the service's route table
// (sender-keyed) when declared, else the service default_action.
type serviceRouteResolver struct {
	base RouteResolver
}

var (
	_ RouteResolver        = (*serviceRouteResolver)(nil)
	_ MessageRouteResolver = (*serviceRouteResolver)(nil)
)

// ResolveChannelRoute keeps the channel-only contract for callers without a
// message in hand. Services that declare a route table cannot be resolved
// here because the sender is unknown; they report a routing failure.
func (r *serviceRouteResolver) ResolveChannelRoute(messengerName, channelID string) (*coremessenger.ProfileMessengerLink, string, error) {
	link, action, err := r.base.ResolveChannelRoute(messengerName, channelID)
	if err == nil {
		return link, action, nil
	}
	if !coremessenger.IsUnknownChannel(err) {
		return nil, "", fmt.Errorf("resolve channel %s on %s: %w", channelID, messengerName, err)
	}
	service, ok := loadMessageService(messengerName)
	if !ok {
		return nil, "", err
	}
	if core.HasRouteTable(service) {
		return nil, "", fmt.Errorf("service %s: %w", service.ID,
			coremessenger.ErrRoutingFailed("", fmt.Errorf("sender route table needs the message; channel-only resolution cannot pick a route")))
	}
	return defaultActionRoute(service, err)
}

// ResolveRouteForMessage resolves with the sender in hand: explicit channel
// mapping, then route table (stamping the decision on platform_metadata
// .routing), then default_action.
func (r *serviceRouteResolver) ResolveRouteForMessage(_ context.Context, messengerName string, msg *coremessenger.NormalizedMessage) (*coremessenger.ProfileMessengerLink, string, error) {
	if msg == nil {
		return nil, "", fmt.Errorf("message is nil")
	}
	link, action, err := r.base.ResolveChannelRoute(messengerName, msg.Channel.ID)
	if err == nil {
		return link, action, nil
	}
	if !coremessenger.IsUnknownChannel(err) {
		return nil, "", fmt.Errorf("resolve channel %s on %s: %w", msg.Channel.ID, messengerName, err)
	}
	service, ok := loadMessageService(messengerName)
	if !ok {
		return nil, "", err
	}
	if !core.HasRouteTable(service) {
		return defaultActionRoute(service, err)
	}
	table, loadErr := core.LoadServiceRouteTable(service)
	if loadErr != nil {
		return nil, "", fmt.Errorf("service %s: %w", service.ID, coremessenger.ErrRoutingFailed(msg.ID, loadErr))
	}
	decision := table.Resolve(msg.Sender.ID)
	mapping := decision.Mapping()
	if mapping == "" {
		return nil, "", fmt.Errorf("service %s: %w", service.ID, coremessenger.ErrRoutingFailed(msg.ID, fmt.Errorf("route table produced no target")))
	}
	if msg.PlatformMetadata == nil {
		msg.PlatformMetadata = map[string]any{}
	}
	msg.PlatformMetadata["routing"] = decision.Metadata()
	return &coremessenger.ProfileMessengerLink{
		ProfileID:     decision.Route.Profile,
		MessengerName: service.ID,
		Enabled:       true,
		DefaultAction: mapping,
	}, mapping, nil
}

// loadMessageService returns the persisted message service behind a route
// key, or false when the key is not a message service with a profile.
func loadMessageService(messengerName string) (*core.ServiceConfig, bool) {
	service, err := core.LoadService(messengerName)
	if err != nil || service.Type != "message" || service.Profile == "" {
		return nil, false
	}
	return service, true
}

// defaultActionRoute expands option default_action into a profile=action
// mapping. unknownErr is returned when the option is absent.
func defaultActionRoute(service *core.ServiceConfig, unknownErr error) (*coremessenger.ProfileMessengerLink, string, error) {
	defaultAction := strings.TrimSpace(service.Options[core.OptionDefaultAction])
	if defaultAction == "" {
		return nil, "", unknownErr
	}
	actionMapping := defaultAction
	if !strings.Contains(defaultAction, "=") && !strings.Contains(defaultAction, ":") {
		actionMapping = service.Profile + "=" + defaultAction
	}
	return &coremessenger.ProfileMessengerLink{
		ProfileID:     service.Profile,
		MessengerName: service.ID,
		Enabled:       true,
		DefaultAction: actionMapping,
	}, actionMapping, nil
}
