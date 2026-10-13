package umabridge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/uma"
	"hop.top/uma/adapter"
	"hop.top/uma/adapter/web/webhookauth"
	"hop.top/uma/companions/bot"
	"hop.top/uma/companions/bot/host"
	"hop.top/uma/model"
	"hop.top/uma/ports"
	"hop.top/uma/store/memory"
)

// Config composes one Bridge.
type Config struct {
	// Service is the persisted aps service: adapter uma, option uma_adapter,
	// the provider options and the env bindings. Required.
	Service *core.ServiceConfig
	// Router and Executor are the messenger.Runtime stages every received
	// message goes through: route resolution and profile action execution
	// with reply policy. Required.
	Router   msgtypes.MessageRouter
	Executor msgtypes.MessageExecutor
	// History records replies as outbound turns and, read back, keeps an
	// event already answered from being run again after a restart;
	// optional.
	History msgtypes.ConversationStore
	// Hooks are the runtime's error, retry and delivery-attempt hooks;
	// optional.
	Hooks msgtypes.RuntimeHooks
	// OnResult, when set, sees the outcome of every handled message: the
	// runtime result (nil on error) and the error. The host uses it to
	// record service events; the bridge does not touch service state itself.
	OnResult func(*msgtypes.RuntimeResult, error)
	// Store resolves "secret:NAME" binding references; nil means the
	// service's profile secret store.
	Store core.SecretLookup
	// HTTP is the client for provider REST calls; nil is a client with a
	// 30 second timeout.
	HTTP ports.HTTPClient
	// Options are provider factory settings (test seams).
	Options Options
	// Blobs stages inbound media; nil leaves media summary-only.
	Blobs ports.BlobStore
	// Clock, IDs and Waiter are engine ports; nil takes the system ones.
	Clock  ports.Clock
	IDs    ports.IDs
	Waiter ports.Waiter
	// Poll is how long the delivery worker waits between idle rounds.
	Poll time.Duration
	// MediaDir, when set, enables inbound media: attachments are staged under
	// it (Blobs defaults to a file store there) and handed to actions as
	// file:// attachments. Empty leaves media summary-only.
	MediaDir string
	// Host is the uma host settings (the relay callback host allowlist); nil
	// reads them from the global aps configuration.
	Host *core.UmaHostConfig
}

// Bridge is one uma engine serving one aps message service.
type Bridge struct {
	cfg     Config
	engine  uma.Engine
	channel model.ChannelRef
	host    *host.Host
	runtime *msgtypes.Runtime
	webhook http.Handler
}

