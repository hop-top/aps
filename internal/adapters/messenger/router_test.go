package messenger

import (
	"context"
	"encoding/json"
	"testing"

	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
)

// mockResolver implements RouteResolver for testing.
type mockResolver struct {
	links   map[string]*msgtypes.ProfileMessengerLink // key: "messenger:channel"
	actions map[string]string                         // key: "messenger:channel"
}

func (m *mockResolver) ResolveChannelRoute(messengerName, channelID string) (*msgtypes.ProfileMessengerLink, string, error) {
	key := messengerName + ":" + channelID
	link, ok := m.links[key]
	if !ok {
		return nil, "", msgtypes.ErrUnknownChannel(messengerName, channelID)
	}
	return link, m.actions[key], nil
}

func newTestRouter(links map[string]*msgtypes.ProfileMessengerLink, actions map[string]string) *MessageRouter {
	resolver := &mockResolver{
		links:   links,
		actions: actions,
	}
	normalizer := NewNormalizer()
	return NewMessageRouterWithExecutor(resolver, normalizer, &fakeActionExecutor{})
}

type fakeActionExecutor struct {
	status protocol.RunStatus
	output string
	err    error
	input  protocol.RunInput
}

func (f *fakeActionExecutor) ExecuteRun(ctx context.Context, input protocol.RunInput, stream protocol.StreamWriter) (*protocol.RunState, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	status := f.status
	if status == "" {
		status = protocol.RunStatusCompleted
	}
	output := f.output
	if output == "" {
		output = "action output"
	}
	return &protocol.RunState{
		ProfileID: input.ProfileID,
		ActionID:  input.ActionID,
		Status:    status,
		Output:    output,
	}, nil
}

func TestMessageRouter_Route_Success(t *testing.T) {
	tests := []struct {
		name           string
		messengerName  string
		channelID      string
		actionMapping  string
		wantProfileID  string
		wantActionName string
		wantStatus     string
	}{
		{
			name:           "telegram channel routed to profile action",
			messengerName:  "telegram",
			channelID:      "-1001234567890",
			actionMapping:  "research-agent=handle_message",
			wantProfileID:  "research-agent",
			wantActionName: "handle_message",
			wantStatus:     "routed",
		},
		{
			name:           "slack channel routed with colon separator",
			messengerName:  "slack",
			channelID:      "C01ABC2DEF",
			actionMapping:  "dev-ops=deploy_notify",
			wantProfileID:  "dev-ops",
			wantActionName: "deploy_notify",
			wantStatus:     "routed",
		},
		{
			name:           "github repo routed",
			messengerName:  "github",
			channelID:      "org/repo",
			actionMapping:  "ci-bot=run_checks",
			wantProfileID:  "ci-bot",
			wantActionName: "run_checks",
			wantStatus:     "routed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.messengerName + ":" + tt.channelID
			router := newTestRouter(
				map[string]*msgtypes.ProfileMessengerLink{
					key: {
						ProfileID:     tt.wantProfileID,
						MessengerName: tt.messengerName,
						Enabled:       true,
					},
				},
				map[string]string{
					key: tt.actionMapping,
				},
			)

			msg := &msgtypes.NormalizedMessage{
				ID:       "msg_test_1",
				Platform: tt.messengerName,
				Sender:   msgtypes.Sender{ID: "sender1"},
				Channel:  msgtypes.Channel{ID: tt.channelID},
			}

			result, err := router.Route(context.Background(), msg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", result.Status, tt.wantStatus)
			}
			if result.ProfileID != tt.wantProfileID {
				t.Errorf("profileID = %q, want %q", result.ProfileID, tt.wantProfileID)
			}
			if result.ActionName != tt.wantActionName {
				t.Errorf("actionName = %q, want %q", result.ActionName, tt.wantActionName)
			}
			if result.MessageID != "msg_test_1" {
				t.Errorf("messageID = %q, want %q", result.MessageID, "msg_test_1")
			}
			if result.Route == "" {
				t.Error("route should not be empty for routed message")
			}
			// Verify the message was stamped with the profile ID.
			if msg.ProfileID != tt.wantProfileID {
				t.Errorf("msg.ProfileID = %q, want %q", msg.ProfileID, tt.wantProfileID)
			}
		})
	}
}

