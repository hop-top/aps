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
	"github.com/spf13/pflag"
	kitcli "hop.top/kit/go/console/cli"
	kitconfigoverrides "hop.top/kit/go/core/config"
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
// In addition to inverting --no-redact into the redact.enabled viper
// key (T-0460), it bootstraps the kit/runtime/policy engine on the
// process bus (T-1292). Misconfig (bad YAML, unknown topic, broken
// CEL) fails loud here so the user's command never runs against an
// unenforced ruleset. KIT_POLICY_DISABLE=1 short-circuits the bootstrap
// for tests / dev mode.
//
// Defined as a top-level function (not a closure inside Config) to
// avoid the initialization cycle: `root` is the value being declared
// and a closure that captured `root.Viper` would force the compiler
// to evaluate `root` while still building it. This function reads
// the parsed flag pointer (noRedactFlag) and the cmd-tree's root
// flags directly, neither of which is on the cycle.
func applyNoRedactToggle(cmd *cobra.Command, _ []string) error {
	logging.SetRedactEnabled(!noRedactFlag)
	// T-0583 — install parsed -c/--config tokens before any subcommand
	// (or kit-installed hook downstream) calls core.LoadConfig. kit/cli
	// has already parsed the flag by the time PrePersistentRunE fires.
	// Read directly from the cobra flag rather than `root.ConfigArgs()`
	// to avoid an init cycle: `var root = kitcli.New(...)` declares the
	// hook by reference, so the function body cannot transitively read
	// `root` without forcing Go's initializer cycle detector to fire.
	if f := cmd.Root().PersistentFlags().Lookup("config"); f != nil {
		if sa, ok := f.Value.(pflag.SliceValue); ok {
			paths, overrides, _ := kitconfigoverrides.ParseConfigArgs(sa.GetSlice())
			core.SetConfigArgs(paths, overrides)
		}
	}
	if _, err := initPolicyEngine(eventBus); err != nil {
		return err
	}
	return nil
}

// profileFlagName is the canonical name of the --profile root global
// flag. Centralised so subcommands, group-mapping tables, and the
// flag's own registration all reference the same identifier.
const profileFlagName = "profile"