// New builds the engine, provisions the service's channel and inbox, opens
// the bot's subscription and composes the runtime. It starts nothing: Run
// does. The caller owns the Bridge and closes it.
func New(ctx context.Context, cfg Config) (*Bridge, error) {
	svc := cfg.Service
	switch {
	case svc == nil:
		return nil, errors.New("uma bridge: service is required")
	case svc.Adapter != core.AdapterUma:
		return nil, fmt.Errorf("uma bridge: service %q has adapter %q, not uma", svc.ID, svc.Adapter)
	case cfg.Router == nil || cfg.Executor == nil:
		return nil, errors.New("uma bridge: router and executor are required")
	}
	// The static contract (options, bindings) is the one service add checked.
	if result := core.ValidateServiceConfig(svc); !result.Valid {
		return nil, fmt.Errorf("uma bridge: service %q is invalid: %s", svc.ID, strings.Join(result.Issues, "; "))
	}
	key := strings.TrimSpace(svc.Options[core.OptionUmaAdapter])
	ent, err := lookupEntry(key)
	if err != nil {
		return nil, fmt.Errorf("uma bridge: %w", err)
	}
	spec, _ := core.UmaAdapterSpecFor(key)

	store := cfg.Store
	if store == nil {
		store = core.ProfileSecretLookup(svc.Profile)
	}
	creds := newCredentials(ent.purposes(svc.Options), svc.Env, store)
	binding := func(name string) (string, bool) { return creds.value(name) }
	extension, destination, err := ent.channel(svc.Options, binding)
	if err != nil {
		return nil, fmt.Errorf("uma bridge: %w", err)
	}
	if ent.webhookPurpose != "" {
		if err := checkWebhookSecret(svc.Options, binding); err != nil {
			return nil, fmt.Errorf("uma bridge: service %q: %w", svc.ID, err)
		}
	}
	host := cfg.Host
	if host == nil {
		loaded := core.LoadUmaHostConfig()
		host = &loaded
	}

	clock := cfg.Clock
	if clock == nil {
		clock = ports.SystemClock{}
	}
	ids := cfg.IDs
	if ids == nil {
		ids = ports.NewUUIDv7(clock)
	}
	httpClient := cfg.HTTP
	if httpClient == nil {
		client := &http.Client{Timeout: 30 * time.Second}
		if ent.noRedirect {
			// The credential goes to the configured URL and nowhere else.
			client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		}
		httpClient = client
	}
	blobs := cfg.Blobs
	if blobs == nil && cfg.MediaDir != "" {
		blobs = fileBlobs{dir: filepath.Join(cfg.MediaDir, "blobs")}
	}
	factory := ent.factory(factoryDeps{
		opts: cfg.Options, host: *host,
		// In memory, as the engine's store: replay rejection lasts as long
		// as the bridge.
		replay: webhookauth.NewMemoryReplayStore(0, clock),
	})
	engine, err := uma.New(uma.Config{
		Store:       memory.New(memory.Options{Clock: clock}),
		Authorizer:  authorizer{},
		Adapters:    []adapter.Factory{factory},
		Credentials: creds,
		HTTP:        httpClient,
		Clock:       clock,
		IDs:         ids,
		Blobs:       blobs,
		Waiter:      cfg.Waiter,
	})
	if err != nil {
		return nil, fmt.Errorf("uma bridge: compose engine: %w", err)
	}
	b := &Bridge{cfg: cfg, engine: engine}
	if err := b.provision(ctx, svc.ID, ent, extension, destination); err != nil {
		return nil, errors.Join(err, engine.Close(context.WithoutCancel(ctx)))
	}

	// The service's allowlists are aps's, not uma's: the same validator the
	// native providers' messages pass through, on the normalized message.
	validator := msgtypes.NewServiceValidator()
	policy := msgtypes.ServiceValidationConfig{ID: svc.ID, Adapter: svc.Adapter, Profile: svc.Profile, Env: svc.Env, Options: svc.Options}
	p := &provider{
		platform: spec.PlatformFor(svc.Options), entry: ent, key: key, service: svc.ID, receiving: destination,
		mediaDir: cfg.MediaDir, receiveOnly: spec.ReceiveOnly,
		validate: func(msg *msgtypes.NormalizedMessage) error { return validator.ValidateMessage(policy, msg) },
	}
	executor := cfg.Executor
	if spec.ReceiveOnly {
		executor = receiveOnlyExecutor{next: executor, key: key}
	}
	if cfg.History != nil {
		executor = dedupExecutor{next: executor, history: cfg.History}
	}
	b.runtime, err = msgtypes.NewRuntime(p, cfg.Router, executor, msgtypes.RuntimeOptions{
		ServiceID: svc.ID, History: cfg.History, Hooks: cfg.Hooks,
	})
	if err != nil {
		return nil, errors.Join(err, engine.Close(context.WithoutCancel(ctx)))
	}
	if err := b.compose(ctx); err != nil {
		return nil, errors.Join(err, engine.Close(context.WithoutCancel(ctx)))
	}
	return b, nil
}

var (
	serviceCall = uma.RuntimeCall{
		Principal: model.Principal{Tenant: tenant, Actor: actorService},
		Call:      model.Call{Correlation: "aps-bridge-runtime"},
	}
	botPrincipal = model.Principal{Tenant: tenant, Actor: actorBot}
)

func command(op model.OperationID) model.Command {
	return model.Command{Call: model.Call{Correlation: model.CorrelationID(op)}, Operation: op}
}

// provision configures the channel and its inbox as the service actor. The
// store is in memory, so this runs on every start.
func (b *Bridge) provision(ctx context.Context, serviceID string, ent entry, extension model.Extension, destination string) error {
	admin, err := b.engine.For(serviceCall.Principal)
	if err != nil {
		return fmt.Errorf("uma bridge: %w", err)
	}
	if issues := b.engine.ValidateChannel(ent.typeKey, extension); len(issues) > 0 {
		msgs := make([]string, 0, len(issues))
		for _, issue := range issues {
			msgs = append(msgs, issue.Field+": "+issue.Message)
		}
		return fmt.Errorf("uma bridge: invalid %s configuration: %s", ent.typeKey, strings.Join(msgs, "; "))
	}
	cfgSvc := admin.Configuration()
	acc, err := cfgSvc.ConfigureChannel(ctx, model.ConfigureChannel{
		Command: command("aps.bridge.channel"),
		Channel: model.Channel{
			Ref: model.ChannelRef{Tenant: tenant}, Key: serviceID, Type: ent.typeKey,
			Configuration: extension, CredentialRef: serviceID,
		},
	})
	if err != nil {
		return fmt.Errorf("uma bridge: configure channel: %w", err)
	}
	b.channel, err = acc.Operation.Result.AsChannel()
	if err != nil {
		return fmt.Errorf("uma bridge: configure channel: %w", err)
	}
	lock := true
	if _, err := cfgSvc.ConfigureInbox(ctx, model.ConfigureInbox{
		Command: command("aps.bridge.inbox"),
		Inbox: model.Inbox{
			Ref: model.InboxRef{Tenant: tenant}, Channel: b.channel, Key: serviceID, Name: serviceID,
			Destinations: []string{destination},
			Selection: model.Selection{
				Mode: model.ModePerContact, LockToSingle: &lock, LifecycleRules: model.RuleSetContactLifecycleV1,
			},
		},
	}); err != nil {
		return fmt.Errorf("uma bridge: configure inbox: %w", err)
	}
	return nil
}

