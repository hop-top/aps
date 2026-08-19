package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/bundle"
	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/core/org"
	"hop.top/aps/internal/styles"
)

// profileSummaryRow is the table/json/yaml row shape for `aps profile list`.
// Higher-priority columns survive narrow terminals (kit/output Table
// drops low-priority columns first when width is constrained).
type profileSummaryRow struct {
	ID           string `table:"ID,priority=10"           json:"id"            yaml:"id"`
	DisplayName  string `table:"DISPLAY NAME,priority=9"  json:"display_name"  yaml:"display_name"`
	Roles        string `table:"ROLES,priority=8"         json:"roles"         yaml:"roles"`
	Capabilities string `table:"CAPABILITIES,priority=7"  json:"capabilities"  yaml:"capabilities"`
	Workspace    string `table:"WORKSPACE,priority=6"     json:"workspace"     yaml:"workspace"`
	Email        string `table:"EMAIL,priority=5"         json:"email"         yaml:"email"`
	HasSecrets   bool   `table:"SECRETS,priority=4"       json:"has_secrets"   yaml:"has_secrets"`
	HasIdentity  bool   `table:"DID,priority=3"           json:"has_identity"  yaml:"has_identity"`
	Color        string `table:"COLOR,priority=2"         json:"color"         yaml:"color"`
	Avatar       string `table:"AVATAR,priority=1"        json:"avatar"        yaml:"avatar"`
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage agent profiles",
	Long:  `Create, list, and inspect agent profiles.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available profiles",
	Long: `List every agent profile discovered under
$APS_DATA_PATH/profiles/, projected into a row shape that includes
id, display name, roles, capabilities, workspace link, email,
has-secrets, has-identity, color, and avatar.

Filters compose with AND semantics: --capability, --role, --squad,
--tone match against the corresponding slice/string field;
--workspace (kit-shipped persistent global) filters by linked
workspace name; --has-identity / --has-secrets are boolean flags
that scope to profiles that do (or do not) have the given module
attached. Output respects the global --format flag (table|json|
yaml).

Read-only: no profile state is mutated. Idempotent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := core.ListProfilesFull()
		if err != nil {
			return fmt.Errorf("listing profiles: %w", err)
		}

		// Compose the filter predicate from CLI flags. Unset flags
		// produce nil predicates which All() treats as match-all.
		capFlag, _ := cmd.Flags().GetString("capability")
		roleFlag, _ := cmd.Flags().GetString("role")
		squadFlag, _ := cmd.Flags().GetString("squad")
		workspaceFlag, _ := cmd.Flags().GetString("workspace")
		toneFlag, _ := cmd.Flags().GetString("tone")
		hasIdentity, _ := cmd.Flags().GetBool("has-identity")
		hasSecrets, _ := cmd.Flags().GetBool("has-secrets")

		pred := listing.All(
			listing.MatchSlice(func(p core.Profile) []string { return p.Capabilities }, capFlag),
			listing.MatchSlice(func(p core.Profile) []string { return p.Roles }, roleFlag),
			listing.MatchSlice(func(p core.Profile) []string { return p.Squads }, squadFlag),
			listing.MatchString(func(p core.Profile) string {
				if p.Workspace == nil {
					return ""
				}
				return p.Workspace.Name
			}, workspaceFlag),
			listing.MatchString(func(p core.Profile) string { return p.Persona.Tone }, toneFlag),
			listing.BoolFlag(cmd.Flags().Changed("has-identity"),
				func(p core.Profile) bool { return p.Identity != nil }, hasIdentity),
			listing.BoolFlag(cmd.Flags().Changed("has-secrets"),
				profileHasSecrets, hasSecrets),
		)

		filtered := listing.Filter(profiles, pred)
		rows := make([]profileSummaryRow, 0, len(filtered))
		for _, p := range filtered {
			rows = append(rows, profileToSummaryRow(p))
		}

		format := root.Viper.GetString("format")
		if format == "" {
			format = output.Table
		}
		return listing.RenderList(os.Stdout, format, rows)
	},
}

