package chat

import (
	"fmt"
	"strings"

	"hop.top/aps/internal/core"
	corechat "hop.top/aps/internal/core/chat"
)

// ParseParticipants combines the primary profile argument, comma shorthand,
// and repeated/comma-separated --invite values into an ordered, deduplicated
// participant list.
func ParseParticipants(primary string, invites []string) ([]string, error) {
	ids, err := splitProfileRefs(primary)
	if err != nil {
		return nil, err
	}
	for _, invite := range invites {
		parsed, err := splitProfileRefs(invite)
		if err != nil {
			return nil, err
		}
		ids = append(ids, parsed...)
	}

	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one profile is required")
	}
	return out, nil
}

// ParseMetaCommand recognizes chat REPL control lines.
func ParseMetaCommand(line string) (corechat.MetaCommand, bool) {
	switch strings.TrimSpace(line) {
	case ":auto":
		return corechat.MetaCommandAuto, true
	case ":human":
		return corechat.MetaCommandHuman, true
	case ":done":
		return corechat.MetaCommandDone, true
	default:
		return corechat.MetaCommandNone, false
	}
}

func splitProfileRefs(value string) ([]string, error) {
	fields := strings.Split(value, ",")
	ids := make([]string, 0, len(fields))
	for _, field := range fields {
		ref := strings.TrimSpace(field)
		if ref == "" {
			continue
		}
		id, err := core.ParseProfileRef(ref)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
