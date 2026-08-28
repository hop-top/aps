package core

import (
	"reflect"
	"testing"
)

// TestConfigFieldDocsPinned asserts the generated Fields table content:
// every Config leaf is classified (nil error) and the documented rows
// match the docs/dev/configuration.md table exactly, in struct order.
func TestConfigFieldDocsPinned(t *testing.T) {
	docs, err := ConfigFieldDocs()
	if err != nil {
		t.Fatalf("ConfigFieldDocs: %v", err)
	}
	want := []ConfigFieldDoc{
		{
			Path:        "prefix",
			Default:     "`APS`",
			Description: "Prefix for environment variables injected into sessions",
		},
		{
			Path:        "isolation.default_level",
			Default:     "`process`",
			Description: "Default isolation level for profiles that don't specify one",
		},
		{
			Path:        "isolation.fallback_enabled",
			Default:     "`true`",
			Description: "Allow degraded-mode operation if preferred isolation is unavailable",
		},
		{
			Path:        "capability_sources",
			Default:     "`[]`",
			Description: "Additional directories to search for capability definitions",
		},
	}
	if !reflect.DeepEqual(docs, want) {
		t.Errorf("ConfigFieldDocs mismatch:\n got: %#v\nwant: %#v", docs, want)
	}
}

// TestConfigFieldExclusionsHaveLocations asserts every out-of-table
// field names the place that documents it instead.
func TestConfigFieldExclusionsHaveLocations(t *testing.T) {
	for path, where := range configFieldsOutOfTable {
		if where == "" {
			t.Errorf("excluded config field %q has no documentation location", path)
		}
	}
}

// TestConfigLeavesRecursion sanity-checks the reflection walk: nested
// struct fields must surface as dotted leaf paths.
func TestConfigLeavesRecursion(t *testing.T) {
	leaves := configLeaves(reflect.TypeOf(Config{}), "")
	got := map[string]bool{}
	for _, l := range leaves {
		got[l] = true
	}
	for _, path := range []string{
		"prefix",
		"isolation.default_level",
		"secrets.token_env",
		"profile.avatar.provider",
		"idempotency.ttl",
	} {
		if !got[path] {
			t.Errorf("configLeaves missing expected leaf %q (leaves: %v)", path, leaves)
		}
	}
}
