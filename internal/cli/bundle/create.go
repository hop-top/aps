package bundle

import (
	"fmt"
	"os"
	"path/filepath"

	"hop.top/aps/internal/cli/clinote"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

const scaffoldTemplate = `name: %s
description: ""
version: "1.0"
capabilities: []
`

func newCreateCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Scaffold a new bundle file in the user bundle directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args[0], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false,
		"Overwrite existing bundle file")
	clinote.AddFlag(cmd) // T-1291

	// `create` mints new state; with --force it overwrites in-place
	// (idempotent on the same input). Mark Conditional to capture both.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	// T-0656 — create writes a fresh bundle file at the canonical user
	// path; preview would only print the destination path that's a
	// pure function of the name argument.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "create writes a new bundle file to a canonical path derived from the name argument; previewing would print a destination already implied by the input."); err != nil {
		panic(err)
	}
	return cmd
}

func runCreate(name string, force bool) error {
	dest, err := userBundlePath(name)
	if err != nil {
		return err
	}

	if !force {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("bundle file already exists: %s (use --force to overwrite)", dest)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("failed to create bundle directory: %w", err)
	}

	content := fmt.Sprintf(scaffoldTemplate, name)
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write bundle file: %w", err)
	}

	fmt.Printf("%s %s\n", successStyle.Render("Created"), dest)
	return nil
}

// userBundlePath returns the canonical path for a user bundle by name.
func userBundlePath(name string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user config dir: %w", err)
	}
	return filepath.Join(configDir, "aps", "bundles", name+".yaml"), nil
}
