package messenger

import (
	"testing"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
)

func TestServiceRouteResolver_DefaultActionFallback(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "slack",
		Profile: "assistant",
		Options: map[string]string{
			"default_action": "reply",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	resolver := &serviceRouteResolver{base: &mockResolver{
		links:   map[string]*coremessenger.ProfileMessengerLink{},
		actions: map[string]string{},
	}}
	link, action, err := resolver.ResolveChannelRoute("support-bot", "C01ABC2DEF")
	if err != nil {
		t.Fatalf("ResolveChannelRoute: %v", err)
	}
	if link.ProfileID != "assistant" {
		t.Fatalf("profile = %q, want assistant", link.ProfileID)
	}
	if action != "assistant=reply" {
		t.Fatalf("action = %q, want assistant=reply", action)
	}
}

func TestServiceRouteResolver_DefaultActionPreservesExplicitMapping(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "community-bot",
		Type:    "message",
		Adapter: "discord",
		Profile: "assistant",
		Options: map[string]string{
			"default_action": "assistant=handle_discord",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	resolver := &serviceRouteResolver{base: &mockResolver{
		links:   map[string]*coremessenger.ProfileMessengerLink{},
		actions: map[string]string{},
	}}
	_, action, err := resolver.ResolveChannelRoute("community-bot", "1200000000000000002")
	if err != nil {
		t.Fatalf("ResolveChannelRoute: %v", err)
	}
	if action != "assistant=handle_discord" {
		t.Fatalf("action = %q, want assistant=handle_discord", action)
	}
}
