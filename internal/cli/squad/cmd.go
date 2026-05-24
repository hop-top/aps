package squad

import (
	coresquad "hop.top/aps/internal/core/squad"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

var defaultManager = coresquad.NewManager()

// NewSquadCmd returns the top-level squad command with all subcommands.
func NewSquadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "squad",
		Aliases: []string{"squads"},
		Short:   "Manage agent squads (topology, membership, scope)",
	}
	// T-0648 — intermediate ancestor for depth-3 leaves under
	// `aps squad members …`. The signature validator requires
	// kit/hierarchical on every intermediate parent of any depth>=3
	// leaf, regardless of whether the parent is a reserved name.
	kitcli.SetHierarchical(cmd)

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newMembersCmd())
	cmd.AddCommand(newCheckCmd())

	return cmd
}
