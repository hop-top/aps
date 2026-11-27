package ticket

import (
	"os"
	"testing"

	"hop.top/aps/internal/core/session"
	"hop.top/aps/internal/testguard"
)

func TestMain(m *testing.M) {
	testguard.OnExit(session.CloseRegistry)
	os.Exit(testguard.Run(m))
}
