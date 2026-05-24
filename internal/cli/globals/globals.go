package globals

// Accessors for tool-level globals other than --offline. Subpackages
// (internal/cli/a2a, internal/cli/skill, …) use these to read the same
// viper keys root.go declares via kit/cli Config.Globals + kit/output's
// auto-registered --format flag, without forming an import cycle by
// pulling internal/cli.

// Profile returns the value of the --profile global (kit/cli registers
// it via Config.Globals in root.go). Empty string when unset.
func Profile() string {
	if v == nil {
		return ""
	}
	return v.GetString("profile")
}

// Format returns the value of the --format global (kit/output auto-
// registers it via cli.New). Empty string when unset; callers passing
// it to listing.RenderList get the table default.
func Format() string {
	if v == nil {
		return ""
	}
	return v.GetString("format")
}

// Workspace returns the value of the --workspace global (kit/cli
// registers it via Config.Globals in root.go). Empty string when unset.
//
// T-0648 — added so adapter/multidevice leaves can stop redeclaring a
// local --workspace flag that shadows the global.
func Workspace() string {
	if v == nil {
		return ""
	}
	return v.GetString("workspace")
}

// Quiet returns the value of the --quiet global (kit/cli auto-registers
// it via cli.New). False when unset.
//
// T-0648 — added so adapter leaves can stop redeclaring a local --quiet
// flag that shadows the kit global.
func Quiet() bool {
	if v == nil {
		return false
	}
	return v.GetBool("quiet")
}

// Verbose returns the value of the --verbose global (kit/cli auto-
// registers it via cli.New). False when unset.
//
// T-0648 — added so adapter leaves can stop redeclaring a local
// --verbose flag that shadows the kit global.
func Verbose() bool {
	if v == nil {
		return false
	}
	return v.GetBool("verbose")
}

// DryRun returns the value of the --dry-run global (kit/cli auto-
// registers it via cli.New per ADR-0020). False when unset.
//
// T-0648 — added so adapter leaves can stop redeclaring a local
// --dry-run flag that shadows the kit global.
func DryRun() bool {
	if v == nil {
		return false
	}
	return v.GetBool("dry-run")
}
