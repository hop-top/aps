package workspace_e2e

import (
	"encoding/json"
	"strings"
	"testing"

	collab "hop.top/aps/internal/core/collaboration"
)

// seedCtxFixtures writes a small set of context variables exercising
// the inferred-type heuristic (string, number, bool, json) and the
// --key-prefix filter.
func seedCtxFixtures(t *testing.T, home, wsID string) {
	t.Helper()
	now := nowUTC()
	vars := []collab.ContextVariable{
		{Key: "feature.alpha", Value: "true", Version: 1, UpdatedBy: "noor", UpdatedAt: now},
		{Key: "feature.beta", Value: "false", Version: 1, UpdatedBy: "noor", UpdatedAt: now},
		{Key: "limits.max", Value: "42", Version: 2, UpdatedBy: "sami", UpdatedAt: now},
		{Key: "config.json", Value: `{"a":1}`, Version: 1, UpdatedBy: "rami", UpdatedAt: now},
		{Key: "name", Value: "intro", Version: 1, UpdatedBy: "kai", UpdatedAt: now},
	}
	seedContext(t, home, wsID, vars)
}

// TestCtxList_FormatJSON dumps every variable in JSON. Asserts the row
// shape carries the inferred Type + Size, plus the json struct tags.
func TestCtxList_FormatJSON(t *testing.T) {
	home := t.TempDir()
	const wsID = "ws-team"
	seedCtxFixtures(t, home, wsID)

	out, errStr, err := runAPS(t, home,
		"workspace", "ctx", "list",
		"--workspace", wsID,
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("ctx list: %v\nstderr: %s", err, errStr)
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\nstdout: %s", err, out)
	}
	if len(rows) != 5 {
		t.Fatalf("expected 5 ctx rows, got %d: %s", len(rows), out)
	}

	byKey := map[string]map[string]any{}
	for _, r := range rows {
		byKey[r["key"].(string)] = r
	}
	wantTypes := map[string]string{
		"feature.alpha": "bool",
		"feature.beta":  "bool",
		"limits.max":    "number",
		"config.json":   "json-object",
		"name":          "string",
	}
	for k, want := range wantTypes {
		got, _ := byKey[k]["type"].(string)
		if got != want {
			t.Errorf("type for %q = %q, want %q", k, got, want)
		}
	}
	if size, _ := byKey["limits.max"]["size"].(float64); size != 2 {
		t.Errorf("size for limits.max = %v, want 2", byKey["limits.max"]["size"])
	}
}

// TestCtxList_KeyPrefixFilter narrows the listing.
func TestCtxList_KeyPrefixFilter(t *testing.T) {
	home := t.TempDir()
	const wsID = "ws-team"
	seedCtxFixtures(t, home, wsID)

	out, errStr, err := runAPS(t, home,
		"workspace", "ctx", "list",
		"--workspace", wsID,
		"--key-prefix", "feature.",
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("ctx list --key-prefix: %v\nstderr: %s", err, errStr)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows under prefix feature., got %d: %s", len(rows), out)
	}
	for _, r := range rows {
		k := r["key"].(string)
		if !strings.HasPrefix(k, "feature.") {
			t.Errorf("--key-prefix leak: %s", k)
		}
	}

	// Non-matching prefix yields zero rows.
	out, _, err = runAPS(t, home,
		"workspace", "ctx", "list",
		"--workspace", wsID,
		"--key-prefix", "missing.",
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("ctx list --key-prefix missing.: %v", err)
	}
	rows = nil
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for missing prefix, got %d", len(rows))
	}
}

// TestCtxList_TableHeaders confirms the default-format table emits the
// rich row headers.
func TestCtxList_TableHeaders(t *testing.T) {
	home := t.TempDir()
	const wsID = "ws-team"
	seedCtxFixtures(t, home, wsID)

	out, errStr, err := runAPS(t, home,
		"workspace", "ctx", "list",
		"--workspace", wsID,
	)
	if err != nil {
		t.Fatalf("ctx list table: %v\nstderr: %s", err, errStr)
	}
	for _, h := range []string{"KEY", "TYPE", "SIZE", "VERSION", "UPDATED"} {
		if !strings.Contains(out, h) {
			t.Errorf("expected header %q in table, got: %s", h, out)
		}
	}
}