func TestMessageRouter_Route_UnknownChannel(t *testing.T) {
	tests := []struct {
		name          string
		messengerName string
		channelID     string
	}{
		{
			name:          "telegram unknown channel",
			messengerName: "telegram",
			channelID:     "-9999999999",
		},
		{
			name:          "slack unknown channel",
			messengerName: "slack",
			channelID:     "C_UNKNOWN",
		},
		{
			name:          "github unknown repo",
			messengerName: "github",
			channelID:     "unknown/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Empty resolver means no routes are defined.
			router := newTestRouter(
				map[string]*msgtypes.ProfileMessengerLink{},
				map[string]string{},
			)

			msg := &msgtypes.NormalizedMessage{
				ID:       "msg_unknown_1",
				Platform: tt.messengerName,
				Sender:   msgtypes.Sender{ID: "sender1"},
				Channel:  msgtypes.Channel{ID: tt.channelID},
			}

			result, err := router.Route(context.Background(), msg)
			if err != nil {
				t.Fatalf("unexpected error (unknown channel should not produce error): %v", err)
			}
			if result.Status != "unknown_channel" {
				t.Errorf("status = %q, want %q", result.Status, "unknown_channel")
			}
			if result.Error == nil {
				t.Error("error should be set for unknown channel")
			}
			if result.ProfileID != "" {
				t.Errorf("profileID should be empty, got %q", result.ProfileID)
			}
		})
	}
}

