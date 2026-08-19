// Package msgroute implements sender-keyed route tables for message services.
//
// A route table replaces a message service's single default_action with an
// ordered list of routes evaluated against the normalized sender key (and,
// when a contacts source is configured, the resolved contact). The last route
// is a mandatory terminal fail-safe (`match: unknown`) so every inbound
// message resolves to exactly one profile action.
package msgroute

import (
	"strings"
)

// Transport prefixes providers prepend to a sender address.
const (
	prefixWhatsApp = "whatsapp:"
	prefixSMS      = "sms:"
	prefixTel      = "tel:"
	prefixMailto   = "mailto:"
)

// channelPrefixes are stripped case-insensitively before any other rule.
var channelPrefixes = []string{prefixWhatsApp, prefixSMS, prefixTel, prefixMailto}

// phonePrefixes are the subset of channel prefixes that imply the remaining
// address is a phone number even on a non-phone platform.
var phonePrefixes = []string{prefixWhatsApp, prefixSMS, prefixTel}

// Platforms whose sender IDs are phone numbers.
const (
	platformSMS      = "sms"
	platformWhatsApp = "whatsapp"
)

// phonePlatforms are the platforms whose sender IDs are phone numbers.
var phonePlatforms = map[string]bool{
	platformSMS:      true,
	platformWhatsApp: true,
}

// IsPhonePlatform reports whether sender IDs on the platform are phone numbers.
func IsPhonePlatform(platform string) bool {
	return phonePlatforms[strings.ToLower(strings.TrimSpace(platform))]
}

// NormalizeSenderKey canonicalizes a raw sender identifier into the key used
// for route matching and contact lookup.
//
// Rules, in order:
//  1. Trim whitespace and strip one leading channel prefix (whatsapp:, sms:,
//     tel:, mailto:) case-insensitively.
//  2. Email-shaped values (containing @) are lowercased.
//  3. Phone-shaped values (digits with optional +, spaces, dashes, dots,
//     parentheses) have separators removed. On phone platforms, or when a
//     phone prefix was stripped, a missing leading + is added so Twilio
//     (`whatsapp:+1555…`) and WhatsApp Cloud (`1555…`) senders share one key.
//  4. Anything else (platform user IDs, handles) is returned as-is.
//
// The function is idempotent: normalizing an already-normalized key is a no-op.
func NormalizeSenderKey(platform, raw string) string {
	return normalize(platform, raw, false)
}

// NormalizeSenderPattern applies NormalizeSenderKey rules to a route pattern
// while preserving glob metacharacters (* ? [ ]) so a pattern such as
// `whatsapp:+1 555 *` compares against normalized keys as `+1555*`.
func NormalizeSenderPattern(platform, raw string) string {
	return normalize(platform, raw, true)
}

func normalize(platform, raw string, pattern bool) string {
	value := strings.TrimSpace(raw)
	value, phoneHint := stripChannelPrefix(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "@") {
		return strings.ToLower(value)
	}
	if isPhoneShaped(value, pattern) {
		return canonicalPhone(value, phoneHint || IsPhonePlatform(platform))
	}
	return value
}

// stripChannelPrefix removes one leading channel prefix and reports whether
// that prefix implies a phone address.
func stripChannelPrefix(value string) (string, bool) {
	lower := strings.ToLower(value)
	for _, prefix := range channelPrefixes {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		rest := strings.TrimSpace(value[len(prefix):])
		phone := false
		for _, p := range phonePrefixes {
			if p == prefix {
				phone = true
				break
			}
		}
		return rest, phone
	}
	return value, false
}

// isPhoneShaped reports whether value looks like a phone number: an optional
// leading +, then digits and separators. With pattern set, glob metacharacters
// are tolerated so wildcards do not defeat canonicalization.
func isPhoneShaped(value string, pattern bool) bool {
	digits := 0
	for i, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' && i == 0:
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		case pattern && (r == '*' || r == '?'):
		default:
			return false
		}
	}
	return digits > 0 || (pattern && strings.ContainsAny(value, "*?"))
}

// canonicalPhone strips separators and, when addPlus is set and the value
// starts with a digit, prepends + so E.164-with-plus and bare-digit forms
// compare equal. Patterns that start with a wildcard are left alone.
func canonicalPhone(value string, addPlus bool) string {
	var b strings.Builder
	b.Grow(len(value) + 1)
	for _, r := range value {
		switch r {
		case ' ', '-', '.', '(', ')':
			continue
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if addPlus && out != "" && out[0] >= '0' && out[0] <= '9' {
		out = "+" + out
	}
	return out
}
