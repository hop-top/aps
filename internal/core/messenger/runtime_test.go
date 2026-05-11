package messenger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeIngressValidate(t *testing.T) {
	tests := []struct {
		name    string
		ingress NativeIngress
		wantErr string
	}{
		{
			name: "valid",
			ingress: NativeIngress{
				ServiceID: "support-bot",
				Provider:  "slack",
				Mode:      IngressModeWebhook,
			},
		},
		{
			name: "missing service",
			ingress: NativeIngress{
				Provider: "slack",
				Mode:     IngressModeWebhook,
			},
			wantErr: "service ID is required",
		},
		{
			name: "missing provider",
			ingress: NativeIngress{
				ServiceID: "support-bot",
				Mode:      IngressModeWebhook,
			},
			wantErr: "provider is required",
		},
		{
			name: "missing mode",
			ingress: NativeIngress{
				ServiceID: "support-bot",
				Provider:  "slack",
			},
			wantErr: "ingress mode is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ingress.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestRuntimeHandleIngress_NormalizesExecutesAndDelivers(t *testing.T) {
	ctx := context.Background()
	received := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	provider := &stubProvider{
		message: &NormalizedMessage{
			ID:       "msg-1",
			Platform: "slack",
			Sender:   Sender{ID: "U123", Name: "Sam"},
			Channel:  Channel{ID: "C123", Name: "support"},
			Text:     "help",
			Thread:   &Thread{ID: "thread-1", Type: "reply"},
		},
		receipt: &DeliveryReceipt{Provider: "slack", DeliveryID: "delivery-1", Status: "sent"},
	}
	router := &stubRouter{route: ExecutionRoute{ProfileID: "assistant", ActionName: "triage"}}
	executor := &stubExecutor{
		result: &ExecutionResult{
			Status: "completed",
			Output: "done",
			Reply:  &DeliveryRequest{Text: "ack"},
		},
	}

	runtime, err := NewRuntime(provider, router, executor, RuntimeOptions{
		ServiceID: "support-bot",
		Now:       func() time.Time { return received },
	})
	require.NoError(t, err)

	result, err := runtime.HandleIngress(ctx, NativeIngress{
		Mode:   IngressModeWebhook,
		Method: "POST",
		Path:   "/services/support-bot/webhook",
		Body:   []byte(`{"event":"message"}`),
	})
	require.NoError(t, err)

	require.NotNil(t, result)
	assert.Equal(t, "msg-1", result.Message.ID)
	assert.Equal(t, received, result.Message.Timestamp)
	assert.Equal(t, "assistant=triage", result.Route.TargetAction())
	assert.Equal(t, "support-bot", result.Handoff.ServiceID)
	assert.Equal(t, "assistant", result.Handoff.ProfileID)
	assert.Equal(t, "triage", result.Handoff.ActionName)
	assert.Equal(t, "assistant=triage", result.Handoff.TargetAction)
	assert.Equal(t, 1, result.Handoff.Attempt)
	assert.Equal(t, "delivery-1", result.Delivery.DeliveryID)

	require.Len(t, provider.deliveries, 1)
	assert.Equal(t, "slack", provider.deliveries[0].Provider)
	assert.Equal(t, "support-bot", provider.deliveries[0].ServiceID)
	assert.Equal(t, "msg-1", provider.deliveries[0].MessageID)
	assert.Equal(t, "C123", provider.deliveries[0].ChannelID)
	assert.Equal(t, "thread-1", provider.deliveries[0].ThreadID)
	assert.Equal(t, "ack", provider.deliveries[0].Text)
	assert.Equal(t, result.Handoff, executor.handoff)
}

func TestRuntimeHandleIngress_ReportsRetryableDeliveryError(t *testing.T) {
	ctx := context.Background()
	deliveryErr := errors.New("rate limited")
	provider := &stubProvider{
		message: &NormalizedMessage{
			ID:       "msg-1",
			Platform: "slack",
			Sender:   Sender{ID: "U123"},
			Channel:  Channel{ID: "C123"},
			Text:     "hello",
		},
		deliverErr: deliveryErr,
	}
	router := &stubRouter{route: ExecutionRoute{ProfileID: "assistant", ActionName: "reply"}}
	executor := &stubExecutor{result: &ExecutionResult{Status: "completed", Reply: &DeliveryRequest{Text: "ack"}}}

	var reported *RuntimeError
	var retry RetryDecision
	runtime, err := NewRuntime(provider, router, executor, RuntimeOptions{
		ServiceID: "support-bot",
		RetryPolicy: RetryPolicy{
			MaxAttempts: 3,
			BaseDelay:   250 * time.Millisecond,
			MaxDelay:    time.Second,
			Stages:      []RuntimeStage{RuntimeStageDeliver},
		},
		Hooks: RuntimeHooks{
			OnError: func(_ context.Context, err *RuntimeError) error {
				reported = err
				return nil
			},
			OnRetry: func(_ context.Context, decision RetryDecision) error {
				retry = decision
				return nil
			},
		},
	})
	require.NoError(t, err)

	_, err = runtime.HandleIngress(ctx, NativeIngress{
		ServiceID: "support-bot",
		Provider:  "slack",
		Mode:      IngressModeWebhook,
		Attempt:   2,
	})
	require.Error(t, err)

	var runtimeErr *RuntimeError
	require.ErrorAs(t, err, &runtimeErr)
	assert.Equal(t, RuntimeStageDeliver, runtimeErr.Stage)
	assert.Equal(t, "support-bot", runtimeErr.Service)
	assert.Equal(t, "slack", runtimeErr.Provider)
	assert.Equal(t, "msg-1", runtimeErr.Message)
	assert.Equal(t, 2, runtimeErr.Attempt)
	assert.ErrorIs(t, err, deliveryErr)

	require.NotNil(t, reported)
	assert.Equal(t, RuntimeStageDeliver, reported.Stage)
	assert.True(t, retry.Retry)
	assert.Equal(t, 2, retry.Attempt)
	assert.Equal(t, 3, retry.MaxAttempts)
	assert.Equal(t, 500*time.Millisecond, retry.Delay)
	assert.Equal(t, RuntimeStageDeliver, retry.Stage)
}

func TestRetryPolicyDecide(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts: 2,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
		Stages:      []RuntimeStage{RuntimeStageExecute},
	}

	decision := policy.Decide(&RuntimeError{Stage: RuntimeStageExecute, Attempt: 1, Err: errors.New("failed")})
	assert.True(t, decision.Retry)
	assert.Equal(t, 100*time.Millisecond, decision.Delay)

	decision = policy.Decide(&RuntimeError{Stage: RuntimeStageExecute, Attempt: 2, Err: errors.New("failed")})
	assert.False(t, decision.Retry)
	assert.Equal(t, "max attempts reached", decision.Reason)

	decision = policy.Decide(&RuntimeError{Stage: RuntimeStageNormalize, Attempt: 1, Err: errors.New("failed")})
	assert.False(t, decision.Retry)
	assert.Equal(t, "stage is not retryable", decision.Reason)
}

type stubProvider struct {
	message      *NormalizedMessage
	normalizeErr error
	receipt      *DeliveryReceipt
	deliverErr   error
	deliveries   []DeliveryRequest
}

func (p *stubProvider) Metadata() ProviderRuntimeMetadata {
	return ProviderRuntimeMetadata{
		Provider:        "slack",
		DisplayName:     "Slack",
		IngressModes:    []IngressMode{IngressModeWebhook},
		DeliveryModes:   []DeliveryMode{DeliveryModeText},
		SupportsThreads: true,
	}
}

func (p *stubProvider) NormalizeIngress(_ context.Context, _ NativeIngress) (*NormalizedMessage, error) {
	if p.normalizeErr != nil {
		return nil, p.normalizeErr
	}
	return p.message, nil
}

func (p *stubProvider) DeliverMessage(_ context.Context, delivery DeliveryRequest) (*DeliveryReceipt, error) {
	p.deliveries = append(p.deliveries, delivery)
	if p.deliverErr != nil {
		return nil, p.deliverErr
	}
	return p.receipt, nil
}

type stubRouter struct {
	route ExecutionRoute
	err   error
}

func (r *stubRouter) ResolveMessageRoute(_ context.Context, _ *NormalizedMessage) (ExecutionRoute, error) {
	if r.err != nil {
		return ExecutionRoute{}, r.err
	}
	return r.route, nil
}

type stubExecutor struct {
	handoff ExecutionHandoff
	result  *ExecutionResult
	err     error
}

func (e *stubExecutor) ExecuteMessage(_ context.Context, handoff ExecutionHandoff) (*ExecutionResult, error) {
	e.handoff = handoff
	if e.err != nil {
		return nil, e.err
	}
	return e.result, nil
}
