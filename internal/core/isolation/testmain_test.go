package isolation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	// Create a temp HOME with a test-profile so that core.LoadProfile("test-profile")
	// succeeds during SetupEnvironment calls.
	tmpHome, err := os.MkdirTemp("", "isolation-test-home-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpHome)

	profileDir := filepath.Join(tmpHome, ".agents", "profiles", "test-profile")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		panic(err)
	}

	profileYAML := []byte("id: test-profile\ndisplay_name: Test Profile\nisolation:\n  level: process\n")
	if err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), profileYAML, 0644); err != nil {
		panic(err)
	}

	// Create an empty secrets.env so LoadSecrets doesn't fail
	if err := os.WriteFile(filepath.Join(profileDir, "secrets.env"), []byte(""), 0600); err != nil {
		panic(err)
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Exit(m.Run())
}
