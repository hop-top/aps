package globals_test

import (
	"testing"

	"github.com/spf13/viper"

	"hop.top/aps/internal/cli/globals"
)

// TestDryRun_NilSafe asserts the accessor returns false before
// SetViper has wired the underlying viper.
func TestDryRun_NilSafe(t *testing.T) {
	globals.SetViper(nil)
	if globals.DryRun() {
		t.Fatal("DryRun() = true with nil viper; want false")
	}
}

// TestDryRun_ReadsKitViperKey covers the wired path. The kit cli
// binds the root --dry-run persistent flag to viper key kit.dry_run
// (not the flag name), so the accessor must read that key or the
// flag is silently ignored and "previews" execute for real.
func TestDryRun_ReadsKitViperKey(t *testing.T) {
	v := viper.New()
	globals.SetViper(v)
	t.Cleanup(func() { globals.SetViper(nil) })

	if globals.DryRun() {
		t.Fatal("DryRun() = true with unset kit.dry_run; want false")
	}
	v.Set("kit.dry_run", true)
	if !globals.DryRun() {
		t.Fatal("DryRun() = false after viper.Set(kit.dry_run, true); want true")
	}
}
