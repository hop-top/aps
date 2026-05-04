package workspace

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/styles"
)

// ctxSummaryRow is the per-context-variable row rendered by
// `aps workspace ctx list`. Type is inferred from Value (string,
// number, bool, json) since the underlying ContextVariable stores
// values as strings without an explicit type tag.
type ctxSummaryRow struct {
	Key       string `table:"KEY,priority=10" json:"key" yaml:"key"`
	Type      string `table:"TYPE,priority=6" json:"type" yaml:"type"`
	Size      int    `table:"SIZE,priority=5" json:"size" yaml:"size"`
	Version   int    `table:"VERSION,priority=7" json:"version" yaml:"version"`
	UpdatedBy string `table:"UPDATED BY,priority=4" json:"updated_by" yaml:"updated_by"`
	UpdatedAt string `table:"UPDATED,priority=3" json:"updated_at" yaml:"updated_at"`
}

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
		Long: `List context variables in the active or selected workspace.

The --workspace flag is a global (T-0376) and inherits from the
active workspace when not supplied. Use --key-prefix to narrow
the listing to keys starting with a given string.`,
		RunE: runCtxList,
	}

	cmd.Flags().String("key-prefix", "",
		"Filter to keys with this prefix (set membership)")

	return cmd
}

func runCtxList(cmd *cobra.Command, _ []string) error {
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

	rows := buildCtxRows(variables)

	prefix, _ := cmd.Flags().GetString("key-prefix")

	pred := listing.All(
		ctxKeyPrefixPredicate(prefix),
	)
	rows = listing.Filter(rows, pred)

	return listing.RenderList(os.Stdout, globals.Format(), rows)
}

func buildCtxRows(vars []collab.ContextVariable) []ctxSummaryRow {
	rows := make([]ctxSummaryRow, len(vars))
	for i, v := range vars {
		rows[i] = ctxSummaryRow{
			Key:       v.Key,
			Type:      inferCtxType(v.Value),
			Size:      len(v.Value),
			Version:   v.Version,
			UpdatedBy: v.UpdatedBy,
			UpdatedAt: v.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return rows
}

// ctxKeyPrefixPredicate is a per-row predicate for --key-prefix. Empty
// prefix returns nil (match-all) per the listing helper convention.
func ctxKeyPrefixPredicate(prefix string) listing.Predicate[ctxSummaryRow] {
	if prefix == "" {
		return nil
	}
	return func(r ctxSummaryRow) bool {
		return strings.HasPrefix(r.Key, prefix)
	}
}

// inferCtxType returns a coarse type tag derived from the raw stored
// string. ContextVariable is string-typed at rest, so callers shape
// values through Set(); we surface the most useful distinction —
// json blob vs scalar — without re-parsing.
func inferCtxType(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "string"
	}
	switch trimmed[0] {
	case '{':
		return "json-object"
	case '[':
		return "json-array"
	case '"':
		return "json-string"
	case 't', 'f':
		if trimmed == "true" || trimmed == "false" {
			return "bool"
		}
	}
	if trimmed[0] == '-' || (trimmed[0] >= '0' && trimmed[0] <= '9') {
		// Cheap numeric heuristic: leading sign/digit and no
		// non-numeric characters except a single decimal point.
		dotSeen := false
		numeric := true
		for i, ch := range trimmed {
			if i == 0 && ch == '-' {
				continue
			}
			if ch == '.' && !dotSeen {
				dotSeen = true
				continue
			}
			if ch < '0' || ch > '9' {
				numeric = false
				break
			}
		}
		if numeric {
			return "number"
		}
	}
	return "string"
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

			fmt.Printf("%s\n\n", styles.Title.Render(
				fmt.Sprintf("History: %s", key)))

			w := newTabWriter()
			fmt.Fprintln(w, collabTableHeader.Render("VERSION")+"\t"+
				collabTableHeader.Render("AGENT")+"\t"+
				collabTableHeader.Render("OLD")+"\t"+
				collabTableHeader.Render("NEW")+"\t"+
				collabTableHeader.Render("TIME"))
			for _, m := range mutations {
				fmt.Fprintf(w, "v%d\t%s\t%s\t%s\t%s\n",
					m.Version,
					m.AgentID,
					truncate(m.OldValue, 30),
					truncate(m.NewValue, 30),
					styles.Dim.Render(m.Timestamp.Format("15:04:05")),
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
