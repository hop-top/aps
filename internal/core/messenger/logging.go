package messenger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WorkspaceMessageLogger writes structured JSONL logs for messenger activity.
// Logs are organized by date and messenger under the workspace directory:
//
//	{workspaceDir}/logs/messengers/{date}/{messenger}/_all.jsonl
//	{workspaceDir}/logs/messengers/{date}/{messenger}/{profile}.jsonl
type WorkspaceMessageLogger struct {
	workspaceDir  string
	messengerName string
	mu            sync.Mutex
}

// NewWorkspaceMessageLogger creates a logger for a specific messenger within a workspace.
func NewWorkspaceMessageLogger(workspaceDir, messengerName string) *WorkspaceMessageLogger {
	return &WorkspaceMessageLogger{
		workspaceDir:  workspaceDir,
		messengerName: messengerName,
	}
}

// LogMessageReceived records a normalized message arriving from the platform.
func (l *WorkspaceMessageLogger) LogMessageReceived(msg *NormalizedMessage) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	entry := map[string]any{
		"event_type":     "message_received",
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": l.messengerName,
		"message_id":     msg.ID,
		"platform":       msg.Platform,
		"sender_id":      msg.Sender.ID,
		"sender_name":    msg.Sender.Name,
		"channel_id":     msg.Channel.ID,
		"channel_name":   msg.Channel.Name,
		"text_preview":   msg.TextPreview(120),
	}

	if msg.Thread != nil {
		entry["thread_id"] = msg.Thread.ID
	}

	if len(msg.Attachments) > 0 {
		entry["attachment_count"] = len(msg.Attachments)
	}

	return l.writeEntry(msg.ProfileID, entry)
}

// LogMessageRouted records that a message was routed to a specific action.
func (l *WorkspaceMessageLogger) LogMessageRouted(msgID, targetAction string) error {
	// Parse the target action to extract the profile ID for per-profile logging
	profileID := ""
	if ta, err := ParseTargetAction(targetAction); err == nil {
		profileID = ta.ProfileID
	}

	entry := map[string]any{
		"event_type":     "message_routed",
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": l.messengerName,
		"message_id":     msgID,
		"target_action":  targetAction,
	}

	return l.writeEntry(profileID, entry)
}

// LogActionExecuted records the result of executing an action for a message.
func (l *WorkspaceMessageLogger) LogActionExecuted(msgID, action, status string, durationMS int64) error {
	profileID := ""
	if ta, err := ParseTargetAction(action); err == nil {
		profileID = ta.ProfileID
	}

	entry := map[string]any{
		"event_type":     "action_executed",
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": l.messengerName,
		"message_id":     msgID,
		"action":         action,
		"status":         status,
		"duration_ms":    durationMS,
	}

	return l.writeEntry(profileID, entry)
}

// LogMessageSent records a response message sent back to the platform.
func (l *WorkspaceMessageLogger) LogMessageSent(msgID, channelID, platformStatus string) error {
	entry := map[string]any{
		"event_type":      "message_sent",
		"timestamp":       time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name":  l.messengerName,
		"message_id":      msgID,
		"channel_id":      channelID,
		"platform_status": platformStatus,
	}

	return l.writeEntry("", entry)
}

// LogError records an error that occurred during message processing.
func (l *WorkspaceMessageLogger) LogError(msgID, step string, err error) error {
	entry := map[string]any{
		"event_type":     "error",
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": l.messengerName,
		"message_id":     msgID,
		"step":           step,
		"error":          err.Error(),
	}

	return l.writeEntry("", entry)
}

// writeEntry serializes an entry as a single JSON line and appends it to the
// appropriate log files. It always writes to _all.jsonl, and additionally to
// {profile}.jsonl when a profile ID is provided.
func (l *WorkspaceMessageLogger) writeEntry(profileID string, entry map[string]any) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	line, err := json.Marshal(entry)
	if err != nil {
		return &MessengerError{
			Name:    l.messengerName,
			Message: "failed to marshal log entry",
			Code:    ErrCodeLogWriteFailed,
			Cause:   err,
		}
	}
	line = append(line, '\n')

	dir, err := l.logDir()
	if err != nil {
		return err
	}

	// Always write to _all.jsonl
	allPath := filepath.Join(dir, "_all.jsonl")
	if err := appendToFile(allPath, line); err != nil {
		return &MessengerError{
			Name:    l.messengerName,
			Message: fmt.Sprintf("failed to write to %s", allPath),
			Code:    ErrCodeLogWriteFailed,
			Cause:   err,
		}
	}

	// Write to per-profile log when profile is known
	if profileID != "" {
		profilePath := filepath.Join(dir, profileID+".jsonl")
		if err := appendToFile(profilePath, line); err != nil {
			return &MessengerError{
				Name:    l.messengerName,
				Message: fmt.Sprintf("failed to write to %s", profilePath),
				Code:    ErrCodeLogWriteFailed,
				Cause:   err,
			}
		}
	}

	return nil
}

// logDir returns the dated log directory for today and creates it if needed.
// Path: {workspaceDir}/logs/messengers/{date}/{messenger}/
func (l *WorkspaceMessageLogger) logDir() (string, error) {
	date := time.Now().UTC().Format("2006-01-02")
	dir := filepath.Join(l.workspaceDir, "logs", "messengers", date, l.messengerName)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", &MessengerError{
			Name:    l.messengerName,
			Message: fmt.Sprintf("failed to create log directory %s", dir),
			Code:    ErrCodeLogWriteFailed,
			Cause:   err,
		}
	}

	return dir, nil
}

// appendToFile opens a file in append mode (creating it if needed) and writes data.
func appendToFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}
