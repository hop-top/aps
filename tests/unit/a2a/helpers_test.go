package a2a_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// freeAddr returns a loopback address bound to a kernel-assigned port.
// The listener is closed before returning, so callers can pass the
// address to whatever they're starting without contending with the
// helper. There is a microsecond-scale TOCTOU window between Close and
// the caller's bind; acceptable for test isolation.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}
