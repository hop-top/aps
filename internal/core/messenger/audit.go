package messenger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditLogger records capability and configuration changes for messengers
// to a JSONL file. This provides a persistent audit trail of when links are
// created/removed, mappings changed, and messengers enabled/disabled.
//
// Audit log path: {workspaceDir}/logs/audit/capability-changes.jsonl
type AuditLogger struct {
	workspaceDir string
	mu           sync.Mutex
}

// NewAuditLogger creates an audit logger for the given workspace directory.
func NewAuditLogger(workspaceDir string) *AuditLogger {
	return &AuditLogger{workspaceDir: workspaceDir}
}

// LogLinkCreated records the creation of a messenger-profile link.
func (a *AuditLogger) LogLinkCreated(messengerName, profileID string, mappings map[string]string) error {
	entry := map[string]any{
		"event_type":     "link_created",
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": messengerName,
		"profile_id":     profileID,
		"mapping_count":  len(mappings),
	}

	if len(mappings) > 0 {
		entry["mappings"] = mappings
	}

	return a.writeAuditEntry(entry)
}

// LogLinkRemoved records the removal of a messenger-profile link.
func (a *AuditLogger) LogLinkRemoved(messengerName, profileID string) error {
	entry := map[string]any{
		"event_type":     "link_removed",
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": messengerName,
		"profile_id":     profileID,
	}

	return a.writeAuditEntry(entry)
}

// LogMappingChanged records a change to a channel mapping. If oldAction is
// empty, this represents a new mapping being added. If newAction is empty,
// this represents a mapping being removed.
func (a *AuditLogger) LogMappingChanged(messengerName, profileID, channelID, oldAction, newAction string) error {
	changeType := "mapping_updated"
	if oldAction == "" {
		changeType = "mapping_added"
	} else if newAction == "" {
		changeType = "mapping_removed"
	}

	entry := map[string]any{
		"event_type":     changeType,
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": messengerName,
		"profile_id":     profileID,
		"channel_id":     channelID,
	}

	if oldAction != "" {
		entry["old_action"] = oldAction
	}
	if newAction != "" {
		entry["new_action"] = newAction
	}

	return a.writeAuditEntry(entry)
}

// LogEnabled records a messenger-profile link being enabled.
func (a *AuditLogger) LogEnabled(messengerName, profileID string) error {
	entry := map[string]any{
		"event_type":     "link_enabled",
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": messengerName,
		"profile_id":     profileID,
	}

	return a.writeAuditEntry(entry)
}

// LogDisabled records a messenger-profile link being disabled.
func (a *AuditLogger) LogDisabled(messengerName, profileID string) error {
	entry := map[string]any{
		"event_type":     "link_disabled",
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"messenger_name": messengerName,
		"profile_id":     profileID,
	}

	return a.writeAuditEntry(entry)
}

// writeAuditEntry serializes the entry as a single JSON line and appends it
// to the capability-changes.jsonl audit log.
func (a *AuditLogger) writeAuditEntry(entry map[string]any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	line, err := json.Marshal(entry)
	if err != nil {
		return &MessengerError{
			Name:    "audit",
			Message: "failed to marshal audit entry",
			Code:    ErrCodeLogWriteFailed,
			Cause:   err,
		}
	}
	line = append(line, '\n')

	auditDir := filepath.Join(a.workspaceDir, "logs", "audit")
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return &MessengerError{
			Name:    "audit",
			Message: fmt.Sprintf("failed to create audit directory %s", auditDir),
			Code:    ErrCodeLogWriteFailed,
			Cause:   err,
		}
	}

	auditPath := filepath.Join(auditDir, "capability-changes.jsonl")
	f, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return &MessengerError{
			Name:    "audit",
			Message: fmt.Sprintf("failed to open audit log %s", auditPath),
			Code:    ErrCodeLogWriteFailed,
			Cause:   err,
		}
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return &MessengerError{
			Name:    "audit",
			Message: "failed to write audit entry",
			Code:    ErrCodeLogWriteFailed,
			Cause:   err,
		}
	}

	return nil
}
