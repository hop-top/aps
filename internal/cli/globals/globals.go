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
