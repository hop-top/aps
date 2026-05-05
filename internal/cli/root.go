package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/logging"
	"hop.top/aps/internal/styles"
	"hop.top/aps/internal/tui"
	"hop.top/aps/internal/version"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/core/upgrade"
)

// noRedactFlag holds the parsed --no-redact value. The flag is
// declared as a kitcli Global so it shows up in --help and binds to
// the root viper. applyNoRedactToggle (the kitcli PrePersistentRunE
// hook) inverts the bool into the redact.enabled viper key that
// internal/logging/redact.go reads.
var noRedactFlag bool

// applyNoRedactToggle is the kitcli PrePersistentRunE hook. It runs
// after kit's built-in chain (chdir → identity → peer init) and
// before the subcommand's RunE. Idempotent and safe to call many
// times.
//
// Defined as a top-level function (not a closure inside Config) to
// avoid the initialization cycle: `root` is the value being declared
// and a closure that captured `root.Viper` would force the compiler
// to evaluate `root` while still building it. This function reads
// the parsed flag pointer (noRedactFlag) and the cmd-tree's root
// flags directly, neither of which is on the cycle.
func applyNoRedactToggle(cmd *cobra.Command, _ []string) error {
	logging.SetRedactEnabled(!noRedactFlag)
	return nil
}

var root = kitcli.New(kitcli.Config{
	Name:    "aps",
	Version: version.Short(),
	Short:   "Agent Profile System CLI",
	// T-0376 — declare tool-level globals: --config, --profile, --workspace.
	// Subcommands read via root.Viper.GetString("<key>") rather than
	// declaring local duplicates.
	Globals: []kitcli.Flag{
		{Name: "config", Usage: "path to YAML config file"},
		{Name: "profile", Usage: "profile id (defaults to active profile)"},
		{Name: "workspace", Usage: "workspace id (defaults to active workspace)"},
		{Name: "offline", Usage: "disable all network calls"},
		{Name: "instance", Usage: "backend instance to target (defaults to config)"},
		// T-0460 — emergency bypass for kit/core/redact filtering.
		// Default false (redaction ON). See docs/cli/redaction.md.
		{
			Name:    "no-redact",
			Usage:   "disable redaction of secrets/PII in logs and output (DEBUG ONLY)",
			BoolVar: &noRedactFlag,
		},
	},
	// T-0460 — use kit's Hooks slot so the redact toggle is wired
	// into the kit-managed PersistentPreRunE chain (chdir → identity
	// → peer init → here). Setting cmd.PersistentPreRun directly is
	// silently superseded when kit installs its own PersistentPreRunE.
	Hooks: kitcli.Hooks{
		PrePersistentRunE: applyNoRedactToggle,
	},
	// T-0392 — resolve -C/--chdir targets that aren't literal dirs against
	// aps's workspace + profile registries before kit falls back to the
	// literal-path "not a directory" error.
	ChdirResolver: resolveAPSContext,
	// T-0366/T-0367 — command grouping per ~/.ops/docs/cli-conventions-with-kit.md §4.1.
	// MANAGEMENT is auto-registered by kit/cli (hidden by default; --help-management
	// or --help-all to view). Per-group help via --help-<id>.
	Help: kitcli.HelpConfig{
		Groups: []kitcli.GroupConfig{
			{ID: "interact", Title: "INTERACT"},
			{ID: "organize", Title: "ORGANIZE"},
			{ID: "pipelines", Title: "PIPELINES"},
			{ID: "security", Title: "SECURITY"},
			{ID: "instance", Title: "INSTANCE"},
		},
	},
})

// rootCmd is an alias so other files can call rootCmd.AddCommand() in init().
var rootCmd = root.Cmd

func init() {
	logging.SetViper(root.Viper)
	// T-0411 — wire tool-level globals so subpackages (a2a, directory,
	// adapter, …) can gate network paths on --offline without importing
	// internal/cli (which would form an import cycle).
	globals.SetViper(root.Viper)
	// T-0450 — install the kit-themed TableStyle so listing.RenderList
	// forwards it via output.WithTableStyle on TTY writers. Non-TTY paths
	// (pipes, files, test buffers) keep emitting plain tabwriter output;
	// the styled renderer is gated on writerIsTTY in kit/output.
	listing.SetTableStyle(root.TableStyle())

	// Note: kit/go/console/cli.New already calls output.RegisterFlags
	// and output.RegisterHintFlags by default (gated by Config.Disable.
	// Format and .Hints). This wires --format (table|json|yaml) and
	// --no-hints persistent flags on rootCmd, both bound to root.Viper.
	// Subcommands read with root.Viper.GetString("format") /
	// output.HintsEnabled(root.Viper). Tests in root_test.go assert this.

	rootCmd.Long = `Agent Profile System CLI

Run aps with no arguments to launch the interactive TUI.

Pass a profile ID to start a session for that profile, or pass a profile
ID followed by a command to run that command under the selected profile.`

	// ArbitraryArgs + profile dispatch
	rootCmd.Args = cobra.ArbitraryArgs
	rootCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		profiles, err := core.ListProfiles()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return profiles, cobra.ShellCompDirectiveNoFileComp
	}
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		// If no args, launch TUI
		if len(args) == 0 {
			tui.Run()
			return
		}

		// Check if first arg is a profile ID
		profileID := args[0]
		profile, err := core.LoadProfile(profileID)

		if err == nil {
			if len(args) == 1 {
				shell := profile.Preferences.Shell
				if shell == "" {
					shell = core.DetectShell()
				}
				fmt.Printf("Starting session for %s using %s...\n", profileID, shell)
				if err := core.RunCommand(profileID, shell, nil); err != nil {
					logging.GetLogger().Error("session ended with error", err)
					os.Exit(1)
				}
				return
			}

			commandName := args[1]
			commandArgs := args[2:]
			if err := core.RunCommand(profileID, commandName, commandArgs); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				logging.GetLogger().Error("command failed", err)
				os.Exit(1)
			}
			return
		}

		logging.GetLogger().Error("unknown command or profile",
			fmt.Errorf("%q", profileID))
		if err := cmd.Help(); err != nil {
			logging.GetLogger().Error("error rendering help", err)
		}
		os.Exit(1)
	}

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		if cmd.Name() == "upgrade" {
			return
		}
		upgrade.NotifyIfAvailable(cmd.Context(), newChecker(), os.Stderr)
	}

	// Register contextual post-command hints (T-0346).
	registerHints(root.Hints)

	// Render hints after command output. Hints auto-suppress on non-TTY,
	// json/yaml formats, and when --no-hints is set (kit handles this
	// inside output.RenderHints).
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		renderPostRunHintsFor(cmd, root)
		return nil
	}
}

// Execute runs the CLI through fang (styled help, version, etc.)
func Execute() error {
	rootCmd.SilenceErrors = true
	// Assign group IDs to every top-level subcommand before kit
	// renders help (root.Execute calls applyGroupVisibility internally
	// after our hook has run).
	applyCommandGroups()
	// Drain in-flight bus events before returning so short-lived CLI
	// invocations don't exit before async network forwarders flush
	// their writes to the hub. drainBus is a no-op when the bus is
	// disabled or no publishes occurred. See internal/cli/bus.go and
	// T-0176. Deferred so it runs on RunE error paths too.
	defer drainBus()
	err := root.Execute(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, styles.Error.Render("Error: "+err.Error()))
	}
	return err
}
