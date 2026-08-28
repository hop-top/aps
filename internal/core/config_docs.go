package core

import (
	"fmt"
	"reflect"
	"strings"
)

// This file backs the generated "Fields" table in
// docs/dev/configuration.md (see internal/tools/configmd and
// `make docs-gen`). ConfigFieldDocs walks Config's yaml tags by
// reflection, so every struct field must be classified below: either it
// gets a row in the table, or it is explicitly listed as documented
// elsewhere. An unclassified field fails both `make docs-gen` and the
// pin tests, so new config fields cannot ship undocumented.

// ConfigFieldDoc is one row of the generated global-config Fields table.
type ConfigFieldDoc struct {
	Path        string // yaml path, e.g. "isolation.default_level"
	Default     string // rendered default value (markdown)
	Description string
}

// configFieldTable documents the Config leaves that appear in the
// Fields table, keyed by yaml path. Path is filled in during the walk.
var configFieldTable = map[string]ConfigFieldDoc{
	"prefix": {
		Default:     "`APS`",
		Description: "Prefix for environment variables injected into sessions",
	},
	"isolation.default_level": {
		Default:     "`process`",
		Description: "Default isolation level for profiles that don't specify one",
	},
	"isolation.fallback_enabled": {
		Default:     "`true`",
		Description: "Allow degraded-mode operation if preferred isolation is unavailable",
	},
	"capability_sources": {
		Default:     "`[]`",
		Description: "Additional directories to search for capability definitions",
	},
}

// Locations for config sections whose leaves are documented outside the
// global Fields table.
const (
	messengersDocRef = "docs/user/messengers.md (Secret Store Backends)"
	profileGodocRef  = "ProfileDefaultsConfig / ProfileAvatarConfig godoc (profile creation defaults)"
	idemGodocRef     = "IdempotencyConfig godoc (--idempotency-key replay store)"
)

// configFieldsOutOfTable lists Config leaves deliberately absent from
// the Fields table, each mapped to the place that documents it instead.
var configFieldsOutOfTable = map[string]string{
	"secrets.backend":     messengersDocRef,
	"secrets.service":     messengersDocRef,
	"secrets.prefix":      messengersDocRef,
	"secrets.addr":        messengersDocRef,
	"secrets.token":       messengersDocRef,
	"secrets.mount":       messengersDocRef,
	"secrets.project":     messengersDocRef,
	"secrets.env":         messengersDocRef,
	"secrets.vault":       messengersDocRef,
	"secrets.connect_url": messengersDocRef,
	"secrets.repo":        messengersDocRef,
	"secrets.token_env":   messengersDocRef,

	"profile.color":           profileGodocRef,
	"profile.avatar.enabled":  profileGodocRef,
	"profile.avatar.provider": profileGodocRef,
	"profile.avatar.style":    profileGodocRef,
	"profile.avatar.size":     profileGodocRef,
	"profile.avatar.format":   profileGodocRef,

	"idempotency.backend": idemGodocRef,
	"idempotency.path":    idemGodocRef,
	"idempotency.ttl":     idemGodocRef,
}

// ConfigFieldDocs returns the documented global config fields in struct
// declaration order. It errors when a Config leaf is neither documented
// nor explicitly excluded, and when either map carries a stale path
// that no longer exists on Config.
func ConfigFieldDocs() ([]ConfigFieldDoc, error) {
	leaves := configLeaves(reflect.TypeOf(Config{}), "")
	seen := make(map[string]bool, len(leaves))
	docs := make([]ConfigFieldDoc, 0, len(configFieldTable))
	for _, path := range leaves {
		seen[path] = true
		doc, documented := configFieldTable[path]
		_, excluded := configFieldsOutOfTable[path]
		switch {
		case documented && excluded:
			return nil, fmt.Errorf("config field %q is both in configFieldTable and configFieldsOutOfTable", path)
		case documented:
			doc.Path = path
			docs = append(docs, doc)
		case excluded:
			// Documented elsewhere; not part of the Fields table.
		default:
			return nil, fmt.Errorf(
				"config field %q is undocumented: add it to configFieldTable or configFieldsOutOfTable", path)
		}
	}
	for path := range configFieldTable {
		if !seen[path] {
			return nil, fmt.Errorf("configFieldTable documents %q, which is not a Config field", path)
		}
	}
	for path := range configFieldsOutOfTable {
		if !seen[path] {
			return nil, fmt.Errorf("configFieldsOutOfTable lists %q, which is not a Config field", path)
		}
	}
	return docs, nil
}

// configLeaves returns the yaml paths of every exported leaf field
// reachable from t, in declaration order. Struct-typed fields recurse;
// everything else (including named scalar types such as IsolationLevel
// and AutoMode) is a leaf.
func configLeaves(t reflect.Type, prefix string) []string {
	leaves := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := yamlFieldName(f)
		if name == "-" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		ft := f.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			leaves = append(leaves, configLeaves(ft, path)...)
			continue
		}
		leaves = append(leaves, path)
	}
	return leaves
}

// yamlFieldName resolves the yaml key for a struct field: the first
// comma-separated element of the yaml tag, falling back to the
// lowercased field name as yaml.v3 does.
func yamlFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("yaml")
	if idx := strings.Index(tag, ","); idx >= 0 {
		tag = tag[:idx]
	}
	if tag == "" {
		return strings.ToLower(f.Name)
	}
	return tag
}
