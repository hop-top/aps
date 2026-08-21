package core

import (
	"fmt"
	"net/mail"
	"strings"
)

// OptionAllowedSenders is the message service option holding the
// comma-separated sender allowlist for address-keyed adapters (email).
//
// Entry semantics:
//   - exact address: "alice@example.com"
//   - domain glob:   "*@example.com" (matches any local part, whole domain only)
//
// Matching is case-insensitive on both local part and domain. A sender
// given in display-name form ("Alice <alice@example.com>") is compared on
// the bare address.
const OptionAllowedSenders = "allowed_senders"

// ValidateAllowedSenderPattern reports whether a single allowed_senders
// entry is well-formed (exact address or *@domain glob).
func ValidateAllowedSenderPattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	local, domain, ok := strings.Cut(pattern, "@")
	if !ok || local == "" || domain == "" || strings.ContainsAny(domain, "*@ ") {
		return fmt.Errorf("allowed_senders entry %q must be an exact address or a *@domain glob", pattern)
	}
	if strings.Contains(local, "*") && local != "*" {
		return fmt.Errorf("allowed_senders entry %q must be an exact address or a *@domain glob", pattern)
	}
	return nil
}

// MatchAllowedSender reports whether any candidate address matches one of
// the allowed_senders patterns. Empty candidates are skipped; malformed
// patterns never match.
func MatchAllowedSender(patterns []string, candidates ...string) bool {
	for _, candidate := range candidates {
		address := normalizeEmailAddress(candidate)
		if address == "" {
			continue
		}
		local, domain, ok := strings.Cut(address, "@")
		if !ok {
			continue
		}
		for _, pattern := range patterns {
			pattern = strings.ToLower(strings.TrimSpace(pattern))
			pLocal, pDomain, ok := strings.Cut(pattern, "@")
			if !ok || pDomain == "" || pDomain != domain {
				continue
			}
			if pLocal == "*" || pLocal == local {
				return true
			}
		}
	}
	return false
}

// normalizeEmailAddress lowercases and strips a display name when present.
func normalizeEmailAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := mail.ParseAddress(value); err == nil {
		return strings.ToLower(parsed.Address)
	}
	return strings.ToLower(value)
}
