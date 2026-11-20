package a2a

import (
	"os"
	"testing"

	"hop.top/aps/internal/testguard"
)

func TestMain(m *testing.M) { os.Exit(testguard.Run(m)) }
