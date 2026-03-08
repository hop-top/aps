package messenger

import (
	"fmt"
	"strconv"
	"time"

	msgtypes "hop.top/aps/internal/core/messenger"
)

// Normalizer transforms platform-specific webhook events into the unified
// NormalizedMessage format and converts ActionResult back to platform-specific
// response payloads.
type Normalizer struct{}

// NewNormalizer returns a new Normalizer instance.
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// Normalize converts a raw platform event (JSON-decoded map) into a NormalizedMessage.
// It dispatches to platform-specific normalizers based on the platform argument.
func (n *Normalizer) Normalize(platform string, raw map[string]any) (*msgtypes.NormalizedMessage, error) {
	if raw == nil {
		return nil, msgtypes.ErrNormalizeFailed(platform, fmt.Errorf("raw event is nil"))
	}

	var msg *msgtypes.NormalizedMessage
	var err error

	switch msgtypes.MessengerPlatform(platform) {
	case msgtypes.PlatformTelegram:
		msg, err = n.normalizeTelegram(raw)
	case msgtypes.PlatformSlack:
		msg, err = n.normalizeSlack(raw)
	case msgtypes.PlatformGitHub:
		msg, err = n.normalizeGitHub(raw)
	case msgtypes.PlatformEmail:
		msg, err = n.normalizeEmail(raw)
	default:
		return nil, msgtypes.ErrNormalizeFailed(platform, fmt.Errorf("unsupported platform %q", platform))
	}

	if err != nil {
		return nil, msgtypes.ErrNormalizeFailed(platform, err)
	}

	if err := msg.Validate(); err != nil {
		return nil, msgtypes.ErrNormalizeFailed(platform, err)
	}

	return msg, nil
}

// normalizeTelegram extracts fields from a Telegram Bot API webhook update.
// Expected structure: {"message": {"from": {...}, "chat": {...}, "text": "..."}}
func (n *Normalizer) normalizeTelegram(raw map[string]any) (*msgtypes.NormalizedMessage, error) {
	message := getMap(raw, "message")
	if message == nil {
		// Some Telegram events use edited_message or callback_query.
		message = getMap(raw, "edited_message")
	}
	if message == nil {
		return nil, fmt.Errorf("no message or edited_message field in telegram event")
	}

	from := getMap(message, "from")
	chat := getMap(message, "chat")
	if from == nil || chat == nil {
		return nil, fmt.Errorf("missing from or chat in telegram message")
	}

	senderID := formatInt64(getInt64(from, "id"))
	chatID := formatInt64(getInt64(chat, "id"))

	// Determine channel type from chat.type field.
	chatType := getString(chat, "type")
	channelType := "direct"
	switch chatType {
	case "group", "supergroup":
		channelType = "group"
	case "channel":
		channelType = "broadcast"
	}

	msg := &msgtypes.NormalizedMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC(),
		Platform:  string(msgtypes.PlatformTelegram),
		Sender: msgtypes.Sender{
			ID:             senderID,
			Name:           buildTelegramName(getString(from, "first_name"), getString(from, "last_name")),
			PlatformHandle: getString(from, "username"),
			PlatformID:     senderID,
		},
		Channel: msgtypes.Channel{
			ID:         chatID,
			Name:       getString(chat, "title"),
			Type:       channelType,
			PlatformID: chatID,
		},
		Text:             getString(message, "text"),
		PlatformMetadata: raw,
	}

	// Extract reply_to_message as thread context.
	if replyTo := getMap(message, "reply_to_message"); replyTo != nil {
		msgID := getInt64(replyTo, "message_id")
		if msgID != 0 {
			msg.Thread = &msgtypes.Thread{
				ID:   formatInt64(msgID),
				Type: "reply",
			}
		}
	}

	return msg, nil
}

