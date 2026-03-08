package squad

import (
	coresquad "hop.top/aps/internal/core/squad"

	"github.com/spf13/cobra"
)

var defaultManager = coresquad.NewManager()

// NewSquadCmd returns the top-level squad command with all subcommands.
func NewSquadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "squad",
		Aliases: []string{"squads"},
		Short:   "Manage agent squads (topology, membership, scope)",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newAddMemberCmd())
	cmd.AddCommand(newRemoveMemberCmd())
	cmd.AddCommand(newCheckCmd())

	return cmd
}
