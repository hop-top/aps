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

type serviceRouteResolver struct {
	base RouteResolver
}

func (r *serviceRouteResolver) ResolveChannelRoute(messengerName, channelID string) (*coremessenger.ProfileMessengerLink, string, error) {
	link, action, err := r.base.ResolveChannelRoute(messengerName, channelID)
	if err == nil {
		return link, action, nil
	}
	if !coremessenger.IsUnknownChannel(err) {
		return nil, "", err
	}

	service, loadErr := core.LoadService(messengerName)
	if loadErr != nil || service.Type != "message" || service.Profile == "" {
		return nil, "", err
	}
	defaultAction := strings.TrimSpace(service.Options["default_action"])
	if defaultAction == "" {
		return nil, "", err
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
