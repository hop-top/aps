package messenger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cardShapedID is a deterministic opaque ID whose digit tail matches the
// credit-card rule when line-redacted.
const cardShapedID = "9f1c2d3e-4a5b-9131-824401265021"

func auditEntries(t *testing.T, dir string) []map[string]any {
	t.Helper()
	return readJSONLEntries(t, filepath.Join(dir, "logs", "audit", "capability-changes.jsonl"))
}

func TestAuditLogger_VerbatimIDsRoundTrip(t *testing.T) {
	for _, p := range []MessengerPlatform{PlatformSlack, PlatformTelegram, PlatformDiscord} {
		t.Run(string(p), func(t *testing.T) {
			dir := t.TempDir()
			a := NewAuditLogger(dir).WithPlatform(p)
			if err := a.LogMappingChanged("m", "prof", cardShapedID, "", "agent:prof:run"); err != nil {
				t.Fatal(err)
			}
			if err := a.LogLinkCreated("m", "prof", map[string]string{cardShapedID: "agent:prof:run"}); err != nil {
				t.Fatal(err)
			}
			es := auditEntries(t, dir)
			if got := es[0]["channel_id"]; got != cardShapedID {
				t.Fatalf("channel_id = %v, want verbatim %s", got, cardShapedID)
			}
			ms := es[1]["mappings"].([]any)
			if got := ms[0].(map[string]any)["channel_id"]; got != cardShapedID {
				t.Fatalf("mapping channel_id = %v, want verbatim", got)
			}
		})
	}
}

func TestWorkspaceMessageLogger_VerbatimIDsRoundTrip(t *testing.T) {
	for _, p := range []MessengerPlatform{PlatformSlack, PlatformTelegram, PlatformDiscord} {
		t.Run(string(p), func(t *testing.T) {
			dir := t.TempDir()
			l := NewWorkspaceMessageLogger(dir, "m").WithPlatform(p)
			if err := l.LogMessageSent(cardShapedID, cardShapedID, "ok"); err != nil {
				t.Fatal(err)
			}
			es := readJSONLEntries(t, filepath.Join(l.mustLogDir(t), "_all.jsonl"))
			if es[0]["message_id"] != cardShapedID || es[0]["channel_id"] != cardShapedID {
				t.Fatalf("ids mangled: %v", es[0])
			}
		})
	}
}

func TestMessageReceived_PlatformFromMessage(t *testing.T) {
	dir := t.TempDir()
	l := NewWorkspaceMessageLogger(dir, "m") // platform unset on logger
	msg := &NormalizedMessage{ID: cardShapedID, Platform: "telegram"}
	msg.Channel.ID = cardShapedID
	if err := l.LogMessageReceived(msg); err != nil {
		t.Fatal(err)
	}
	es := readJSONLEntries(t, filepath.Join(l.mustLogDir(t), "_all.jsonl"))
	if es[0]["message_id"] != cardShapedID {
		t.Fatalf("message_id mangled: %v", es[0]["message_id"])
	}
}

func TestPhonePlatforms_IDsRedacted(t *testing.T) {
	const phone = "+14155550123"
	for _, p := range []MessengerPlatform{PlatformWhatsApp, PlatformSMS, "", PlatformEmail} {
		t.Run("audit/"+string(p), func(t *testing.T) {
			dir := t.TempDir()
			a := NewAuditLogger(dir).WithPlatform(p)
			if err := a.LogMappingChanged("m", "prof", phone, "", "agent:prof:run"); err != nil {
				t.Fatal(err)
			}
			if err := a.LogLinkCreated("m", "prof", map[string]string{phone: "agent:prof:run"}); err != nil {
				t.Fatal(err)
			}
			raw := readFileT(t, filepath.Join(dir, "logs", "audit", "capability-changes.jsonl"))
			if strings.Contains(raw, "4155550123") {
				t.Fatalf("phone persisted: %s", raw)
			}
		})
		t.Run("log/"+string(p), func(t *testing.T) {
			dir := t.TempDir()
			l := NewWorkspaceMessageLogger(dir, "m").WithPlatform(p)
			if err := l.LogMessageSent(phone, phone, "ok"); err != nil {
				t.Fatal(err)
			}
			raw := readFileT(t, filepath.Join(l.mustLogDir(t), "_all.jsonl"))
			if strings.Contains(raw, "4155550123") {
				t.Fatalf("phone persisted: %s", raw)
			}
		})
	}
}

func readFileT(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func (l *WorkspaceMessageLogger) mustLogDir(t *testing.T) string {
	t.Helper()
	d, err := l.logDir()
	if err != nil {
		t.Fatal(err)
	}
	return d
}