// dedupeStrings returns s with duplicates removed, preserving order of
// first occurrence. Used by `profile edit` to avoid listing the same
// field twice when both --avatar and --auto-avatar are passed.
func dedupeStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	out := make([]string, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// reportsToCycleCheck builds the would-be reporting graph with profile
// id's reports_to set to reportsTo and reports any cycle the mutation
// would introduce. profiles is the current on-disk set; the entry for
// id is replaced (or appended when id is being created) before the walk.
func reportsToCycleCheck(id, reportsTo string, profiles []core.Profile) error {
	mutated := make([]core.Profile, 0, len(profiles)+1)
	replaced := false
	for _, p := range profiles {
		if p.ID == id {
			p.ReportsTo = reportsTo
			replaced = true
		}
		mutated = append(mutated, p)
	}
	if !replaced {
		mutated = append(mutated, core.Profile{ID: id, ReportsTo: reportsTo})
	}
	if _, err := org.Build(mutated).Chain(id); err != nil {
		return fmt.Errorf("reports-to %q would introduce a %w", reportsTo, err)
	}
	return nil
}

// validateReportsTo enforces the write-time posture on reports_to for
// the create/edit flag paths: the target must exist, must not be the
// profile itself, and must not close a reporting cycle. Empty clears
// and is always valid.
func validateReportsTo(id, reportsTo string) error {
	if reportsTo == "" {
		return nil
	}
	if reportsTo == id {
		return fmt.Errorf("profile %q cannot report to itself", id)
	}
	profiles, err := core.ListProfilesFull()
	if err != nil {
		return fmt.Errorf("listing profiles: %w", err)
	}
	exists := false
	for _, p := range profiles {
		if p.ID == reportsTo {
			exists = true
			break
		}
	}
	if !exists {
		return fmt.Errorf("reports-to %q does not match an existing profile", reportsTo)
	}
	return reportsToCycleCheck(id, reportsTo, profiles)
}

// profileHasSecrets reports whether the profile has at least one
// non-empty secret entry. Used by the --has-secrets filter; absence
// of the file (or an empty file) is treated as "no secrets".
func profileHasSecrets(p core.Profile) bool {
	secrets, err := core.LoadProfileSecrets(p.ID)
	if err != nil {
		return false
	}
	return len(secrets) > 0
}

// profileToSummaryRow projects a Profile into the row shape rendered
// by `aps profile list`. Slice fields are joined with ", " for table
// readability; json/yaml output preserves the same string (callers
// wanting structured slices should query individual profiles).
func profileToSummaryRow(p core.Profile) profileSummaryRow {
	wsName := ""
	if p.Workspace != nil {
		wsName = p.Workspace.Name
	}
	return profileSummaryRow{
		ID:           p.ID,
		DisplayName:  p.DisplayName,
		Roles:        strings.Join(p.Roles, ", "),
		Capabilities: strings.Join(p.Capabilities, ", "),
		Workspace:    wsName,
		Email:        p.Email,
		HasSecrets:   profileHasSecrets(p),
		HasIdentity:  p.Identity != nil,
		Color:        p.Color,
		Avatar:       p.Avatar,
	}
}

var profileCreateCmd = &cobra.Command{
	Use:   "create [id]",
	Short: "Create a new profile",
	Long: `Create a new agent profile under
$APS_DATA_PATH/profiles/<id>/. The id argument doubles as the
directory name; display name, email, avatar URL, and color hex
default to interactive prompts when omitted and stdin is a TTY.
Pass --force to overwrite an existing profile directory.

The --type flag sets the profile type discriminator (agent or
human; empty means agent). The --reports-to flag links the profile
into the reporting hierarchy: the target must be an existing
profile, and self-references or reporting cycles are rejected
before anything is written.

The --auto-avatar / --auto-color flags generate deterministic
values from the profile id (avatar via the configured provider,
default dicebear; color from a fixed palette hash). Provider knobs
(--avatar-provider, --avatar-style, --avatar-size, --avatar-format)
override per-call config. ProfileDefaults config drives the auto
behavior when the flags are unset.

Mutating: writes the profile.yaml record and (optionally) seeds the
git config block. Emits a ProfileCreated bus event with the --note
metadata attached. Not idempotent — each call creates a fresh
record (use --force to replace).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		displayName, _ := cmd.Flags().GetString("display-name")
		email, _ := cmd.Flags().GetString("email")
		avatarVal, _ := cmd.Flags().GetString("avatar")
		colorVal, _ := cmd.Flags().GetString("color")
		typeVal, _ := cmd.Flags().GetString("type")
		reportsTo, _ := cmd.Flags().GetString("reports-to")
		force, _ := cmd.Flags().GetBool("force")

		// Resolve auto-assignment policy. Explicit --auto-avatar/--auto-color
		// flags override config; otherwise fall back to ProfileDefaultsConfig.
		cfg, _ := core.LoadConfig()
		avatarMode := core.AutoModeFalse
		colorMode := core.AutoModeFalse
		avatarCfg := core.ProfileAvatarConfig{}
		if cfg != nil {
			avatarMode = cfg.Profile.Avatar.Enabled
			colorMode = cfg.Profile.Color
			avatarCfg = cfg.Profile.Avatar
		}
		// Per-call flag overrides for the avatar generator.
		if v, _ := cmd.Flags().GetString("avatar-provider"); v != "" {
			avatarCfg.Provider = v
		}
		if v, _ := cmd.Flags().GetString("avatar-style"); v != "" {
			avatarCfg.Style = v
		}
		if cmd.Flags().Changed("avatar-size") {
			v, _ := cmd.Flags().GetInt("avatar-size")
			avatarCfg.Size = v
		}
		if v, _ := cmd.Flags().GetString("avatar-format"); v != "" {
			avatarCfg.Format = v
		}
		if cmd.Flags().Changed("auto-avatar") {
			if v, _ := cmd.Flags().GetBool("auto-avatar"); v {
				avatarMode = core.AutoModeTrue
			} else {
				avatarMode = core.AutoModeFalse
			}
		}
		if cmd.Flags().Changed("auto-color") {
			if v, _ := cmd.Flags().GetBool("auto-color"); v {
				colorMode = core.AutoModeTrue
			} else {
				colorMode = core.AutoModeFalse
			}
		}

		// Interactive prompts when flags not provided and stdin is a terminal
		interactive := term.IsTerminal(int(os.Stdin.Fd()))
		if displayName == "" && interactive {
			if err := huh.NewInput().
				Title("Display name").
				Placeholder(id).
				Value(&displayName).
				Run(); err != nil {
				return err
			}
		}
		if email == "" && interactive {
			if err := huh.NewInput().
				Title("Email (for profile + git config)").
				Value(&email).
				Run(); err != nil {
				return err
			}
		}

		// Auto-assign when no explicit value given. We treat the
		// avatar/color prompts as non-interactive (no huh prompt for
		// them), so auto mode generates rather than defers.
		if avatarVal == "" && avatarMode.ShouldAutoAssign(false) {
			avatarVal = core.GenerateProfileAvatar(id, avatarCfg)
		}
		if colorVal == "" && colorMode.ShouldAutoAssign(false) {
			colorVal = core.GenerateProfileColor(id)
		}

		config := core.Profile{
			DisplayName: displayName,
			Email:       email,
			Type:        typeVal,
			ReportsTo:   reportsTo,
			Avatar:      avatarVal,
			Color:       colorVal,
			Git: core.GitConfig{
				Enabled: email != "",
			},
		}
		if config.DisplayName == "" {
			config.DisplayName = id
		}

		// Write-strict validation BEFORE any mutation (including the
		// --force removal below), so a rejected create never destroys
		// the existing profile it would have replaced.
		if err := config.ValidateType(); err != nil {
			return err
		}
		if err := validateReportsTo(id, reportsTo); err != nil {
			return err
		}

		if force {
			dir, err := core.GetProfileDir(id)
			if err != nil {
				return fmt.Errorf("resolving profile dir: %w", err)
			}
			if _, err := os.Stat(dir); err == nil {
				if err := os.RemoveAll(dir); err != nil {
					return fmt.Errorf("removing existing profile: %w", err)
				}
			}
		}

		// T-1291 — attach --note to ctx via policy.ContextAttrsKey
		// BEFORE the entity-mutating call so kit's policy engine and
		// the bus event payload can both surface it.
		ctx := WithNote(cmd.Context(), NoteFromCmd(cmd))
		if err := core.CreateProfileWithContext(ctx, id, config); err != nil {
			return fmt.Errorf("creating profile: %w", err)
		}
		// ProfileCreated event is emitted by core.CreateProfileWithContext.

		fmt.Printf("Profile '%s' created successfully.\n", id)
		return nil
	},
}

var profileEditCmd = &cobra.Command{
	Use:   "edit [id]",
	Short: "Edit fields on an existing profile",
	Long: `Update display name, email, avatar, color, type, or reports-to on an
existing profile.

Only flags that are explicitly passed are applied; unset flags leave the
existing value unchanged. To clear a field, pass the flag with an empty
string (e.g. --avatar "").

--type is validated write-strict (agent or human). --reports-to must
name an existing profile and may not introduce a self-reference or a
reporting cycle; the would-be graph is checked before saving.

The --auto-avatar / --auto-color flags generate a deterministic value
from the profile id and overwrite the existing value when set.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, err := core.LoadProfile(id)
		if err != nil {
			return fmt.Errorf("loading profile: %w", err)
		}

		var fields []string
		if cmd.Flags().Changed("display-name") {
			profile.DisplayName, _ = cmd.Flags().GetString("display-name")
			fields = append(fields, "display_name")
		}
		if cmd.Flags().Changed("email") {
			profile.Email, _ = cmd.Flags().GetString("email")
			fields = append(fields, "email")
		}
		if cmd.Flags().Changed("avatar") {
			profile.Avatar, _ = cmd.Flags().GetString("avatar")
			fields = append(fields, "avatar")
		}
		if cmd.Flags().Changed("color") {
			profile.Color, _ = cmd.Flags().GetString("color")
			fields = append(fields, "color")
		}
		if cmd.Flags().Changed("type") {
			profile.Type, _ = cmd.Flags().GetString("type")
			fields = append(fields, "type")
			// Write-strict: read paths tolerate unknown types, the
			// write path does not.
			if err := profile.ValidateType(); err != nil {
				return err
			}
		}
		if cmd.Flags().Changed("reports-to") {
			profile.ReportsTo, _ = cmd.Flags().GetString("reports-to")
			fields = append(fields, "reports_to")
			// Target must exist, no self-reference, no cycle — checked
			// against the would-be graph before anything is saved.
			if err := validateReportsTo(id, profile.ReportsTo); err != nil {
				return err
			}
		}
		if v, _ := cmd.Flags().GetBool("auto-avatar"); v {
			cfg, _ := core.LoadConfig()
			avatarCfg := core.ProfileAvatarConfig{}
			if cfg != nil {
				avatarCfg = cfg.Profile.Avatar
			}
			if v, _ := cmd.Flags().GetString("avatar-provider"); v != "" {
				avatarCfg.Provider = v
			}
			if v, _ := cmd.Flags().GetString("avatar-style"); v != "" {
				avatarCfg.Style = v
			}
			if cmd.Flags().Changed("avatar-size") {
				v, _ := cmd.Flags().GetInt("avatar-size")
				avatarCfg.Size = v
			}
			if v, _ := cmd.Flags().GetString("avatar-format"); v != "" {
				avatarCfg.Format = v
			}
			profile.Avatar = core.GenerateProfileAvatar(id, avatarCfg)
			fields = append(fields, "avatar")
		}
		if v, _ := cmd.Flags().GetBool("auto-color"); v {
			profile.Color = core.GenerateProfileColor(id)
			fields = append(fields, "color")
		}

		if len(fields) == 0 {
			return fmt.Errorf("no fields specified; pass at least one of --display-name, --email, --avatar, --color, --type, --reports-to, --auto-avatar, --auto-color")
		}

		// T-1291 — attach --note to ctx BEFORE the save so the
		// ProfileUpdated event payload carries the audit note.
		ctx := WithNote(cmd.Context(), NoteFromCmd(cmd))
		if err := core.SaveProfile(profile); err != nil {
			return fmt.Errorf("saving profile: %w", err)
		}
		core.PublishProfileUpdatedWithContext(ctx, id, dedupeStrings(fields))

		fmt.Printf("Profile '%s' updated (%s).\n", id, strings.Join(dedupeStrings(fields), ", "))
		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show profile details",
	Long: `Show the full profile record as YAML: identity fields,
workspace link, capability list (annotated as builtin or external
with description), and the status of optional modules
(secrets present/missing, redacted secret keys). Capability
descriptions are looked up from the builtin registry or the
external capability path.

Read-only: loads $APS_DATA_PATH/profiles/<id>/profile.yaml and
prints a human-friendly render. Idempotent.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, err := core.LoadProfile(id)
		if err != nil {
			return fmt.Errorf("loading profile: %w", err)
		}
		return writeProfileShow(cmd.OutOrStdout(), globals.Format(), profile)
	},
}

// writeProfileShow renders a profile record to w in the requested
// format.
//
// The machine formats emit the profile record and nothing else, so
// stdout parses as a single document: this command used to print YAML
// followed by an ANSI-styled human block no matter what --format
// asked for, which meant an agent requesting JSON got bytes no JSON
// parser accepts. The human view keeps the rich render — the
// annotated capability list and module status operators rely on.
func writeProfileShow(w io.Writer, format string, profile *core.Profile) error {
	switch strings.ToLower(format) {
	case output.JSON, output.YAML:
		return output.Render(w, strings.ToLower(format), profile)
	}
	return writeProfileShowHuman(w, profile)
}

// writeProfileShowHuman renders the operator-facing view: the record
// as YAML, then the workspace link, the capability list annotated with
// builtin/external provenance, and module status.
func writeProfileShowHuman(w io.Writer, profile *core.Profile) error {
	data, err := yaml.Marshal(profile)
	if err != nil {
		return fmt.Errorf("marshaling profile: %w", err)
	}
	_, _ = fmt.Fprintln(w, string(data))

	if profile.Workspace != nil {
		_, _ = fmt.Fprintf(w, "\nWorkspace: %s (%s)\n",
			styles.Bold.Render(profile.Workspace.Name),
			profile.Workspace.Scope)
	}

	if len(profile.Capabilities) > 0 {
		_, _ = fmt.Fprintln(w, "capabilities:")
		for _, capName := range profile.Capabilities {
			_, _ = fmt.Fprintln(w, profileCapabilityLine(capName))
		}
	}

	_, _ = fmt.Fprintln(w, "\nModules:")
	dir, _ := core.GetProfileDir(profile.ID)
	if _, err := os.Stat(filepath.Join(dir, "secrets.env")); err == nil {
		_, _ = fmt.Fprintln(w, "- Secrets: present")
		secrets, _ := core.LoadProfileSecrets(profile.ID)
		for k := range secrets {
			_, _ = fmt.Fprintf(w, "  - %s: ***redacted***\n", k)
		}
	} else {
		_, _ = fmt.Fprintln(w, "- Secrets: missing")
	}
	return nil
}

// profileCapabilityLine renders one annotated capability row: status
// dot, name, builtin/external badge, and a dimmed description when the
// registry knows one (external capabilities fall back to their path).
func profileCapabilityLine(capName string) string {
	dot := styles.StatusDot(true)
	kind := "external"
	desc := ""
	if b, e := capability.GetBuiltin(capName); e == nil {
		kind = "builtin"
		desc = b.Description
	} else if ext, e := capability.LoadCapability(capName); e == nil {
		desc = ext.Description
		if desc == "" {
			desc = ext.Path
		}
	}
	line := fmt.Sprintf("  %s %-18s %s", dot, capName, styles.KindBadge(kind))
	if desc != "" {
		line += "  " + styles.Dim.Render(desc)
	}
	return line
}

// profileCapabilityCmd is the `aps profile capability` mid-level
// command group (add, remove).
var profileCapabilityCmd = &cobra.Command{
	Use:   "capability",
	Short: "Manage capabilities on a profile",
}

var profileAddCapCmd = &cobra.Command{
	Use:   "add <profile> <capability>",
	Short: "Add a capability to a profile",
	Long: `Attach a capability to a profile's capability list. The
capability argument must resolve in either the builtin registry or
the external capabilities tree under
$APS_DATA_PATH/capabilities/; unknown names are rejected before
any state changes.

Mutating: writes profile.yaml and emits a ProfileUpdated bus event
with the --note metadata attached. Idempotent at the field level
(re-adding an already-attached capability is a no-op).`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID, capName := args[0], args[1]
		if !capability.Exists(capName) {
			return fmt.Errorf("capability '%s' does not exist", capName)
		}
		// T-1291 — attach --note before mutating profile capabilities.
		ctx := WithNote(cmd.Context(), NoteFromCmd(cmd))
		if err := core.AddCapabilityToProfileWithContext(ctx, profileID, capName); err != nil {
			return err
		}
		// ProfileUpdated event is emitted by core.AddCapabilityToProfile.

		fmt.Printf("%s %s added to %s\n",
			styles.StatusDot(true), capName, profileID)
		return nil
	},
}

var profileRemoveCapCmd = &cobra.Command{
	Use:   "remove <profile> <capability>",
	Short: "Remove a capability from a profile",
	Long: `Detach a capability from a profile's capability list. The
capability files on disk are not removed — only the link on the
profile record is dropped, so other profiles with the same
capability are unaffected.

Mutating: writes profile.yaml and emits a ProfileUpdated bus event
with the --note metadata attached. Idempotent: removing a
capability that isn't on the profile is a no-op.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID, capName := args[0], args[1]
		// T-1291 — attach --note before mutating profile capabilities.
		ctx := WithNote(cmd.Context(), NoteFromCmd(cmd))
		if err := core.RemoveCapabilityFromProfileWithContext(ctx, profileID, capName); err != nil {
			return err
		}
		// ProfileUpdated event is emitted by core.RemoveCapabilityFromProfile.

		fmt.Printf("%s %s removed from %s\n",
			styles.StatusDot(false), capName, profileID)
		return nil
	},
}

