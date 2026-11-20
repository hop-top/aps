//go:build windows

package testguard

// openHandles is a no-op on Windows: RemoveAll itself fails on open
// files there, which Run reports.
func openHandles(string) []string { return nil }
