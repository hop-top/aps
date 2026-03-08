package bundle

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	corebundle "hop.top/aps/internal/core/bundle"

	"github.com/spf13/cobra"
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
		fmt.Printf("Delete user bundle %q at %s? [y/N] ", name, path)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
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
