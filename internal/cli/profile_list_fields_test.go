package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
)

func listFieldsFixtures() []core.Profile {
	return []core.Profile{
		{ID: "owner-1", DisplayName: "Owner", Type: "human", Email: "owner@example.test"},
		{ID: "bot-1", DisplayName: "Bot", Email: "bot@example.test", ReportsTo: "owner-1"},
	}
}

func isolateAPSData(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_STATE_HOME", dir)
	t.Setenv("XDG_CACHE_HOME", dir)
}

func renderListRows(t *testing.T, format string, profiles []core.Profile) string {
	t.Helper()
	rows := make([]profileSummaryRow, 0, len(profiles))
	for _, p := range profiles {
		rows = append(rows, profileToSummaryRow(p))
	}
	var buf bytes.Buffer
	if err := listing.RenderList(&buf, format, rows); err != nil {
		t.Fatalf("RenderList(%q): %v", format, err)
	}
	return buf.String()
}

// TestProfileListJSONCarriesTypeAndReportsTo pins list/show agreement
// on type and reports_to: same keys, same presence when unset.
func TestProfileListJSONCarriesTypeAndReportsTo(t *testing.T) {
	isolateAPSData(t)
	profiles := listFieldsFixtures()

	var rows []map[string]any
	if err := json.Unmarshal([]byte(renderListRows(t, "json", profiles)), &rows); err != nil {
		t.Fatalf("list JSON not parseable: %v", err)
	}
	if len(rows) != len(profiles) {
		t.Fatalf("rows = %d, want %d", len(rows), len(profiles))
	}
	for i := range profiles {
		p := profiles[i]
		var show map[string]any
		if err := json.Unmarshal([]byte(renderShow(t, "json", &p)), &show); err != nil {
			t.Fatalf("show JSON not parseable: %v", err)
		}
		for _, key := range []string{"type", "reports_to"} {
			sv, sok := show[key]
			lv, lok := rows[i][key]
			if sok != lok || sv != lv {
				t.Errorf("%s %s: list=(%v,%v) show=(%v,%v)", p.ID, key, lv, lok, sv, sok)
			}
		}
	}
	if rows[0]["type"] != "human" {
		t.Errorf("owner-1 type = %v, want human", rows[0]["type"])
	}
	if rows[1]["reports_to"] != "owner-1" {
		t.Errorf("bot-1 reports_to = %v, want owner-1", rows[1]["reports_to"])
	}
}

func TestProfileListYAMLCarriesTypeAndReportsTo(t *testing.T) {
	isolateAPSData(t)
	var rows []map[string]any
	if err := yaml.Unmarshal([]byte(renderListRows(t, "yaml", listFieldsFixtures())), &rows); err != nil {
		t.Fatalf("list YAML not parseable: %v", err)
	}
	if rows[0]["type"] != "human" {
		t.Errorf("owner-1 type = %v, want human", rows[0]["type"])
	}
	if rows[1]["reports_to"] != "owner-1" {
		t.Errorf("bot-1 reports_to = %v, want owner-1", rows[1]["reports_to"])
	}
}

// TestProfileListTableOmitsTypeAndReportsTo keeps the default table
// layout unchanged.
func TestProfileListTableOmitsTypeAndReportsTo(t *testing.T) {
	isolateAPSData(t)
	out := renderListRows(t, "table", listFieldsFixtures())
	for _, hdr := range []string{"TYPE", "REPORTS"} {
		if strings.Contains(strings.ToUpper(out), hdr) {
			t.Errorf("table output unexpectedly contains %q:\n%s", hdr, out)
		}
	}
}
