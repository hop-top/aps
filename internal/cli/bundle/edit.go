package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
	corebundle "hop.top/aps/internal/core/bundle"

	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name>",
		Short: "Open a bundle in $EDITOR; copies built-in to user dir first",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEdit(args[0])
		},
	}
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
