package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"hop.top/aps/internal/cli/clinote"
	corebundle "hop.top/aps/internal/core/bundle"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Open a bundle in $EDITOR; copies built-in to user dir first",
		Long: `Open the named bundle in $EDITOR (or vi when $EDITOR is unset).
If no user override exists at aps/bundles/<name>.yaml under the
user config directory, the built-in bundle of the same name is
copied there first so the edit lands in the user-owned override
rather than the kit-shipped source. A "Note: built-in bundle …
copied to …" line surfaces when that copy happens.

The command shells out to the editor and waits for it to exit;
whether the file changed is up to the user, so the effect is
idempotency-conditional. --dry-run is opted out because the
operation is fundamentally an interactive editor session — preview
would either suppress the editor or describe a write the user
hasn't authored yet.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEdit(args[0])
		},
	}
	clinote.AddFlag(cmd) // T-1291
	// Editor outcome is user-driven; idempotency depends on whether the
	// user edits anything. Conditional captures this best.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	// T-0656 — edit shells out to $EDITOR for a user-driven session;
	// preview would have to either suppress the editor (defeating the
	// command) or describe a write that hasn't been authored yet.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "edit hands control to $EDITOR for a user-driven session; previewing would have to either suppress the editor (defeating the command) or describe a write that has not been authored yet."); err != nil {
		panic(err)
	}
	return cmd
}

func runEdit(name string) error {
	dest, err := userBundlePath(name)
	if err != nil {
		return err
	}

	// If user override does not exist yet, check built-ins and copy.
	if _, statErr := os.Stat(dest); os.IsNotExist(statErr) {
		if err := copyBuiltinToUser(name, dest); err != nil {
			return err
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	//nolint:gosec // user-controlled editor is expected
	if err := exec.Command(editor, dest).Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	return nil
}

// copyBuiltinToUser loads the named bundle from the registry (must be a built-in)
// and writes it to dest in the user bundle directory.
func copyBuiltinToUser(name, dest string) error {
	builtins, err := corebundle.LoadBuiltins()
	if err != nil {
		return fmt.Errorf("failed to load built-in bundles: %w", err)
	}

	var found *corebundle.Bundle
	for i := range builtins {
		if builtins[i].Name == name {
			found = &builtins[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("bundle %q not found (no built-in and no user override)", name)
	}

	data, err := yaml.Marshal(found)
	if err != nil {
		return fmt.Errorf("failed to marshal bundle: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("failed to create bundle directory: %w", err)
	}

	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("failed to write user bundle file: %w", err)
	}

	fmt.Printf("%s built-in bundle %q copied to %s\n",
		dimStyle.Render("Note:"), name, dest)
	return nil
}
