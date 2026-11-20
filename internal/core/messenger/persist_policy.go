package messenger

import "hop.top/aps/internal/logging"

// idKeys are the persisted-record keys that carry platform identifiers.
var idKeys = []string{"message_id", "sender_id", "channel_id", "thread_id"}

// verbatimIDPlatforms are platforms whose identifiers are opaque,
// non-personal tokens (workspace/chat/snowflake IDs, org/repo). Their
// ID fields are stored verbatim so audit and log records stay joinable.
var verbatimIDPlatforms = map[MessengerPlatform]bool{
	PlatformSlack:    true,
	PlatformTelegram: true,
	PlatformDiscord:  true,
	PlatformGitHub:   true,
}

// persistPolicy returns the JSONL redaction policy for records of the
// given platform. Fail closed: phone-number platforms (WhatsApp, SMS),
// email, and unknown or empty platforms keep ID fields under normal PII
// redaction.
func persistPolicy(platform string) logging.JSONLPolicy {
	if verbatimIDPlatforms[MessengerPlatform(platform)] {
		return logging.JSONLPolicy{}
	}
	return logging.JSONLPolicy{RedactKeys: idKeys}
}
