package service

import (
	"sync/atomic"

	"hop.top/aps/internal/core"
)

// NativeTransportNotice is the deprecation notice for a service on an
// aps-native SMS or WhatsApp transport. It is registered as a next-step hint
// (internal/cli/hints.go), so kit renders it only in table format on a
// terminal and honors --no-hints, --quiet and HOP_QUIET_HINTS.
const NativeTransportNotice = "Deprecated: this aps-native SMS/WhatsApp transport has a uma-hosted " +
	"replacement (--type uma, --uma-adapter twilio.sms, twilio.whatsapp or meta.whatsapp). " +
	"Steps and rollback: docs/user/uma-migration.md"

// nativeTransportNoted records that this invocation handled a service the
// notice applies to. The hint's condition reads it after the command ran.
var nativeTransportNoted atomic.Bool

// NativeTransportNoted reports whether the command that just ran handled a
// native SMS or WhatsApp service.
func NativeTransportNoted() bool { return nativeTransportNoted.Load() }

// ResetNativeTransportNotice clears the record; tests use it between runs.
func ResetNativeTransportNotice() { nativeTransportNoted.Store(false) }

// noteNativeTransport records service when the notice applies to it.
func noteNativeTransport(service *core.ServiceConfig) {
	if _, ok := core.NativeTransportUmaAdapter(service); ok {
		nativeTransportNoted.Store(true)
	}
}
