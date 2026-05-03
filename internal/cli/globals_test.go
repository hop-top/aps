package cli

// T-0376 — declare aps tool-level globals via cli.Config.Globals.
// --config, --profile, --workspace become persistent flags on root,
// bound to viper at the global key. Subcommand-local duplicates that
// exclusively duplicated these globals are removed.

import "testing"

func TestRoot_GlobalConfigFlag(t *testing.T) {
	if f := rootCmd.PersistentFlags().Lookup("config"); f == nil {
		t.Fatal("--config persistent flag not registered on root (T-0376)")
	}
	if got := root.Viper.GetString("config"); got != "" {
		t.Errorf(`viper.GetString("config") = %q, want ""`, got)
	}
}

func TestRoot_GlobalProfileFlag(t *testing.T) {
	if f := rootCmd.PersistentFlags().Lookup("profile"); f == nil {
		t.Fatal("--profile persistent flag not registered on root (T-0376)")
	}
	if got := root.Viper.GetString("profile"); got != "" {
		t.Errorf(`viper.GetString("profile") = %q, want ""`, got)
	}
}

func TestRoot_GlobalWorkspaceFlag(t *testing.T) {
	if f := rootCmd.PersistentFlags().Lookup("workspace"); f == nil {
		t.Fatal("--workspace persistent flag not registered on root (T-0376)")
	}
	if got := root.Viper.GetString("workspace"); got != "" {
		t.Errorf(`viper.GetString("workspace") = %q, want ""`, got)
	}
}

// T-0386 — --offline and --instance globals mirror tlc's parity addition.
func TestRoot_GlobalOfflineFlag(t *testing.T) {
	if f := rootCmd.PersistentFlags().Lookup("offline"); f == nil {
		t.Fatal("--offline persistent flag not registered on root (T-0386)")
	}
	if got := root.Viper.GetBool("offline"); got {
		t.Errorf(`viper.GetBool("offline") = %v, want false`, got)
	}
}

func TestRoot_GlobalInstanceFlag(t *testing.T) {
	if f := rootCmd.PersistentFlags().Lookup("instance"); f == nil {
		t.Fatal("--instance persistent flag not registered on root (T-0386)")
	}
	if got := root.Viper.GetString("instance"); got != "" {
		t.Errorf(`viper.GetString("instance") = %q, want ""`, got)
	}
}

// TestSessionList_NoLocalProfileFlag asserts the per-command --profile flag
// on `aps session list` is removed in favor of the global persistent flag.
func TestSessionList_NoLocalProfileFlag(t *testing.T) {
	cmd := findSubcommand(rootCmd, "session", "list")
	if cmd == nil {
		t.Fatal("aps session list not registered")
	}
	if f := cmd.LocalFlags().Lookup("profile"); f != nil {
		t.Errorf("aps session list still owns local --profile; expected use of global (T-0376)")
	}
}

// TestSessionList_NoLocalWorkspaceFlag asserts the per-command --workspace
// flag on `aps session list` is removed in favor of the global.
func TestSessionList_NoLocalWorkspaceFlag(t *testing.T) {
	cmd := findSubcommand(rootCmd, "session", "list")
	if cmd == nil {
		t.Fatal("aps session list not registered")
	}
	if f := cmd.LocalFlags().Lookup("workspace"); f != nil {
		t.Errorf("aps session list still owns local --workspace; expected use of global (T-0376)")
	}
}
