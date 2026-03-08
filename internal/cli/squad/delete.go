package squad

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a squad",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0])
		},
	}
}

func runDelete(id string) error {
	if err := defaultManager.Delete(id); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Deleted squad %q\n", id)
	return nil
}