// profileStatusCmd implements `aps profile status <id>` (T-0054, T-0055).
// It shows per-bundle binary results (skipped/blocked/warned) and, with --verbose,
// the full resolved scope and injected env var keys for each bundle.
var profileStatusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "Show bundle resolution status for a profile",
	Long: `Show per-bundle binary resolution for a profile: which required
binaries are active, missing/skipped, or blocked. Bundles are
inferred from the profile's capability list via
core.ExtractBundleNames.

With the inherited --verbose global, also emits each bundle's
resolved scope (operations, file_patterns, networks) and the set
of env-var keys the bundle would inject when activated. Warnings
from the resolver (e.g. version conflicts) surface inline.

Read-only: loads profile.yaml and runs the bundle resolver; no
disk writes. Idempotent up to filesystem drift between calls.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		// --verbose is a kit-shipped root-persistent global; read the
		// inherited flag rather than redeclaring locally (T-0648 batch 8
		// local-globals dedup).
		verbose, _ := cmd.Flags().GetBool("verbose")

		profile, err := core.LoadProfile(id)
		if err != nil {
			return fmt.Errorf("failed to load profile %s: %w", id, err)
		}

		bundleNames, _ := core.ExtractBundleNames(profile.Capabilities)

		fmt.Printf("Profile: %s\n", styles.Bold.Render(id))

		if len(bundleNames) == 0 {
			fmt.Println("Bundles: (none)")
			return nil
		}

		fmt.Printf("Bundles: %s\n\n", styles.Dim.Render(joinStrings(bundleNames, ", ")))

		resolvedBundles, err := core.ResolveBundlesForProfile(profile)
		if err != nil {
			// Even on error, show what we can.
			fmt.Fprintf(os.Stderr, "Warning: bundle resolution error: %v\n", err)
		}

		// Map bundle name → resolved bundle for display.
		rbByName := make(map[string]*bundle.ResolvedBundle, len(resolvedBundles))
		for _, rb := range resolvedBundles {
			rbByName[rb.Bundle.Name] = rb
		}

		for _, name := range bundleNames {
			rb, ok := rbByName[name]
			if !ok {
				fmt.Printf(
					"  %s bundle:%s  %s\n",
					styles.Error.Render("✗"),
					name,
					styles.Dim.Render("(not resolved)"),
				)
				continue
			}

			// T-0054 — Binary results.
			if len(rb.BinaryResults) == 0 {
				fmt.Printf("  bundle:%s  %s\n", name, styles.Dim.Render("no binary requirements"))
			} else {
				for _, br := range rb.BinaryResults {
					icon, status := binaryResultStatus(br)
					fmt.Printf("  %s %-20s %s\n", icon, br.Binary, styles.Dim.Render(status))
				}
			}

			// Warnings.
			for _, w := range rb.Warnings {
				fmt.Printf("  %s %s\n", styles.Warn.Render("!"), styles.Dim.Render(w))
			}

			// T-0055 — Verbose: show full resolved scope and env var keys.
			if verbose {
				fmt.Printf("\n  [bundle:%s — resolved scope]\n", name)
				if len(rb.Scope.Operations) > 0 {
					fmt.Printf("    operations:    %s\n", joinStrings(rb.Scope.Operations, ", "))
				}
				if len(rb.Scope.FilePatterns) > 0 {
					fmt.Printf("    file_patterns: %s\n", joinStrings(rb.Scope.FilePatterns, ", "))
				}
				if len(rb.Scope.Networks) > 0 {
					fmt.Printf("    networks:      %s\n", joinStrings(rb.Scope.Networks, ", "))
				}
				if len(rb.Env) > 0 {
					fmt.Printf("    env vars:      ")
					keys := make([]string, 0, len(rb.Env))
					for k := range rb.Env {
						keys = append(keys, k)
					}
					fmt.Printf("%s\n", joinStrings(keys, ", "))
				}
				fmt.Println()
			}
		}

		return nil
	},
}

// binaryResultStatus returns a display icon and status string for a BinaryResult.
func binaryResultStatus(br bundle.BinaryResult) (icon, status string) {
	switch {
	case br.Blocked:
		msg := "blocked"
		if br.Message != "" {
			msg = "blocked  (" + br.Message + ")"
		}
		return styles.Error.Render("✗"), msg
	case br.Skipped:
		return styles.Error.Render("✗"), "skipped  (binary not found)"
	case !br.Found:
		msg := "not found"
		if br.Message != "" {
			msg = "warning  (" + br.Message + ")"
		}
		return styles.Warn.Render("!"), msg
	default:
		return styles.Success.Render("✓"), "active"
	}
}

// joinStrings joins a slice of strings with sep.
func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

var profileShareCmd = &cobra.Command{
	Use:   "share [id]",
	Short: "Export a shareable profile bundle",
	Long: `Export a profile as a portable bundle file that another aps
