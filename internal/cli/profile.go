package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
	"hop.top/kit/go/console/output"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/bundle"
	"hop.top/aps/internal/core/capability"
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
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		displayName, _ := cmd.Flags().GetString("display-name")
		email, _ := cmd.Flags().GetString("email")
		avatarVal, _ := cmd.Flags().GetString("avatar")
		colorVal, _ := cmd.Flags().GetString("color")
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
			Avatar:      avatarVal,
			Color:       colorVal,
			Git: core.GitConfig{
				Enabled: email != "",
			},
		}
		if config.DisplayName == "" {
			config.DisplayName = id
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
	Long: `Update display name, email, avatar, or color on an existing profile.

Only flags that are explicitly passed are applied; unset flags leave the
existing value unchanged. To clear a field, pass the flag with an empty
string (e.g. --avatar "").

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
			return fmt.Errorf("no fields specified; pass at least one of --display-name, --email, --avatar, --color, --auto-avatar, --auto-color")
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
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, err := core.LoadProfile(id)
		if err != nil {
			return fmt.Errorf("loading profile: %w", err)
		}

		data, err := yaml.Marshal(profile)
		if err != nil {
			return fmt.Errorf("marshaling profile: %w", err)
		}
		fmt.Println(string(data))

		// Workspace link
		if profile.Workspace != nil {
			fmt.Printf("\nWorkspace: %s (%s)\n",
				styles.Bold.Render(profile.Workspace.Name),
				profile.Workspace.Scope)
		}

		// Rich capabilities section
		if len(profile.Capabilities) > 0 {
			fmt.Println("capabilities:")
			for _, capName := range profile.Capabilities {
				dot := styles.StatusDot(true)
				kind := "external"
				desc := ""
				if b, e := capability.GetBuiltin(capName); e == nil {
					kind = "builtin"
					desc = b.Description
				} else if ext, e := capability.LoadCapability(capName); e == nil {
					if ext.Description != "" {
						desc = ext.Description
					} else {
						desc = ext.Path
					}
				}
				badge := styles.KindBadge(kind)
				line := fmt.Sprintf("  %s %-18s %s", dot, capName, badge)
				if desc != "" {
					line += "  " + styles.Dim.Render(desc)
				}
				fmt.Println(line)
			}
		}

		// Show modules status
		fmt.Println("\nModules:")
		dir, _ := core.GetProfileDir(id)
		if _, err := os.Stat(filepath.Join(dir, "secrets.env")); err == nil {
			fmt.Println("- Secrets: present")
			secrets, _ := core.LoadProfileSecrets(id)
			for k := range secrets {
				fmt.Printf("  - %s: ***redacted***\n", k)
			}
		} else {
			fmt.Println("- Secrets: missing")
		}
		return nil
	},
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
	Args:  cobra.ExactArgs(2),
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
	Args:  cobra.ExactArgs(2),
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
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
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
				fmt.Printf("  %s bundle:%s  %s\n",
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
	Args:  cobra.ExactArgs(1),
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
	Use:   "import [bundle]",
	Short: "Import a shared profile bundle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bundlePath := args[0]
		id, _ := cmd.Flags().GetString("id")
		force, _ := cmd.Flags().GetBool("force")

		// T-1291 — attach --note before importing (which calls Create).
		ctx := WithNote(cmd.Context(), NoteFromCmd(cmd))
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
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		force, _ := cmd.Flags().GetBool("force")
		yes, _ := cmd.Flags().GetBool("yes")

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
	profileEditCmd.Flags().String("avatar", "", "URL or local path to profile image (pass empty string to clear)")
	profileEditCmd.Flags().String("color", "", "Hex color (e.g. #3b82f6) for UI rendering (pass empty string to clear)")
	profileEditCmd.Flags().Bool("auto-avatar", false, "Generate and apply a deterministic avatar via the configured provider")
	profileEditCmd.Flags().Bool("auto-color", false, "Generate and apply a deterministic palette color")
	profileEditCmd.Flags().String("avatar-provider", "", "Avatar provider name for --auto-avatar")
	profileEditCmd.Flags().String("avatar-style", "", "Provider-specific style for --auto-avatar")
	profileEditCmd.Flags().Int("avatar-size", 0, "Avatar size in pixels for --auto-avatar")
	profileEditCmd.Flags().String("avatar-format", "", "Avatar format for --auto-avatar")
	profileStatusCmd.Flags().Bool("verbose", false, "Show full resolved scope and env var keys per bundle")
	profileShareCmd.Flags().String("out", "", "Output path for the bundle")
	profileImportCmd.Flags().String("id", "", "Override profile ID from bundle")
	profileImportCmd.Flags().Bool("force", false, "Overwrite existing profile")
	profileDeleteCmd.Flags().Bool("force", false, "Delete even if there are active sessions (orphans them — they keep running but lose profile context)")
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
}
