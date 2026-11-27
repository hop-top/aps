package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A session ID whose tail is a 16-digit run across a hyphen must be
// persisted verbatim; the kit credit-card rule matches that shape.
func TestTelemetry_SessionIDRoundTrip(t *testing.T) {
	const sid = "bffab4c3-9d88-4349-9131-824401265021"
	logFile := filepath.Join(t.TempDir(), "events.jsonl")
	tel, err := NewTelemetry(&TelemetryConfig{Enabled: true, EventLog: logFile})
	if err != nil {
		t.Fatal(err)
	}
	if err := tel.writeEvent(TelemetryEvent{
		Timestamp: time.Now(),
		EventType: "skill_invoked",
		SkillName: "s",
		ProfileID: "p",
		SessionID: sid,
	}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"session_id":"`+sid+`"`) {
		t.Fatalf("session_id mangled: %s", b)
	}
}
