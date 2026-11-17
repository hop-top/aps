package core

import (
	"fmt"
	"os"
	"testing"
)

// TestMain points the user config dir at an empty temp dir for the whole
// package, so tests that call LoadConfig without their own override never
// read the developer's real config (a stray prefix there would rewrite
// every injected APS_* variable).
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "aps-core-test-config-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "TestMain:", err)
		os.Exit(1)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
