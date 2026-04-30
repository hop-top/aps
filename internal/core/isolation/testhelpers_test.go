package isolation

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestProfile writes a minimal valid profile.yaml + empty
// secrets.env into dir and returns the YAML path. Use this when
// constructing ExecutionContext inline in a test so the test is
// self-contained — no reliance on $HOME or any global APS data
// path. Tests that pass dir=t.TempDir() get automatic cleanup
// via the testing framework.
func setupTestProfile(t *testing.T, dir string) string {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setupTestProfile: mkdir %s: %v", dir, err)
	}

	yaml := []byte("id: test-profile\ndisplay_name: Test Profile\nisolation:\n  level: process\n")
	yamlPath := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(yamlPath, yaml, 0o644); err != nil {
		t.Fatalf("setupTestProfile: write %s: %v", yamlPath, err)
	}

	secretsPath := filepath.Join(dir, "secrets.env")
	if err := os.WriteFile(secretsPath, []byte(""), 0o600); err != nil {
		t.Fatalf("setupTestProfile: write %s: %v", secretsPath, err)
	}

	return yamlPath
}
