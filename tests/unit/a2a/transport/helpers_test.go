package a2a_transport

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// freeAddr returns a loopback address bound to a kernel-assigned port.
// The listener is closed before returning, so callers can pass the
// address to whatever they're starting without contending with the
// helper. The microsecond-scale TOCTOU window between Close and the
// caller's bind is an accepted trade-off: it eliminates 100% of the
// hardcoded-port flake class (the prior cross-test :8081 collision),
// at the cost of a vanishingly small theoretical re-bind race.
//
// Duplicated verbatim in tests/unit/a2a/helpers_test.go to avoid an
// import edge from tests/unit/a2a/transport -> tests/e2e. If a third
// package needs this helper, promote both copies to a shared
// tests/internal/freeport package and import from there.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}
