package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/skills"
)

// NewSkillCmd creates the skill command group
func NewSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage Agent Skills",
		Long:  `Manage Agent Skills - discover, install, run, and configure skills for your profiles.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newStatsCmd())
	cmd.AddCommand(newSuggestCmd())

	return cmd
}

// newListCmd creates the 'skill list' command
func newListCmd() *cobra.Command {
	var profileID string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Long:  `List all skills available in configured skill directories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg := skills.DefaultConfig()
			// TODO: Load from ~/.config/aps/config.yaml and merge

			// Create registry
			registry := skills.NewRegistry(profileID, cfg.SkillSources, cfg.AutoDetectIDEPaths)

			// Discover skills
			if err := registry.Discover(); err != nil {
				return fmt.Errorf("failed to discover skills: %w", err)
			}

			// Display by source
			bySource := registry.ListBySource()

			if len(bySource) == 0 {
				fmt.Println("No skills found.")
				fmt.Println()
				fmt.Println("To install skills:")
				fmt.Println("  aps skill install <path> [--global]")
				return nil
			}

			fmt.Printf("Found %d skill(s):\n\n", registry.Count())

			for source, skillList := range bySource {
				fmt.Printf("%s (%d):\n", source, len(skillList))
				for _, skill := range skillList {
					if verbose {
						fmt.Printf("  • %s\n", skill.Name)
						fmt.Printf("    %s\n", skill.Description)
						if skill.License != "" {
							fmt.Printf("    License: %s\n", skill.License)
						}
						fmt.Printf("    Location: %s\n", skill.BasePath)
						fmt.Println()
					} else {
						fmt.Printf("  %-30s %s\n", skill.Name, skill.Description)
					}
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID to list skills for")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information")

	return cmd
}

// newShowCmd creates the 'skill show' command
func newShowCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "show <skill-name>",
		Short: "Show detailed skill information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillName := args[0]

			// Load config
			cfg := skills.DefaultConfig()

			// Create registry
			registry := skills.NewRegistry(profileID, cfg.SkillSources, cfg.AutoDetectIDEPaths)

			// Discover skills
			if err := registry.Discover(); err != nil {
				return fmt.Errorf("failed to discover skills: %w", err)
			}

			// Get skill
			skill, found := registry.Get(skillName)
			if !found {
				return fmt.Errorf("skill '%s' not found", skillName)
			}

			// Display details
			fmt.Printf("Name:          %s\n", skill.Name)
			fmt.Printf("Description:   %s\n", skill.Description)
			if skill.License != "" {
				fmt.Printf("License:       %s\n", skill.License)
			}
			if skill.Compatibility != "" {
				fmt.Printf("Compatibility: %s\n", skill.Compatibility)
			}
			if len(skill.Metadata) > 0 {
				fmt.Println("Metadata:")
				for k, v := range skill.Metadata {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}
			fmt.Printf("\nLocation:      %s\n", skill.BasePath)
			fmt.Printf("Source:        %s\n", skill.SourcePath)

			// List scripts
			scripts, _ := skill.ListScripts()
			if len(scripts) > 0 {
				fmt.Println("\nScripts:")
				for _, script := range scripts {
					fmt.Printf("  • %s\n", script)
				}
			}

			// List references
			refs, _ := skill.ListReferences()
			if len(refs) > 0 {
				fmt.Println("\nReferences:")
				for _, ref := range refs {
					refPath := skill.GetReferencePath(ref)
					info, _ := os.Stat(refPath)
					size := ""
					if info != nil {
						size = fmt.Sprintf(" (%d bytes)", info.Size())
					}
					fmt.Printf("  • %s%s\n", ref, size)
				}
			}

			// Show body content preview (first 500 chars)
			if skill.BodyContent != "" {
				fmt.Println("\nInstructions (preview):")
				preview := skill.BodyContent
				if len(preview) > 500 {
					preview = preview[:500] + "..."
				}
				fmt.Println(strings.TrimSpace(preview))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID")

	return cmd
}

// newInstallCmd creates the 'skill install' command
func newInstallCmd() *cobra.Command {
	var global bool
	var profileID string

	cmd := &cobra.Command{
		Use:   "install <path>",
		Short: "Install a skill",
		Long:  `Install a skill from a local directory or archive.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourcePath := args[0]

			// Validate source
			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				return fmt.Errorf("source path does not exist: %s", sourcePath)
			}

			// Parse skill to validate
			skill, err := skills.ParseSkill(sourcePath)
			if err != nil {
				return fmt.Errorf("invalid skill: %w", err)
			}

			// Determine target directory
			var targetBase string
			if global {
				targetBase = skills.NewSkillPaths("").GlobalPath
			} else {
				if profileID == "" {
					return fmt.Errorf("--profile is required when not using --global")
				}
				targetBase = skills.NewSkillPaths(profileID).ProfilePath
			}

			targetPath := filepath.Join(targetBase, skill.Name)

			// Check if already exists
			if _, err := os.Stat(targetPath); err == nil {
				return fmt.Errorf("skill '%s' already exists at %s", skill.Name, targetPath)
			}

			// Create target directory
			if err := os.MkdirAll(targetBase, 0755); err != nil {
				return fmt.Errorf("failed to create target directory: %w", err)
			}

			// Copy skill directory
			if err := copyDir(sourcePath, targetPath); err != nil {
				return fmt.Errorf("failed to copy skill: %w", err)
			}

			fmt.Printf("✓ Installed skill '%s' to %s\n", skill.Name, targetPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Install to global skills directory")
	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID to install skill for")

	return cmd
}

// newValidateCmd creates the 'skill validate' command
func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <path>",
		Short: "Validate a skill",
		Long:  `Validate that a skill directory follows the Agent Skills specification.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillPath := args[0]

			skill, err := skills.ParseSkill(skillPath)
			if err != nil {
				fmt.Printf("✗ Invalid skill: %v\n", err)
				return err
			}

			fmt.Println("✓ Valid Agent Skill")
			fmt.Printf("  Name:        %s\n", skill.Name)
			fmt.Printf("  Description: %s\n", skill.Description)
			return nil
		},
	}

	return cmd
}

