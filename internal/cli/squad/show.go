package squad

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show squad details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(args[0])
		},
	}
}

func runShow(id string) error {
	s, err := defaultManager.Get(id)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Squad: %s\n", s.Name)
	fmt.Fprintf(os.Stdout, "  ID:          %s\n", s.ID)
	fmt.Fprintf(os.Stdout, "  Type:        %s\n", s.Type)
	fmt.Fprintf(os.Stdout, "  Domain:      %s\n", s.Domain)
	if s.Description != "" {
		fmt.Fprintf(os.Stdout, "  Description: %s\n", s.Description)
	}
	fmt.Fprintf(os.Stdout, "  Members:     %s\n", formatMembers(s.Members))
	fmt.Fprintf(os.Stdout, "  Created:     %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(os.Stdout, "  Updated:     %s\n", s.UpdatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func formatMembers(members []string) string {
	if len(members) == 0 {
		return "(none)"
	}
	return strings.Join(members, ", ")
}
