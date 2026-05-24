// Package cli — signature-annotation post-init pass (T-0648 batch 8).
//
// Walks the fully-built rootCmd tree once at package init time (this file
// is named zz_* so it sorts after every other top-level cli source file
// in Go's alphabetical init order) and stamps kit/hierarchical=true on
// every intermediate parent that lacks it.
//
// Why this lives in a single top-level file:
//
//   - The kit/cli signature validator (Root.ValidateSignature) flags
//     every leaf at depth >= 3 whose chain of non-root, non-leaf
//     ancestors is missing kit/hierarchical. To drop the
//     `depth-hierarchical` violation count to zero, EVERY intermediate
//     needs the annotation — including sub-parents declared inside
//     subdir packages (e.g. `aps adapter messenger`, `aps a2a tasks`).
//   - Batch 8's edit boundary is "top-level internal/cli/*.go only";
//     editing the subdir constructors is reserved for batches 1-7.
//   - A runtime tree walk from a top-level file can mark sub-parents
//     without touching subdir source. kitcli.SetHierarchical is
//     idempotent (setAnnotationTrue), so a later subdir-batch PR that
//     adds an explicit SetHierarchical call in the subdir's
//     constructor produces no diff in behaviour or test result.
//
// Leaf annotations (kit/side-effect, kit/idempotent) intentionally are
// NOT auto-applied here. Each subdir batch owns the classification
// decision for its own leaves; defaulting them centrally would
// pre-empt that judgment and ship the wrong tier for write/destructive
// commands that read like list/show by name.
package cli

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func init() {
	markIntermediatesHierarchical(rootCmd)
}

// markIntermediatesHierarchical walks the subtree rooted at root and
// sets kit/hierarchical=true on every non-root, non-leaf, non-builtin
// descendant that lacks the annotation. Idempotent: calling it twice
// is a no-op on the second pass.
func markIntermediatesHierarchical(root *cobra.Command) {
	if root == nil {
		return
	}
	var walk func(c *cobra.Command, depth int)
	walk = func(c *cobra.Command, depth int) {
		for _, sub := range c.Commands() {
			if isBuiltinCmd(sub) {
				continue
			}
			if sub.HasSubCommands() {
				// Intermediate node — annotate, then recurse.
				if !kitcli.IsHierarchical(sub) {
					kitcli.SetHierarchical(sub)
				}
				walk(sub, depth+1)
			}
		}
	}
	walk(root, 0)
}

// isBuiltinCmd mirrors kit/cli's internal isBuiltin (which is
// unexported). Returns true for the completion subtree, the auto-help
// command, the hidden __complete helper, and any cmd opting out via
// kit/exempt-validation. Keeps this file from accidentally tagging
// kit-shipped helpers that the validator already exempts.
func isBuiltinCmd(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	switch cmd.Name() {
	case "completion", "help", "__complete", "__completeNoDesc":
		return true
	}
	if p := cmd.Parent(); p != nil && p.Name() == "completion" {
		return true
	}
	if cmd.Annotations != nil && cmd.Annotations["kit/exempt-validation"] == "true" {
		return true
	}
	return false
}
