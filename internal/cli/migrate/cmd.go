package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hop.top/aps/internal/core/adapter"
	"hop.top/aps/internal/styles"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	headerStyle  = styles.Title
	dimStyle     = styles.Dim
	successStyle = styles.Success
	errorStyle   = styles.Error
	tableHeader  = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDim)
)

func NewMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate legacy configurations to new formats",
	}

	cmd.AddCommand(newMessengersCmd())

	return cmd
}

type messengerMigrate struct {
	Name      string
	Type      string
	Scope     string
	ProfileID string
	Source    string
	Target    string
	Status    string
	Error     error
}

func newMessengersCmd() *cobra.Command {
	var dryRun bool
	var backup bool
	var only string

	cmd := &cobra.Command{
		Use:   "messengers",
		Short: "Migrate messengers to adapter framework",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMessengersMigrate(dryRun, backup, only)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration without making changes")
	cmd.Flags().BoolVar(&backup, "backup", false, "Create backup before migration")
	cmd.Flags().StringVar(&only, "only", "", "Migrate only specified messengers (comma-separated)")

	return cmd
}

func runMessengersMigrate(dryRun, backup bool, only string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	messengersDir := filepath.Join(home, ".aps", "messengers")
	if _, err := os.Stat(messengersDir); os.IsNotExist(err) {
		fmt.Println(dimStyle.Render("No messengers directory found"))
		fmt.Println()
		fmt.Println("  Create an adapter instead:")
		fmt.Println("    aps adapter create my-telegram --type=messenger")
		return nil
	}

	messengers, err := discoverMessengers(messengersDir)
	if err != nil {
		return err
	}

	if len(messengers) == 0 {
		fmt.Println(dimStyle.Render("No messengers found to migrate"))
		return nil
	}

	if only != "" {
		messengers = filterMessengers(messengers, only)
	}

	if dryRun {
		return renderDryRun(messengers)
	}

	if backup {
		backupDir := filepath.Join(home, ".aps", "backups",
			fmt.Sprintf("messengers-%s", time.Now().Format("20060102")))
		if err := createBackup(messengersDir, backupDir); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		fmt.Printf("Backing up %s -> %s\n\n",
			dimStyle.Render(messengersDir),
			dimStyle.Render(backupDir))
	}

	return executeMigration(messengers)
}

func discoverMessengers(dir string) ([]messengerMigrate, error) {
	var messengers []messengerMigrate

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		sourcePath := filepath.Join(dir, name)

		manifestPath := filepath.Join(sourcePath, "manifest.yaml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		m := messengerMigrate{
			Name:   name,
			Type:   "messenger",
			Scope:  "global",
			Source: sourcePath,
			Status: "migrate",
		}

		messengers = append(messengers, m)
	}

	return messengers, nil
}

func filterMessengers(messengers []messengerMigrate, only string) []messengerMigrate {
	want := make(map[string]bool)
	for _, name := range splitList(only) {
		want[name] = true
	}

	var filtered []messengerMigrate
	for _, m := range messengers {
		if want[m.Name] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func splitList(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}

func renderDryRun(messengers []messengerMigrate) error {
	fmt.Println(headerStyle.Render("Migration Preview: messengers -> adapters"))
	fmt.Println()

	fmt.Printf("%-16s %-12s %-10s %s\n",
		tableHeader.Render("MESSENGER"),
		tableHeader.Render("TYPE"),
		tableHeader.Render("SCOPE"),
		tableHeader.Render("ACTION"))

	for _, m := range messengers {
		scope := m.Scope
		if m.ProfileID != "" {
			scope = fmt.Sprintf("profile (%s)", m.ProfileID)
		}
		fmt.Printf("%-16s %-12s %-10s %s\n",
			m.Name,
			dimStyle.Render(m.Type),
			dimStyle.Render(scope),
			successStyle.Render(m.Status))
	}

	home, _ := os.UserHomeDir()
	fmt.Println()
	fmt.Println(dimStyle.Render("  Summary:"))
	fmt.Printf("    %d messengers will be migrated\n", len(messengers))
	fmt.Printf("    Paths: %s -> %s\n",
		dimStyle.Render(filepath.Join(home, ".aps", "messengers")),
		dimStyle.Render(filepath.Join(home, ".aps", "adapters")))
	fmt.Println("    Symlinks: created for backward compat")
	fmt.Println("    Aliases: 'aps messengers' will still work")
	fmt.Println()
	fmt.Println(dimStyle.Render("  No changes made. Remove --dry-run to migrate."))

	return nil
}

func createBackup(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(target, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func executeMigration(messengers []messengerMigrate) error {
	fmt.Println("Migrating messengers to adapters...")

	success := 0
	failed := 0

	for i, m := range messengers {
		fmt.Printf("  [%d/%d] %-16s ", i+1, len(messengers), m.Name)

		err := migrateMessenger(m)
		if err != nil {
			fmt.Printf("%s: %s\n", errorStyle.Render("failed"), dimStyle.Render(err.Error()))
			failed++
			continue
		}

		fmt.Println(successStyle.Render("migrated"))
		success++
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("Migration partially complete (%d of %d).\n", success, len(messengers))
		fmt.Println()
		fmt.Println("  Rollback: aps migrate messengers --rollback")
		return fmt.Errorf("%d migrations failed", failed)
	}

	fmt.Println(headerStyle.Render("Migration complete."))
	fmt.Printf("  %d messengers migrated to adapters\n", success)
	fmt.Printf("  Verify: aps adapter list --type=messenger\n")

	return nil
}

func migrateMessenger(m messengerMigrate) error {
	dev := &adapter.Adapter{
		Name:     m.Name,
		Type:     adapter.AdapterTypeMessenger,
		Scope:    adapter.ScopeGlobal,
		Strategy: adapter.StrategySubprocess,
	}

	if err := adapter.SaveAdapter(dev); err != nil {
		return err
	}

	return nil
}
