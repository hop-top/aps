package core

import (
	"reflect"
	"strings"
	"testing"
)

// tagName returns the serialized field name and whether omitempty is
// set, for the given struct tag key.
func tagName(f reflect.StructField, key string) (name string, omitempty, present bool) {
	raw, ok := f.Tag.Lookup(key)
	if !ok {
		return "", false, false
	}
	parts := strings.Split(raw, ",")
	name = parts[0]
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty, true
}

// TestProfileJSONMatchesYAMLTags pins Factor 3 shape stability: the
// same record must serialize under the same key names and the same
// emptiness rules regardless of the requested format.
//
// Profile originally carried yaml tags only, so encoding/json fell
// back to Go field names: `--format json` produced "ID" and
// "DisplayName" while `--format yaml` produced "id" and
// "display_name", and omitempty was ignored in JSON — one command
// describing one profile with two incompatible schemas.
func TestProfileJSONMatchesYAMLTags(t *testing.T) {
	typ := reflect.TypeOf(Profile{})

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}

		yamlName, yamlOmit, hasYAML := tagName(f, "yaml")
		if !hasYAML {
			continue
		}
		jsonName, jsonOmit, hasJSON := tagName(f, "json")

		if !hasJSON {
			t.Errorf("field %s has a yaml tag (%q) but no json tag: "+
				"JSON will fall back to the Go field name", f.Name, yamlName)
			continue
		}
		if jsonName != yamlName {
			t.Errorf("field %s: json name %q != yaml name %q", f.Name, jsonName, yamlName)
		}
		if jsonOmit != yamlOmit {
			t.Errorf("field %s: omitempty differs (json=%v yaml=%v); "+
				"an absent field in one format must be absent in the other",
				f.Name, jsonOmit, yamlOmit)
		}
	}
}
