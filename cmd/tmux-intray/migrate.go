package main

import (
	"fmt"
	"path/filepath"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

type migrateClient interface {
	GetStateDir() string
	MigrateToSQLite(opts sqlite.MigrationOptions) (sqlite.MigrationStats, error)
	RollbackMigration(tsvPath, sqlitePath, backupPath string) error
}

var (
	migrateTSVPathFlag    string
	migrateSQLitePathFlag string
	migrateBackupPathFlag string
	migrateDryRunFlag     bool
	migrateRollbackFlag   bool
)

// NewMigrateCmd creates the migrate command with explicit dependencies.
func NewMigrateCmd(client migrateClient) *cobra.Command {
	if client == nil {
		panic("NewMigrateCmd: client dependency cannot be nil")
	}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate notifications from TSV to SQLite",
		Long: `Migrate notifications from TSV to SQLite safely.

The migration validates each TSV row, skips malformed rows with warnings,
creates a backup before writing, and imports inside a single transaction.

Use --dry-run to preview migration statistics without creating backups or
writing to SQLite.

Use --rollback to restore the TSV file from backup and remove the SQLite DB.`,
		RunE: func(c *cobra.Command, args []string) error {
			stateDir := client.GetStateDir()
			if stateDir == "" {
				return fmt.Errorf("migrate: state directory is not configured")
			}

			tsvPath := migrateTSVPathFlag
			if tsvPath == "" {
				tsvPath = filepath.Join(stateDir, "notifications.tsv")
			}

			sqlitePath := migrateSQLitePathFlag
			if sqlitePath == "" {
				sqlitePath = filepath.Join(stateDir, "notifications.db")
			}

			backupPath := migrateBackupPathFlag
			if backupPath == "" {
				backupPath = tsvPath + ".sqlite-migration.bak"
			}

			if migrateRollbackFlag {
				if migrateDryRunFlag {
					return fmt.Errorf("migrate: --dry-run cannot be combined with --rollback")
				}
				if err := client.RollbackMigration(tsvPath, sqlitePath, backupPath); err != nil {
					return err
				}
				c.Printf("rollback completed: restored %s and removed %s\n", tsvPath, sqlitePath)
				return nil
			}

			stats, err := client.MigrateToSQLite(sqlite.MigrationOptions{
				TSVPath:    tsvPath,
				SQLitePath: sqlitePath,
				BackupPath: backupPath,
				DryRun:     migrateDryRunFlag,
			})
			if err != nil {
				return err
			}

			c.Printf("migration completed\n")
			c.Printf("total=%d migrated=%d skipped=%d failed=%d duplicates=%d\n", stats.TotalRows, stats.MigratedRows, stats.SkippedRows, stats.FailedRows, stats.DuplicateRows)
			if stats.BackupCreated {
				c.Printf("backup=%s\n", stats.BackupPath)
			}
			for _, warning := range stats.Warnings {
				c.Printf("warning: %s\n", warning)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&migrateTSVPathFlag, "tsv-path", "", "Path to source notifications TSV file")
	cmd.Flags().StringVar(&migrateSQLitePathFlag, "sqlite-path", "", "Path to destination SQLite database")
	cmd.Flags().StringVar(&migrateBackupPathFlag, "backup-path", "", "Path for TSV backup (default: <tsv-path>.sqlite-migration.bak)")
	cmd.Flags().BoolVar(&migrateDryRunFlag, "dry-run", false, "Validate and report migration without writing files")
	cmd.Flags().BoolVar(&migrateRollbackFlag, "rollback", false, "Restore TSV from backup and remove SQLite database")

	return cmd
}

// defaultMigrateClient is the default implementation using sqlite and storage packages.
type defaultMigrateClient struct{}

func (d *defaultMigrateClient) GetStateDir() string {
	return storage.GetStateDir()
}

func (d *defaultMigrateClient) MigrateToSQLite(opts sqlite.MigrationOptions) (sqlite.MigrationStats, error) {
	return sqlite.MigrateTSVToSQLite(opts)
}

func (d *defaultMigrateClient) RollbackMigration(tsvPath, sqlitePath, backupPath string) error {
	return sqlite.RollbackTSVMigration(tsvPath, sqlitePath, backupPath)
}

// migrateCmd represents the migrate command
var migrateCmd = NewMigrateCmd(&defaultMigrateClient{})

func init() {
	cmd.RootCmd.AddCommand(migrateCmd)
}
