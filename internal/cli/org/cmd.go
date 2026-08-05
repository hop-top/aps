// Package org implements the `aps org` command group: read-only views
// over the reporting hierarchy that profiles declare via reports_to.
// All graph semantics live in internal/core/org; this package only
// loads profiles off disk, projects graph results into rows, and
// renders them.
package org

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

// NewOrgCmd returns the top-level org command with all subcommands.
func NewOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Inspect the reporting hierarchy across profiles",
	}
	kitcli.SetHierarchical(cmd)

	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newSnapshotCmd())

	return cmd
}
