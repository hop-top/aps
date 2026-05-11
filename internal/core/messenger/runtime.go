package messenger

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// IngressMode identifies how a provider supplies native events to APS.
type IngressMode string

const (
	IngressModeWebhook IngressMode = "webhook"
	IngressModePolling IngressMode = "polling"
	IngressModeStream  IngressMode = "stream"
)

// RuntimeStage identifies a message provider runtime processing step.
type RuntimeStage string

const (
	RuntimeStageIngress   RuntimeStage = "ingress"
	RuntimeStageNormalize RuntimeStage = "normalize"
	RuntimeStageRoute     RuntimeStage = "route"
	RuntimeStageExecute   RuntimeStage = "execute"
	RuntimeStageDeliver   RuntimeStage = "deliver"
)

// DeliveryMode identifies outbound delivery capabilities exposed by a provider.
type DeliveryMode string

const (
	DeliveryModeText     DeliveryMode = "text"
	DeliveryModeReaction DeliveryMode = "reaction"
	DeliveryModeFile     DeliveryMode = "file"
)

// NativeIngress is the provider-native event envelope accepted by the shared
// message runtime before any provider-specific normalization happens.
type NativeIngress struct {
	ID         string              `json:"id,omitempty"`
	ServiceID  string              `json:"service_id"`
	Provider   string              `json:"provider"`
	Mode       IngressMode         `json:"mode"`
	Method     string              `json:"method,omitempty"`
	Path       string              `json:"path,omitempty"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Query      map[string][]string `json:"query,omitempty"`
	Body       []byte              `json:"body,omitempty"`
	RemoteAddr string              `json:"remote_addr,omitempty"`
	ReceivedAt time.Time           `json:"received_at"`
	Attempt    int                 `json:"attempt,omitempty"`
	Metadata   map[string]any      `json:"metadata,omitempty"`
}

// Validate checks the common fields required before handing ingress to a
// provider normalizer.
func (i NativeIngress) Validate() error {
	if strings.TrimSpace(i.ServiceID) == "" {
		return fmt.Errorf("service ID is required")
	}
	if strings.TrimSpace(i.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if i.Mode == "" {
		return fmt.Errorf("ingress mode is required")
	}
	return nil
}

// ProviderRuntimeMetadata describes a provider's shared runtime capabilities.
type ProviderRuntimeMetadata struct {
	Provider            string         `json:"provider"`
	DisplayName         string         `json:"display_name,omitempty"`
	IngressModes        []IngressMode  `json:"ingress_modes,omitempty"`
	DeliveryModes       []DeliveryMode `json:"delivery_modes,omitempty"`
	SupportsThreads     bool           `json:"supports_threads,omitempty"`
	SupportsAttachments bool           `json:"supports_attachments,omitempty"`
	SupportsReactions   bool           `json:"supports_reactions,omitempty"`
}

// Validate checks that provider metadata can identify the runtime owner.
func (m ProviderRuntimeMetadata) Validate() error {
	if strings.TrimSpace(m.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	return nil
}

// MessageProvider normalizes provider-native ingress into APS' common message
// shape. Delivery is intentionally a separate interface so receive-only
// providers can still participate in the runtime.
type MessageProvider interface {
	Metadata() ProviderRuntimeMetadata
	NormalizeIngress(ctx context.Context, ingress NativeIngress) (*NormalizedMessage, error)
}

// ProviderDelivery sends an outbound message through a provider.
type ProviderDelivery interface {
	DeliverMessage(ctx context.Context, delivery DeliveryRequest) (*DeliveryReceipt, error)
}

// DeliveryRequest is the provider-neutral outbound message contract.
type DeliveryRequest struct {
	Provider    string         `json:"provider"`
	ServiceID   string         `json:"service_id"`
	MessageID   string         `json:"message_id,omitempty"`
	ChannelID   string         `json:"channel_id"`
	ThreadID    string         `json:"thread_id,omitempty"`
	Text        string         `json:"text,omitempty"`
	Attachments []Attachment   `json:"attachments,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Validate checks the common fields required for outbound provider delivery.
func (d DeliveryRequest) Validate() error {
	if strings.TrimSpace(d.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(d.ServiceID) == "" {
		return fmt.Errorf("service ID is required")
	}
	if strings.TrimSpace(d.ChannelID) == "" {
		return fmt.Errorf("channel ID is required")
	}
	if strings.TrimSpace(d.Text) == "" && len(d.Attachments) == 0 {
		return fmt.Errorf("delivery content is required")
	}
	return nil
}

// DeliveryReceipt is the provider-neutral outbound delivery result.
type DeliveryReceipt struct {
	Provider     string         `json:"provider"`
	DeliveryID   string         `json:"delivery_id,omitempty"`
	Status       string         `json:"status"`
	Retriable    bool           `json:"retriable,omitempty"`
	DeliveredAt  time.Time      `json:"delivered_at,omitempty"`
	ProviderData map[string]any `json:"provider_data,omitempty"`
}

// ExecutionRoute is the normalized profile action target for a message.
type ExecutionRoute struct {
	ProfileID  string `json:"profile_id"`
	ActionName string `json:"action_name"`
	Mapping    string `json:"mapping,omitempty"`
}

// Validate checks that a route can hand off to profile execution.
func (r ExecutionRoute) Validate() error {
	if strings.TrimSpace(r.ProfileID) == "" {
		return fmt.Errorf("profile ID is required")
	}
	if strings.TrimSpace(r.ActionName) == "" {
		return fmt.Errorf("action name is required")
	}
	return nil
}

// TargetAction returns the canonical profile=action mapping string.
func (r ExecutionRoute) TargetAction() string {
	if r.Mapping != "" {
		return r.Mapping
	}
	return TargetAction{ProfileID: r.ProfileID, ActionName: r.ActionName}.String()
}

// MessageRouter resolves a normalized message to a profile action target.
type MessageRouter interface {
	ResolveMessageRoute(ctx context.Context, msg *NormalizedMessage) (ExecutionRoute, error)
}

// ExecutionHandoff is the normalized runtime payload passed to profile
// execution after provider ingress has been validated, normalized, and routed.
type ExecutionHandoff struct {
	ServiceID    string             `json:"service_id"`
	Provider     string             `json:"provider"`
	ProfileID    string             `json:"profile_id"`
	ActionName   string             `json:"action_name"`
	TargetAction string             `json:"target_action"`
	Message      *NormalizedMessage `json:"message"`
	Ingress      NativeIngress      `json:"ingress"`
	Attempt      int                `json:"attempt,omitempty"`
	Metadata     map[string]any     `json:"metadata,omitempty"`
}

// MessageExecutor executes a normalized message against a profile action.
type MessageExecutor interface {
	ExecuteMessage(ctx context.Context, handoff ExecutionHandoff) (*ExecutionResult, error)
}

// ExecutionResult is the provider-neutral result from profile execution.
type ExecutionResult struct {
	Status   string            `json:"status"`
	Output   string            `json:"output,omitempty"`
	Reply    *DeliveryRequest  `json:"reply,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RuntimeResult captures a completed shared runtime pass.
type RuntimeResult struct {
	Message  *NormalizedMessage `json:"message,omitempty"`
	Route    ExecutionRoute     `json:"route"`
	Handoff  ExecutionHandoff   `json:"handoff"`
	Result   *ExecutionResult   `json:"result,omitempty"`
	Delivery *DeliveryReceipt   `json:"delivery,omitempty"`
}

// RetryPolicy decides whether runtime errors should be retried by the caller.
type RetryPolicy struct {
	MaxAttempts int            `json:"max_attempts"`
	BaseDelay   time.Duration  `json:"base_delay"`
	MaxDelay    time.Duration  `json:"max_delay"`
	Stages      []RuntimeStage `json:"stages,omitempty"`
	ErrorCodes  []ErrorCode    `json:"error_codes,omitempty"`
}

// RetryDecision records retry advice emitted by the runtime.
type RetryDecision struct {
	Retry       bool          `json:"retry"`
	Attempt     int           `json:"attempt"`
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay,omitempty"`
	Stage       RuntimeStage  `json:"stage"`
	Reason      string        `json:"reason,omitempty"`
}

// RuntimeError wraps errors with runtime stage and retry context.
type RuntimeError struct {
	Stage    RuntimeStage
	Service  string
	Provider string
	Message  string
	Attempt  int
	Decision RetryDecision
	Err      error
}

func (e *RuntimeError) Error() string {
	if e == nil {
		return "<nil>"
	}
	base := fmt.Sprintf("%s runtime stage failed", e.Stage)
	if e.Service != "" {
		base = fmt.Sprintf("%s for service %s", base, e.Service)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", base, e.Err)
	}
	return base
}

func (e *RuntimeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// RuntimeHooks are advisory extension points for reporting and retry scheduling.
type RuntimeHooks struct {
	OnError func(context.Context, *RuntimeError) error
	OnRetry func(context.Context, RetryDecision) error
}

// RuntimeOptions configures the shared provider runtime.
type RuntimeOptions struct {
	ServiceID   string
	RetryPolicy RetryPolicy
	Hooks       RuntimeHooks
	Now         func() time.Time
}

// Runtime coordinates provider ingress, routing, execution, delivery, and
// retry/error reporting hooks without knowing any provider-specific API.
type Runtime struct {
	provider MessageProvider
	router   MessageRouter
	executor MessageExecutor
	options  RuntimeOptions
}

// NewRuntime creates a shared message provider runtime.
func NewRuntime(provider MessageProvider, router MessageRouter, executor MessageExecutor, options RuntimeOptions) (*Runtime, error) {
	if provider == nil {
		return nil, fmt.Errorf("message provider is required")
	}
	if err := provider.Metadata().Validate(); err != nil {
		return nil, err
	}
	if router == nil {
		return nil, fmt.Errorf("message router is required")
	}
	if executor == nil {
		return nil, fmt.Errorf("message executor is required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.RetryPolicy.MaxAttempts == 0 {
		options.RetryPolicy = DefaultRetryPolicy()
	}
	return &Runtime{
		provider: provider,
		router:   router,
		executor: executor,
		options:  options,
	}, nil
}

// HandleIngress runs a provider-native ingress envelope through the normalized
// message provider runtime.
func (r *Runtime) HandleIngress(ctx context.Context, ingress NativeIngress) (*RuntimeResult, error) {
	meta := r.provider.Metadata()
	if ingress.Provider == "" {
		ingress.Provider = meta.Provider
	}
	if ingress.ServiceID == "" {
		ingress.ServiceID = r.options.ServiceID
	}
	if ingress.ReceivedAt.IsZero() {
		ingress.ReceivedAt = r.options.Now().UTC()
	}
	if ingress.Attempt < 1 {
		ingress.Attempt = 1
	}
	if err := ingress.Validate(); err != nil {
		return nil, r.runtimeError(ctx, RuntimeStageIngress, ingress, "", err)
	}

	msg, err := r.provider.NormalizeIngress(ctx, ingress)
	if err != nil {
		return nil, r.runtimeError(ctx, RuntimeStageNormalize, ingress, "", err)
	}
	if msg == nil {
		return nil, r.runtimeError(ctx, RuntimeStageNormalize, ingress, "", fmt.Errorf("normalized message is nil"))
	}
	if msg.Platform == "" {
		msg.Platform = ingress.Provider
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = ingress.ReceivedAt
	}
	if err := msg.Validate(); err != nil {
		return nil, r.runtimeError(ctx, RuntimeStageNormalize, ingress, msg.ID, err)
	}

	route, err := r.router.ResolveMessageRoute(ctx, msg)
	if err != nil {
		return nil, r.runtimeError(ctx, RuntimeStageRoute, ingress, msg.ID, err)
	}
	if err := route.Validate(); err != nil {
		return nil, r.runtimeError(ctx, RuntimeStageRoute, ingress, msg.ID, err)
	}

	handoff := ExecutionHandoff{
		ServiceID:    ingress.ServiceID,
		Provider:     ingress.Provider,
		ProfileID:    route.ProfileID,
		ActionName:   route.ActionName,
		TargetAction: route.TargetAction(),
		Message:      msg,
		Ingress:      ingress,
		Attempt:      ingress.Attempt,
	}

	result, err := r.executor.ExecuteMessage(ctx, handoff)
	if err != nil {
		return nil, r.runtimeError(ctx, RuntimeStageExecute, ingress, msg.ID, err)
	}
	if result == nil {
		result = &ExecutionResult{Status: "completed"}
	}

	runtimeResult := &RuntimeResult{
		Message: msg,
		Route:   route,
		Handoff: handoff,
		Result:  result,
	}
	if result.Reply == nil {
		return runtimeResult, nil
	}

	delivery, ok := r.provider.(ProviderDelivery)
	if !ok {
		return nil, r.runtimeError(ctx, RuntimeStageDeliver, ingress, msg.ID, fmt.Errorf("provider %q does not implement delivery", ingress.Provider))
	}
	reply := *result.Reply
	completeDeliveryRequest(&reply, ingress, msg)
	if err := reply.Validate(); err != nil {
		return nil, r.runtimeError(ctx, RuntimeStageDeliver, ingress, msg.ID, err)
	}
	receipt, err := delivery.DeliverMessage(ctx, reply)
	if err != nil {
		return nil, r.runtimeError(ctx, RuntimeStageDeliver, ingress, msg.ID, err)
	}
	runtimeResult.Delivery = receipt
	return runtimeResult, nil
}

// DefaultRetryPolicy returns conservative retry advice for caller-managed retry
// loops. It does not sleep or re-enter provider APIs by itself.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Stages:      []RuntimeStage{RuntimeStageRoute, RuntimeStageExecute, RuntimeStageDeliver},
	}
}

// Decide returns retry advice for a runtime error.
func (p RetryPolicy) Decide(err *RuntimeError) RetryDecision {
	if err == nil {
		return RetryDecision{}
	}
	if p.MaxAttempts == 0 {
		p = DefaultRetryPolicy()
	}
	decision := RetryDecision{
		Attempt:     err.Attempt,
		MaxAttempts: p.MaxAttempts,
		Stage:       err.Stage,
	}
	if err.Attempt >= p.MaxAttempts {
		decision.Reason = "max attempts reached"
		return decision
	}
	if len(p.Stages) > 0 && !runtimeStageAllowed(p.Stages, err.Stage) {
		decision.Reason = "stage is not retryable"
		return decision
	}
	decision.Retry = true
	decision.Delay = retryDelay(p, err.Attempt)
	decision.Reason = "retryable runtime error"
	return decision
}

func (r *Runtime) runtimeError(ctx context.Context, stage RuntimeStage, ingress NativeIngress, msgID string, err error) error {
	runtimeErr := &RuntimeError{
		Stage:    stage,
		Service:  ingress.ServiceID,
		Provider: ingress.Provider,
		Message:  msgID,
		Attempt:  ingress.Attempt,
		Err:      err,
	}
	runtimeErr.Decision = r.options.RetryPolicy.Decide(runtimeErr)
	if r.options.Hooks.OnError != nil {
		_ = r.options.Hooks.OnError(ctx, runtimeErr)
	}
	if runtimeErr.Decision.Retry && r.options.Hooks.OnRetry != nil {
		_ = r.options.Hooks.OnRetry(ctx, runtimeErr.Decision)
	}
	return runtimeErr
}

func completeDeliveryRequest(reply *DeliveryRequest, ingress NativeIngress, msg *NormalizedMessage) {
	if reply.Provider == "" {
		reply.Provider = ingress.Provider
	}
	if reply.ServiceID == "" {
		reply.ServiceID = ingress.ServiceID
	}
	if reply.MessageID == "" {
		reply.MessageID = msg.ID
	}
	if reply.ChannelID == "" {
		reply.ChannelID = msg.Channel.ID
	}
	if reply.ThreadID == "" && msg.Thread != nil {
		reply.ThreadID = msg.Thread.ID
	}
}

func runtimeStageAllowed(stages []RuntimeStage, stage RuntimeStage) bool {
	for _, candidate := range stages {
		if candidate == stage {
			return true
		}
	}
	return false
}

func retryDelay(policy RetryPolicy, attempt int) time.Duration {
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = time.Second
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 30 * time.Second
	}
	if attempt < 1 {
		attempt = 1
	}
	delay := policy.BaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= policy.MaxDelay {
			return policy.MaxDelay
		}
	}
	return delay
}
