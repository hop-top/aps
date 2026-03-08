package squad

import (
	"fmt"
	"os"
	"strings"

	coresquad "oss-aps-cli/internal/core/squad"

	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var squadType string
	var domain string
	var description string
	var members []string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new squad",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args[0], squadType, domain, description, members)
		},
	}

	cmd.Flags().StringVar(&squadType, "type", "", "Squad type (stream-aligned, enabling, complicated-subsystem, platform)")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain boundary")
	cmd.Flags().StringVar(&description, "description", "", "Squad description")
	cmd.Flags().StringSliceVar(&members, "members", nil, "Comma-separated profile IDs")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("domain")

	return cmd
}

func runCreate(name, squadType, domain, description string, members []string) error {
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	st := coresquad.SquadType(squadType)
	if err := st.Validate(); err != nil {
		return err
	}

	s := coresquad.Squad{
		ID:          id,
		Name:        name,
		Type:        st,
		Domain:      domain,
		Description: description,
		Members:     members,
	}

	if err := defaultManager.Create(s); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Created squad %q (%s)\n", name, id)
	fmt.Fprintf(os.Stdout, "  Type:    %s\n", squadType)
	fmt.Fprintf(os.Stdout, "  Domain:  %s\n", domain)
	if len(members) > 0 {
		fmt.Fprintf(os.Stdout, "  Members: %s\n", strings.Join(members, ", "))
	}

	return nil
}
