package adapter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// declaredValues parses every non-test source file in this package and
// returns the string value of each constant declared with the named
// type. Enumerating from source keeps the meta tables pinned to the
// const blocks in both directions: a new const fails its pin test
// until a meta row exists, and a row without a matching const fails
// as stale.
func declaredValues(t *testing.T, typeName string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	fset := token.NewFileSet()
	values := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				ident, ok := vs.Type.(*ast.Ident)
				if !ok || ident.Name != typeName {
					continue
				}
				for _, v := range vs.Values {
					lit, ok := v.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					s, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatalf("unquote %s: %v", lit.Value, err)
					}
					values[s] = true
				}
			}
		}
	}
	if len(values) == 0 {
		t.Fatalf("no %s constants found; enumeration broken", typeName)
	}
	return values
}

// assertPinned checks a description table against the declared consts
// in both directions and rejects empty descriptions.
func assertPinned[T ~string](t *testing.T, tableName, typeName string, table map[T]string) {
	t.Helper()
	decl := declaredValues(t, typeName)
	for v := range decl {
		if _, ok := table[T(v)]; !ok {
			t.Errorf("%s missing row for declared %s const %q", tableName, typeName, v)
		}
	}
	for k, desc := range table {
		if !decl[string(k)] {
			t.Errorf("%s row %q matches no declared %s const", tableName, k, typeName)
		}
		if desc == "" {
			t.Errorf("%s row %q has empty description", tableName, k)
		}
	}
}

func TestAdapterTypesPinned(t *testing.T) {
	decl := declaredValues(t, "AdapterType")
	for v := range decl {
		if _, ok := AdapterTypes[AdapterType(v)]; !ok {
			t.Errorf("AdapterTypes missing row for declared AdapterType const %q", v)
		}
	}
	for k, meta := range AdapterTypes {
		if !decl[string(k)] {
			t.Errorf("AdapterTypes row %q matches no declared AdapterType const", k)
		}
		if meta.Type != k {
			t.Errorf("AdapterTypes row %q has mismatched Type %q", k, meta.Type)
		}
		if meta.Description == "" {
			t.Errorf("AdapterTypes row %q has empty description", k)
		}
	}
}

func TestLoadingStrategiesPinned(t *testing.T) {
	decl := declaredValues(t, "LoadingStrategy")
	for v := range decl {
		if _, ok := LoadingStrategies[LoadingStrategy(v)]; !ok {
			t.Errorf("LoadingStrategies missing row for declared LoadingStrategy const %q", v)
		}
	}
	for k, meta := range LoadingStrategies {
		if !decl[string(k)] {
			t.Errorf("LoadingStrategies row %q matches no declared LoadingStrategy const", k)
		}
		if meta.Display == "" {
			t.Errorf("LoadingStrategies row %q has empty display name", k)
		}
		if meta.Description == "" {
			t.Errorf("LoadingStrategies row %q has empty description", k)
		}
		if meta.Persistence == "" {
			t.Errorf("LoadingStrategies row %q has empty persistence", k)
		}
	}
}

func TestAdapterScopesPinned(t *testing.T) {
	assertPinned(t, "AdapterScopes", "AdapterScope", AdapterScopes)
}

func TestAdapterStatesPinned(t *testing.T) {
	assertPinned(t, "AdapterStates", "AdapterState", AdapterStates)
}

func TestHealthStatusesPinned(t *testing.T) {
	assertPinned(t, "HealthStatuses", "HealthStatus", HealthStatuses)
}
