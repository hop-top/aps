package messenger

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	coremessenger "hop.top/aps/internal/core/messenger"
)

// metadataReceiverTypes extracts the receiver type name of every
// Metadata() ProviderRuntimeMetadata method declared in this package's
// non-test sources, so first-class providers are enumerated from the code
// itself rather than from a hand-maintained list. Unexported receivers
// are skipped: they are internal decorators delegating to a first-class
// provider, not providers of their own.
func metadataReceiverTypes(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	fset := token.NewFileSet()
	var receivers []string
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
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Metadata" || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
				continue
			}
			selector, ok := fn.Type.Results.List[0].Type.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "ProviderRuntimeMetadata" {
				continue
			}
			recv := fn.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			ident, ok := recv.(*ast.Ident)
			if !ok || !ident.IsExported() {
				continue
			}
			receivers = append(receivers, ident.Name)
		}
	}
	if len(receivers) == 0 {
		t.Fatal("no Metadata() ProviderRuntimeMetadata methods found in package sources")
	}
	return receivers
}

// TestFirstClassProvidersHavePlatformMetaRows pins every first-class
// message provider in this package to a presentation row in the core
// platform metadata table: a new provider fails here until it is
// registered below and a metadata row lands for its platform.
func TestFirstClassProvidersHavePlatformMetaRows(t *testing.T) {
	providers := map[string]coremessenger.MessageProvider{
		"TelegramProvider": NewTelegramProvider(TelegramProviderConfig{}),
		"SlackProvider":    NewSlackProvider(SlackProviderConfig{}),
		"TeamsProvider":    NewTeamsProvider(TeamsProviderConfig{}),
		"DiscordProvider":  NewDiscordProvider(DiscordProviderConfig{}),
	}

	declared := metadataReceiverTypes(t)
	seen := map[string]bool{}
	for _, name := range declared {
		seen[name] = true
		provider, ok := providers[name]
		if !ok {
			t.Errorf("provider %s is not covered here: register it above and add a platform metadata row", name)
			continue
		}
		metadata := provider.Metadata()
		platform := coremessenger.MessengerPlatform(metadata.Provider)
		row, ok := coremessenger.PlatformMetaFor(platform)
		if !ok {
			t.Errorf("provider %s (platform %q) has no platform metadata row", name, platform)
			continue
		}
		if row.Display == "" {
			t.Errorf("provider %s (platform %q) has a metadata row without a display name", name, platform)
		}
	}
	for name := range providers {
		if !seen[name] {
			t.Errorf("stale provider entry %s: no Metadata() method found in package sources", name)
		}
	}
}
