package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/version"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/core/upgrade"
	"hop.top/kit/go/core/upgrade/skill"
	"hop.top/kit/go/core/xdg"
)

const apsGitHubRepo = "hop-top/aps"

func newChecker() *upgrade.Checker {
	return upgrade.New(
		upgrade.WithBinary("aps", version.Short()),
		upgrade.WithGitHub(apsGitHubRepo),
	)
}

func newUpgradeCmd() *cobra.Command {
	var auto bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Check for and install updates",
		Long:  `Check for a newer version of aps and optionally install it.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// --quiet is provided by kit/go/console/cli as a persistent
			// root flag (T-0347 dropped the local -q --quiet shadow).
			quiet := root.Viper.GetBool("quiet")
			return upgrade.RunCLI(cmd.Context(), newChecker(), upgrade.CLIOptions{
				AutoUpgrade: auto,
				Quiet:       quiet,
			})
		},
	}

	cmd.Flags().BoolVar(&auto, "auto", false, "Install without prompting")
	cmd.AddCommand(newUpgradePreambleCmd())
	// T-0648 — `upgrade` writes the new binary into the install
	// location; repeated calls converge once at latest.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — upgrade downloads and replaces the running binary;
	// preview would have to discover the latest release version, which
	// is the same network call the real upgrade performs.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "upgrade downloads the latest release and atomically replaces the running binary; previewing would require the same release-discovery network call, with no useful local state to inspect."); err != nil {
		panic(err)
	}
	return cmd
}

func newUpgradePreambleCmd() *cobra.Command {
	var auto, never, install bool

	cmd := &cobra.Command{
		Use:   "preamble",
		Short: "Print the upgrade preamble fragment for skill files",
		Long: `Print a markdown preamble fragment for embedding in APS skill files.
Agents read this to know how to self-upgrade aps before executing tasks.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			level := skill.SnoozeOnce
			if auto {
				level = skill.SnoozeNever
			} else if never {
				level = skill.SnoozeAlways
			}

			preamble := skill.Generate(skill.PreambleOptions{
				BinaryName: "aps",
				Snooze:     level,
			})

			if install {
				return installAPSPreamble(preamble)
			}

			fmt.Print(preamble)
			return nil
		},
	}

	cmd.Flags().BoolVar(&auto, "auto", false, "Emit auto-upgrade (SnoozeNever) variant")
	cmd.Flags().BoolVar(&never, "never", false, "Emit check-only (SnoozeAlways) variant")
	cmd.Flags().BoolVar(&install, "install", false, "Write preamble to ~/.config/aps/skills/")
	// T-0648 — default path prints; --install writes the same content
	// to ~/.config/aps/skills/upgrade-preamble.md (idempotent overwrite).
	// Conservatively tag as WriteLocal to cover the --install path.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — preamble is a generator; without --install it already
	// prints to stdout (the canonical preview), and --install writes a
	// deterministic file at a fixed path.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "preamble without --install already prints the generated content to stdout (the canonical preview); --install writes the same bytes to a fixed path under ~/.config/aps/skills/."); err != nil {
		panic(err)
	}
	return cmd
}

func installAPSPreamble(preamble string) error {
	configDir, err := xdg.ConfigDir("aps")
	if err != nil {
		return fmt.Errorf("upgrade preamble: %w", err)
	}
	dir := filepath.Join(configDir, "skills")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("upgrade preamble: mkdir: %w", err)
	}
	path := filepath.Join(dir, "upgrade-preamble.md")
	if err := os.WriteFile(path, []byte(preamble), 0o600); err != nil {
		return fmt.Errorf("upgrade preamble: write: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Installed upgrade preamble → %s\n", path)
	return nil
}

func init() {
	rootCmd.AddCommand(newUpgradeCmd())
}
