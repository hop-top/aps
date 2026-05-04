package cli

// commandGroups maps each top-level subcommand name to its help-output
// group ID. Subcommands not listed fall through to the default
// "COMMANDS" group. Edit here to move a command between groups; no
// need to touch individual <name>.go files.
//
// Group taxonomy follows ~/.ops/docs/cli-conventions-with-kit.md §4.1.
// See T-0366/T-0367.
var commandGroups = map[string]string{
	// INTERACT — long-running surfaces.
	"run":     "interact",
	"serve":   "interact",
	"voice":   "interact",
	"session": "interact",
	"listen":  "interact",

	// ORGANIZE — taxonomy + scoping (the nouns aps owns).
	"profile":    "organize",
	"capability": "organize",
	"bundle":     "organize",
	"squad":      "organize",
	"workspace":  "organize",
	"contact":    "organize",

	// PIPELINES — integration / dispatch machinery.
	"a2a":       "pipelines",
	"acp":       "pipelines",
	"adapter":   "pipelines",
	"webhook":   "pipelines",
	"directory": "pipelines",
	"action":    "pipelines",
	"skill":     "pipelines",

	// SECURITY — trust + access.
	"identity": "security",
	"policy":   "security",

	// INSTANCE — per-backend ops.
	"observability": "instance",

	// MANAGEMENT — kit auto-registers this group (Hidden=true).
	// Members hidden from default --help; shown via --help-management
	// or --help-all.
	"alias":    "management",
	"docs":     "management",
	"env":      "management",
	"migrate":  "management",
	"upgrade":  "management",
	"toolspec": "management",
	"version":  "management",
	// "completion" is added to the management group by kit's
	// applyGroupVisibility; do not register it here.
}

// applyCommandGroups assigns GroupID to every top-level subcommand
// based on the commandGroups map. Run after every init() has called
// rootCmd.AddCommand and before fang renders help.
func applyCommandGroups() {
	for _, c := range rootCmd.Commands() {
		if g, ok := commandGroups[c.Name()]; ok {
			c.GroupID = g
		}
	}
}
