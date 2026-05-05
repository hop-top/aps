package squad

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/clinote"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a squad",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0])
		},
	}
	clinote.AddFlag(cmd) // T-1291
	return cmd
}

func runDelete(id string) error {
	if err := defaultManager.Delete(id); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Deleted squad %q\n", id)
	return nil
}
