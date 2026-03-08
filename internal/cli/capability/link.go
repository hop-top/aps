package capability

import (
	"fmt"

	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "link <name> [--target <path>]",
		Short: "Symlink a capability to a target path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if target == "" {
				if pattern, err := capability.GetSmartPattern(name); err == nil {
					target = pattern.ToolName
					fmt.Println(dimStyle.Render(fmt.Sprintf(
						"Smart link: %s -> %s", name, pattern.DefaultPath)))
				} else {
					return fmt.Errorf(
						"--target required unless using a Smart Pattern name")
				}
			}

			if err := capability.Link(name, target); err != nil {
				return err
			}
			fmt.Println(successStyle.Render("Linked") + " " +
				boldStyle.Render(name) + " -> " + target)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Target path for symlink")

	return cmd
}
