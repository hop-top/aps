package core

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

	"hop.top/kit/go/core/avatar"

	"hop.top/aps/internal/events"
)

// avatarPalette is a curated set of accessible mid-tone colors readable
// on both light and dark backgrounds. Indexed deterministically by the
// profile id so the same id always lands on the same color.
//
// Sourced from Tailwind CSS 500-shade equivalents. Keep this list small
// and visually distinct; collisions are acceptable beyond ~12 profiles.
var avatarPalette = []string{
	"#3b82f6", // blue
	"#10b981", // emerald
	"#f59e0b", // amber
	"#ef4444", // red
	"#8b5cf6", // violet
	"#ec4899", // pink
	"#14b8a6", // teal
	"#f97316", // orange
	"#6366f1", // indigo
	"#84cc16", // lime
	"#06b6d4", // cyan
	"#a855f7", // purple
}

// GenerateProfileColor returns a deterministic color for the given id,
// drawn from avatarPalette. Empty id returns the first palette entry.
func GenerateProfileColor(id string) string {
	if id == "" {
		return avatarPalette[0]
	}
	sum := sha256.Sum256([]byte(id))
	idx := binary.BigEndian.Uint32(sum[:4]) % uint32(len(avatarPalette))
	return avatarPalette[idx]
}

// GenerateProfileAvatar returns a deterministic avatar URL via the
// kit/avatar facade. cfg controls provider/style/size/format; an empty
// cfg uses kit/avatar defaults (currently dicebear with style "shapes").
//
// On error (unknown provider, missing seed) it returns an empty string;
// callers treat absence as "no avatar set" rather than fail-on-config.
func GenerateProfileAvatar(id string, cfg ProfileAvatarConfig) string {
	url, err := avatar.Generate(context.Background(), avatar.Options{
		Provider: cfg.Provider,
		Seed:     id,
		Style:    cfg.Style,
		Size:     cfg.Size,
		Format:   cfg.Format,
	})
	if err != nil {
		return ""
	}
	return url
}

// PublishProfileUpdated emits a ProfileUpdated event listing the changed fields.
// This is the public wrapper around the package-private publish() so callers
// outside core (notably the CLI edit command) can announce mutations without
// duplicating the bus plumbing.
func PublishProfileUpdated(profileID string, fields []string) {
	publish(context.Background(), string(events.TopicProfileUpdated), "", events.ProfileUpdatedPayload{
		ProfileID: profileID,
		Fields:    fields,
	})
}
