package adapter

import (
	"strings"
	"testing"

	coreadapter "hop.top/aps/internal/core/adapter"
)

func TestRunCreateCreatesAdvertisedAdapterTypes(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	t.Setenv("XDG_DATA_HOME", "")

	tests := map[string]coreadapter.LoadingStrategy{
		"messenger": coreadapter.StrategySubprocess,
		"protocol":  coreadapter.StrategyBuiltin,
		"mobile":    coreadapter.StrategyBuiltin,
		"desktop":   coreadapter.StrategySubprocess,
		"sense":     coreadapter.StrategyScript,
		"actuator":  coreadapter.StrategyScript,
	}

	for typ, wantStrategy := range tests {
		t.Run(typ, func(t *testing.T) {
			name := "create-" + typ
			if err := runCreate(name, typ, "", "", true); err != nil {
				t.Fatalf("runCreate: %v", err)
			}

			dev, err := coreadapter.LoadAdapter(name)
			if err != nil {
				t.Fatalf("LoadAdapter: %v", err)
			}
			if dev.Type != coreadapter.AdapterType(typ) {
				t.Fatalf("type = %s, want %s", dev.Type, typ)
			}
			if dev.Strategy != wantStrategy {
				t.Fatalf("strategy = %s, want %s", dev.Strategy, wantStrategy)
			}
		})
	}
}

func TestRunCreateRejectsInvalidStrategyBeforeCreatingAdapter(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	t.Setenv("XDG_DATA_HOME", "")

	err := runCreate("bad-create", "desktop", "daemon", "", true)
	if err == nil {
		t.Fatal("runCreate returned nil error")
	}
	if !strings.Contains(err.Error(), "invalid loading strategy") {
		t.Fatalf("runCreate error = %v, want invalid loading strategy", err)
	}
	if _, loadErr := coreadapter.LoadAdapter("bad-create"); !coreadapter.IsAdapterNotFound(loadErr) {
		t.Fatalf("LoadAdapter after failed create = %v, want not found", loadErr)
	}
}

func TestRunCreateInvalidTypeDoesNotUseImplementationPlaceholder(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	t.Setenv("XDG_DATA_HOME", "")

	err := runCreate("bad-type", "wearable", "", "", true)
	if err == nil {
		t.Fatal("runCreate returned nil error")
	}
	if strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("runCreate error = %v, should be a validation error", err)
	}
	if _, loadErr := coreadapter.LoadAdapter("bad-type"); !coreadapter.IsAdapterNotFound(loadErr) {
		t.Fatalf("LoadAdapter after failed create = %v, want not found", loadErr)
	}
}
