package workspace_e2e

import (
	"encoding/json"
	"strings"
	"testing"

	"hop.top/aps/internal/core/multidevice"
)

// seedConflictFixtures populates a workspace with a mix of conflict
// statuses so --unresolved can split the rows.
func seedConflictFixtures(t *testing.T, home, wsID string) {
	t.Helper()
	now := nowUTC()
	conflicts := []*multidevice.Conflict{
		{
			ID:          "c-pending",
			WorkspaceID: wsID,
			Type:        multidevice.ConflictConcurrentWrite,
			Status:      multidevice.ConflictPending,
			Resource:    "doc/intro",
			DetectedAt:  now,
			Events: []*multidevice.WorkspaceEvent{
				{DeviceID: "dev-a"},
				{DeviceID: "dev-b"},
			},
		},
		{
			ID:          "c-resolved",
			WorkspaceID: wsID,
			Type:        multidevice.ConflictOrdering,
			Status:      multidevice.ConflictResolved,
			Resource:    "doc/intro",
			DetectedAt:  now,
			Events: []*multidevice.WorkspaceEvent{
				{DeviceID: "dev-a"},
			},
		},
		{
			ID:          "c-auto",
			WorkspaceID: wsID,
			Type:        multidevice.ConflictMetadata,
			Status:      multidevice.ConflictAutoResolved,
			Resource:    "doc/cover",
			DetectedAt:  now,
			Events: []*multidevice.WorkspaceEvent{
				{DeviceID: "dev-c"},
			},
		},
	}
	seedConflicts(t, home, wsID, conflicts)
}

// TestConflictsList_FormatJSON verifies the rich row + JSON dispatch.
func TestConflictsList_FormatJSON(t *testing.T) {
	home := t.TempDir()
	const wsID = "ws-team"
	seedConflictFixtures(t, home, wsID)

	out, errStr, err := runAPS(t, home,
		"workspace", "conflicts", "list",
		"--workspace", wsID,
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("workspace conflicts list: %v\nstderr: %s", err, errStr)
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\nstdout: %s", err, out)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (no filter), got %d: %s", len(rows), out)
	}

	want := map[string]string{
		"id":        "c-pending",
		"workspace": wsID,
		"type":      string(multidevice.ConflictConcurrentWrite),
		"status":    string(multidevice.ConflictPending),
		"resource":  "doc/intro",
	}
	var pendingRow map[string]any
	for _, r := range rows {
		if r["id"] == "c-pending" {
			pendingRow = r
			break
		}
	}
	if pendingRow == nil {
		t.Fatalf("expected row for c-pending; got: %s", out)
	}
	for k, v := range want {
		if got, _ := pendingRow[k].(string); got != v {
			t.Errorf("c-pending field %q = %q, want %q", k, got, v)
		}
	}
	// Devices count is numeric (json.Number-able as float64).
	if dev, ok := pendingRow["devices"].(float64); !ok || dev != 2 {
		t.Errorf("c-pending devices = %v, want 2", pendingRow["devices"])
	}
}

// TestConflictsList_UnresolvedFilter drops resolved + auto-resolved rows.
func TestConflictsList_UnresolvedFilter(t *testing.T) {
	home := t.TempDir()
	const wsID = "ws-team"
	seedConflictFixtures(t, home, wsID)

	out, errStr, err := runAPS(t, home,
		"workspace", "conflicts", "list",
		"--workspace", wsID,
		"--unresolved",
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("conflicts list --unresolved: %v\nstderr: %s", err, errStr)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\nstdout: %s", err, out)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 unresolved row, got %d: %s", len(rows), out)
	}
	if rows[0]["id"] != "c-pending" {
		t.Errorf("expected c-pending, got %v", rows[0]["id"])
	}
}

// TestConflictsList_TableHeaders confirms the default-format table
// emits the rich row headers.
func TestConflictsList_TableHeaders(t *testing.T) {
	home := t.TempDir()
	const wsID = "ws-team"
	seedConflictFixtures(t, home, wsID)

	out, errStr, err := runAPS(t, home,
		"workspace", "conflicts", "list",
		"--workspace", wsID,
	)
	if err != nil {
		t.Fatalf("conflicts list table: %v\nstderr: %s", err, errStr)
	}
	for _, h := range []string{"ID", "WORKSPACE", "TYPE", "STATUS", "RESOURCE"} {
		if !strings.Contains(out, h) {
			t.Errorf("expected header %q in table output, got: %s", h, out)
		}
	}
}
