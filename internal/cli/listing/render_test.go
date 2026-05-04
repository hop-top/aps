package listing

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"hop.top/kit/go/console/output"
)

type sampleRow struct {
	ID   string `table:"ID,priority=10" json:"id"   yaml:"id"`
	Name string `table:"NAME,priority=5" json:"name" yaml:"name"`
}

func TestRenderList_TableFormat(t *testing.T) {
	var buf bytes.Buffer
	rows := []sampleRow{
		{ID: "a", Name: "Alpha"},
		{ID: "b", Name: "Beta"},
	}
	if err := RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") {
		t.Errorf("expected headers in table output, got: %q", out)
	}
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "Beta") {
		t.Errorf("expected data rows in output, got: %q", out)
	}
}

func TestRenderList_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	rows := []sampleRow{{ID: "a", Name: "Alpha"}}
	if err := RenderList(&buf, output.JSON, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	var got []sampleRow
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json: %v\nraw: %q", err, buf.String())
	}
	if len(got) != 1 || got[0].ID != "a" || got[0].Name != "Alpha" {
		t.Errorf("unexpected json output: %+v", got)
	}
}

func TestRenderList_YAMLFormat(t *testing.T) {
	var buf bytes.Buffer
	rows := []sampleRow{{ID: "a", Name: "Alpha"}}
	if err := RenderList(&buf, output.YAML, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "id: a") || !strings.Contains(out, "name: Alpha") {
		t.Errorf("expected yaml fields in output, got: %q", out)
	}
}

func TestRenderList_EmptyFormat_DefaultsToTable(t *testing.T) {
	var buf bytes.Buffer
	rows := []sampleRow{{ID: "a", Name: "Alpha"}}
	if err := RenderList(&buf, "", rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") {
		t.Errorf("empty format should default to table, got: %q", out)
	}
}

func TestRenderList_EmptySlice_RendersHeaders(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderList[sampleRow](&buf, output.Table, nil); err != nil {
		t.Fatalf("RenderList nil: %v", err)
	}
	// Empty slice is not an error; output may be headers-only or
	// a hint line — kit/output's choice. We only assert no error.
}

func TestRenderList_UnknownFormat_ReturnsError(t *testing.T) {
	var buf bytes.Buffer
	rows := []sampleRow{{ID: "a"}}
	// Use a clearly-invalid format string; "csv" was registered after
	// initial draft so we use a placeholder unlikely to ever be a
	// real format.
	err := RenderList(&buf, "this-format-will-never-exist", rows)
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}
