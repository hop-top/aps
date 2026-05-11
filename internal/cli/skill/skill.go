package skill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/skills"
	"hop.top/aps/internal/styles"
	"hop.top/kit/go/console/output"
)

// skillSummaryRow is the table row shape for `aps skill list` (T-0440).
//
// Description is truncated to keep the table scannable; JSON/YAML
// formats keep the full string by reading from the raw skill instead.
// Scripts is the count of files under <skill>/scripts/ — gives quick
// signal for which skills carry executable runners vs. instructions.
type skillSummaryRow struct {
	Name        string `table:"NAME,priority=10"        json:"name"        yaml:"name"`
	Description string `table:"DESCRIPTION,priority=8"  json:"description" yaml:"description"`
	Source      string `table:"SOURCE,priority=7"       json:"source"      yaml:"source"`
	Profile     string `table:"PROFILE,priority=6"      json:"profile"     yaml:"profile"`
	Scripts     int    `table:"SCRIPTS,priority=4"      json:"scripts"     yaml:"scripts"`
}

const skillDescriptionWidth = 60

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

// newListCmd creates the 'skill list' command.
//
// T-0440 audit: the pre-uplift command exposed --profile and --verbose
// only. --verbose is dropped — kit/output's table renderer carries
// column priorities so narrow terminals already drop low-value
// columns; the JSON/YAML formats expose every field. Per the T-0427
// convention the surviving filters are --profile (inherits from the
// --profile global) and --source (set membership over the source
// label: Profile / Global / User / Claude Code / Cursor / …).
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Long:  `List all skills available in configured skill directories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID := globals.Profile()
			sourceFilter, _ := cmd.Flags().GetString("source")

			cfg := skills.DefaultConfig()
			registry := skills.NewRegistry(profileID, cfg.SkillSources, cfg.AutoDetectIDEPaths)
			if err := registry.Discover(); err != nil {
				return fmt.Errorf("failed to discover skills: %w", err)
			}

			rows := buildSkillRows(registry, profileID)
			pred := listing.All(
				listing.MatchString(func(r skillSummaryRow) string { return r.Source }, sourceFilter),
			)
			rows = listing.Filter(rows, pred)

			format := globals.Format()
			if len(rows) == 0 && (format == "" || format == "table") {
				fmt.Println(styles.Dim.Render("No skills found."))
				fmt.Println()
				fmt.Println(styles.Dim.Render("To install skills:"))
				fmt.Println(styles.Dim.Render("  aps skill install <path> [--global]"))
				return nil
			}

			return listing.RenderList(os.Stdout, format, rows)
		},
	}

	cmd.Flags().String("source", "",
		"Filter by source label (Profile, Global, User, Claude Code, Cursor, Zed, VS Code, Windsurf)")
	return cmd
}

// buildSkillRows projects the registry's discovered skills into table
// rows. Description is right-truncated to skillDescriptionWidth so a
// 120-col terminal still fits Source + Profile + Scripts on one line;
// JSON/YAML callers see the full description (kit/output uses the json
// tag, not the truncated table cell).
func buildSkillRows(registry *skills.Registry, profileID string) []skillSummaryRow {
	all := registry.List()
	rows := make([]skillSummaryRow, 0, len(all))
	for _, s := range all {
		desc := s.Description
		if n := skillDescriptionWidth; len(desc) > n {
			desc = desc[:n-1] + "…"
		}
		scripts, _ := s.ListScripts()
		rows = append(rows, skillSummaryRow{
			Name:        s.Name,
			Description: desc,
			Source:      registry.SourceLabel(s.SourcePath),
			Profile:     profileID,
			Scripts:     len(scripts),
		})
	}
	return rows
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
			return runSkillScript(cmd.Context(), profileID, args, cmd.ArgsLenAtDash(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID")

	return cmd
}

func runSkillScript(ctx context.Context, profileID string, args []string, dashIdx int, stdin io.Reader, stdout, stderr io.Writer) error {
	if dashIdx == -1 {
		return fmt.Errorf("missing '--' separator\nUsage: aps skill run <skill-name> -- <script> [args...]")
	}
	if dashIdx != 1 {
		return fmt.Errorf("expected exactly one skill name before '--'")
	}

	skillName := args[0]
	commandArgs := args[dashIdx:]
	if len(commandArgs) == 0 {
		return fmt.Errorf("no script specified")
	}

	scriptName := commandArgs[0]
	if err := validateSkillScriptName(scriptName); err != nil {
		return err
	}

	cfg := skills.DefaultConfig()
	registry := skills.NewRegistry(profileID, cfg.SkillSources, cfg.AutoDetectIDEPaths)
	if err := registry.Discover(); err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	skill, found := registry.Get(skillName)
	if !found {
		return fmt.Errorf("skill %q not found: %w", skillName, os.ErrNotExist)
	}
	if !skill.HasScript(scriptName) {
		return fmt.Errorf("script %q not found in skill %q: %w", scriptName, skillName, os.ErrNotExist)
	}

	scriptPath, err := checkedSkillScriptPath(skill, scriptName)
	if err != nil {
		return err
	}

	start := time.Now()
	telemetry := newSkillTelemetry(cfg)
	if telemetry != nil {
		_ = telemetry.TrackInvocation(skillName, profileID, "", "cli", "process")
	}

	err = execSkillScript(ctx, skill, scriptName, scriptPath, commandArgs[1:], profileID, stdin, stdout, stderr)
	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		if telemetry != nil {
			_ = telemetry.TrackFailure(skillName, profileID, "", scriptName, durationMs, sanitizedSkillScriptError(err))
		}
		return skillScriptRunError(skillName, scriptName, err)
	}

	if telemetry != nil {
		_ = telemetry.TrackCompletion(skillName, profileID, "", scriptName, durationMs, nil)
	}
	return nil
}

func validateSkillScriptName(scriptName string) error {
	switch {
	case scriptName == "":
		return fmt.Errorf("script name is required")
	case filepath.IsAbs(scriptName):
		return fmt.Errorf("script %q must be a filename under scripts/", scriptName)
	case filepath.Clean(scriptName) != scriptName:
		return fmt.Errorf("script %q must be a filename under scripts/", scriptName)
	case strings.Contains(scriptName, "/") || strings.Contains(scriptName, `\`):
		return fmt.Errorf("script %q must be a filename under scripts/", scriptName)
	}
	return nil
}

func checkedSkillScriptPath(skill *skills.Skill, scriptName string) (string, error) {
	scriptsDir := filepath.Join(skill.BasePath, "scripts")
	scriptPath := skill.GetScriptPath(scriptName)
	info, err := os.Lstat(scriptPath)
	if err != nil {
		return "", fmt.Errorf("script %q not found in skill %q: %w", scriptName, skill.Name, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("script %q in skill %q is a directory", scriptName, skill.Name)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("script %q in skill %q must not be a symlink", scriptName, skill.Name)
	}

	absDir, err := filepath.Abs(scriptsDir)
	if err != nil {
		return "", fmt.Errorf("resolve scripts directory: %w", err)
	}
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return "", fmt.Errorf("resolve script path: %w", err)
	}
	if filepath.Dir(absPath) != absDir {
		return "", fmt.Errorf("script %q must resolve under scripts/", scriptName)
	}
	return absPath, nil
}

func execSkillScript(ctx context.Context, skill *skills.Skill, scriptName, scriptPath string, args []string, profileID string, stdin io.Reader, stdout, stderr io.Writer) error {
	command, commandArgs := skillScriptCommand(scriptName, scriptPath, args)
	cmd := exec.CommandContext(ctx, command, commandArgs...)
	cmd.Dir = skill.BasePath
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(),
		"APS_PROFILE_ID="+profileID,
		"APS_SKILL_NAME="+skill.Name,
		"APS_SKILL_DIR="+skill.BasePath,
		"APS_SKILL_SCRIPT="+scriptName,
	)
	return cmd.Run()
}

func skillScriptCommand(scriptName, scriptPath string, args []string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(scriptName))
	switch ext {
	case ".sh":
		return "sh", append([]string{scriptPath}, args...)
	case ".bash":
		return "bash", append([]string{scriptPath}, args...)
	case ".py":
		return "python3", append([]string{scriptPath}, args...)
	case ".js", ".cjs", ".mjs":
		return "node", append([]string{scriptPath}, args...)
	default:
		return scriptPath, args
	}
}

func skillScriptRunError(skillName, scriptName string, err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &output.Error{
			Code:     output.CodeGeneric,
			Message:  fmt.Sprintf("skill %q script %q failed: %v", skillName, scriptName, err),
			ExitCode: exitErr.ExitCode(),
		}
	}
	return fmt.Errorf("run skill %q script %q: %w", skillName, scriptName, err)
}

func sanitizedSkillScriptError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("script exited with code %d", exitErr.ExitCode())
	}
	return err
}

func newSkillTelemetry(cfg *skills.Config) *skills.Telemetry {
	telemetry, err := skills.NewTelemetry(&cfg.Telemetry)
	if err != nil {
		return nil
	}
	return telemetry
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
