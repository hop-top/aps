package adapter

import (
	"os"
	"testing"
)

// TestSaveLoadAdapter_LinkedToRoundtrip verifies that Adapter.LinkedTo
// is preserved across SaveAdapter → loadAdapterFromPath. Pre-fix this
// failed because AdapterManifest had no LinkedTo field, so SaveAdapter
// dropped the slice on write and loadAdapterFromPath returned an empty
// slice on read. Cross-process e2e (T-0163 → T-0181) surfaced this.
func TestSaveLoadAdapter_LinkedToRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	// Defensive: clear XDG_DATA_HOME so the override is unambiguous.
	t.Setenv("XDG_DATA_HOME", "")

	want := []string{"profile-a", "profile-b"}
	dev := &Adapter{
		Name:     "test-protocol",
		Type:     AdapterTypeProtocol,
		Scope:    ScopeGlobal,
		Strategy: StrategyBuiltin,
		LinkedTo: append([]string(nil), want...),
	}

	if err := SaveAdapter(dev); err != nil {
		t.Fatalf("SaveAdapter: %v", err)
	}

	// Sanity: manifest file must exist on disk.
	if _, err := os.Stat(dev.ManifestPath); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}

	loaded, err := LoadAdapter("test-protocol")
	if err != nil {
		t.Fatalf("LoadAdapter: %v", err)
	}

	if len(loaded.LinkedTo) != len(want) {
		t.Fatalf("LinkedTo len = %d, want %d (got=%v)", len(loaded.LinkedTo), len(want), loaded.LinkedTo)
	}
	for i, p := range want {
		if loaded.LinkedTo[i] != p {
			t.Errorf("LinkedTo[%d] = %q, want %q", i, loaded.LinkedTo[i], p)
		}
	}
	if !loaded.IsLinkedToProfile("profile-a") {
		t.Errorf("IsLinkedToProfile(profile-a) = false; want true")
	}
}

// TestSaveLoadAdapter_LinkedToEmptyOmitted verifies that an adapter
// with no profile links round-trips with an empty/nil LinkedTo and
// the YAML key is omitted (omitempty). Guards against accidentally
// writing `linked_to: []` everywhere.
func TestSaveLoadAdapter_LinkedToEmptyOmitted(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	t.Setenv("XDG_DATA_HOME", "")

	dev := &Adapter{
		Name:     "bare-protocol",
		Type:     AdapterTypeProtocol,
		Scope:    ScopeGlobal,
		Strategy: StrategyBuiltin,
	}
	if err := SaveAdapter(dev); err != nil {
		t.Fatalf("SaveAdapter: %v", err)
	}

	data, err := os.ReadFile(dev.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if got := string(data); contains(got, "linked_to") {
		t.Errorf("expected linked_to omitted; manifest:\n%s", got)
	}

	loaded, err := LoadAdapter("bare-protocol")
	if err != nil {
		t.Fatalf("LoadAdapter: %v", err)
	}
	if len(loaded.LinkedTo) != 0 {
		t.Errorf("LinkedTo = %v, want empty", loaded.LinkedTo)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
