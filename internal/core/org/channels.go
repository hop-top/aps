// Package org derives organizational views from profile configuration.
package org

import "hop.top/aps/internal/core"

// Channel names a profile can be reached through, in the fixed order
// Channels emits them.
const (
	ChannelA2A      = "a2a"
	ChannelACP      = "acp"
	ChannelEmail    = "email"
	ChannelWebhooks = "webhooks"
)

// Channels returns the communication channels reachable from the profile's
// configuration alone, in a stable fixed order (a2a, acp, email, webhooks).
// It always returns a non-nil slice.
//
// Derivation rules:
//   - a2a: A2A config present (A2AConfig has no enabled flag; presence means
//     the protocol is configured).
//   - acp: ACP config present and Enabled set.
//   - email: Email non-empty.
//   - webhooks: at least one allowed event configured.
//
// Messenger links (WhatsApp, Telegram, Discord, SMS) are intentionally
// excluded: they live in a per-profile filesystem link store
// (internal/core/messenger.LinkStore), not on the Profile struct, so
// deriving them would require probing adapter/runtime state and break the
// pure-function contract of this helper.
func Channels(p core.Profile) []string {
	channels := []string{}
	if p.A2A != nil {
		channels = append(channels, ChannelA2A)
	}
	if p.ACP != nil && p.ACP.Enabled {
		channels = append(channels, ChannelACP)
	}
	if p.Email != "" {
		channels = append(channels, ChannelEmail)
	}
	if len(p.Webhooks.AllowedEvents) > 0 {
		channels = append(channels, ChannelWebhooks)
	}
	return channels
}
