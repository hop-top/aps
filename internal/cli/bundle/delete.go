package bundle

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/prompt"
	corebundle "hop.top/aps/internal/core/bundle"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a user-defined bundle (refuses on built-ins)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false,
		"Skip confirmation prompt")
	clinote.AddFlag(cmd) // T-1291

	return cmd
}

func runDelete(name string, force bool) error {
	path, err := userBundlePath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Not a user bundle — check if it's a built-in.
		if isBuiltin(name) {
			return fmt.Errorf(
				"cannot delete built-in bundle; use 'aps bundle edit' to create a user override")
		}
		return fmt.Errorf("bundle %q not found", name)
	}

	if !force {
		confirmed, err := prompt.Confirm(
			fmt.Sprintf("Delete user bundle %q at %s?", name, path))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(dimStyle.Render("Aborted."))
			return nil
		}
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete bundle file: %w", err)
	}

	fmt.Printf("%s bundle %q\n", successStyle.Render("Deleted"), name)
	return nil
}

func isBuiltin(name string) bool {
	builtins, err := corebundle.LoadBuiltins()
	if err != nil {
		return false
	}
	for _, b := range builtins {
		if b.Name == name {
			return true
		}
	}
	return false
}
