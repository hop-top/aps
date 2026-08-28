package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// TestSecretsBackendsMetaMatchesBackends pins SecretsBackendsMeta to
// SecretsBackends: same backends, same order, no extra or stale rows.
func TestSecretsBackendsMetaMatchesBackends(t *testing.T) {
	if got, want := len(SecretsBackendsMeta), len(SecretsBackends); got != want {
		t.Fatalf("SecretsBackendsMeta has %d rows, SecretsBackends has %d backends", got, want)
	}
	for i, name := range SecretsBackends {
		if SecretsBackendsMeta[i].Name != name {
			t.Errorf("SecretsBackendsMeta[%d].Name = %q, want %q (SecretsBackends order)",
				i, SecretsBackendsMeta[i].Name, name)
		}
	}
}

// TestSecretsBackendConstsHaveMeta parses secrets.go and asserts every
// SecretsBackend* const has a SecretsBackendsMeta row and vice versa, so
// a new backend const cannot ship without documentation metadata.
func TestSecretsBackendConstsHaveMeta(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "secrets.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing secrets.go: %v", err)
	}

	consts := map[string]string{} // const name -> string value
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, ident := range vs.Names {
				if !strings.HasPrefix(ident.Name, "SecretsBackend") || i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				val, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquoting %s value %s: %v", ident.Name, lit.Value, err)
				}
				consts[ident.Name] = val
			}
		}
	}
	if len(consts) == 0 {
		t.Fatal("no SecretsBackend* consts found in secrets.go; did they move?")
	}

	metaNames := map[string]bool{}
	for _, m := range SecretsBackendsMeta {
		metaNames[m.Name] = true
	}
	constValues := map[string]bool{}
	for name, val := range consts {
		constValues[val] = true
		if !metaNames[val] {
			t.Errorf("const %s (%q) has no SecretsBackendsMeta row", name, val)
		}
	}
	for _, m := range SecretsBackendsMeta {
		if !constValues[m.Name] {
			t.Errorf("SecretsBackendsMeta row %q has no SecretsBackend* const", m.Name)
		}
	}
}

// TestSecretsBackendsMetaContent asserts every row is fully filled in,
// exactly one backend is the default, and BuildTag agrees with
// optionalBackends.
func TestSecretsBackendsMetaContent(t *testing.T) {
	defaults := 0
	for _, m := range SecretsBackendsMeta {
		if m.Where == "" {
			t.Errorf("backend %q has an empty Where description", m.Name)
		}
		if m.Required == "" {
			t.Errorf("backend %q has an empty Required description", m.Name)
		}
		if m.Default {
			defaults++
			if m.Name != SecretsBackendFile {
				t.Errorf("backend %q claims to be the default; want %q", m.Name, SecretsBackendFile)
			}
		}
		if got, want := m.BuildTag, optionalBackends[m.Name]; got != want {
			t.Errorf("backend %q BuildTag = %q, optionalBackends says %q", m.Name, got, want)
		}
	}
	if defaults != 1 {
		t.Errorf("found %d default backends, want exactly 1", defaults)
	}
}
