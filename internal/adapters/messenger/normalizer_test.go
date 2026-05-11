package messenger

import (
	"testing"
	"time"

	msgtypes "hop.top/aps/internal/core/messenger"
)

func TestNormalizer_NormalizeTelegram(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		name    string
		raw     map[string]any
		wantErr bool
		check   func(t *testing.T, msg *msgtypes.NormalizedMessage)
	}{
		{
			name: "typical group message",
			raw: map[string]any{
				"message": map[string]any{
					"message_id": float64(123),
					"from": map[string]any{
						"id":         float64(456),
						"first_name": "Alice",
						"username":   "alice_bot",
					},
					"chat": map[string]any{
						"id":    float64(-1001234567890),
						"title": "Research Team",
						"type":  "group",
					},
					"text": "Hello research agent!",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Platform != "telegram" {
					t.Errorf("platform = %q, want %q", msg.Platform, "telegram")
				}
				if msg.Sender.ID != "456" {
					t.Errorf("sender.ID = %q, want %q", msg.Sender.ID, "456")
				}
				if msg.Sender.Name != "Alice" {
					t.Errorf("sender.Name = %q, want %q", msg.Sender.Name, "Alice")
				}
				if msg.Sender.PlatformHandle != "alice_bot" {
					t.Errorf("sender.PlatformHandle = %q, want %q", msg.Sender.PlatformHandle, "alice_bot")
				}
				if msg.Channel.ID != "-1001234567890" {
					t.Errorf("channel.ID = %q, want %q", msg.Channel.ID, "-1001234567890")
				}
				if msg.Channel.Name != "Research Team" {
					t.Errorf("channel.Name = %q, want %q", msg.Channel.Name, "Research Team")
				}
				if msg.Channel.Type != "group" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "group")
				}
				if msg.Text != "Hello research agent!" {
					t.Errorf("text = %q, want %q", msg.Text, "Hello research agent!")
				}
				if msg.Thread != nil {
					t.Errorf("thread should be nil for non-reply message")
				}
			},
		},
		{
			name: "private message",
			raw: map[string]any{
				"message": map[string]any{
					"message_id": float64(789),
					"from": map[string]any{
						"id":         float64(111),
						"first_name": "Bob",
						"last_name":  "Smith",
						"username":   "bob_s",
					},
					"chat": map[string]any{
						"id":   float64(111),
						"type": "private",
					},
					"text": "private hello",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Channel.Type != "direct" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "direct")
				}
				if msg.Sender.Name != "Bob Smith" {
					t.Errorf("sender.Name = %q, want %q", msg.Sender.Name, "Bob Smith")
				}
			},
		},
		{
			name: "supergroup message",
			raw: map[string]any{
				"message": map[string]any{
					"message_id": float64(1),
					"from": map[string]any{
						"id":         float64(222),
						"first_name": "Eve",
					},
					"chat": map[string]any{
						"id":    float64(-1009876543210),
						"title": "Big Group",
						"type":  "supergroup",
					},
					"text": "hi",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Channel.Type != "group" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "group")
				}
			},
		},
		{
			name: "channel message",
			raw: map[string]any{
				"message": map[string]any{
					"message_id": float64(1),
					"from": map[string]any{
						"id":         float64(333),
						"first_name": "ChannelBot",
					},
					"chat": map[string]any{
						"id":    float64(-1005555555555),
						"title": "Announcements",
						"type":  "channel",
					},
					"text": "announcement",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Channel.Type != "broadcast" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "broadcast")
				}
			},
		},
		{
			name: "reply message with thread",
			raw: map[string]any{
				"message": map[string]any{
					"message_id": float64(200),
					"from": map[string]any{
						"id":         float64(456),
						"first_name": "Alice",
					},
					"chat": map[string]any{
						"id":   float64(-1001234567890),
						"type": "group",
					},
					"text": "replying here",
					"reply_to_message": map[string]any{
						"message_id": float64(100),
					},
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Thread == nil {
					t.Fatal("thread should not be nil for reply message")
				}
				if msg.Thread.ID != "100" {
					t.Errorf("thread.ID = %q, want %q", msg.Thread.ID, "100")
				}
				if msg.Thread.Type != "reply" {
					t.Errorf("thread.Type = %q, want %q", msg.Thread.Type, "reply")
				}
			},
		},
		{
			name: "edited message",
			raw: map[string]any{
				"edited_message": map[string]any{
					"message_id": float64(50),
					"from": map[string]any{
						"id":         float64(456),
						"first_name": "Alice",
					},
					"chat": map[string]any{
						"id":   float64(-1001234567890),
						"type": "group",
					},
					"text": "edited text",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Text != "edited text" {
					t.Errorf("text = %q, want %q", msg.Text, "edited text")
				}
			},
		},
		{
			name:    "missing message field",
			raw:     map[string]any{"update_id": float64(12345)},
			wantErr: true,
		},
		{
			name: "missing from field",
			raw: map[string]any{
				"message": map[string]any{
					"chat": map[string]any{
						"id":   float64(111),
						"type": "private",
					},
					"text": "no sender",
				},
			},
			wantErr: true,
		},
		{
			name: "missing chat field",
			raw: map[string]any{
				"message": map[string]any{
					"from": map[string]any{
						"id":         float64(456),
						"first_name": "Alice",
					},
					"text": "no chat",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := n.Normalize("telegram", tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if msg.ID == "" {
				t.Error("message ID should not be empty")
			}
			if msg.Timestamp.IsZero() {
				t.Error("timestamp should not be zero")
			}
			if msg.PlatformMetadata == nil {
				t.Error("platform metadata should be preserved")
			}
			tt.check(t, msg)
		})
	}
}

func TestNormalizer_NormalizeSlack(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		name    string
		raw     map[string]any
		wantErr bool
		check   func(t *testing.T, msg *msgtypes.NormalizedMessage)
	}{
		{
			name: "typical channel message",
			raw: map[string]any{
				"event": map[string]any{
					"user":         "U12345",
					"channel":      "C01ABC2DEF",
					"channel_type": "channel",
					"text":         "Hello from Slack!",
					"ts":           "1234567890.123456",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Platform != "slack" {
					t.Errorf("platform = %q, want %q", msg.Platform, "slack")
				}
				if msg.Sender.ID != "U12345" {
					t.Errorf("sender.ID = %q, want %q", msg.Sender.ID, "U12345")
				}
				if msg.Channel.ID != "C01ABC2DEF" {
					t.Errorf("channel.ID = %q, want %q", msg.Channel.ID, "C01ABC2DEF")
				}
				if msg.Channel.Type != "group" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "group")
				}
				if msg.Text != "Hello from Slack!" {
					t.Errorf("text = %q, want %q", msg.Text, "Hello from Slack!")
				}
				if msg.Thread != nil {
					t.Error("thread should be nil for non-threaded message")
				}
			},
		},
		{
			name: "direct message",
			raw: map[string]any{
				"event": map[string]any{
					"user":         "U99999",
					"channel":      "D01XYZ",
					"channel_type": "im",
					"text":         "DM text",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Channel.Type != "direct" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "direct")
				}
			},
		},
		{
			name: "threaded message",
			raw: map[string]any{
				"event": map[string]any{
					"user":      "U12345",
					"channel":   "C01ABC2DEF",
					"text":      "thread reply",
					"thread_ts": "1234567890.000001",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Thread == nil {
					t.Fatal("thread should not be nil for threaded message")
				}
				if msg.Thread.ID != "1234567890.000001" {
					t.Errorf("thread.ID = %q, want %q", msg.Thread.ID, "1234567890.000001")
				}
				if msg.Thread.Type != "reply" {
					t.Errorf("thread.Type = %q, want %q", msg.Thread.Type, "reply")
				}
			},
		},
		{
			name: "message with file attachments",
			raw: map[string]any{
				"event": map[string]any{
					"user":    "U12345",
					"channel": "C01ABC2DEF",
					"text":    "check this file",
					"files": []any{
						map[string]any{
							"filetype":    "pdf",
							"url_private": "https://files.slack.com/files/doc.pdf",
							"mimetype":    "application/pdf",
						},
						map[string]any{
							"filetype":    "png",
							"url_private": "https://files.slack.com/files/img.png",
							"mimetype":    "image/png",
						},
					},
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if len(msg.Attachments) != 2 {
					t.Fatalf("attachments count = %d, want 2", len(msg.Attachments))
				}
				if msg.Attachments[0].Type != "pdf" {
					t.Errorf("attachments[0].Type = %q, want %q", msg.Attachments[0].Type, "pdf")
				}
				if msg.Attachments[0].URL != "https://files.slack.com/files/doc.pdf" {
					t.Errorf("attachments[0].URL = %q, want url", msg.Attachments[0].URL)
				}
				if msg.Attachments[1].MimeType != "image/png" {
					t.Errorf("attachments[1].MimeType = %q, want %q", msg.Attachments[1].MimeType, "image/png")
				}
			},
		},
		{
			name: "multi-party IM",
			raw: map[string]any{
				"event": map[string]any{
					"user":         "U12345",
					"channel":      "G01MPIM",
					"channel_type": "mpim",
					"text":         "mpim message",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Channel.Type != "group" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "group")
				}
			},
		},
		{
			name: "app mention strips leading bot mention",
			raw: map[string]any{
				"team_id":  "T123",
				"event_id": "Ev123",
				"event": map[string]any{
					"type":         "app_mention",
					"user":         "U12345",
					"channel":      "C01ABC2DEF",
					"channel_type": "channel",
					"text":         "<@U999BOT> help me",
					"ts":           "1710000000.000001",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.ID != "Ev123" {
					t.Errorf("id = %q, want event id", msg.ID)
				}
				if msg.WorkspaceID != "T123" {
					t.Errorf("workspace = %q, want T123", msg.WorkspaceID)
				}
				if msg.Text != "help me" {
					t.Errorf("text = %q, want stripped mention", msg.Text)
				}
				if mentioned, _ := msg.PlatformMetadata["slack_bot_mentioned"].(bool); !mentioned {
					t.Errorf("slack_bot_mentioned metadata = false, want true")
				}
			},
		},
		{
			name:    "missing event field",
			raw:     map[string]any{"token": "xoxb-abc"},
			wantErr: true,
		},
		{
			name: "missing user field",
			raw: map[string]any{
				"event": map[string]any{
					"channel": "C01ABC2DEF",
					"text":    "no user",
				},
			},
			wantErr: true,
		},
		{
			name: "missing channel field",
			raw: map[string]any{
				"event": map[string]any{
					"user": "U12345",
					"text": "no channel",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := n.Normalize("slack", tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if msg.ID == "" {
				t.Error("message ID should not be empty")
			}
			tt.check(t, msg)
		})
	}
}

func TestNormalizer_NormalizeGitHub(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		name    string
		raw     map[string]any
		wantErr bool
		check   func(t *testing.T, msg *msgtypes.NormalizedMessage)
	}{
		{
			name: "issue opened event",
			raw: map[string]any{
				"action": "opened",
				"sender": map[string]any{
					"login": "octocat",
					"id":    float64(1),
				},
				"repository": map[string]any{
					"full_name": "octocat/hello-world",
					"name":      "hello-world",
				},
				"issue": map[string]any{
					"number": float64(42),
					"title":  "Found a bug",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Platform != "github" {
					t.Errorf("platform = %q, want %q", msg.Platform, "github")
				}
				if msg.Sender.ID != "1" {
					t.Errorf("sender.ID = %q, want %q", msg.Sender.ID, "1")
				}
				if msg.Sender.Name != "octocat" {
					t.Errorf("sender.Name = %q, want %q", msg.Sender.Name, "octocat")
				}
				if msg.Sender.PlatformHandle != "octocat" {
					t.Errorf("sender.PlatformHandle = %q, want %q", msg.Sender.PlatformHandle, "octocat")
				}
				if msg.Channel.ID != "octocat/hello-world" {
					t.Errorf("channel.ID = %q, want %q", msg.Channel.ID, "octocat/hello-world")
				}
				if msg.Channel.Name != "hello-world" {
					t.Errorf("channel.Name = %q, want %q", msg.Channel.Name, "hello-world")
				}
				if msg.Channel.Type != "topic" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "topic")
				}
				if msg.Text != "[opened] Found a bug" {
					t.Errorf("text = %q, want %q", msg.Text, "[opened] Found a bug")
				}
				if msg.Thread == nil {
					t.Fatal("thread should not be nil for issue event")
				}
				if msg.Thread.ID != "42" {
					t.Errorf("thread.ID = %q, want %q", msg.Thread.ID, "42")
				}
				if msg.Thread.Type != "issue" {
					t.Errorf("thread.Type = %q, want %q", msg.Thread.Type, "issue")
				}
			},
		},
		{
			name: "pull request opened event",
			raw: map[string]any{
				"action": "opened",
				"sender": map[string]any{
					"login": "dev1",
					"id":    float64(99),
				},
				"repository": map[string]any{
					"full_name": "org/repo",
					"name":      "repo",
				},
				"pull_request": map[string]any{
					"number": float64(10),
					"title":  "Add feature X",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Text != "[opened] Add feature X" {
					t.Errorf("text = %q, want %q", msg.Text, "[opened] Add feature X")
				}
				if msg.Thread == nil {
					t.Fatal("thread should not be nil for PR event")
				}
				if msg.Thread.ID != "10" {
					t.Errorf("thread.ID = %q, want %q", msg.Thread.ID, "10")
				}
			},
		},
		{
			name: "comment event extracts body",
			raw: map[string]any{
				"action": "created",
				"sender": map[string]any{
					"login": "reviewer",
					"id":    float64(55),
				},
				"repository": map[string]any{
					"full_name": "org/repo",
					"name":      "repo",
				},
				"comment": map[string]any{
					"body": "Looks good to me!",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Text != "Looks good to me!" {
					t.Errorf("text = %q, want %q", msg.Text, "Looks good to me!")
				}
			},
		},
		{
			name: "sender without numeric id uses login",
			raw: map[string]any{
				"action": "pushed",
				"sender": map[string]any{
					"login": "bot-user",
				},
				"repository": map[string]any{
					"full_name": "org/repo",
					"name":      "repo",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Sender.ID != "bot-user" {
					t.Errorf("sender.ID = %q, want %q (login fallback)", msg.Sender.ID, "bot-user")
				}
			},
		},
		{
			name: "organization-level event without repository",
			raw: map[string]any{
				"action": "member_added",
				"sender": map[string]any{
					"login": "admin",
					"id":    float64(1),
				},
				"organization": map[string]any{
					"login": "myorg",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Channel.ID != "myorg" {
					t.Errorf("channel.ID = %q, want %q", msg.Channel.ID, "myorg")
				}
			},
		},
		{
			name:    "missing sender",
			raw:     map[string]any{"action": "opened"},
			wantErr: true,
		},
		{
			name: "missing repository and organization",
			raw: map[string]any{
				"action": "something",
				"sender": map[string]any{
					"login": "user",
					"id":    float64(1),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := n.Normalize("github", tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if msg.ID == "" {
				t.Error("message ID should not be empty")
			}
			tt.check(t, msg)
		})
	}
}

func TestNormalizer_NormalizeEmail(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		name    string
		raw     map[string]any
		wantErr bool
		check   func(t *testing.T, msg *msgtypes.NormalizedMessage)
	}{
		{
			name: "typical email",
			raw: map[string]any{
				"from":    "alice@example.com",
				"to":      "bob@example.com",
				"subject": "Meeting Notes",
				"body":    "Here are the meeting notes.",
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Platform != "email" {
					t.Errorf("platform = %q, want %q", msg.Platform, "email")
				}
				if msg.Sender.ID != "alice@example.com" {
					t.Errorf("sender.ID = %q, want %q", msg.Sender.ID, "alice@example.com")
				}
				if msg.Sender.Name != "alice@example.com" {
					t.Errorf("sender.Name = %q, want %q", msg.Sender.Name, "alice@example.com")
				}
				if msg.Channel.ID != "bob@example.com" {
					t.Errorf("channel.ID = %q, want %q", msg.Channel.ID, "bob@example.com")
				}
				if msg.Channel.Type != "direct" {
					t.Errorf("channel.Type = %q, want %q", msg.Channel.Type, "direct")
				}
				if msg.Text != "Here are the meeting notes." {
					t.Errorf("text = %q, want %q", msg.Text, "Here are the meeting notes.")
				}
				if msg.Thread == nil {
					t.Fatal("thread should not be nil when subject is set")
				}
				if msg.Thread.ID != "Meeting Notes" {
					t.Errorf("thread.ID = %q, want %q", msg.Thread.ID, "Meeting Notes")
				}
				if msg.Thread.Type != "topic" {
					t.Errorf("thread.Type = %q, want %q", msg.Thread.Type, "topic")
				}
			},
		},
		{
			name: "email without subject",
			raw: map[string]any{
				"from": "alice@example.com",
				"to":   "bob@example.com",
				"body": "No subject here.",
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Thread != nil {
					t.Error("thread should be nil when no subject")
				}
			},
		},
		{
			name: "email with attachments",
			raw: map[string]any{
				"from":    "alice@example.com",
				"to":      "bob@example.com",
				"subject": "Files",
				"body":    "See attached.",
				"attachments": []any{
					map[string]any{
						"type":       "document",
						"url":        "https://files.example.com/report.pdf",
						"mime_type":  "application/pdf",
						"size_bytes": float64(1024),
					},
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if len(msg.Attachments) != 1 {
					t.Fatalf("attachments count = %d, want 1", len(msg.Attachments))
				}
				if msg.Attachments[0].Type != "document" {
					t.Errorf("attachments[0].Type = %q, want %q", msg.Attachments[0].Type, "document")
				}
				if msg.Attachments[0].URL != "https://files.example.com/report.pdf" {
					t.Errorf("attachments[0].URL = %q, want url", msg.Attachments[0].URL)
				}
				if msg.Attachments[0].SizeBytes != 1024 {
					t.Errorf("attachments[0].SizeBytes = %d, want 1024", msg.Attachments[0].SizeBytes)
				}
			},
		},
		{
			name: "missing from",
			raw: map[string]any{
				"to":   "bob@example.com",
				"body": "no sender",
			},
			wantErr: true,
		},
		{
			name: "missing to",
			raw: map[string]any{
				"from": "alice@example.com",
				"body": "no recipient",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := n.Normalize("email", tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if msg.ID == "" {
				t.Error("message ID should not be empty")
			}
			tt.check(t, msg)
		})
	}
}

func TestNormalizer_NormalizeDiscord(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		name    string
		raw     map[string]any
		wantErr bool
		check   func(t *testing.T, msg *msgtypes.NormalizedMessage)
	}{
		{
			name: "guild channel message",
			raw: map[string]any{
				"id":         "1100000000000000001",
				"channel_id": "1200000000000000002",
				"guild_id":   "1300000000000000003",
				"content":    "Hello from Discord",
				"author": map[string]any{
					"id":          "1400000000000000004",
					"username":    "alice",
					"global_name": "Alice A.",
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Platform != "discord" {
					t.Errorf("platform = %q, want discord", msg.Platform)
				}
				if msg.WorkspaceID != "1300000000000000003" {
					t.Errorf("workspaceID = %q, want guild id", msg.WorkspaceID)
				}
				if msg.Sender.ID != "1400000000000000004" {
					t.Errorf("sender.ID = %q, want author id", msg.Sender.ID)
				}
				if msg.Sender.Name != "Alice A." {
					t.Errorf("sender.Name = %q, want global name", msg.Sender.Name)
				}
				if msg.Channel.ID != "1200000000000000002" {
					t.Errorf("channel.ID = %q, want channel id", msg.Channel.ID)
				}
				if msg.Channel.Type != "group" {
					t.Errorf("channel.Type = %q, want group", msg.Channel.Type)
				}
				if msg.Text != "Hello from Discord" {
					t.Errorf("text = %q, want message content", msg.Text)
				}
			},
		},
		{
			name: "gateway message create envelope",
			raw: map[string]any{
				"op": float64(0),
				"t":  "MESSAGE_CREATE",
				"d": map[string]any{
					"id":         "1100000000000000001",
					"channel_id": "1200000000000000002",
					"guild_id":   "1300000000000000003",
					"content":    "Gateway hello",
					"timestamp":  "2026-05-11T12:00:00.000000+00:00",
					"author": map[string]any{
						"id":       "1400000000000000004",
						"username": "alice",
					},
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.ID != "1100000000000000001" {
					t.Errorf("id = %q, want gateway message id", msg.ID)
				}
				if msg.Text != "Gateway hello" {
					t.Errorf("text = %q, want Gateway hello", msg.Text)
				}
				if msg.Timestamp.Format(time.RFC3339) != "2026-05-11T12:00:00Z" {
					t.Errorf("timestamp = %s, want gateway timestamp", msg.Timestamp.Format(time.RFC3339))
				}
			},
		},
		{
			name: "direct message with reply and attachment",
			raw: map[string]any{
				"id":         "1100000000000000001",
				"channel_id": "1200000000000000002",
				"content":    "see attached",
				"author": map[string]any{
					"id":       "1400000000000000004",
					"username": "alice",
				},
				"message_reference": map[string]any{
					"message_id": "1000000000000000000",
				},
				"attachments": []any{
					map[string]any{
						"url":          "https://cdn.discordapp.com/file.png",
						"content_type": "image/png",
						"filename":     "file.png",
						"size":         float64(2048),
					},
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Channel.Type != "direct" {
					t.Errorf("channel.Type = %q, want direct", msg.Channel.Type)
				}
				if msg.Thread == nil || msg.Thread.ID != "1000000000000000000" {
					t.Fatalf("thread = %#v, want reply thread", msg.Thread)
				}
				if len(msg.Attachments) != 1 {
					t.Fatalf("attachments count = %d, want 1", len(msg.Attachments))
				}
				if msg.Attachments[0].Type != "image" {
					t.Errorf("attachment type = %q, want image", msg.Attachments[0].Type)
				}
				if msg.Attachments[0].SizeBytes != 2048 {
					t.Errorf("attachment size = %d, want 2048", msg.Attachments[0].SizeBytes)
				}
			},
		},
		{
			name:    "missing author",
			raw:     map[string]any{"channel_id": "1200000000000000002", "content": "no author"},
			wantErr: true,
		},
		{
			name: "missing channel",
			raw: map[string]any{
				"author": map[string]any{"id": "1400000000000000004"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := n.Normalize("discord", tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, msg)
		})
	}
}

func TestNormalizer_NormalizeSMS(t *testing.T) {
	n := NewNormalizer()

	msg, err := n.Normalize("sms", map[string]any{
		"MessageSid":        "SM123",
		"From":              "+15551230001",
		"To":                "+15559870002",
		"Body":              "Hello over SMS",
		"NumMedia":          "1",
		"MediaUrl0":         "https://api.twilio.com/media/ME123",
		"MediaContentType0": "image/jpeg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Platform != "sms" {
		t.Errorf("platform = %q, want sms", msg.Platform)
	}
	if msg.ID != "SM123" {
		t.Errorf("id = %q, want SM123", msg.ID)
	}
	if msg.Sender.ID != "+15551230001" {
		t.Errorf("sender.ID = %q, want from number", msg.Sender.ID)
	}
	if msg.Channel.ID != "+15559870002" {
		t.Errorf("channel.ID = %q, want receiving number", msg.Channel.ID)
	}
	if msg.Text != "Hello over SMS" {
		t.Errorf("text = %q, want body", msg.Text)
	}
	if len(msg.Attachments) != 1 || msg.Attachments[0].Type != "image" {
		t.Fatalf("attachments = %#v, want one image attachment", msg.Attachments)
	}
}

func TestNormalizer_NormalizeWhatsApp(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		name  string
		raw   map[string]any
		check func(t *testing.T, msg *msgtypes.NormalizedMessage)
	}{
		{
			name: "cloud api text message",
			raw: map[string]any{
				"entry": []any{
					map[string]any{
						"changes": []any{
							map[string]any{
								"value": map[string]any{
									"metadata": map[string]any{
										"phone_number_id":      "123456789012345",
										"display_phone_number": "+15559870002",
									},
									"contacts": []any{
										map[string]any{
											"profile": map[string]any{"name": "Alice"},
										},
									},
									"messages": []any{
										map[string]any{
											"id":        "wamid.123",
											"from":      "+15551230001",
											"timestamp": "1710000000",
											"type":      "text",
											"text":      map[string]any{"body": "Hello on WhatsApp"},
											"context":   map[string]any{"id": "wamid.parent"},
										},
									},
								},
							},
						},
					},
				},
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.ID != "wamid.123" {
					t.Errorf("id = %q, want cloud message id", msg.ID)
				}
				if msg.Channel.ID != "123456789012345" {
					t.Errorf("channel.ID = %q, want phone number id", msg.Channel.ID)
				}
				if msg.Channel.Name != "+15559870002" {
					t.Errorf("channel.Name = %q, want display phone", msg.Channel.Name)
				}
				if msg.Sender.Name != "Alice" {
					t.Errorf("sender.Name = %q, want contact name", msg.Sender.Name)
				}
				if msg.Text != "Hello on WhatsApp" {
					t.Errorf("text = %q, want text body", msg.Text)
				}
				if msg.Thread == nil || msg.Thread.ID != "wamid.parent" {
					t.Fatalf("thread = %#v, want context reply", msg.Thread)
				}
			},
		},
		{
			name: "twilio style whatsapp message",
			raw: map[string]any{
				"MessageSid": "SM123",
				"From":       "whatsapp:+15551230001",
				"To":         "whatsapp:+15559870002",
				"Body":       "Hello through Twilio WhatsApp",
			},
			check: func(t *testing.T, msg *msgtypes.NormalizedMessage) {
				t.Helper()
				if msg.Channel.ID != "whatsapp:+15559870002" {
					t.Errorf("channel.ID = %q, want receiving WhatsApp number", msg.Channel.ID)
				}
				if msg.Text != "Hello through Twilio WhatsApp" {
					t.Errorf("text = %q, want body", msg.Text)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := n.Normalize("whatsapp", tt.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if msg.Platform != "whatsapp" {
				t.Errorf("platform = %q, want whatsapp", msg.Platform)
			}
			tt.check(t, msg)
		})
	}
}

func TestNormalizer_NormalizeUnsupportedPlatform(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		name     string
		platform string
		raw      map[string]any
	}{
		{
			name:     "unknown platform",
			platform: "matrix",
			raw:      map[string]any{"message": "hello"},
		},
		{
			name:     "empty platform string",
			platform: "",
			raw:      map[string]any{"message": "hello"},
		},
		{
			name:     "arbitrary string",
			platform: "some-random-platform",
			raw:      map[string]any{"data": "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := n.Normalize(tt.platform, tt.raw)
			if err == nil {
				t.Fatal("expected error for unsupported platform, got nil")
			}
			if msg != nil {
				t.Error("expected nil message for unsupported platform")
			}
		})
	}
}

func TestNormalizer_NormalizeNilPayload(t *testing.T) {
	n := NewNormalizer()

	platforms := []string{"telegram", "slack", "discord", "github", "email", "sms", "whatsapp"}
	for _, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			msg, err := n.Normalize(platform, nil)
			if err == nil {
				t.Fatal("expected error for nil payload, got nil")
			}
			if msg != nil {
				t.Error("expected nil message for nil payload")
			}
		})
	}
}

func TestNormalizer_Denormalize(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		name     string
		platform string
		result   *ActionResult
		wantErr  bool
		check    func(t *testing.T, resp map[string]any)
	}{
		{
			name:     "telegram success",
			platform: "telegram",
			result: &ActionResult{
				Status: "success",
				Output: "Action completed.",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["method"] != "sendMessage" {
					t.Errorf("method = %v, want %q", resp["method"], "sendMessage")
				}
				if resp["text"] != "Action completed." {
					t.Errorf("text = %v, want %q", resp["text"], "Action completed.")
				}
				if resp["parse_mode"] != "Markdown" {
					t.Errorf("parse_mode = %v, want %q", resp["parse_mode"], "Markdown")
				}
			},
		},
		{
			name:     "telegram with output data",
			platform: "telegram",
			result: &ActionResult{
				Status:     "success",
				Output:     "Result",
				OutputData: map[string]any{"key": "value"},
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["data"] == nil {
					t.Error("data should be set when OutputData is present")
				}
			},
		},
		{
			name:     "slack success",
			platform: "slack",
			result: &ActionResult{
				Status: "success",
				Output: "Slack response text.",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["response_type"] != "in_channel" {
					t.Errorf("response_type = %v, want %q", resp["response_type"], "in_channel")
				}
				if resp["text"] != "Slack response text." {
					t.Errorf("text = %v, want %q", resp["text"], "Slack response text.")
				}
			},
		},
		{
			name:     "slack with blocks",
			platform: "slack",
			result: &ActionResult{
				Status:     "success",
				Output:     "With blocks",
				OutputData: []map[string]any{{"type": "section"}},
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["blocks"] == nil {
					t.Error("blocks should be set when OutputData is present")
				}
			},
		},
		{
			name:     "discord success",
			platform: "discord",
			result: &ActionResult{
				Status: "success",
				Output: "Discord response text.",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["content"] != "Discord response text." {
					t.Errorf("content = %v, want Discord response text", resp["content"])
				}
				if resp["allowed_mentions"] == nil {
					t.Error("allowed_mentions should be set")
				}
			},
		},
		{
			name:     "github success",
			platform: "github",
			result: &ActionResult{
				Status: "success",
				Output: "Check passed.",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["body"] != "Check passed." {
					t.Errorf("body = %v, want %q", resp["body"], "Check passed.")
				}
				if resp["state"] != "success" {
					t.Errorf("state = %v, want %q", resp["state"], "success")
				}
			},
		},
		{
			name:     "github failed",
			platform: "github",
			result: &ActionResult{
				Status: "failed",
				Output: "Check failed.",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["state"] != "failure" {
					t.Errorf("state = %v, want %q", resp["state"], "failure")
				}
			},
		},
		{
			name:     "email success",
			platform: "email",
			result: &ActionResult{
				Status: "success",
				Output: "Reply body.",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["body"] != "Reply body." {
					t.Errorf("body = %v, want %q", resp["body"], "Reply body.")
				}
				if resp["subject"] != "Re: Action Result" {
					t.Errorf("subject = %v, want %q", resp["subject"], "Re: Action Result")
				}
			},
		},
		{
			name:     "email with output data",
			platform: "email",
			result: &ActionResult{
				Status:     "success",
				Output:     "Body",
				OutputData: map[string]any{"extra": true},
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["data"] == nil {
					t.Error("data should be set when OutputData is present")
				}
			},
		},
		{
			name:     "sms success",
			platform: "sms",
			result: &ActionResult{
				Status: "success",
				Output: "SMS response body.",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["body"] != "SMS response body." {
					t.Errorf("body = %v, want SMS response body", resp["body"])
				}
			},
		},
		{
			name:     "whatsapp success",
			platform: "whatsapp",
			result: &ActionResult{
				Status: "success",
				Output: "WhatsApp response body.",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["type"] != "text" {
					t.Errorf("type = %v, want text", resp["type"])
				}
				text, ok := resp["text"].(map[string]any)
				if !ok {
					t.Fatalf("text = %#v, want text object", resp["text"])
				}
				if text["body"] != "WhatsApp response body." {
					t.Errorf("text.body = %v, want WhatsApp response body", text["body"])
				}
			},
		},
		{
			name:     "unknown platform returns generic response",
			platform: "matrix",
			result: &ActionResult{
				Status: "success",
				Output: "generic output",
			},
			check: func(t *testing.T, resp map[string]any) {
				t.Helper()
				if resp["status"] != "success" {
					t.Errorf("status = %v, want %q", resp["status"], "success")
				}
				if resp["output"] != "generic output" {
					t.Errorf("output = %v, want %q", resp["output"], "generic output")
				}
			},
		},
		{
			name:     "nil result",
			platform: "telegram",
			result:   nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := n.Denormalize(tt.platform, tt.result)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatal("response should not be nil")
			}
			tt.check(t, resp)
		})
	}
}

func TestHelpers(t *testing.T) {
	t.Run("getString", func(t *testing.T) {
		tests := []struct {
			name string
			m    map[string]any
			key  string
			want string
		}{
			{"string value", map[string]any{"k": "hello"}, "k", "hello"},
			{"float64 integer", map[string]any{"k": float64(42)}, "k", "42"},
			{"float64 decimal", map[string]any{"k": float64(3.14)}, "k", "3.14"},
			{"missing key", map[string]any{}, "k", ""},
			{"nil value", map[string]any{"k": nil}, "k", ""},
			{"bool value", map[string]any{"k": true}, "k", "true"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := getString(tt.m, tt.key)
				if got != tt.want {
					t.Errorf("getString(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
				}
			})
		}
	})

	t.Run("getInt64", func(t *testing.T) {
		tests := []struct {
			name string
			m    map[string]any
			key  string
			want int64
		}{
			{"float64 value", map[string]any{"k": float64(123)}, "k", 123},
			{"int64 value", map[string]any{"k": int64(456)}, "k", 456},
			{"int value", map[string]any{"k": int(789)}, "k", 789},
			{"string number", map[string]any{"k": "100"}, "k", 100},
			{"missing key", map[string]any{}, "k", 0},
			{"nil value", map[string]any{"k": nil}, "k", 0},
			{"non-numeric string", map[string]any{"k": "abc"}, "k", 0},
			{"bool value", map[string]any{"k": true}, "k", 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := getInt64(tt.m, tt.key)
				if got != tt.want {
					t.Errorf("getInt64(%v, %q) = %d, want %d", tt.m, tt.key, got, tt.want)
				}
			})
		}
	})

	t.Run("getMap", func(t *testing.T) {
		tests := []struct {
			name    string
			m       map[string]any
			key     string
			wantNil bool
		}{
			{"map value", map[string]any{"k": map[string]any{"a": 1}}, "k", false},
			{"missing key", map[string]any{}, "k", true},
			{"nil value", map[string]any{"k": nil}, "k", true},
			{"non-map value", map[string]any{"k": "string"}, "k", true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := getMap(tt.m, tt.key)
				if tt.wantNil && got != nil {
					t.Errorf("getMap() = %v, want nil", got)
				}
				if !tt.wantNil && got == nil {
					t.Error("getMap() = nil, want non-nil")
				}
			})
		}
	})

	t.Run("buildTelegramName", func(t *testing.T) {
		tests := []struct {
			first, last, want string
		}{
			{"Alice", "Smith", "Alice Smith"},
			{"Alice", "", "Alice"},
			{"", "Smith", "Smith"},
			{"", "", ""},
		}
		for _, tt := range tests {
			t.Run(tt.first+"_"+tt.last, func(t *testing.T) {
				got := buildTelegramName(tt.first, tt.last)
				if got != tt.want {
					t.Errorf("buildTelegramName(%q, %q) = %q, want %q", tt.first, tt.last, got, tt.want)
				}
			})
		}
	})

	t.Run("formatInt64", func(t *testing.T) {
		tests := []struct {
			in   int64
			want string
		}{
			{0, "0"},
			{42, "42"},
			{-1001234567890, "-1001234567890"},
		}
		for _, tt := range tests {
			t.Run(tt.want, func(t *testing.T) {
				got := formatInt64(tt.in)
				if got != tt.want {
					t.Errorf("formatInt64(%d) = %q, want %q", tt.in, got, tt.want)
				}
			})
		}
	})
}
