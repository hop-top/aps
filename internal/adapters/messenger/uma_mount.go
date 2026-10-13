package messenger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"hop.top/aps/internal/adapters/umabridge"
	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/logging"
)

// umaStopTimeout bounds how long closing the adapter waits for bridges to
// wind down.
const umaStopTimeout = 10 * time.Second

var errUmaStopped = errors.New("uma bridges are stopped")

// umaMounts hosts one umabridge.Bridge per uma message service and serves
// each one's provider webhook at the service's webhook route.
//
// Services are loaded per request, as the native providers' are, so a bridge
// is built on the first request for its service, not when routes register: a
// service added while `aps serve` runs is served without a restart. A bridge
// is rebuilt when the service's provider configuration (profile, options,
// env bindings) changes; its in-memory engine state goes with it, so events
// pending at that moment are redelivered by the provider.
type umaMounts struct {
	router     *MessageRouter
	chatRunner ChatTurnRunner
	options    umabridge.Options

	// ctx outlives any request: it owns every bridge's engine and worker.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	attempts umaAttempts

	mu      sync.Mutex
	closed  bool
	bridges map[string]*umaInstance
}

// umaInstance is one running bridge.
type umaInstance struct {
	fingerprint string
	bridge      *umabridge.Bridge
	webhook     http.Handler
	cancel      context.CancelFunc
	done        chan struct{}
}

func newUmaMounts(router *MessageRouter, options umabridge.Options) *umaMounts {
	ctx, cancel := context.WithCancel(context.Background())
	return &umaMounts{
		router:  router,
		options: options,
		ctx:     ctx,
		cancel:  cancel,
		bridges: map[string]*umaInstance{},
	}
}

// serve answers one webhook request for a uma service by handing it, byte
// for byte, to the service's bridge: uma verifies the provider signature over
// the raw body and answers the provider. No aps check runs first or after;
// aps's own policy (allowlists, routing, reply policy) applies to the
// message the bridge normalizes.
func (m *umaMounts) serve(w http.ResponseWriter, r *http.Request, service *core.ServiceConfig) {
	webhook, err := m.webhook(service)
	if err != nil {
		logging.GetLogger().Error("uma service unavailable", err, "service", service.ID)
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("uma service %q is not available", service.ID))
		return
	}
	webhook.ServeHTTP(w, r)
}

// webhook returns the provider webhook of the service's bridge, building or
// rebuilding the bridge when there is none or its configuration changed.
func (m *umaMounts) webhook(service *core.ServiceConfig) (http.Handler, error) {
	fingerprint := umaFingerprint(service)

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, errUmaStopped
	}
	if inst := m.bridges[service.ID]; inst != nil {
		if inst.fingerprint == fingerprint {
			return inst.webhook, nil
		}
		delete(m.bridges, service.ID)
		m.retire(inst)
	}
	inst, err := m.start(service, fingerprint)
	if err != nil {
		return nil, err
	}
	m.bridges[service.ID] = inst
	return inst.webhook, nil
}

func (m *umaMounts) start(service *core.ServiceConfig, fingerprint string) (*umaInstance, error) {
	cfg := umabridge.Config{
		Service:  service,
		Router:   m.router,
		Executor: &serviceRuntimeExecutor{router: m.router, service: service, chatRunner: m.chatRunner},
		History:  m.router.ConversationStore(),
		Hooks: msgtypes.RuntimeHooks{
			OnDeliveryAttempt: func(_ context.Context, attempt msgtypes.DeliveryAttempt) error {
				m.attempts.add(attempt)
				return nil
			},
		},
		OnResult: func(result *msgtypes.RuntimeResult, err error) {
			m.record(service.ID, result, err)
		},
		Options:  m.options,
		MediaDir: umaMediaDir(service.ID),
	}
	bridge, err := umabridge.New(m.ctx, cfg)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(m.ctx)
	inst := &umaInstance{
		fingerprint: fingerprint,
		bridge:      bridge,
		webhook:     bridge.Webhook(),
		cancel:      cancel,
		done:        make(chan struct{}),
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(inst.done)
		if err := bridge.Run(runCtx); err != nil {
			logging.GetLogger().Error("uma bridge stopped", err, "service", service.ID)
		}
	}()
	return inst, nil
}