// newRunCmd creates the 'skill run' command
func newRunCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "run <skill-name> -- <script> [args...]",
		Short: "Run a skill script",
		Long:  `Execute a script from a skill.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement skill script execution
			// This will integrate with the existing action executor and isolation system
			return fmt.Errorf("not yet implemented")
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID")

	return cmd
}

// newStatsCmd creates the 'skill stats' command
func newStatsCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show skill usage statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load telemetry
			cfg := skills.DefaultConfig()
			telemetry, err := skills.NewTelemetry(&cfg.Telemetry)
			if err != nil {
				return fmt.Errorf("failed to initialize telemetry: %w", err)
			}

			// Get stats
			stats, err := telemetry.GetStats(profileID, time.Time{})
			if err != nil {
				return fmt.Errorf("failed to get stats: %w", err)
			}

			if stats.TotalInvocations == 0 {
				fmt.Println("No skill usage recorded yet.")
				return nil
			}

			fmt.Printf("Total Invocations: %d\n", stats.TotalInvocations)
			fmt.Printf("Total Completions: %d\n", stats.TotalCompletions)
			fmt.Printf("Total Failures:    %d\n", stats.TotalFailures)
			fmt.Println()

			fmt.Println("By Skill:")
			for skillName, skillStats := range stats.BySkill {
				fmt.Printf("  %s:\n", skillName)
				fmt.Printf("    Invocations: %d\n", skillStats.Invocations)
				fmt.Printf("    Completions: %d\n", skillStats.Completions)
				fmt.Printf("    Failures:    %d\n", skillStats.Failures)
				fmt.Printf("    Success Rate: %.1f%%\n", skillStats.SuccessRate()*100)
				if skillStats.Completions > 0 {
					fmt.Printf("    Avg Duration: %.1fms\n", skillStats.AverageDurationMs())
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID")

	return cmd
}

// newSuggestCmd creates the 'skill suggest' command
func newSuggestCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "suggest",
		Short: "Suggest IDE skill paths to configure",
		Long:  `Detect IDE/TDE skill directories and suggest adding them to configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := skills.NewSkillPaths(profileID)
			suggestions := paths.SuggestIDEPaths()

			if len(suggestions) == 0 {
				fmt.Println("No IDE skill paths detected.")
				return nil
			}

			fmt.Println("Detected IDE skill paths:")
			fmt.Println()
			for _, path := range suggestions {
				fmt.Printf("  %s\n", path)
			}
			fmt.Println()
			fmt.Println("To add these paths, update ~/.config/aps/config.yaml:")
			fmt.Println()
			fmt.Println("skill_sources:")
			for _, path := range suggestions {
				fmt.Printf("  - %s\n", path)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID")

	return cmd
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, input, info.Mode())
	})
}
