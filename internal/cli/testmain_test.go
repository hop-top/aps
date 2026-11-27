package cli

import (
	"os"
	"testing"

	"hop.top/aps/internal/core/session"
	"hop.top/aps/internal/testguard"
)

func TestMain(m *testing.M) {
	// Re-exec'd as the CLI under test (see TestExecuteHelperProcess): the
	// parent test already chose the child's HOME and XDG dirs, so the
	// sandbox must not replace them.
	if os.Getenv(execHelperEnv) != "" {
		os.Exit(m.Run())
	}
	testguard.OnExit(session.CloseRegistry)
	os.Exit(testguard.Run(m))
}
