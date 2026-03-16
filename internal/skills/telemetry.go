package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hop.top/aps/internal/core"
)

// TelemetryEvent represents a skill usage event
type TelemetryEvent struct {
	// Core fields
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"` // "skill_invoked", "skill_completed", "skill_failed"
	SkillName   string    `json:"skill_name"`
	ProfileID   string    `json:"profile_id"`
	SessionID   string    `json:"session_id,omitempty"`
	Duration    *int64    `json:"duration_ms,omitempty"` // For completion/failure events

	// Context
	Protocol    string    `json:"protocol,omitempty"`    // "agent-protocol", "a2a", "acp"
	IsolationLevel string `json:"isolation_level,omitempty"` // "process", "platform", "container"

	// Execution details
	ScriptName  string    `json:"script_name,omitempty"`
	Success     *bool     `json:"success,omitempty"`
	ErrorMsg    string    `json:"error,omitempty"`

	// Optional metadata (if enabled in config)
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Telemetry handles skill usage tracking
type Telemetry struct {
	config   *TelemetryConfig
	logFile  string
	enabled  bool
}

// NewTelemetry creates a new telemetry tracker
func NewTelemetry(config *TelemetryConfig) (*Telemetry, error) {
	if !config.Enabled {
		return &Telemetry{enabled: false}, nil
	}

	logFile := config.EventLog
	if logFile == "" {
		dataDir, err := core.GetDataDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get data directory: %w", err)
		}
		logFile = filepath.Join(dataDir, "skills", "usage.jsonl")
	}

	// Ensure directory exists
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create telemetry directory: %w", err)
	}

	return &Telemetry{
		config:  config,
		logFile: logFile,
		enabled: true,
	}, nil
}

// TrackInvocation tracks a skill invocation
func (t *Telemetry) TrackInvocation(skillName, profileID, sessionID, protocol, isolationLevel string) error {
	if !t.enabled {
		return nil
	}

	event := TelemetryEvent{
		Timestamp:      time.Now(),
		EventType:      "skill_invoked",
		SkillName:      skillName,
		ProfileID:      profileID,
		SessionID:      sessionID,
		Protocol:       protocol,
		IsolationLevel: isolationLevel,
	}

	return t.writeEvent(event)
}

// TrackCompletion tracks successful skill completion
func (t *Telemetry) TrackCompletion(skillName, profileID, sessionID, scriptName string, durationMs int64, metadata map[string]interface{}) error {
	if !t.enabled {
		return nil
	}

	success := true
	event := TelemetryEvent{
		Timestamp:  time.Now(),
		EventType:  "skill_completed",
		SkillName:  skillName,
		ProfileID:  profileID,
		SessionID:  sessionID,
		ScriptName: scriptName,
		Duration:   &durationMs,
		Success:    &success,
	}

	if t.config.IncludeMetadata {
		event.Metadata = metadata
	}

	return t.writeEvent(event)
}

// TrackFailure tracks skill execution failure
func (t *Telemetry) TrackFailure(skillName, profileID, sessionID, scriptName string, durationMs int64, err error) error {
	if !t.enabled {
		return nil
	}

	success := false
	event := TelemetryEvent{
		Timestamp:  time.Now(),
		EventType:  "skill_failed",
		SkillName:  skillName,
		ProfileID:  profileID,
		SessionID:  sessionID,
		ScriptName: scriptName,
		Duration:   &durationMs,
		Success:    &success,
		ErrorMsg:   err.Error(),
	}

	return t.writeEvent(event)
}

// writeEvent appends an event to the log file (JSONL format)
func (t *Telemetry) writeEvent(event TelemetryEvent) error {
	// Open file in append mode
	f, err := os.OpenFile(t.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open telemetry log: %w", err)
	}
	defer f.Close()

	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write as single line (JSONL)
	if _, err := f.Write(append(eventJSON, '\n')); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// GetStats retrieves usage statistics from the event log
func (t *Telemetry) GetStats(profileID string, since time.Time) (*UsageStats, error) {
	if !t.enabled {
		return &UsageStats{}, nil
	}

	// Read and parse event log
	f, err := os.Open(t.logFile)
	if os.IsNotExist(err) {
		return &UsageStats{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open telemetry log: %w", err)
	}
	defer f.Close()

	stats := &UsageStats{
		BySkill: make(map[string]*SkillStats),
	}

	decoder := json.NewDecoder(f)
	for {
		var event TelemetryEvent
		if err := decoder.Decode(&event); err != nil {
			break // EOF or parse error
		}

		// Filter by profile and time
		if profileID != "" && event.ProfileID != profileID {
			continue
		}
		if !since.IsZero() && event.Timestamp.Before(since) {
			continue
		}

		// Aggregate stats
		switch event.EventType {
		case "skill_invoked":
			stats.TotalInvocations++
			skillStats := stats.getOrCreateSkillStats(event.SkillName)
			skillStats.Invocations++

		case "skill_completed":
			stats.TotalCompletions++
			skillStats := stats.getOrCreateSkillStats(event.SkillName)
			skillStats.Completions++
			if event.Duration != nil {
				skillStats.TotalDurationMs += *event.Duration
			}

		case "skill_failed":
			stats.TotalFailures++
			skillStats := stats.getOrCreateSkillStats(event.SkillName)
			skillStats.Failures++
		}
	}

	return stats, nil
}

// UsageStats represents aggregated usage statistics
type UsageStats struct {
	TotalInvocations int64                  `json:"total_invocations"`
	TotalCompletions int64                  `json:"total_completions"`
	TotalFailures    int64                  `json:"total_failures"`
	BySkill          map[string]*SkillStats `json:"by_skill"`
}

// SkillStats represents statistics for a single skill
type SkillStats struct {
	Invocations     int64 `json:"invocations"`
	Completions     int64 `json:"completions"`
	Failures        int64 `json:"failures"`
	TotalDurationMs int64 `json:"total_duration_ms"`
}

func (us *UsageStats) getOrCreateSkillStats(skillName string) *SkillStats {
	if us.BySkill[skillName] == nil {
		us.BySkill[skillName] = &SkillStats{}
	}
	return us.BySkill[skillName]
}

// SuccessRate returns the success rate (0-1) for a skill
func (ss *SkillStats) SuccessRate() float64 {
	total := ss.Completions + ss.Failures
	if total == 0 {
		return 0
	}
	return float64(ss.Completions) / float64(total)
}

// AverageDurationMs returns average execution time in milliseconds
func (ss *SkillStats) AverageDurationMs() float64 {
	if ss.Completions == 0 {
		return 0
	}
	return float64(ss.TotalDurationMs) / float64(ss.Completions)
}
