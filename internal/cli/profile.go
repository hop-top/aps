package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/bundle"
	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/styles"
)

var profileTableHeader = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDim)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage agent profiles",
	Long:  `Create, list, and inspect agent profiles.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := core.ListProfiles()
		if err != nil {
			return fmt.Errorf("listing profiles: %w", err)
		}

		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(profiles)
		}

		if len(profiles) == 0 {
			fmt.Fprintln(os.Stderr, styles.Dim.Render("No profiles found."))
			return nil
		}

		fmt.Printf("%s\n\n", styles.Title.Render("Profiles"))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, profileTableHeader.Render("ID"))
		for _, p := range profiles {
			fmt.Fprintln(w, p)
		}
		w.Flush()

		fmt.Printf("\n%s\n", styles.Dim.Render(
			fmt.Sprintf("%d profiles", len(profiles))))
		return nil
	},
}

var profileNewCmd = &cobra.Command{
	Use:   "new [id]",
	Short: "Create a new profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		displayName, _ := cmd.Flags().GetString("display-name")
		email, _ := cmd.Flags().GetString("email")
		force, _ := cmd.Flags().GetBool("force")

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
				Title("Email (for git config, optional)").
				Value(&email).
				Run(); err != nil {
				return err
			}
		}

		config := core.Profile{
			DisplayName: displayName,
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

		if err := core.CreateProfile(id, config); err != nil {
			return fmt.Errorf("creating profile: %w", err)
		}

		fmt.Printf("Profile '%s' created successfully.\n", id)
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
		secretsPath, _ := core.GetProfileDir(id)
		if _, err := os.Stat(secretsPath + "/secrets.env"); err == nil {
			fmt.Println("- Secrets: present")
			secrets, _ := core.LoadSecrets(secretsPath + "/secrets.env")
			for k := range secrets {
				fmt.Printf("  - %s: ***redacted***\n", k)
			}
		} else {
			fmt.Println("- Secrets: missing")
		}
		return nil
	},
}

var profileAddCapCmd = &cobra.Command{
	Use:   "add-capability <profile> <capability>",
	Short: "Add a capability to a profile",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID, capName := args[0], args[1]
		if !capability.Exists(capName) {
			return fmt.Errorf("capability '%s' does not exist", capName)
		}
		if err := core.AddCapabilityToProfile(profileID, capName); err != nil {
			return err
		}
		fmt.Printf("%s %s added to %s\n",
			styles.StatusDot(true), capName, profileID)
		return nil
	},
}

var profileRemoveCapCmd = &cobra.Command{
	Use:   "remove-capability <profile> <capability>",
	Short: "Remove a capability from a profile",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID, capName := args[0], args[1]
		if err := core.RemoveCapabilityFromProfile(profileID, capName); err != nil {
			return err
		}
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

		profile, bundle, err := core.ImportProfileBundle(bundlePath, id, force)
		if err != nil {
			return fmt.Errorf("importing profile bundle: %w", err)
		}

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

		if err := core.DeleteProfile(id, force); err != nil {
			if errors.Is(err, core.ErrProfileHasActiveSessions) {
				return fmt.Errorf(
					"cannot delete profile '%s': %w\n\nHint: terminate the blocking sessions first, or pass --force to delete anyway",
					id,
					err,
				)
			}
			return fmt.Errorf("deleting profile %q: %w", id, err)
		}

		fmt.Printf("Profile '%s' deleted.\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileNewCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileStatusCmd)
	profileCmd.AddCommand(profileShareCmd)
	profileCmd.AddCommand(profileImportCmd)
	profileCmd.AddCommand(profileAddCapCmd)
	profileCmd.AddCommand(profileRemoveCapCmd)
	profileCmd.AddCommand(profileDeleteCmd)

	profileListCmd.Flags().Bool("json", false, "Output as JSON")
	profileNewCmd.Flags().String("display-name", "", "Display name for the profile")
	profileNewCmd.Flags().String("email", "", "Email for git config")
	profileNewCmd.Flags().Bool("force", false, "Overwrite existing profile")
	profileStatusCmd.Flags().BoolP("verbose", "v", false, "Show full resolved scope and env var keys per bundle")
	profileShareCmd.Flags().String("out", "", "Output path for the bundle")
	profileImportCmd.Flags().String("id", "", "Override profile ID from bundle")
	profileImportCmd.Flags().Bool("force", false, "Overwrite existing profile")
	profileDeleteCmd.Flags().BoolP("force", "f", false, "Delete even if there are active sessions (orphans them — they keep running but lose profile context)")
	profileDeleteCmd.Flags().BoolP("yes", "y", false, "Skip interactive confirmation")
}
