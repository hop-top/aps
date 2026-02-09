package skills_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"oss-aps-cli/internal/skills"
)

func TestTelemetry_Disabled(t *testing.T) {
	config := &skills.TelemetryConfig{
		Enabled: false,
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)
	assert.NotNil(t, telemetry)

	// Track should succeed but do nothing
	err = telemetry.TrackInvocation("test-skill", "test-profile", "sess-123", "acp", "process")
	assert.NoError(t, err)
}

func TestTelemetry_TrackInvocation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "usage.jsonl")

	config := &skills.TelemetryConfig{
		Enabled:  true,
		EventLog: logFile,
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)

	// Track invocation
	err = telemetry.TrackInvocation("test-skill", "test-profile", "sess-123", "acp", "process")
	require.NoError(t, err)

	// Verify log file created
	assert.FileExists(t, logFile)

	// Read and parse log
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	var event skills.TelemetryEvent
	err = json.Unmarshal(content, &event)
	require.NoError(t, err)

	assert.Equal(t, "skill_invoked", event.EventType)
	assert.Equal(t, "test-skill", event.SkillName)
	assert.Equal(t, "test-profile", event.ProfileID)
	assert.Equal(t, "sess-123", event.SessionID)
	assert.Equal(t, "acp", event.Protocol)
	assert.Equal(t, "process", event.IsolationLevel)
}

func TestTelemetry_TrackCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "usage.jsonl")

	config := &skills.TelemetryConfig{
		Enabled:  true,
		EventLog: logFile,
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)

	// Track completion
	metadata := map[string]interface{}{
		"extra": "data",
	}
	err = telemetry.TrackCompletion("test-skill", "test-profile", "sess-123", "script.sh", 1500, metadata)
	require.NoError(t, err)

	// Read and parse log
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	var event skills.TelemetryEvent
	err = json.Unmarshal(content, &event)
	require.NoError(t, err)

	assert.Equal(t, "skill_completed", event.EventType)
	assert.Equal(t, "test-skill", event.SkillName)
	assert.Equal(t, "script.sh", event.ScriptName)
	assert.NotNil(t, event.Duration)
	assert.Equal(t, int64(1500), *event.Duration)
	assert.NotNil(t, event.Success)
	assert.True(t, *event.Success)
}

func TestTelemetry_TrackFailure(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "usage.jsonl")

	config := &skills.TelemetryConfig{
		Enabled:  true,
		EventLog: logFile,
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)

	// Track failure
	err = telemetry.TrackFailure("test-skill", "test-profile", "sess-123", "script.sh", 500, assert.AnError)
	require.NoError(t, err)

	// Read and parse log
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	var event skills.TelemetryEvent
	err = json.Unmarshal(content, &event)
	require.NoError(t, err)

	assert.Equal(t, "skill_failed", event.EventType)
	assert.Equal(t, "test-skill", event.SkillName)
	assert.NotNil(t, event.Success)
	assert.False(t, *event.Success)
	assert.NotEmpty(t, event.ErrorMsg)
}

func TestTelemetry_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "usage.jsonl")

	config := &skills.TelemetryConfig{
		Enabled:  true,
		EventLog: logFile,
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)

	// Track multiple events
	telemetry.TrackInvocation("skill-a", "profile-1", "", "", "")
	telemetry.TrackCompletion("skill-a", "profile-1", "", "", 1000, nil)

	telemetry.TrackInvocation("skill-b", "profile-1", "", "", "")
	telemetry.TrackCompletion("skill-b", "profile-1", "", "", 2000, nil)

	telemetry.TrackInvocation("skill-a", "profile-2", "", "", "")
	telemetry.TrackFailure("skill-a", "profile-2", "", "", 500, assert.AnError)

	// Get stats for all profiles
	stats, err := telemetry.GetStats("", time.Time{})
	require.NoError(t, err)

	assert.Equal(t, int64(3), stats.TotalInvocations)
	assert.Equal(t, int64(2), stats.TotalCompletions)
	assert.Equal(t, int64(1), stats.TotalFailures)

	// Get stats for specific profile
	statsProfile1, err := telemetry.GetStats("profile-1", time.Time{})
	require.NoError(t, err)

	assert.Equal(t, int64(2), statsProfile1.TotalInvocations)
	assert.Equal(t, int64(2), statsProfile1.TotalCompletions)
	assert.Equal(t, int64(0), statsProfile1.TotalFailures)
}

func TestTelemetry_SkillStats(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "usage.jsonl")

	config := &skills.TelemetryConfig{
		Enabled:  true,
		EventLog: logFile,
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)

	// Track events for a specific skill
	telemetry.TrackInvocation("test-skill", "profile-1", "", "", "")
	telemetry.TrackCompletion("test-skill", "profile-1", "", "", 1000, nil)

	telemetry.TrackInvocation("test-skill", "profile-1", "", "", "")
	telemetry.TrackCompletion("test-skill", "profile-1", "", "", 2000, nil)

	telemetry.TrackInvocation("test-skill", "profile-1", "", "", "")
	telemetry.TrackFailure("test-skill", "profile-1", "", "", 500, assert.AnError)

	// Get stats
	stats, err := telemetry.GetStats("", time.Time{})
	require.NoError(t, err)

	skillStats := stats.BySkill["test-skill"]
	require.NotNil(t, skillStats)

	assert.Equal(t, int64(3), skillStats.Invocations)
	assert.Equal(t, int64(2), skillStats.Completions)
	assert.Equal(t, int64(1), skillStats.Failures)
	assert.Equal(t, int64(3000), skillStats.TotalDurationMs)

	// Test success rate
	successRate := skillStats.SuccessRate()
	assert.InDelta(t, 0.666, successRate, 0.01) // 2/3 ≈ 0.666

	// Test average duration
	avgDuration := skillStats.AverageDurationMs()
	assert.Equal(t, 1500.0, avgDuration) // (1000 + 2000) / 2 = 1500
}

func TestTelemetry_MultipleEvents(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "usage.jsonl")

	config := &skills.TelemetryConfig{
		Enabled:  true,
		EventLog: logFile,
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)

	// Track multiple events (JSONL format)
	telemetry.TrackInvocation("skill-1", "profile-1", "", "", "")
	telemetry.TrackInvocation("skill-2", "profile-1", "", "", "")
	telemetry.TrackInvocation("skill-3", "profile-1", "", "", "")

	// Read log file and verify JSONL format (one event per line)
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	lines := string(content)
	assert.Equal(t, 3, len(strings.Split(strings.TrimSpace(lines), "\n")))

	// Verify each line is valid JSON
	for _, line := range strings.Split(strings.TrimSpace(lines), "\n") {
		var event skills.TelemetryEvent
		err := json.Unmarshal([]byte(line), &event)
		assert.NoError(t, err)
	}
}

func TestTelemetry_DefaultLogPath(t *testing.T) {
	config := &skills.TelemetryConfig{
		Enabled:  true,
		EventLog: "", // Empty, should use default
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)
	assert.NotNil(t, telemetry)

	// Should create default log directory
	// (Can't easily verify without knowing home dir in test)
}