var root = kitcli.New(kitcli.Config{
	Name:    "aps",
	Version: version.Short(),
	Short:   "Agent Profile System CLI",
	// kit 0.4 defaults EnforceValidate=true, which rejects all 169 aps
	// leaves lacking kit/side-effect annotations. Annotation rollout is
	// scoped to the aps-kit-12fcc-conformance track (T-0647..T-0662);
	// flip this back to the default once that track lands.
	DisableValidate: true,
	// T-0654 — refuse destructive leaves that have not opted into the
	// typed-token confirm flow (kit/destructive-token=required, set via
	// kitcli.SetDestructiveToken). DisableValidate above short-circuits
	// the runtime pre-flight; this flag still feeds the validator gate
	// the TestRootValidate_StrictGatesPass regression net exercises, and
	// arms the policy-gate confirm flow on every destructive leaf so the
	// CLI refuses to run them on non-TTY without --confirm-token=<sha>.
	EnforceDestructiveToken: true,
	// T-0655 — refuse runnable leaves that lack kit/examples, and non-read
	// leaves that lack kit/next-steps. Every aps leaf is annotated via
	// zz_guidance_annotations.go's init-time pass (174 leaves, 97 of
	// which carry next-steps). Same pre-flight semantics as
	// EnforceDestructiveToken: gated by DisableValidate today, armed for
	// when T-0657 flips DisableValidate back to false. See kit cli.go:218
	// for the field definition and cli.go:1080 for the runtime gate.
	EnforceGuidance: true,
	// T-0656 — refuse write/destructive leaves that called
	// kitcli.OptOutDryRun(cmd) without pairing a kit/dry-run-rationale
	// annotation. Each opt-out in the aps tree carries an honest
	// reason explaining why preview would be uninformative or
	// impossible; the rationale is what the user reads when --dry-run
	// is rejected on the leaf. Leaves that genuinely honor --dry-run
	// (action run, adapter link/revoke/stop/unlink, migrate messengers,
	// workspace conflicts resolve) need neither the opt-out nor the
	// rationale. See kit cli.go:210 for the field, contract.go:118 for
	// SetDryRunRationale, and the EnforceDryRunRationale gate at
	// cli.go:1064.
	EnforceDryRunRationale: true,
	// T-0657 — paired with DisableValidate above. While DisableValidate
	// is true (the production guardrail during the 12fcc annotation
	// rollout) kit/cli short-circuits the pre-flight validator and this
	// mode is unreachable in production. We set it ahead of time so that
	// when T-0648/T-0653 progressively tighten strictness and eventually
	// flip DisableValidate back to false, validation failures bubble out
	// of Execute() as typed *kitcli.ValidationError values that tests
	// can errors.As against — instead of kit's default behavior of
	// writing to stderr and calling os.Exit(2). See kit cli.go:105 for
	// the constant definition.
	ValidationFailureMode: kitcli.ValidationFailureError,
	// T-0653 — flip the signature validator from silent (zero value) to
	// reject. The four signature checks (kit/signature/reserved-name,
	// kit/signature/passthrough, kit/signature/local-globals,
	// kit/signature/depth-hierarchical) are all at 0 across the aps
	// tree after T-0648's umbrella conformance pass, so reject is now
	// the documented production target per the aps-kit-12fcc-conformance
	// plan. Note this is INDEPENDENT of DisableValidate above: kit gates
	// SignatureStrictness on its own conditional (see kit cli.go:784),
	// separate from the EnforceValidate / Layer-A annotation pre-flight
	// that DisableValidate=true still short-circuits. The Layer-A flip
	// lands in T-0657 once the kit/side-effect + kit/idempotent
	// annotations are guaranteed across the tree. See kit cli.go:144
	// for the SignatureStrictnessReject constant definition.
	SignatureStrictness: kitcli.SignatureStrictnessReject,
	// T-0680 — kit defaults MaxHierarchyDepth=3 (see kit shape.go:14),
	// which rejects the `aps adapter messenger link {add,delete,list}`
	// subtree (depth-4 leaves: aps[0] → adapter[1] → messenger[2] →
	// link[3] → add|delete|list[4]). Raise the cap to 4 to admit the
	// existing link subtree; kit hard-clamps at 5 (shape.go:18), so 4
	// is within bounds. See kit cli.go:234 for the field and
	// cli.go:950-968 for the validator gate.
	MaxHierarchyDepth: 4,
	// T-0376 — declare tool-level globals: --config, --profile, --workspace.
	// Subcommands read via root.Viper.GetString("<key>") rather than
	// declaring local duplicates.
	Globals: []kitcli.Flag{
		// NOTE: -c/--config is auto-registered by kit/cli (StringArrayP,
		// repeatable, supports both bare paths and key=value overrides;
		// see kit cli.go §Disable.Config). aps consumes the parsed tokens
		// in applyNoRedactToggle via root.ConfigArgs() and threads them
		// into core.LoadConfig (T-0583). Do not redeclare here — pflag
		// panics on duplicate flag names.
		// T-0648 batch 8 — `-p` shorthand promoted to the global so
		// subcommands that previously declared local `--profile,-p` flags
		// can drop the duplicate without losing the `-p` UX. Subcommands
		// migrate by removing the local registration and reading via
		// root.Viper.GetString("profile") or the inherited persistent
		// flag set.
		{Name: profileFlagName, Short: "p", Usage: "profile id (defaults to active profile)"},
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
	//
	// closePolicy unsubscribes the kit/runtime/policy engine from the
	// bus before drainBus closes it, so the unsubscribe is a no-op on
	// a still-open bus (matches tlc T-1192 ordering).
	defer drainBus()
	defer closePolicy()
	err := root.Execute(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, styles.Error.Render("Error: "+err.Error()))
	}
	return err
}
