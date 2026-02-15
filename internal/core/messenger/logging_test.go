package messenger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// readJSONLEntries reads a JSONL file and returns parsed entries.
func readJSONLEntries(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read JSONL file %s: %v", path, err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var entries []map[string]any
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("failed to parse JSONL line %q: %v", line, err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestWorkspaceMessageLogger_LogMessageReceived(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	msg := &NormalizedMessage{
		ID:        "msg-001",
		Timestamp: time.Now(),
		Platform:  "telegram",
		Sender:    Sender{ID: "user-1", Name: "Alice"},
		Channel:   Channel{ID: "chan-1", Name: "general"},
		Text:      "Hello, world!",
	}

	err := logger.LogMessageReceived(msg)
	if err != nil {
		t.Fatalf("LogMessageReceived failed: %v", err)
	}

	// Find the _all.jsonl file
	date := time.Now().UTC().Format("2006-01-02")
	allPath := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram", "_all.jsonl")

	entries := readJSONLEntries(t, allPath)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	tests := []struct {
		field string
		want  any
	}{
		{"event_type", "message_received"},
		{"messenger_name", "test-telegram"},
		{"message_id", "msg-001"},
		{"platform", "telegram"},
		{"sender_id", "user-1"},
		{"sender_name", "Alice"},
		{"channel_id", "chan-1"},
		{"channel_name", "general"},
		{"text_preview", "Hello, world!"},
	}

	for _, tt := range tests {
		t.Run("field_"+tt.field, func(t *testing.T) {
			got, ok := entry[tt.field]
			if !ok {
				t.Fatalf("field %q missing from entry", tt.field)
			}
			if got != tt.want {
				t.Errorf("%s = %v, want %v", tt.field, got, tt.want)
			}
		})
	}

	// Verify timestamp field exists
	if _, ok := entry["timestamp"]; !ok {
		t.Error("timestamp field missing from entry")
	}
}

func TestWorkspaceMessageLogger_LogMessageReceived_NilMessage(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	err := logger.LogMessageReceived(nil)
	if err == nil {
		t.Fatal("expected error for nil message, got nil")
	}
}

func TestWorkspaceMessageLogger_LogMessageReceived_WithThread(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	msg := &NormalizedMessage{
		ID:       "msg-002",
		Platform: "telegram",
		Sender:   Sender{ID: "user-1", Name: "Alice"},
		Channel:  Channel{ID: "chan-1"},
		Text:     "Reply in thread",
		Thread:   &Thread{ID: "thread-123", Type: "reply"},
	}

	err := logger.LogMessageReceived(msg)
	if err != nil {
		t.Fatalf("LogMessageReceived failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	allPath := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram", "_all.jsonl")
	entries := readJSONLEntries(t, allPath)

	if entries[0]["thread_id"] != "thread-123" {
		t.Errorf("thread_id = %v, want %q", entries[0]["thread_id"], "thread-123")
	}
}

func TestWorkspaceMessageLogger_LogMessageReceived_WithAttachments(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	msg := &NormalizedMessage{
		ID:       "msg-003",
		Platform: "telegram",
		Sender:   Sender{ID: "user-1", Name: "Alice"},
		Channel:  Channel{ID: "chan-1"},
		Text:     "Image attached",
		Attachments: []Attachment{
			{Type: "image", URL: "https://example.com/img.png"},
			{Type: "file", URL: "https://example.com/doc.pdf"},
		},
	}

	err := logger.LogMessageReceived(msg)
	if err != nil {
		t.Fatalf("LogMessageReceived failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	allPath := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram", "_all.jsonl")
	entries := readJSONLEntries(t, allPath)

	// attachment_count is a float64 when unmarshalled from JSON
	count, ok := entries[0]["attachment_count"]
	if !ok {
		t.Fatal("attachment_count field missing")
	}
	if count != float64(2) {
		t.Errorf("attachment_count = %v, want 2", count)
	}
}

func TestWorkspaceMessageLogger_LogActionExecuted(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	err := logger.LogActionExecuted("msg-001", "dev=deploy", "success", 250)
	if err != nil {
		t.Fatalf("LogActionExecuted failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	allPath := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram", "_all.jsonl")

	entries := readJSONLEntries(t, allPath)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry["event_type"] != "action_executed" {
		t.Errorf("event_type = %v, want %q", entry["event_type"], "action_executed")
	}
	if entry["message_id"] != "msg-001" {
		t.Errorf("message_id = %v, want %q", entry["message_id"], "msg-001")
	}
	if entry["action"] != "dev=deploy" {
		t.Errorf("action = %v, want %q", entry["action"], "dev=deploy")
	}
	if entry["status"] != "success" {
		t.Errorf("status = %v, want %q", entry["status"], "success")
	}
	if entry["duration_ms"] != float64(250) {
		t.Errorf("duration_ms = %v, want 250", entry["duration_ms"])
	}
}

func TestWorkspaceMessageLogger_DualWrite(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	msg := &NormalizedMessage{
		ID:        "msg-001",
		Platform:  "telegram",
		ProfileID: "dev-profile",
		Sender:    Sender{ID: "user-1", Name: "Alice"},
		Channel:   Channel{ID: "chan-1"},
		Text:      "message for profile",
	}

	err := logger.LogMessageReceived(msg)
	if err != nil {
		t.Fatalf("LogMessageReceived failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	baseDir := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram")

	// Verify _all.jsonl has the entry
	allEntries := readJSONLEntries(t, filepath.Join(baseDir, "_all.jsonl"))
	if len(allEntries) != 1 {
		t.Fatalf("expected 1 entry in _all.jsonl, got %d", len(allEntries))
	}

	// Verify {profile}.jsonl has the entry
	profileEntries := readJSONLEntries(t, filepath.Join(baseDir, "dev-profile.jsonl"))
	if len(profileEntries) != 1 {
		t.Fatalf("expected 1 entry in dev-profile.jsonl, got %d", len(profileEntries))
	}

	// Both entries should have the same content
	if allEntries[0]["message_id"] != profileEntries[0]["message_id"] {
		t.Error("entries in _all.jsonl and dev-profile.jsonl should match")
	}
}

func TestWorkspaceMessageLogger_NoDualWrite_WithoutProfileID(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	msg := &NormalizedMessage{
		ID:       "msg-001",
		Platform: "telegram",
		// No ProfileID set
		Sender:  Sender{ID: "user-1", Name: "Alice"},
		Channel: Channel{ID: "chan-1"},
		Text:    "message without profile",
	}

	err := logger.LogMessageReceived(msg)
	if err != nil {
		t.Fatalf("LogMessageReceived failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	baseDir := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram")

	// _all.jsonl should exist
	if _, err := os.Stat(filepath.Join(baseDir, "_all.jsonl")); err != nil {
		t.Fatalf("_all.jsonl should exist: %v", err)
	}

	// No profile-specific file should exist
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("failed to read log dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "_all.jsonl" {
			t.Errorf("unexpected file %q (should only have _all.jsonl)", e.Name())
		}
	}
}

func TestWorkspaceMessageLogger_DirectoryCreation(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "nested-messenger")

	msg := &NormalizedMessage{
		ID:       "msg-001",
		Platform: "telegram",
		Sender:   Sender{ID: "user-1"},
		Channel:  Channel{ID: "chan-1"},
	}

	err := logger.LogMessageReceived(msg)
	if err != nil {
		t.Fatalf("LogMessageReceived failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	expectedDir := filepath.Join(workspaceDir, "logs", "messengers", date, "nested-messenger")

	info, err := os.Stat(expectedDir)
	if err != nil {
		t.Fatalf("expected directory %s to exist: %v", expectedDir, err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", expectedDir)
	}
}

func TestWorkspaceMessageLogger_LogMessageRouted(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	err := logger.LogMessageRouted("msg-001", "dev-profile=process-incoming")
	if err != nil {
		t.Fatalf("LogMessageRouted failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	baseDir := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram")

	allEntries := readJSONLEntries(t, filepath.Join(baseDir, "_all.jsonl"))
	if len(allEntries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(allEntries))
	}
	if allEntries[0]["event_type"] != "message_routed" {
		t.Errorf("event_type = %v, want %q", allEntries[0]["event_type"], "message_routed")
	}
	if allEntries[0]["target_action"] != "dev-profile=process-incoming" {
		t.Errorf("target_action = %v, want %q", allEntries[0]["target_action"], "dev-profile=process-incoming")
	}

	// Should also write to profile-specific log since action contains profile ID
	profileEntries := readJSONLEntries(t, filepath.Join(baseDir, "dev-profile.jsonl"))
	if len(profileEntries) != 1 {
		t.Fatalf("expected 1 entry in dev-profile.jsonl, got %d", len(profileEntries))
	}
}

func TestWorkspaceMessageLogger_LogMessageSent(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	err := logger.LogMessageSent("msg-001", "chan-1", "delivered")
	if err != nil {
		t.Fatalf("LogMessageSent failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	allPath := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram", "_all.jsonl")
	entries := readJSONLEntries(t, allPath)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["event_type"] != "message_sent" {
		t.Errorf("event_type = %v, want %q", entries[0]["event_type"], "message_sent")
	}
	if entries[0]["platform_status"] != "delivered" {
		t.Errorf("platform_status = %v, want %q", entries[0]["platform_status"], "delivered")
	}
}

func TestWorkspaceMessageLogger_LogError(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	err := logger.LogError("msg-001", "routing", ErrUnknownChannel("test-telegram", "chan-999"))
	if err != nil {
		t.Fatalf("LogError failed: %v", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	allPath := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram", "_all.jsonl")
	entries := readJSONLEntries(t, allPath)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["event_type"] != "error" {
		t.Errorf("event_type = %v, want %q", entries[0]["event_type"], "error")
	}
	if entries[0]["step"] != "routing" {
		t.Errorf("step = %v, want %q", entries[0]["step"], "routing")
	}
	errStr, ok := entries[0]["error"].(string)
	if !ok {
		t.Fatal("error field missing or not a string")
	}
	if errStr == "" {
		t.Error("error field should not be empty")
	}
}

func TestWorkspaceMessageLogger_MultipleEntries(t *testing.T) {
	workspaceDir := t.TempDir()
	logger := NewWorkspaceMessageLogger(workspaceDir, "test-telegram")

	// Write 3 entries
	for i := range 3 {
		msg := &NormalizedMessage{
			ID:       "msg-" + string(rune('0'+i)),
			Platform: "telegram",
			Sender:   Sender{ID: "user-1"},
			Channel:  Channel{ID: "chan-1"},
		}
		if err := logger.LogMessageReceived(msg); err != nil {
			t.Fatalf("LogMessageReceived #%d failed: %v", i, err)
		}
	}

	date := time.Now().UTC().Format("2006-01-02")
	allPath := filepath.Join(workspaceDir, "logs", "messengers", date, "test-telegram", "_all.jsonl")
	entries := readJSONLEntries(t, allPath)

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}