// normalizeSlack extracts fields from a Slack Events API event envelope.
// Expected structure: {"event": {"user": "...", "channel": "...", "text": "...", "ts": "..."}}
func (n *Normalizer) normalizeSlack(raw map[string]any) (*msgtypes.NormalizedMessage, error) {
	event := getMap(raw, "event")
	if event == nil {
		return nil, fmt.Errorf("no event field in slack payload")
	}

	userID := getString(event, "user")
	channelID := getString(event, "channel")
	if userID == "" || channelID == "" {
		return nil, fmt.Errorf("missing user or channel in slack event")
	}

	// Determine channel type from channel_type field.
	channelType := getString(event, "channel_type")
	normalizedType := "group"
	switch channelType {
	case "im":
		normalizedType = "direct"
	case "mpim":
		normalizedType = "group"
	case "channel", "group":
		normalizedType = "group"
	}

	msg := &msgtypes.NormalizedMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC(),
		Platform:  string(msgtypes.PlatformSlack),
		Sender: msgtypes.Sender{
			ID:         userID,
			PlatformID: userID,
		},
		Channel: msgtypes.Channel{
			ID:         channelID,
			Name:       getString(event, "channel"),
			Type:       normalizedType,
			PlatformID: channelID,
		},
		Text:             getString(event, "text"),
		PlatformMetadata: raw,
	}

	// Thread support via thread_ts.
	threadTS := getString(event, "thread_ts")
	if threadTS != "" {
		msg.Thread = &msgtypes.Thread{
			ID:   threadTS,
			Type: "reply",
		}
	}

	// Extract files as attachments.
	if files, ok := event["files"].([]any); ok {
		for _, f := range files {
			if fileMap, ok := f.(map[string]any); ok {
				att := msgtypes.Attachment{
					Type:     getString(fileMap, "filetype"),
					URL:      getString(fileMap, "url_private"),
					MimeType: getString(fileMap, "mimetype"),
				}
				if att.URL != "" {
					msg.Attachments = append(msg.Attachments, att)
				}
			}
		}
	}

	return msg, nil
}

// normalizeGitHub extracts fields from a GitHub webhook event.
// Expected structure: {"action": "...", "sender": {"login": "..."}, "repository": {"full_name": "..."}}
func (n *Normalizer) normalizeGitHub(raw map[string]any) (*msgtypes.NormalizedMessage, error) {
	sender := getMap(raw, "sender")
	repo := getMap(raw, "repository")
	if sender == nil {
		return nil, fmt.Errorf("missing sender in github event")
	}

	senderLogin := getString(sender, "login")
	senderID := formatInt64(getInt64(sender, "id"))
	if senderID == "0" {
		senderID = senderLogin
	}

	// Use repository full_name as the channel ID (matches ChannelIDFormat).
	channelID := ""
	channelName := ""
	if repo != nil {
		channelID = getString(repo, "full_name")
		channelName = getString(repo, "name")
	}
	if channelID == "" {
		// Fallback for org-level events.
		if org := getMap(raw, "organization"); org != nil {
			channelID = getString(org, "login")
			channelName = channelID
		}
	}
	if channelID == "" {
		return nil, fmt.Errorf("unable to determine channel (repository or organization) from github event")
	}

	action := getString(raw, "action")

	// Build text from event type and action. Try to extract a comment body
	// for comment-related events.
	text := action
	if comment := getMap(raw, "comment"); comment != nil {
		body := getString(comment, "body")
		if body != "" {
			text = body
		}
	} else if issue := getMap(raw, "issue"); issue != nil {
		title := getString(issue, "title")
		if title != "" {
			text = fmt.Sprintf("[%s] %s", action, title)
		}
	} else if pr := getMap(raw, "pull_request"); pr != nil {
		title := getString(pr, "title")
		if title != "" {
			text = fmt.Sprintf("[%s] %s", action, title)
		}
	}

	msg := &msgtypes.NormalizedMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC(),
		Platform:  string(msgtypes.PlatformGitHub),
		Sender: msgtypes.Sender{
			ID:             senderID,
			Name:           senderLogin,
			PlatformHandle: senderLogin,
			PlatformID:     senderID,
		},
		Channel: msgtypes.Channel{
			ID:         channelID,
			Name:       channelName,
			Type:       "topic",
			PlatformID: channelID,
		},
		Text:             text,
		PlatformMetadata: raw,
	}

	// Use issue or PR number as thread.
	if issue := getMap(raw, "issue"); issue != nil {
		number := getInt64(issue, "number")
		if number != 0 {
			msg.Thread = &msgtypes.Thread{
				ID:   formatInt64(number),
				Type: "issue",
			}
		}
	} else if pr := getMap(raw, "pull_request"); pr != nil {
		number := getInt64(pr, "number")
		if number != 0 {
			msg.Thread = &msgtypes.Thread{
				ID:   formatInt64(number),
				Type: "issue",
			}
		}
	}

	return msg, nil
}

