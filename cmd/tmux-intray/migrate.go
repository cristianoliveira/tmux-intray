package main

import (
	"fmt"
	"path/filepath"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

var (
	migrateTSVPathFlag    string
	migrateSQLitePathFlag string
	migrateBackupPathFlag string
	migrateDryRunFlag     bool
	migrateRollbackFlag   bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate notifications from TSV to SQLite",
	Long: `Migrate notifications from TSV to SQLite safely.

The migration validates each TSV row, skips malformed rows with warnings,
creates a backup before writing, and imports inside a single transaction.

Use --dry-run to preview migration statistics without creating backups or
writing to SQLite.

Use --rollback to restore the TSV file from backup and remove the SQLite DB.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stateDir := storage.GetStateDir()
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
			if err := sqlite.RollbackTSVMigration(tsvPath, sqlitePath, backupPath); err != nil {
				return err
			}
			cmd.Printf("rollback completed: restored %s and removed %s\n", tsvPath, sqlitePath)
			return nil
		}

		stats, err := sqlite.MigrateTSVToSQLite(sqlite.MigrationOptions{
			TSVPath:    tsvPath,
			SQLitePath: sqlitePath,
			BackupPath: backupPath,
			DryRun:     migrateDryRunFlag,
		})
		if err != nil {
			return err
		}

		cmd.Printf("migration completed\n")
		cmd.Printf("total=%d migrated=%d skipped=%d failed=%d duplicates=%d\n", stats.TotalRows, stats.MigratedRows, stats.SkippedRows, stats.FailedRows, stats.DuplicateRows)
		if stats.BackupCreated {
			cmd.Printf("backup=%s\n", stats.BackupPath)
		}
		for _, warning := range stats.Warnings {
			cmd.Printf("warning: %s\n", warning)
		}

		return nil
	},
}

func init() {
	cmd.RootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().StringVar(&migrateTSVPathFlag, "tsv-path", "", "Path to source notifications TSV file")
	migrateCmd.Flags().StringVar(&migrateSQLitePathFlag, "sqlite-path", "", "Path to destination SQLite database")
	migrateCmd.Flags().StringVar(&migrateBackupPathFlag, "backup-path", "", "Path for TSV backup (default: <tsv-path>.sqlite-migration.bak)")
	migrateCmd.Flags().BoolVar(&migrateDryRunFlag, "dry-run", false, "Validate and report migration without writing files")
	migrateCmd.Flags().BoolVar(&migrateRollbackFlag, "rollback", false, "Restore TSV from backup and remove SQLite database")
}