install can ingest via aps profile import. The default output path
is "<id>.aps-profile.yaml" in the current working directory;
--out overrides the destination.

The bundle includes the profile.yaml plus any side-car files
(secrets are NOT included — the importer must re-supply them per
the profile_share_created tracking event). A profile_share_created
event is emitted on success with the bundle version recorded.

Mutating (filesystem write): creates the bundle file. Pair with
aps profile import on the receiving install.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		outPath, _ := cmd.Flags().GetString("out")
		if outPath == "" {
			outPath = fmt.Sprintf("%s.aps-profile.yaml", id)
		}

		bundle, err := core.ExportProfileBundle(id, outPath)
		if err != nil {
			return fmt.Errorf("exporting profile bundle: %w", err)
		}

		if err := core.TrackEvent("profile_share_created", map[string]string{
			"profile_id":     id,
			"bundle_version": bundle.Version,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record share event: %v\n", err)
		}

		fmt.Printf("Share bundle created: %s\n", outPath)
		fmt.Printf("Import with: aps profile import %s\n", outPath)
		return nil
	},
}

var profileImportCmd = &cobra.Command{
	Use:   "import [bundle|AGENTS.md]",
	Short: "Import a shared profile bundle or an agent role manifest",
	Long: `Import a profile bundle previously produced by aps profile
share, or an agent role manifest (AGENTS.md — YAML frontmatter +
markdown body). Dispatch is by extension: a .md argument is
treated as a manifest, anything else as a .aps-profile.yaml
bundle. By default the new profile keeps the source id; pass --id
to rename it (e.g. when the local install already has a profile
with the source id). For bundle imports --force overwrites an
existing profile directory with the same target id; for manifest
imports it overrides the reportsTo check (see below).

Manifest imports map title (falling back to name) to the display
name, slug (falling back to the slugified name) to the profile
id, description and reportsTo to the matching profile fields,
the markdown body to notes.md, and each skills entry to a
capability link when the shortname resolves in the capability
registry — unresolvable shortnames are warned to stderr and
skipped, never failing the import. A reportsTo naming no existing
profile does fail the import, before anything is written — import
the supervising profile first, or pass --force to downgrade it to
a warning and store the value anyway. A reportsTo that would
introduce a self-reference or a reporting cycle always fails the
import; --force never downgrades a cycle. Secrets, isolation, and
machine-specific paths are never taken from a manifest; the
profile receives the normal create-path defaults. --dry-run
prints the resulting profile.yaml plus intended links and skips
without writing anything, and reports the same reportsTo verdict
a real import would.

Mutating: creates $APS_DATA_PATH/profiles/<target-id>/ and emits
both a ProfileCreated bus event and a profile_share_imported
tracking event. The --note metadata is attached to the
ProfileCreated payload. Secrets are NOT imported (they are not
part of the bundle); set them separately after import.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bundlePath := args[0]
		id, _ := cmd.Flags().GetString("id")
		force, _ := cmd.Flags().GetBool("force")

		// T-1291 — attach --note before importing (which calls Create).
		ctx := WithNote(cmd.Context(), NoteFromCmd(cmd))

		// Agent role manifest (.md) → manifest import path. The kit
		// root-persistent --dry-run global previews the profile.yaml
		// plus intended capability links without writing.
		if isAgentManifestPath(bundlePath) {
			return runManifestImport(ctx, bundlePath, id, globals.DryRun(), force, os.Stdout, os.Stderr)
		}
		if globals.DryRun() {
			return errors.New("--dry-run is only supported for agent manifest (.md) imports; bundle imports copy the source directly")
		}
		profile, bundle, err := core.ImportProfileBundleWithContext(ctx, bundlePath, id, force)
		if err != nil {
			return fmt.Errorf("importing profile bundle: %w", err)
		}
		// ProfileCreated event is emitted by core.CreateProfile (called from ImportProfileBundle).

		if err := core.TrackEvent("profile_share_imported", map[string]string{
			"profile_id":     profile.ID,
			"source_id":      bundle.SourceID,
			"bundle_version": bundle.Version,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record import event: %v\n", err)
		}

		fmt.Printf("Profile '%s' imported successfully.\n", profile.ID)
		return nil
	},
}

// profileDeleteCmd implements `aps profile delete <id> [--force] [--yes]` (T7).
// It wraps core.DeleteProfile, prompting for confirmation interactively unless
// --yes is set, and surfacing a helpful hint when blocked by active sessions.
var profileDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a profile",
	Long: `Delete a profile directory under $APS_DATA_PATH/profiles/<id>/
along with its profile.yaml, secrets, and any side-car files.
Interactive confirmation prompts unless --yes is set or stdin is
not a TTY.

Destructive: irreversible without a prior aps profile share
export. Blocked when the profile has active sessions in the
session store, or when other profiles report to it (their ids are
listed); --force overrides both guards and removes the profile
anyway, leaving orphaned sessions to fail on next lookup and
inbound reports_to links dangling. A ProfileDeleted bus event
fires with the --note metadata attached.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		force, _ := cmd.Flags().GetBool("force")
		yes, _ := cmd.Flags().GetBool("yes")

		// Reporting-hierarchy guard: deleting a profile that others
		// report to would leave their reports_to dangling. Blocked
		// without --force; forced deletes warn with the reporter list.
		profiles, err := core.ListProfilesFull()
		if err != nil {
			return fmt.Errorf("listing profiles: %w", err)
		}
		if inbound := org.Build(profiles).DirectReports(id); len(inbound) > 0 {
			ids := make([]string, 0, len(inbound))
			for _, p := range inbound {
				ids = append(ids, p.ID)
			}
			if !force {
				return fmt.Errorf(
					"cannot delete profile %q: %d profile(s) report to it: %s\n\nHint: reassign their reports_to first, or pass --force to delete anyway (their reports_to will dangle)",
					id, len(ids), strings.Join(ids, ", "),
				)
			}
			fmt.Fprintf(os.Stderr,
				"Warning: %d profile(s) report to %q and will be left with a dangling reports_to: %s\n",
				len(ids), id, strings.Join(ids, ", "))
		}

		// Interactive confirmation unless --yes or non-tty.
		if !yes && term.IsTerminal(int(os.Stdin.Fd())) {
			confirmed := false
			prompt := fmt.Sprintf("Delete profile '%s'? This cannot be undone.", id)
			if err := huh.NewConfirm().
				Title(prompt).
				Value(&confirmed).
				Run(); err != nil {
				return fmt.Errorf("confirmation prompt: %w", err)
			}
			if !confirmed {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// T-1291 — attach --note to ctx BEFORE the delete.
		ctx := WithNote(cmd.Context(), NoteFromCmd(cmd))
		if err := core.DeleteProfileWithContext(ctx, id, force); err != nil {
			if errors.Is(err, core.ErrProfileHasActiveSessions) {
				return fmt.Errorf(
					"cannot delete profile '%s': %w\n\nHint: terminate the blocking sessions first, or pass --force to delete anyway",
					id,
					err,
				)
			}
			return fmt.Errorf("deleting profile %q: %w", id, err)
		}
		// ProfileDeleted event is emitted by core.DeleteProfile.

		fmt.Printf("Profile '%s' deleted.\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileEditCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileStatusCmd)
	profileCmd.AddCommand(profileShareCmd)
	profileCmd.AddCommand(profileImportCmd)
	profileCmd.AddCommand(profileCapabilityCmd)
	profileCapabilityCmd.AddCommand(profileAddCapCmd)
	profileCapabilityCmd.AddCommand(profileRemoveCapCmd)
	profileCmd.AddCommand(profileDeleteCmd)

	// `aps profile list` filter flags. --workspace is a kit-owned
	// global (T-0376) inherited via PersistentFlags; the others are
	// declared here.
	profileListCmd.Flags().String("capability", "", "Filter by capability membership")
	profileListCmd.Flags().String("role", "", "Filter by role membership (owner, assignee, evaluator, auditor)")
	profileListCmd.Flags().String("squad", "", "Filter by squad membership")
	profileListCmd.Flags().String("tone", "", "Filter by persona tone")
	profileListCmd.Flags().Bool("has-identity", false, "Filter to profiles with (true) or without (false) a DID identity")
	profileListCmd.Flags().Bool("has-secrets", false, "Filter to profiles with (true) or without (false) at least one secret")

	profileCreateCmd.Flags().String("display-name", "", "Display name for the profile")
	profileCreateCmd.Flags().String("email", "", "Email for profile and git config")
	profileCreateCmd.Flags().String("type", "", "Profile type: agent or human (empty means agent)")
	profileCreateCmd.Flags().String("reports-to", "", "Profile id this profile reports to (must exist; no self-reference or cycles)")
	profileCreateCmd.Flags().String("avatar", "", "URL or local path to profile image")
	profileCreateCmd.Flags().String("color", "", "Hex color (e.g. #3b82f6) for UI rendering")
	profileCreateCmd.Flags().Bool("auto-avatar", false, "Generate a deterministic avatar via the configured provider (overrides config)")
	profileCreateCmd.Flags().Bool("auto-color", false, "Generate a deterministic palette color (overrides config)")
	profileCreateCmd.Flags().String("avatar-provider", "", "Avatar provider name (default: kit/avatar's default — dicebear)")
	profileCreateCmd.Flags().String("avatar-style", "", "Provider-specific style (e.g. dicebear: shapes, bottts, identicon)")
	profileCreateCmd.Flags().Int("avatar-size", 0, "Avatar size in pixels (0 = provider default)")
	profileCreateCmd.Flags().String("avatar-format", "", "Avatar format: svg, png, webp (provider-dependent)")
	profileCreateCmd.Flags().Bool("force", false, "Overwrite existing profile")

	profileEditCmd.Flags().String("display-name", "", "Display name for the profile")
	profileEditCmd.Flags().String("email", "", "Email for profile and git config")
	profileEditCmd.Flags().String("type", "", "Profile type: agent or human (pass empty string to clear back to the agent default)")
	profileEditCmd.Flags().String("reports-to", "", "Profile id this profile reports to (pass empty string to clear; must exist; no self-reference or cycles)")
	profileEditCmd.Flags().String("avatar", "", "URL or local path to profile image (pass empty string to clear)")
	profileEditCmd.Flags().String("color", "", "Hex color (e.g. #3b82f6) for UI rendering (pass empty string to clear)")
	profileEditCmd.Flags().Bool("auto-avatar", false, "Generate and apply a deterministic avatar via the configured provider")
	profileEditCmd.Flags().Bool("auto-color", false, "Generate and apply a deterministic palette color")
	profileEditCmd.Flags().String("avatar-provider", "", "Avatar provider name for --auto-avatar")
	profileEditCmd.Flags().String("avatar-style", "", "Provider-specific style for --auto-avatar")
	profileEditCmd.Flags().Int("avatar-size", 0, "Avatar size in pixels for --auto-avatar")
	profileEditCmd.Flags().String("avatar-format", "", "Avatar format for --auto-avatar")
	// T-0648 batch 8 — local --verbose dropped to stop shadowing kit's
	// root-persistent --verbose (signature check local-globals). The
	// RunE reads cmd.Flags().GetBool("verbose") which now resolves to
	// the inherited global. Behaviour is unchanged.
	profileShareCmd.Flags().String("out", "", "Output path for the bundle")
	profileImportCmd.Flags().String("id", "", "Override profile ID from bundle")
	profileImportCmd.Flags().Bool("force", false, "Overwrite an existing profile (bundle imports); import despite a reportsTo that names no existing profile (manifest imports)")
	profileDeleteCmd.Flags().Bool("force", false, "Delete even if there are active sessions (orphans them — they keep running but lose profile context) or inbound reports_to links (they are left dangling)")
	profileDeleteCmd.Flags().BoolP("yes", "y", false, "Skip interactive confirmation")

	// T-1291 — --note|-n on every state-changing profile subcommand.
	// The note is attached to ctx via policy.ContextAttrsKey before the
	// core mutator runs (so policy.engine sees `context.note` from CEL)
	// and surfaces in the bus event payload for audit downstream.
	AddNoteFlag(profileCreateCmd)
	AddNoteFlag(profileEditCmd)
	AddNoteFlag(profileDeleteCmd)
	AddNoteFlag(profileImportCmd)
	AddNoteFlag(profileAddCapCmd)
	AddNoteFlag(profileRemoveCapCmd)

	// T-0648 — kit signature annotations. Read leaves (list, show,
	// status, trust) safe to retry; write leaves mutate local
	// $APS_DATA_PATH/profiles state; delete is irreversible-local.
	kitcli.SetSideEffect(profileListCmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(profileListCmd, kitcli.IdempotencyYes)
	kitcli.SetSideEffect(profileShowCmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(profileShowCmd, kitcli.IdempotencyYes)
	kitcli.SetSideEffect(profileStatusCmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(profileStatusCmd, kitcli.IdempotencyYes)
	kitcli.SetSideEffect(profileCreateCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(profileCreateCmd, kitcli.IdempotencyNo)
	// T-0656 — create is atomic-by-design; the prospective profile ID
	// is the name argument the user already supplied.
	kitcli.OptOutDryRun(profileCreateCmd)
	if err := kitcli.SetDryRunRationale(profileCreateCmd, "create is atomic-by-design; the prospective profile ID is the name argument the user supplied, and preview would only restate it."); err != nil {
		panic(err)
	}
	kitcli.SetSideEffect(profileEditCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(profileEditCmd, kitcli.IdempotencyYes)
	// T-0656 — edit shells out to $EDITOR; there is no batch boundary
	// on which to scope a preview before the user authors the diff.
	kitcli.OptOutDryRun(profileEditCmd)
	if err := kitcli.SetDryRunRationale(profileEditCmd, "edit opens the profile YAML in $EDITOR for a user-driven session; previewing would have to either suppress the editor or describe a diff that has not been authored yet."); err != nil {
		panic(err)
	}
	kitcli.SetSideEffect(profileShareCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(profileShareCmd, kitcli.IdempotencyYes)
	// T-0656 — share writes a share manifest at a canonical path
	// derived from the profile ID.
	kitcli.OptOutDryRun(profileShareCmd)
	if err := kitcli.SetDryRunRationale(profileShareCmd, "share writes a share-manifest file under a canonical path derived from the profile ID; previewing would only restate the destination path."); err != nil {
		panic(err)
	}
	kitcli.SetSideEffect(profileImportCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(profileImportCmd, kitcli.IdempotencyConditional)
	// --dry-run is honored on the agent-manifest (.md) import path
	// (previews profile.yaml + capability links without writing);
	// bundle imports reject it in RunE since previewing would perform
	// the same disk read as the real import.
	// T-0654 — profile delete removes the profile directory and all
	// associated state irreversibly; delete-by-id is idempotent.
	kitcli.SetSideEffect(profileDeleteCmd, kitcli.SideEffectDestructiveLocal)
	kitcli.SetIdempotency(profileDeleteCmd, kitcli.IdempotencyConditional)
	kitcli.SetDestructiveToken(profileDeleteCmd)
	// T-0656 — destructive-token confirm already gates the apply path;
	// preview would only restate the profile ID being removed.
	kitcli.OptOutDryRun(profileDeleteCmd)
	if err := kitcli.SetDryRunRationale(profileDeleteCmd, "delete removes the entire profile directory irreversibly; the destructive-token confirm flow already gates the apply path, and preview would only restate the profile ID."); err != nil {
		panic(err)
	}
	kitcli.SetSideEffect(profileAddCapCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(profileAddCapCmd, kitcli.IdempotencyYes)
	// T-0656 — add-cap appends a single entry to the profile manifest;
	// preview would only restate the capability name.
	kitcli.OptOutDryRun(profileAddCapCmd)
	if err := kitcli.SetDryRunRationale(profileAddCapCmd, "add-cap appends a single capability entry to the profile manifest; the result is fully determined by the capability name argument."); err != nil {
		panic(err)
	}
	kitcli.SetSideEffect(profileRemoveCapCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(profileRemoveCapCmd, kitcli.IdempotencyYes)
	// T-0656 — remove-cap drops a single entry from the profile
	// manifest; preview would only restate the capability name.
	kitcli.OptOutDryRun(profileRemoveCapCmd)
	if err := kitcli.SetDryRunRationale(profileRemoveCapCmd, "remove-cap drops a single capability entry from the profile manifest; the result is fully determined by the capability name argument."); err != nil {
		panic(err)
	}
}
