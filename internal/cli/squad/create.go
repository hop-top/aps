package squad

import (
	"fmt"
	"os"
	"strings"

	"hop.top/aps/internal/cli/clinote"
	coresquad "hop.top/aps/internal/core/squad"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newCreateCmd() *cobra.Command {
	var squadType string
	var domain string
	var description string
	var members []string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new squad",
		Long: `Create a new squad entry in the local squad store. The ID is
derived from the <name> argument by lower-casing and replacing
spaces with dashes. --type and --domain are required: --type must
be one of the four team-topology values (stream-aligned, enabling,
complicated-subsystem, platform), and --domain names the squad's
domain boundary. --description and --members are optional; the
members slice takes a comma-separated list of profile IDs.

Mints a new local record. Not idempotent — re-running with the
same name on an existing squad fails. --dry-run is opted out
because the resulting record is fully determined by the flags.
Use --note to attach an audit reason that flows to the event bus
alongside the mutation.`,
		Args: cobra.ExactArgs(1),
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
	clinote.AddFlag(cmd) // T-1291

	// T-0648 — kit 0.4 signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	// T-0656 — create writes a single squad entry; the result is fully
	// determined by the --type/--domain/--description/--members flags.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "create writes a single squad entry to the local squad store; the resulting record is fully determined by the type, domain, and members flags."); err != nil {
		panic(err)
	}

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
