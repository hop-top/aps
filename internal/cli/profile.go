package cli

import (
	"fmt"
	"os"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/capability"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage agent profiles",
	Long:  `Create, list, and inspect agent profiles.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available profiles",
	Run: func(cmd *cobra.Command, args []string) {
		profiles, err := core.ListProfiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing profiles: %v\n", err)
			os.Exit(1)
		}
		for _, p := range profiles {
			fmt.Println(p)
		}
	},
}

var profileNewCmd = &cobra.Command{
	Use:   "new [id]",
	Short: "Create a new profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		displayName, _ := cmd.Flags().GetString("display-name")
		email, _ := cmd.Flags().GetString("email")
		force, _ := cmd.Flags().GetBool("force")

		// Prepare initial config
		config := core.Profile{
			DisplayName: displayName,
			Git: core.GitConfig{
				Enabled: email != "",
			},
		}
		if config.DisplayName == "" {
			config.DisplayName = id
		}

		// Handle Force: if force is true and profile exists, we might need to remove it first or just overwrite?
		// Spec T013 just says "Implement aps profile new command handler with flags"
		// Spec 12.4 says "Refuse overwrite unless --force is provided"
		// CreateProfile returns error if exists.
		if force {
			dir, _ := core.GetProfileDir(id)
			if _, err := os.Stat(dir); err == nil {
				os.RemoveAll(dir) // DANGER: destructive, but requested by force
			}
		}

		if err := core.CreateProfile(id, config); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating profile: %v\n", err)
			os.Exit(1)
		}

		// If email provided, we might want to update the gitconfig we just wrote
		// But CreateProfile wrote a placeholder. Let's strictly follow MVP.
		fmt.Printf("Profile '%s' created successfully.\n", id)
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show profile details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		profile, err := core.LoadProfile(id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading profile: %v\n", err)
			os.Exit(1)
		}

		data, err := yaml.Marshal(profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling profile: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))

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

var profileShareCmd = &cobra.Command{
	Use:   "share [id]",
	Short: "Export a shareable profile bundle",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		outPath, _ := cmd.Flags().GetString("out")
		if outPath == "" {
			outPath = fmt.Sprintf("%s.aps-profile.yaml", id)
		}

		bundle, err := core.ExportProfileBundle(id, outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting profile bundle: %v\n", err)
			os.Exit(1)
		}

		if err := core.TrackEvent("profile_share_created", map[string]string{
			"profile_id":     id,
			"bundle_version": bundle.Version,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record share event: %v\n", err)
		}

		fmt.Printf("Share bundle created: %s\n", outPath)
		fmt.Printf("Import with: aps profile import %s\n", outPath)
	},
}

var profileImportCmd = &cobra.Command{
	Use:   "import [bundle]",
	Short: "Import a shared profile bundle",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bundlePath := args[0]
		id, _ := cmd.Flags().GetString("id")
		force, _ := cmd.Flags().GetBool("force")

		profile, bundle, err := core.ImportProfileBundle(bundlePath, id, force)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error importing profile bundle: %v\n", err)
			os.Exit(1)
		}

		if err := core.TrackEvent("profile_share_imported", map[string]string{
			"profile_id":     profile.ID,
			"source_id":      bundle.SourceID,
			"bundle_version": bundle.Version,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record import event: %v\n", err)
		}

		fmt.Printf("Profile '%s' imported successfully.\n", profile.ID)
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileNewCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileShareCmd)
	profileCmd.AddCommand(profileImportCmd)
	profileCmd.AddCommand(profileAddCapCmd)
	profileCmd.AddCommand(profileRemoveCapCmd)

	profileNewCmd.Flags().String("display-name", "", "Display name for the profile")
	profileNewCmd.Flags().String("email", "", "Email for git config")
	profileNewCmd.Flags().Bool("force", false, "Overwrite existing profile")
	profileShareCmd.Flags().String("out", "", "Output path for the bundle")
	profileImportCmd.Flags().String("id", "", "Override profile ID from bundle")
	profileImportCmd.Flags().Bool("force", false, "Overwrite existing profile")
}
