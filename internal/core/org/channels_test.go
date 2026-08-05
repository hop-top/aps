package org

import (
	"reflect"
	"testing"

	"hop.top/aps/internal/core"
)

func TestChannels(t *testing.T) {
	tests := []struct {
		name    string
		profile core.Profile
		want    []string
	}{
		{
			name:    "none configured",
			profile: core.Profile{ID: "empty"},
			want:    []string{},
		},
		{
			name:    "a2a only",
			profile: core.Profile{ID: "a", A2A: &core.A2AConfig{}},
			want:    []string{ChannelA2A},
		},
		{
			name:    "acp enabled only",
			profile: core.Profile{ID: "a", ACP: &core.ACPConfig{Enabled: true}},
			want:    []string{ChannelACP},
		},
		{
			name:    "acp present but not enabled",
			profile: core.Profile{ID: "a", ACP: &core.ACPConfig{Transport: "stdio"}},
			want:    []string{},
		},
		{
			name:    "email only",
			profile: core.Profile{ID: "h", Email: "h@example.com"},
			want:    []string{ChannelEmail},
		},
		{
			name:    "webhooks only",
			profile: core.Profile{ID: "w", Webhooks: core.WebhookConfig{AllowedEvents: []string{"push"}}},
			want:    []string{ChannelWebhooks},
		},
		{
			name:    "webhooks empty events is not configured",
			profile: core.Profile{ID: "w", Webhooks: core.WebhookConfig{}},
			want:    []string{},
		},
		{
			name:    "human with only email",
			profile: core.Profile{ID: "jad", Type: core.ProfileTypeHuman, Email: "jad@example.com"},
			want:    []string{ChannelEmail},
		},
		{
			name: "agent with a2a and acp",
			profile: core.Profile{
				ID:  "bot",
				A2A: &core.A2AConfig{ListenAddr: "127.0.0.1:0"},
				ACP: &core.ACPConfig{Enabled: true, Transport: "http"},
			},
			want: []string{ChannelA2A, ChannelACP},
		},
		{
			name: "all channels in fixed order",
			profile: core.Profile{
				ID:       "full",
				Email:    "full@example.com",
				A2A:      &core.A2AConfig{},
				ACP:      &core.ACPConfig{Enabled: true},
				Webhooks: core.WebhookConfig{AllowedEvents: []string{"*"}},
			},
			want: []string{ChannelA2A, ChannelACP, ChannelEmail, ChannelWebhooks},
		},
		{
			name: "email and webhooks skip disabled acp",
			profile: core.Profile{
				ID:       "mix",
				Email:    "mix@example.com",
				ACP:      &core.ACPConfig{Enabled: false},
				Webhooks: core.WebhookConfig{AllowedEvents: []string{"deploy"}},
			},
			want: []string{ChannelEmail, ChannelWebhooks},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Channels(tt.profile)
			if got == nil {
				t.Fatal("Channels returned nil, want non-nil slice")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Channels() = %v, want %v", got, tt.want)
			}
		})
	}
}