// retire stops a bridge in the background: its handler answers 503 from the
// moment it is closed, and the provider retries.
func (m *umaMounts) retire(inst *umaInstance) {
	inst.cancel()
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		<-inst.done
		closeCtx, cancel := context.WithTimeout(context.Background(), umaStopTimeout)
		defer cancel()
		if err := inst.bridge.Close(closeCtx); err != nil {
			logging.GetLogger().Error("uma bridge close", err)
		}
	}()
}

// close stops every bridge and refuses new ones. It is idempotent.
func (m *umaMounts) close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	bridges := m.bridges
	m.bridges = map[string]*umaInstance{}
	m.mu.Unlock()

	for _, inst := range bridges {
		m.retire(inst)
	}
	m.cancel()
	finished := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(umaStopTimeout):
		logging.GetLogger().Warn("uma bridges did not stop in time")
	}
}

// umaFingerprint identifies the part of a service a bridge is built from.
// Event bookkeeping the runtime writes back to the service (last inbound and
// outbound) and routing, which is resolved per message, are not part of it.
func umaFingerprint(service *core.ServiceConfig) string {
	view := struct {
		Type, Adapter, Profile string
		Env, Options           map[string]string
		Delivery               *core.ServiceDelivery
	}{service.Type, service.Adapter, service.Profile, service.Env, service.Options, service.Delivery}
	data, _ := json.Marshal(view)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// umaMediaDir is where a service's inbound media is staged: under the aps
// data dir, owner-only. Empty (summary-only media) when the data dir cannot
// be resolved.
func umaMediaDir(serviceID string) string {
	dir, err := core.GetDataDir()
	if err != nil {
		logging.GetLogger().Warn("uma media dir unavailable; inbound media is summary-only", "service", serviceID, "error", err)
		return ""
	}
	return filepath.Join(dir, "messenger", "uma", serviceID)
}

// umaAttempts carries delivery attempts from the runtime hook to the result
// callback of the same message: the runtime is shared by concurrent turns, so
// attempts are keyed by message id, not collected per call.
type umaAttempts struct {
	mu   sync.Mutex
	byID map[string][]msgtypes.DeliveryAttempt
}

func (a *umaAttempts) add(attempt msgtypes.DeliveryAttempt) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.byID == nil {
		a.byID = map[string][]msgtypes.DeliveryAttempt{}
	}
	a.byID[attempt.MessageID] = append(a.byID[attempt.MessageID], attempt)
}

func (a *umaAttempts) take(messageID string) []msgtypes.DeliveryAttempt {
	a.mu.Lock()
	defer a.mu.Unlock()
	attempts := a.byID[messageID]
	delete(a.byID, messageID)
	return attempts
}

// record writes the service bookkeeping the native providers write after a
// message: inbound and outbound events, delivery attempts, failures logged.
// It runs once per handled message, on the bridge's worker.
func (m *umaMounts) record(serviceID string, result *msgtypes.RuntimeResult, err error) {
	if err != nil {
		var rerr *msgtypes.RuntimeError
		if errors.As(err, &rerr) {
			_ = recordServiceDeliveryAttempts(serviceID, m.attempts.take(rerr.Message), nil)
		}
		if msgtypes.IsSenderNotAllowed(err) {
			logging.GetLogger().Warn("uma message refused", "service", serviceID, "error", err)
			return
		}
		logging.GetLogger().Error("uma message failed", err, "service", serviceID)
		return
	}
	if result == nil || result.Message == nil {
		return
	}
	msg := result.Message
	_ = m.attempts.take(msg.ID)
	if result.Result != nil && result.Result.Status == umabridge.StatusDuplicate {
		// An event already answered, redelivered after a restart: nothing
		// ran and nothing was sent, so the service's last inbound and
		// outbound events stand.
		return
	}
	_ = core.RecordServiceInboundEvent(serviceID, core.ServiceEventMeta{
		MessageID: msg.ID,
		Platform:  msg.Platform,
		ChannelID: msg.Channel.ID,
		SenderID:  msg.Sender.ID,
		Status:    "received",
	})
	switch {
	case result.Delivery != nil:
		_ = recordServiceDeliveryAttempts(serviceID, result.DeliveryAttempts, msg)
		if len(result.DeliveryAttempts) == 0 {
			_ = core.RecordServiceOutboundEvent(serviceID, core.ServiceEventMeta{
				MessageID: msg.ID,
				Platform:  msg.Platform,
				ChannelID: msg.Channel.ID,
				SenderID:  msg.Sender.ID,
				Status:    result.Delivery.Status,
			})
		}
	case result.Result != nil:
		_ = recordServiceExecutionEvent(serviceID, msg, result.Result)
	}
}
