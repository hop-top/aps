// Package globals exposes tool-level CLI globals (declared on root via
// kit/cli Config.Globals) to subpackages without forcing them to import
// the root cli package — that import would form a cycle.
//
// Subpackages access globals through accessors here. The root cli
// package wires the underlying *viper.Viper at init time via SetViper.
//
// T-0411 — sweeps --offline across network-touching commands. The first
// global wired this way is offline; --instance and others can follow
// the same pattern as needs arise.
package globals

import (
	"errors"

	"github.com/spf13/viper"
)

// ErrOffline is the sentinel returned by network-touching commands when
// --offline is set. Wrap with fmt.Errorf("...: %w", globals.ErrOffline)
// so callers can detect via errors.Is.
//
// This is not in kit/runtime/domain because it's CLI-flag-specific
// rather than a generic domain failure mode.
var ErrOffline = errors.New("offline mode: network calls disabled")

// v is the viper instance backing the tool-level globals. Set once by
// internal/cli at init time. Nil-safe: accessors return zero values
// (e.g. IsOffline() == false) until SetViper is called, which keeps
// unit tests that don't boot the full CLI safe.
var v *viper.Viper

// SetViper wires the viper instance backing the tool-level globals.
// Called from internal/cli root init alongside logging.SetViper.
func SetViper(vp *viper.Viper) {
	v = vp
}

// IsOffline reports whether --offline (root.Viper key "offline") is set.
//
// Subcommands that hit the network should call this at the top of their
// RunE and return ErrOffline when true:
//
//	if globals.IsOffline() {
//	    return fmt.Errorf("a2a send: %w", globals.ErrOffline)
//	}
func IsOffline() bool {
	if v == nil {
		return false
	}
	return v.GetBool("offline")
}
