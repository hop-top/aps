package workspace

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/cli/policygate"
	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/styles"
	"hop.top/kit/go/console/output"
)

// ctxHistoryRow is the table row shape for `aps workspace ctx history`.
// T-0456 — moved off hand-rolled tabwriter so styled tables activate
// on a TTY.
type ctxHistoryRow struct {
	Version string `table:"VERSION,priority=10" json:"version"  yaml:"version"`
	Agent   string `table:"AGENT,priority=9"    json:"agent"    yaml:"agent"`
	Old     string `table:"OLD,priority=8"      json:"old"      yaml:"old"`
	New     string `table:"NEW,priority=7"      json:"new"      yaml:"new"`
	Time    string `table:"TIME,priority=6"     json:"time"     yaml:"time"`
}

// ctxSummaryRow is the per-context-variable row rendered by
// `aps workspace ctx list`. Type is inferred from Value (string,
// number, bool, json) since the underlying ContextVariable stores
// values as strings without an explicit type tag.
//
// T-1309: Visibility is rendered with priority=2 so it drops first
// on narrow terminals — operators normally only care when a private
// variable is in play, and the writer can always tell from --json.
type ctxSummaryRow struct {
	Key        string `table:"KEY,priority=10" json:"key" yaml:"key"`
	Type       string `table:"TYPE,priority=6" json:"type" yaml:"type"`
	Size       int    `table:"SIZE,priority=5" json:"size" yaml:"size"`
	Version    int    `table:"VERSION,priority=7" json:"version" yaml:"version"`
	UpdatedBy  string `table:"UPDATED BY,priority=4" json:"updated_by" yaml:"updated_by"`
	UpdatedAt  string `table:"UPDATED,priority=3" json:"updated_at" yaml:"updated_at"`
	Visibility string `table:"VISIBILITY,priority=2" json:"visibility" yaml:"visibility"`
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
		Long: `Set a context variable in the workspace.

By default variables are workspace-wide ("shared"): visible to every
member, gated by the per-key ACL. Pass --private to scope the
variable to the current profile only — private variables are
invisible to other profiles' workspace ctx get/list (T-1309).`,
		Args: cobra.ExactArgs(2),
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

			wc := collab.NewWorkspaceContextFromState(variables, nil)

			// T-1309 — surface Visibility on the policygate request_attrs
			// vocabulary so CEL rules can read context.request_attrs.visibility.
			private, _ := cmd.Flags().GetBool("private")
			visibility := collab.VisibilityShared
			if private {
				visibility = collab.VisibilityPrivate
			}

			// T-1291 — attach --note; T-1292 — kind discriminator;
			// T-1309 — visibility for context_variable rules.
			policyCtx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd))
			policyCtx = policygate.WithContextVariableAttrs(policyCtx, string(visibility))

			var setOpts []collab.SetOption
			if private {
				setOpts = append(setOpts, collab.WithVisibility(collab.VisibilityPrivate))
			}

			v, err := wc.SetWithContext(policyCtx, key, value, profile, collab.RoleOwner, setOpts...)
			if err != nil {
				return err
			}

			// Save back
			snapshot, _ := wc.Snapshot()
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
	clinote.AddFlag(cmd) // T-1291
	cmd.Flags().Bool("private", false,
		"Scope variable to the current profile (default: shared workspace-wide)")

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

			// T-1309 — visibility filter. When --profile is unset we
			// fall through to the raw view (compat with shell scripts
			// that don't pass a profile).
			profile, _ := resolveProfile(cmd)

			store, err := getStorage()
			if err != nil {
				return err
			}

			variables, err := store.LoadContext(wsID)
			if err != nil {
				return fmt.Errorf("loading context: %w", err)
			}

			ctx := collab.NewWorkspaceContextFromState(variables, nil)

			v, ok := ctx.GetForProfile(key, profile)
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
	addProfileFlag(cmd)
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

	// T-1309 — apply the visibility filter when a profile is in
	// scope. resolveProfile returns an error when --profile and
	// APS_PROFILE are both empty; a profile-less list is the legacy
	// raw view (used by tooling and tests), so we tolerate that.
	profile, _ := resolveProfile(cmd)

	store, err := getStorage()
	if err != nil {
		return err
	}

	variables, err := store.LoadContext(wsID)
	if err != nil {
		return fmt.Errorf("loading context: %w", err)
	}

	wc := collab.NewWorkspaceContextFromState(variables, nil)
	visible := wc.ListForProfile(profile)

	rows := buildCtxRows(visible)

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
			Key:        v.Key,
			Type:       inferCtxType(v.Value),
			Size:       len(v.Value),
			Version:    v.Version,
			UpdatedBy:  v.UpdatedBy,
			UpdatedAt:  v.UpdatedAt.Format("2006-01-02 15:04:05"),
			Visibility: string(v.EffectiveVisibility()),
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

			wc := collab.NewWorkspaceContextFromState(variables, nil)

			// T-1291 — attach --note before mutating shared context.
			policyCtx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd))

			// T-1309 — surface the variable's visibility on the
			// request_attrs vocabulary so CEL rules keying on
			// context.request_attrs.visibility see the correct value.
			// Falls back to "shared" when the variable is missing
			// (the subsequent delete will fail with not-found anyway).
			vizForGate := string(collab.VisibilityShared)
			if existing, ok := wc.GetForProfile(key, profile); ok {
				vizForGate = string(existing.EffectiveVisibility())
			}
			policyCtx = policygate.WithContextVariableAttrs(policyCtx, vizForGate)

			// T-1292 — synchronous policy gate. Publishes the kit
			// pre_persisted topic with Op=delete and kind=workspace_context
			// before the in-memory delete + persist fans out. Vetoes
			// surface as *policy.PolicyDeniedError, mapped to exit 4
			// via the domain.ErrConflict unwrap in internal/cli/exit.
			policyCtx, err = policygate.PublishDeletePrePersisted(policyCtx, "workspace_context", key)
			if err != nil {
				return err
			}

			if err := wc.DeleteWithContext(policyCtx, key, profile, collab.RoleOwner); err != nil {
				return err
			}

			snapshot, _ := wc.Snapshot()
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
	clinote.AddFlag(cmd) // T-1291

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

			rows := make([]ctxHistoryRow, 0, len(mutations))
			for _, m := range mutations {
				rows = append(rows, ctxHistoryRow{
					Version: fmt.Sprintf("v%d", m.Version),
					Agent:   m.AgentID,
					Old:     truncate(m.OldValue, 30),
					New:     truncate(m.NewValue, 30),
					Time:    m.Timestamp.Format("15:04:05"),
				})
			}
			return listing.RenderList(os.Stdout, output.Table, rows)
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