// normalizeEmail extracts fields from an email event.
// Expected structure: {"from": "...", "to": "...", "subject": "...", "body": "..."}
func (n *Normalizer) normalizeEmail(raw map[string]any) (*msgtypes.NormalizedMessage, error) {
	from := getString(raw, "from")
	to := getString(raw, "to")
	if from == "" {
		return nil, fmt.Errorf("missing from field in email event")
	}
	if to == "" {
		return nil, fmt.Errorf("missing to field in email event")
	}

	subject := getString(raw, "subject")
	body := getString(raw, "body")

	msg := &msgtypes.NormalizedMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC(),
		Platform:  string(msgtypes.PlatformEmail),
		Sender: msgtypes.Sender{
			ID:             from,
			Name:           from,
			PlatformHandle: from,
			PlatformID:     from,
		},
		Channel: msgtypes.Channel{
			ID:         to,
			Name:       to,
			Type:       "direct",
			PlatformID: to,
		},
		Text:             body,
		PlatformMetadata: raw,
	}

	// Use subject as thread context if present.
	if subject != "" {
		msg.Thread = &msgtypes.Thread{
			ID:   subject,
			Type: "topic",
		}
	}

	// Extract attachments if present.
	if attachments, ok := raw["attachments"].([]any); ok {
		for _, a := range attachments {
			if attMap, ok := a.(map[string]any); ok {
				att := msgtypes.Attachment{
					Type:     getString(attMap, "type"),
					URL:      getString(attMap, "url"),
					MimeType: getString(attMap, "mime_type"),
				}
				if sizeBytes := getInt64(attMap, "size_bytes"); sizeBytes > 0 {
					att.SizeBytes = sizeBytes
				}
				if att.URL != "" || att.Type != "" {
					msg.Attachments = append(msg.Attachments, att)
				}
			}
		}
	}

	return msg, nil
}

// Denormalize converts an ActionResult into a platform-specific response payload
// suitable for sending back through the platform's API.
func (n *Normalizer) Denormalize(platform string, result *ActionResult) (map[string]any, error) {
	if result == nil {
		return nil, fmt.Errorf("action result is nil")
	}

	switch msgtypes.MessengerPlatform(platform) {
	case msgtypes.PlatformTelegram:
		return n.denormalizeTelegram(result), nil
	case msgtypes.PlatformSlack:
		return n.denormalizeSlack(result), nil
	case msgtypes.PlatformGitHub:
		return n.denormalizeGitHub(result), nil
	case msgtypes.PlatformEmail:
		return n.denormalizeEmail(result), nil
	default:
		return map[string]any{
			"status": result.Status,
			"output": result.Output,
		}, nil
	}
}

func (n *Normalizer) denormalizeTelegram(result *ActionResult) map[string]any {
	resp := map[string]any{
		"method":     "sendMessage",
		"text":       result.Output,
		"parse_mode": "Markdown",
	}
	if result.OutputData != nil {
		resp["data"] = result.OutputData
	}
	return resp
}

func (n *Normalizer) denormalizeSlack(result *ActionResult) map[string]any {
	resp := map[string]any{
		"response_type": "in_channel",
		"text":          result.Output,
	}
	if result.OutputData != nil {
		resp["blocks"] = result.OutputData
	}
	return resp
}

func (n *Normalizer) denormalizeGitHub(result *ActionResult) map[string]any {
	resp := map[string]any{
		"body": result.Output,
	}
	if result.Status == "failed" {
		resp["state"] = "failure"
	} else {
		resp["state"] = "success"
	}
	return resp
}

func (n *Normalizer) denormalizeEmail(result *ActionResult) map[string]any {
	resp := map[string]any{
		"body":    result.Output,
		"subject": "Re: Action Result",
	}
	if result.OutputData != nil {
		resp["data"] = result.OutputData
	}
	return resp
}

// getString safely extracts a string value from a map. Returns empty string
// if the key is absent or the value is not a string.
func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		if s == float64(int64(s)) {
			return strconv.FormatInt(int64(s), 10)
		}
		return strconv.FormatFloat(s, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// getInt64 safely extracts an int64 value from a map. JSON numbers are
// typically decoded as float64, so this handles that conversion.
func getInt64(m map[string]any, key string) int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	default:
		return 0
	}
}

// getMap safely extracts a nested map from a map. Returns nil if the key
// is absent or the value is not a map.
func getMap(m map[string]any, key string) map[string]any {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	if sub, ok := v.(map[string]any); ok {
		return sub
	}
	return nil
}

// formatInt64 converts an int64 to its string representation.
func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

// buildTelegramName combines first and last name fields.
func buildTelegramName(first, last string) string {
	if last == "" {
		return first
	}
	if first == "" {
		return last
	}
	return first + " " + last
}
