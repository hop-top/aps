package messenger

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// platformConstValues extracts every MessengerPlatform const value declared
// in this package's non-test sources, so the metadata table is pinned to
// the enum itself rather than to a hand-maintained list.
func platformConstValues(t *testing.T) []MessengerPlatform {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	fset := token.NewFileSet()
	var values []MessengerPlatform
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if ident, ok := value.Type.(*ast.Ident); !ok || ident.Name != "MessengerPlatform" {
					continue
				}
				for _, expr := range value.Values {
					lit, ok := expr.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					text, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatalf("unquote %s in %s: %v", lit.Value, name, err)
					}
					values = append(values, MessengerPlatform(text))
				}
			}
		}
	}
	if len(values) == 0 {
		t.Fatal("no MessengerPlatform consts found in package sources")
	}
	return values
}

// TestPlatformMetaCompleteAgainstEnum pins the metadata table to the
// MessengerPlatform const block in both directions: a new platform fails
// here until a metadata row lands, and a stale row fails when the enum
// drops a platform.
func TestPlatformMetaCompleteAgainstEnum(t *testing.T) {
	enum := map[MessengerPlatform]bool{}
	for _, platform := range platformConstValues(t) {
		if enum[platform] {
			t.Errorf("duplicate MessengerPlatform const value %q", platform)
		}
		enum[platform] = true
	}

	rows := map[MessengerPlatform]bool{}
	for _, row := range platformMeta {
		if rows[row.Platform] {
			t.Errorf("duplicate platform meta row %q", row.Platform)
		}
		rows[row.Platform] = true
	}

	for platform := range enum {
		if !rows[platform] {
			t.Errorf("platform %q has no metadata row", platform)
		}
	}
	for platform := range rows {
		if !enum[platform] {
			t.Errorf("stale metadata row %q: platform is not in the enum", platform)
		}
	}
}

// TestPlatformMetaRowsPopulated requires the presentation columns docs
// render. Every row carries a display name and channel ID format; aliased
// message adapters must be fully documented.
func TestPlatformMetaRowsPopulated(t *testing.T) {
	for _, row := range platformMeta {
		if row.Display == "" {
			t.Errorf("platform %q has empty Display", row.Platform)
		}
		if row.ChannelID == "" {
			t.Errorf("platform %q has empty ChannelID", row.Platform)
		}
		if row.Alias == "" {
			continue
		}
		required := map[string]string{
			"ChannelControl": row.ChannelControl,
			"AuthSource":     row.AuthSource,
			"Ingress":        row.Ingress,
			"Normalize":      row.Normalize,
			"Reply":          row.Reply,
			"Signature":      row.Signature,
			"Support":        row.Support,
			"Maturity":       row.Maturity,
		}
		for column, value := range required {
			if value == "" {
				t.Errorf("aliased platform %q has empty %s", row.Platform, column)
			}
		}
	}
}

func TestAllPlatformMetaSorted(t *testing.T) {
	all := AllPlatformMeta()
	if len(all) != len(platformMeta) {
		t.Fatalf("AllPlatformMeta returned %d rows, want %d", len(all), len(platformMeta))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].Platform >= all[i].Platform {
			t.Errorf("AllPlatformMeta not sorted at %q >= %q", all[i-1].Platform, all[i].Platform)
		}
	}
}

func TestPlatformMetaFor(t *testing.T) {
	row, ok := PlatformMetaFor(PlatformTelegram)
	if !ok || row.Platform != PlatformTelegram {
		t.Fatalf("PlatformMetaFor(telegram) = %+v, %v", row, ok)
	}
	if _, ok := PlatformMetaFor(MessengerPlatform("nope")); ok {
		t.Fatal("PlatformMetaFor should miss unknown platforms")
	}
	if PlatformDisplay(PlatformWhatsApp) != "WhatsApp" {
		t.Fatalf("PlatformDisplay(whatsapp) = %q", PlatformDisplay(PlatformWhatsApp))
	}
	if PlatformDisplay(MessengerPlatform("nope")) != "" {
		t.Fatal("PlatformDisplay should be empty for unknown platforms")
	}
}

// TestChannelIDFormatDerivedFromMeta guards the derived map so the CLI
// format hint always matches the metadata table.
func TestChannelIDFormatDerivedFromMeta(t *testing.T) {
	if len(ChannelIDFormat) != len(platformMeta) {
		t.Fatalf("ChannelIDFormat has %d entries, meta has %d", len(ChannelIDFormat), len(platformMeta))
	}
	for _, row := range platformMeta {
		if ChannelIDFormat[row.Platform] != row.ChannelID {
			t.Errorf("ChannelIDFormat[%q] = %q, want %q", row.Platform, ChannelIDFormat[row.Platform], row.ChannelID)
		}
	}
}
