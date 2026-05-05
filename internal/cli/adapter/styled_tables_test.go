package adapter

// Golden-style tests for T-0474/T-0475/T-0476: the adapter package's
// three tabwriter callsites (presence, pending, channels) now route
// through listing.RenderList. Non-TTY callers see plain tabwriter
// output with no ANSI / box-drawing leakage. JSON/YAML round-trip
// through each typed row preserves its field set.

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/kit/go/console/output"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func assertPlainTable(t *testing.T, label, out string, want []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("%s: expected %q in output, got: %q", label, w, out)
		}
	}
	if ansiRe.MatchString(out) {
		t.Errorf("%s: non-TTY output leaked ANSI escapes: %q", label, out)
	}
	for _, r := range []rune{'┌', '┐', '└', '┘', '│', '─'} {
		if strings.ContainsRune(out, r) {
			t.Errorf("%s: non-TTY output leaked box-drawing rune %q: %q", label, r, out)
		}
	}
}

func TestPresenceTableRow_NonTTYPlainOutput(t *testing.T) {
	rows := []presenceTableRow{
		{DeviceID: "dev-1", State: "online", LastSeen: "2s ago", SyncLag: "--", Queue: "--"},
		{DeviceID: "dev-2", State: "offline", LastSeen: "5m ago", SyncLag: "3 events", Queue: "1 pending"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	assertPlainTable(t, "presence", buf.String(), []string{
		"DEVICE", "STATUS", "LAST SEEN", "SYNC LAG", "QUEUE",
		"dev-1", "online", "2s ago", "dev-2", "offline", "3 events", "1 pending",
	})
}

func TestPresenceRow_JSONRoundTrip(t *testing.T) {
	// presenceRow is the json/yaml shape — int fields are preserved.
	rows := []presenceRow{
		{DeviceID: "dev-1", State: "online", LastSeen: "2s ago", SyncLag: 0, OfflineQueue: 0},
		{DeviceID: "dev-2", State: "offline", LastSeen: "5m ago", SyncLag: 3, OfflineQueue: 1},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.JSON, rows); err != nil {
		t.Fatalf("RenderList JSON: %v", err)
	}
	var got []presenceRow
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, buf.String())
	}
	if len(got) != 2 || got[0] != rows[0] || got[1] != rows[1] {
		t.Errorf("presence JSON round-trip: got %+v, want %+v", got, rows)
	}
}

func TestPendingTableRow_NonTTYPlainOutput(t *testing.T) {
	rows := []pendingTableRow{
		{DeviceID: "dev-1", Requested: "5m ago", DeviceInfo: "Pixel 8, android 14"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	assertPlainTable(t, "pending", buf.String(), []string{
		"DEVICE", "REQUESTED", "DEVICE INFO",
		"dev-1", "5m ago", "Pixel 8, android 14",
	})
}

func TestPendingJSONRow_JSONRoundTrip(t *testing.T) {
	rows := []pendingJSONRow{
		{
			DeviceID:    "dev-1",
			ProfileID:   "alpha",
			DeviceName:  "Pixel 8",
			DeviceOS:    "android 14",
			RequestedAt: "2026-05-04T12:00:00Z",
		},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.JSON, rows); err != nil {
		t.Fatalf("RenderList JSON: %v", err)
	}
	var got []pendingJSONRow
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, buf.String())
	}
	if len(got) != 1 || got[0] != rows[0] {
		t.Errorf("pending JSON round-trip: got %+v, want %+v", got, rows)
	}
	// Field-set guard: the JSON must have exactly the 5 keys the
	// pre-T-0475 inline `pendingDevice` struct emitted.
	for _, want := range []string{`"device_id"`, `"profile_id"`, `"device_name"`, `"device_os"`, `"requested_at"`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("pending JSON missing key %q: %s", want, buf.String())
		}
	}
}

func TestChannelRow_NonTTYPlainOutput(t *testing.T) {
	rows := []channelRow{
		{ChannelID: "C123", MappedTo: "deploy"},
		{ChannelID: "C456", MappedTo: "(unmapped)"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	assertPlainTable(t, "channels", buf.String(), []string{
		"CHANNEL ID", "MAPPED TO",
		"C123", "deploy", "C456", "(unmapped)",
	})
}

func TestChannelRow_JSONRoundTrip(t *testing.T) {
	rows := []channelRow{
		{ChannelID: "C123", MappedTo: "deploy", ProfileID: "alpha"},
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, output.JSON, rows); err != nil {
		t.Fatalf("RenderList JSON: %v", err)
	}
	var got []channelRow
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, buf.String())
	}
	if len(got) != 1 || got[0] != rows[0] {
		t.Errorf("channel JSON round-trip: got %+v, want %+v", got, rows)
	}
}
