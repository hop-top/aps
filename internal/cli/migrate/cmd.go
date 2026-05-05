package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core/adapter"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/core/xdg"
)

// T-0456 — `tableHeader` (lipgloss bold-dim style) was removed when
// the dry-run preview migrated to listing.RenderList. Header styling
// now flows from the active kit/cli theme via the styled table
// renderer on TTY writers.
var (
	headerStyle  = styles.Title
	dimStyle     = styles.Dim
	successStyle = styles.Success
	errorStyle   = styles.Error
)

// migrateDryRunRow is the table row shape for `aps migrate messengers
// --dry-run`. Higher-priority columns survive narrow terminals.
type migrateDryRunRow struct {
	Messenger string `table:"MESSENGER,priority=10" json:"messenger" yaml:"messenger"`
	Type      string `table:"TYPE,priority=8"       json:"type"      yaml:"type"`
	Scope     string `table:"SCOPE,priority=7"      json:"scope"     yaml:"scope"`
	Action    string `table:"ACTION,priority=9"     json:"action"    yaml:"action"`
}

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

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview migration without making changes")
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
		// T-0390 — route backup through kit/xdg.DataDir per
		// cli-conventions-with-kit §7.2.
		dataDir, err := xdg.DataDir("aps")
		if err != nil {
			return fmt.Errorf("xdg data dir: %w", err)
		}
		backupDir := filepath.Join(dataDir, "backups",
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

	rows := make([]migrateDryRunRow, 0, len(messengers))
	for _, m := range messengers {
		scope := m.Scope
		if m.ProfileID != "" {
			scope = fmt.Sprintf("profile (%s)", m.ProfileID)
		}
		rows = append(rows, migrateDryRunRow{
			Messenger: m.Name,
			Type:      m.Type,
			Scope:     scope,
			Action:    m.Status,
		})
	}
	if err := listing.RenderList(os.Stdout, output.Table, rows); err != nil {
		return err
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
