package listing

import (
	"bytes"
	"encoding/json"
	"image/color"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"hop.top/kit/go/console/output"
)

// ansiRe matches ANSI color/style escape sequences. Used by SetTableStyle
// non-TTY tests to assert the styled renderer falls through to plain
// tabwriter when the writer is not a *os.File terminal.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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

// TestRenderList_StyledNonTTY_FallsThroughToPlain documents the contract
// that SetTableStyle is a no-op for non-TTY writers. The styled path in
// kit/output is gated on writerIsTTY (an *os.File + isatty.IsTerminal
// check); a bytes.Buffer is neither, so the plain tabwriter renderer
// runs and the output stays diff-friendly. This guards every callsite
// that captures listing output for tests or pipes it into another tool.
func TestRenderList_StyledNonTTY_FallsThroughToPlain(t *testing.T) {
	prev, hadPrev := activeTableStyle()
	t.Cleanup(func() {
		if hadPrev {
			SetTableStyle(prev)
			return
		}
		// Reset to a fresh value; there's no Unset path because
		// production never unsets after root init wires it.
		tableStyleMu.Lock()
		tableStyle = output.TableStyle{}
		tableStyleSet = false
		tableStyleMu.Unlock()
	})

	SetTableStyle(output.TableStyle{
		Border:           lipgloss.NormalBorder(),
		BorderForeground: color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Header:           color.RGBA{R: 200, G: 200, B: 200, A: 255},
		Primary:          color.RGBA{R: 126, G: 217, B: 87, A: 255},
		Secondary:        color.RGBA{R: 255, G: 102, B: 196, A: 255},
		Muted:            color.RGBA{R: 100, G: 100, B: 100, A: 255},
	})

	var buf bytes.Buffer
	rows := []sampleRow{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}}
	if err := RenderList(&buf, output.Table, rows); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := buf.String()

	if ansiRe.MatchString(out) {
		t.Errorf("non-TTY styled path leaked ANSI escapes: %q", out)
	}
	for _, r := range []rune{'┌', '┐', '└', '┘', '│', '─'} {
		if strings.ContainsRune(out, r) {
			t.Errorf("non-TTY styled path leaked box-drawing rune %q: %q", r, out)
		}
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") {
		t.Errorf("expected headers in output, got: %q", out)
	}
	for _, r := range rows {
		if !strings.Contains(out, r.Name) {
			t.Errorf("expected row %q in output, got: %q", r.Name, out)
		}
	}
}

// TestSetTableStyle_Replaces verifies the setter swaps the active style.
// Production calls it once during root init, but tests may want to reset
// or repoint it; the contract is "last write wins".
func TestSetTableStyle_Replaces(t *testing.T) {
	prev, hadPrev := activeTableStyle()
	t.Cleanup(func() {
		if hadPrev {
			SetTableStyle(prev)
			return
		}
		tableStyleMu.Lock()
		tableStyle = output.TableStyle{}
		tableStyleSet = false
		tableStyleMu.Unlock()
	})

	first := output.TableStyle{Header: color.RGBA{R: 1}}
	SetTableStyle(first)
	got, ok := activeTableStyle()
	if !ok {
		t.Fatal("SetTableStyle: expected style to be set")
	}
	if got.Header != first.Header {
		t.Errorf("first SetTableStyle: header = %v, want %v", got.Header, first.Header)
	}

	second := output.TableStyle{Header: color.RGBA{R: 2}}
	SetTableStyle(second)
	got, _ = activeTableStyle()
	if got.Header != second.Header {
		t.Errorf("second SetTableStyle: header = %v, want %v", got.Header, second.Header)
	}
}
