package cli

import (
	"log/slog"
	"testing"

	charmlog "charm.land/log/v2"
)

// TestRoot_FormatFlag verifies kit's output.RegisterFlags wired --format
// onto the root command via cli.New (Disable.Format = false default).
func TestRoot_FormatFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("format")
	if f == nil {
		t.Fatal("--format persistent flag not registered on root")
	}
	if f.DefValue != "table" {
		t.Errorf("--format default = %q, want %q", f.DefValue, "table")
	}
}

// TestRoot_NoHintsFlag verifies kit's output.RegisterHintFlags wired
// --no-hints onto the root command (Disable.Hints = false default).
func TestRoot_NoHintsFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("no-hints")
	if f == nil {
		t.Fatal("--no-hints persistent flag not registered on root")
	}
	if f.DefValue != "false" {
		t.Errorf("--no-hints default = %q, want %q", f.DefValue, "false")
	}
}

// TestRoot_FormatBoundToViper verifies the --format flag is bound to
// the root viper key "format" so subcommands can read it.
func TestRoot_FormatBoundToViper(t *testing.T) {
	if got := root.Viper.GetString("format"); got != "table" {
		t.Errorf(`viper.GetString("format") = %q, want %q`, got, "table")
	}
}

// TestRoot_SlogDefaultIsKitHandler verifies that the package init()
// in root.go installed kit's charm.land/log/v2 logger as the process
// slog.Default (T-0647). After init, any stdlib slog.Info/Warn/Error
// call in aps or its deps must route through kit's handler so it
// honours --quiet/--no-color and aps's redaction wrapper.
func TestRoot_SlogDefaultIsKitHandler(t *testing.T) {
	h := slog.Default().Handler()
	if h == nil {
		t.Fatal("slog.Default().Handler() = nil; expected kit handler")
	}
	if _, ok := h.(*charmlog.Logger); !ok {
		t.Errorf("slog.Default().Handler() type = %T, want *charm.land/log/v2.Logger", h)
	}
}