func TestMessageRouter_Route_UsesConfiguredMessengerName(t *testing.T) {
	key := "support-bot:C01ABC2DEF"
	router := newTestRouter(
		map[string]*msgtypes.ProfileMessengerLink{
			key: {
				ProfileID:     "assistant",
				MessengerName: "support-bot",
				Enabled:       true,
			},
		},
		map[string]string{key: "assistant=reply"},
	)

	msg := &msgtypes.NormalizedMessage{
		ID:       "msg_service_route",
		Platform: "slack",
		Sender:   msgtypes.Sender{ID: "U123"},
		Channel:  msgtypes.Channel{ID: "C01ABC2DEF"},
		PlatformMetadata: map[string]any{
			"messenger_name": "support-bot",
		},
	}

	result, err := router.Route(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "routed" {
		t.Fatalf("status = %q, want routed", result.Status)
	}
	if result.ProfileID != "assistant" || result.ActionName != "reply" {
		t.Fatalf("route = %s/%s, want assistant/reply", result.ProfileID, result.ActionName)
	}
}

func TestMessageRouter_Route_NoAction(t *testing.T) {
	key := "telegram:-1001234567890"
	router := newTestRouter(
		map[string]*msgtypes.ProfileMessengerLink{
			key: {
				ProfileID:     "research-agent",
				MessengerName: "telegram",
				Enabled:       true,
			},
		},
		map[string]string{
			key: "", // empty action mapping
		},
	)

	msg := &msgtypes.NormalizedMessage{
		ID:       "msg_noaction_1",
		Platform: "telegram",
		Sender:   msgtypes.Sender{ID: "sender1"},
		Channel:  msgtypes.Channel{ID: "-1001234567890"},
	}

	result, err := router.Route(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "no_action" {
		t.Errorf("status = %q, want %q", result.Status, "no_action")
	}
	if result.ProfileID != "research-agent" {
		t.Errorf("profileID = %q, want %q", result.ProfileID, "research-agent")
	}
}

func TestMessageRouter_Route_NilMessage(t *testing.T) {
	router := newTestRouter(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
	)

	_, err := router.Route(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil message, got nil")
	}
}

func TestMessageRouter_Route_ContextCancelled(t *testing.T) {
	router := newTestRouter(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	msg := &msgtypes.NormalizedMessage{
		ID:       "msg_cancelled",
		Platform: "telegram",
		Sender:   msgtypes.Sender{ID: "sender1"},
		Channel:  msgtypes.Channel{ID: "-1001234567890"},
	}

	_, err := router.Route(ctx, msg)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestMessageRouter_HandleMessage_Success(t *testing.T) {
	key := "telegram:-1001234567890"
	router := newTestRouter(
		map[string]*msgtypes.ProfileMessengerLink{
			key: {
				ProfileID:     "research-agent",
				MessengerName: "telegram",
				Enabled:       true,
			},
		},
		map[string]string{
			key: "research-agent=handle_message",
		},
	)

	msg := &msgtypes.NormalizedMessage{
		ID:       "msg_handle_1",
		Platform: "telegram",
		Sender:   msgtypes.Sender{ID: "456"},
		Channel:  msgtypes.Channel{ID: "-1001234567890"},
		Text:     "hello",
	}

	result, err := router.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("status = %q, want %q", result.Status, "success")
	}
	if result.Output == "" {
		t.Error("output should not be empty on success")
	}
	if result.ExecutionTime <= 0 {
		t.Error("execution time should be positive")
	}
}

func TestMessageRouter_HandleMessage_UnknownChannel(t *testing.T) {
	// Empty resolver, no routes.
	router := newTestRouter(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
	)

	msg := &msgtypes.NormalizedMessage{
		ID:       "msg_handle_unknown",
		Platform: "telegram",
		Sender:   msgtypes.Sender{ID: "456"},
		Channel:  msgtypes.Channel{ID: "-9999999999"},
		Text:     "unrouted message",
	}

	result, err := router.HandleMessage(context.Background(), msg)
	// HandleMessage returns an ActionResult with "failed" status but no error
	// when the route is simply unknown (not an infrastructure failure).
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("status = %q, want %q", result.Status, "failed")
	}
	if result.Output == "" {
		t.Error("output should contain failure information")
	}
}

func TestMessageRouter_HandleMessage_ContextCancelled(t *testing.T) {
	router := newTestRouter(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := &msgtypes.NormalizedMessage{
		ID:       "msg_handle_cancelled",
		Platform: "telegram",
		Sender:   msgtypes.Sender{ID: "456"},
		Channel:  msgtypes.Channel{ID: "-1001234567890"},
	}

	_, err := router.HandleMessage(ctx, msg)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestMessageRouter_ExecuteAction(t *testing.T) {
	executor := &fakeActionExecutor{output: "real action output"}
	router := NewMessageRouterWithExecutor(&mockResolver{}, NewNormalizer(), executor)

	tests := []struct {
		name       string
		profileID  string
		actionName string
		msg        *msgtypes.NormalizedMessage
		wantStatus string
	}{
		{
			name:       "basic execution",
			profileID:  "test-profile",
			actionName: "test-action",
			msg: &msgtypes.NormalizedMessage{
				ID:       "msg_exec_1",
				Platform: "telegram",
				Sender:   msgtypes.Sender{ID: "sender1"},
				Channel:  msgtypes.Channel{ID: "chan1"},
			},
			wantStatus: "success",
		},
		{
			name:       "github execution",
			profileID:  "ci-bot",
			actionName: "run_checks",
			msg: &msgtypes.NormalizedMessage{
				ID:       "msg_exec_2",
				Platform: "github",
				Sender:   msgtypes.Sender{ID: "octocat"},
				Channel:  msgtypes.Channel{ID: "org/repo"},
			},
			wantStatus: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := router.ExecuteAction(context.Background(), tt.profileID, tt.actionName, tt.msg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", result.Status, tt.wantStatus)
			}
			if result.Output == "" {
				t.Error("output should contain dispatch information")
			}
			if result.Output != "real action output" {
				t.Errorf("output = %q, want real action output", result.Output)
			}
			var payload msgtypes.NormalizedMessage
			if err := json.Unmarshal(executor.input.Payload, &payload); err != nil {
				t.Fatalf("payload should be normalized message JSON: %v", err)
			}
			if payload.ID != tt.msg.ID {
				t.Errorf("payload ID = %q, want %q", payload.ID, tt.msg.ID)
			}
			if result.ExecutionTime < 0 {
				t.Error("execution time should not be negative")
			}
		})
	}
}

func TestMessageRouter_ExecuteAction_ContextCancelled(t *testing.T) {
	router := newTestRouter(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := &msgtypes.NormalizedMessage{
		ID:       "msg_exec_cancel",
		Platform: "telegram",
		Sender:   msgtypes.Sender{ID: "sender1"},
		Channel:  msgtypes.Channel{ID: "chan1"},
	}

	_, err := router.ExecuteAction(ctx, "profile", "action", msg)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
