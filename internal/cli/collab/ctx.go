package collab

import (
	"fmt"
	"os"

	collab "oss-aps-cli/internal/core/collaboration"

	"github.com/spf13/cobra"
)

// NewCtxCmd creates the "collab ctx" command group.
func NewCtxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ctx",
		Aliases: []string{"context"},
		Short:   "Manage workspace shared context",
		Long: `Manage shared context variables in a collaboration workspace.

Context variables are key-value pairs visible to all agents.
Changes are tracked with version history and ACL enforcement.`,
	}

	cmd.AddCommand(newCtxSetCmd())
	cmd.AddCommand(newCtxGetCmd())
	cmd.AddCommand(newCtxListCmd())
	cmd.AddCommand(newCtxDeleteCmd())
	cmd.AddCommand(newCtxHistoryCmd())

	return cmd
}

func newCtxSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a context variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			profile, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			variables, err := store.LoadContext(wsID)
			if err != nil {
				variables = []collab.ContextVariable{}
			}

			ctx := collab.NewWorkspaceContextFromState(variables, nil)

			v, err := ctx.Set(key, value, profile, collab.RoleOwner)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			// Save back
			snapshot, _ := ctx.Snapshot()
			if err := store.SaveContext(wsID, snapshot); err != nil {
				return fmt.Errorf("saving context: %w", err)
			}

			if isJSON(cmd) {
				return outputJSON(v)
			}

			fmt.Printf("Set '%s' = '%s' (v%d)\n", key, value, v.Version)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addProfileFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}

func newCtxGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a context variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			variables, err := store.LoadContext(wsID)
			if err != nil {
				return fmt.Errorf("loading context: %w", err)
			}

			ctx := collab.NewWorkspaceContextFromState(variables, nil)

			v, ok := ctx.Get(key)
			if !ok {
				return fmt.Errorf("context variable %q not found", key)
			}

			if isJSON(cmd) {
				return outputJSON(v)
			}

			fmt.Println(v.Value)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}

func newCtxListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all context variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			variables, err := store.LoadContext(wsID)
			if err != nil {
				return fmt.Errorf("loading context: %w", err)
			}

			if isJSON(cmd) {
				return outputJSON(variables)
			}

			if len(variables) == 0 {
				fmt.Println("No context variables set.")
				fmt.Println()
				fmt.Println("  Set one:")
				fmt.Println("    aps collab ctx set <key> <value>")
				return nil
			}

			w := newTabWriter()
			fmt.Fprintf(w, "KEY\tVALUE\tVERSION\tUPDATED BY\n")
			for _, v := range variables {
				fmt.Fprintf(w, "%s\t%s\tv%d\t%s\n",
					v.Key,
					v.Value,
					v.Version,
					v.UpdatedBy,
				)
			}
			w.Flush()

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}

func newCtxDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a context variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			profile, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			variables, err := store.LoadContext(wsID)
			if err != nil {
				return fmt.Errorf("loading context: %w", err)
			}

			ctx := collab.NewWorkspaceContextFromState(variables, nil)

			if err := ctx.Delete(key, profile, collab.RoleOwner); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			snapshot, _ := ctx.Snapshot()
			if err := store.SaveContext(wsID, snapshot); err != nil {
				return fmt.Errorf("saving context: %w", err)
			}

			if isJSON(cmd) {
				return outputJSON(map[string]string{
					"key":    key,
					"status": "deleted",
				})
			}

			fmt.Printf("Deleted '%s'\n", key)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addProfileFlag(cmd)
	addForceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}

func newCtxHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history <key>",
		Short: "Show mutation history for a variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")

			store, err := getStorage()
			if err != nil {
				return err
			}

			variables, err := store.LoadContext(wsID)
			if err != nil {
				return fmt.Errorf("loading context: %w", err)
			}

			ctx := collab.NewWorkspaceContextFromState(variables, nil)

			mutations := ctx.MutationsForKey(key)

			if limit > 0 && limit < len(mutations) {
				mutations = mutations[len(mutations)-limit:]
			}

			if isJSON(cmd) {
				return outputJSON(mutations)
			}

			if len(mutations) == 0 {
				fmt.Printf("No history for '%s'\n", key)
				return nil
			}

			w := newTabWriter()
			fmt.Fprintf(w, "VERSION\tAGENT\tOLD\tNEW\tTIME\n")
			for _, m := range mutations {
				fmt.Fprintf(w, "v%d\t%s\t%s\t%s\t%s\n",
					m.Version,
					m.AgentID,
					truncate(m.OldValue, 30),
					truncate(m.NewValue, 30),
					m.Timestamp.Format("15:04:05"),
				)
			}
			w.Flush()

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)
	addLimitFlag(cmd)

	return cmd
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
