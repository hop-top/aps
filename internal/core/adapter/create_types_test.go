package adapter

import "testing"

func TestDefaultStrategyForImplementedAdapterTypes(t *testing.T) {
	tests := map[AdapterType]LoadingStrategy{
		AdapterTypeMessenger: StrategySubprocess,
		AdapterTypeProtocol:  StrategyBuiltin,
		AdapterTypeMobile:    StrategyBuiltin,
		AdapterTypeDesktop:   StrategySubprocess,
		AdapterTypeSense:     StrategyScript,
		AdapterTypeActuator:  StrategyScript,
	}

	for typ, want := range tests {
		t.Run(string(typ), func(t *testing.T) {
			if !IsAdapterTypeImplemented(typ) {
				t.Fatalf("%s should be creatable", typ)
			}
			if got := DefaultStrategyForType(typ); got != want {
				t.Fatalf("DefaultStrategyForType(%s) = %s, want %s", typ, got, want)
			}
		})
	}
}

func TestCreateAdapterSupportsEveryImplementedType(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	t.Setenv("XDG_DATA_HOME", "")

	mgr := NewManager()
	for _, typ := range []AdapterType{
		AdapterTypeMessenger,
		AdapterTypeProtocol,
		AdapterTypeMobile,
		AdapterTypeDesktop,
		AdapterTypeSense,
		AdapterTypeActuator,
	} {
		t.Run(string(typ), func(t *testing.T) {
			name := "test-" + string(typ)
			dev, err := mgr.CreateAdapter(name, typ, "", ScopeGlobal, "")
			if err != nil {
				t.Fatalf("CreateAdapter: %v", err)
			}
			if dev.Type != typ {
				t.Fatalf("created type = %s, want %s", dev.Type, typ)
			}
			if dev.Strategy != DefaultStrategyForType(typ) {
				t.Fatalf("strategy = %s, want %s", dev.Strategy, DefaultStrategyForType(typ))
			}

			loaded, err := LoadAdapter(name)
			if err != nil {
				t.Fatalf("LoadAdapter: %v", err)
			}
			if loaded.Type != typ {
				t.Fatalf("loaded type = %s, want %s", loaded.Type, typ)
			}
		})
	}
}

func TestCreateAdapterRejectsInvalidStrategyBeforeSave(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	t.Setenv("XDG_DATA_HOME", "")

	mgr := NewManager()
	_, err := mgr.CreateAdapter(
		"bad-strategy",
		AdapterTypeActuator,
		LoadingStrategy("daemon"),
		ScopeGlobal,
		"",
	)
	if err == nil {
		t.Fatal("CreateAdapter returned nil error")
	}
	if !IsStrategyInvalid(err) {
		t.Fatalf("CreateAdapter error = %v, want strategy invalid", err)
	}
	if _, loadErr := LoadAdapter("bad-strategy"); !IsAdapterNotFound(loadErr) {
		t.Fatalf("LoadAdapter after failed create = %v, want not found", loadErr)
	}
}