// compose subscribes the bot to message.received and builds the host.
func (b *Bridge) compose(ctx context.Context) error {
	client, err := b.engine.For(botPrincipal)
	if err != nil {
		return fmt.Errorf("uma bridge: %w", err)
	}
	prep, err := host.Prepare(ctx, host.PrepareConfig{
		Client: client, Principal: botPrincipal, Consumer: "aps-bridge",
		Filter: model.Filter{Kinds: []string{model.EventMessageReceived}},
		// The store is in memory: every start is a first run.
		Intent: host.Bootstrap(model.StartBeginning, ""),
	})
	if err != nil {
		return fmt.Errorf("uma bridge: open subscription: %w", err)
	}
	poll := b.cfg.Poll
	if poll <= 0 {
		poll = time.Second
	}
	h, err := host.New(ctx, host.Config{
		Client: client, Principal: botPrincipal,
		Runtime: b.engine.Runtime(), RuntimeCall: serviceCall,
		Worker:   uma.WorkerOptions{Owner: "aps-bridge-worker", Limit: 10, Poll: poll},
		Prepared: prep,
		Bot: bot.Config{
			Wait:     10 * time.Second,
			Fallback: func(ctx context.Context, t *bot.Turn) error { return b.handle(ctx, client, t) },
			Waiter:   b.cfg.Waiter,
		},
	})
	if err != nil {
		return fmt.Errorf("uma bridge: compose bot host: %w", err)
	}
	b.host = h
	b.webhook, err = b.engine.Runtime().Webhook(ctx, serviceCall, b.channel)
	if err != nil {
		return fmt.Errorf("uma bridge: bind provider webhook: %w", err)
	}
	return nil
}

// handle runs one received message through messenger.Runtime. Errors the
// runtime returns are final (it already retried delivery), so they are
// dropped after being reported; only failures before the action ran, which
// repeat safely, are left to the bot's own retry.
func (b *Bridge) handle(ctx context.Context, client uma.Client, t *bot.Turn) error {
	if t.Message == nil {
		return nil
	}
	ctx = context.WithValue(ctx, stateKey{}, &turnState{turn: t, client: client})
	result, err := b.runtime.HandleIngress(ctx, msgtypes.NativeIngress{
		ServiceID: b.cfg.Service.ID,
		Provider:  b.providerName(),
		Mode:      msgtypes.IngressModeWebhook,
		Metadata:  map[string]any{"engine": "uma", "uma_adapter": b.cfg.Service.Options[core.OptionUmaAdapter]},
	})
	if b.cfg.OnResult != nil {
		b.cfg.OnResult(result, err)
	}
	if err == nil {
		return nil
	}
	var rerr *msgtypes.RuntimeError
	if errors.As(err, &rerr) && (rerr.Stage == msgtypes.RuntimeStageIngress || rerr.Stage == msgtypes.RuntimeStageNormalize) {
		// Nothing ran yet: an engine failure repeats safely; anything else
		// (a message that cannot be normalized) never will succeed.
		var merr *model.Error
		if errors.As(rerr.Err, &merr) {
			return rerr.Err
		}
	}
	return bot.Drop(err)
}

func (b *Bridge) providerName() string {
	spec, _ := core.UmaAdapterSpecFor(b.cfg.Service.Options[core.OptionUmaAdapter])
	return spec.PlatformFor(b.cfg.Service.Options)
}

// Webhook is the provider webhook receiver for the service's channel: it
// verifies the provider's signature, records the event in uma and answers
// the provider. Mounting it at a public URL is the host's job; the bridge
// serves it from nowhere itself.
func (b *Bridge) Webhook() http.Handler { return b.webhook }

// Run delivers queued replies and dispatches received messages until ctx
// ends or either part stops. It returns nil on a clean stop.
func (b *Bridge) Run(ctx context.Context) error {
	if err := b.host.Run(ctx, host.Options{}); err != nil {
		return fmt.Errorf("uma bridge: %w", err)
	}
	return nil
}

// Close closes the engine. Call it after Run returned.
func (b *Bridge) Close(ctx context.Context) error {
	if err := b.engine.Close(ctx); err != nil {
		return fmt.Errorf("uma bridge: close engine: %w", err)
	}
	return nil
}
