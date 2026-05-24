// Package adapter — link_parent.go owns the `aps adapter link` parent
// command introduced in T-0398. The previous flat trio
// (link / links / unlink) violated convention §3.2 (singular/plural
// verb-pair for read+write); they were replaced by add/list/delete
// under this parent.
package adapter

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newLinkParentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Manage device-profile links (add, list, delete)",
		Long: `Manage links between adapter devices and profiles.

A "link" is the relationship between a device and a profile; the device
and profile are the parties.`,
	}
	// T-0648 — link is an intermediate grouping node (depth 2 under
	// `adapter`, depth 3 under `messenger` alias); its add/list/delete
	// leaves trip the depth-hierarchical check without this annotation.
	kitcli.SetHierarchical(cmd)

	cmd.AddCommand(newLinkAddCmd())
	cmd.AddCommand(newLinkListCmd())
	cmd.AddCommand(newLinkDeleteCmd())

	return cmd
}
