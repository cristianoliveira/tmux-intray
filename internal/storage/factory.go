// Package storage provides storage backend selection and implementations.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
)

const (
	// BackendTSV selects file-based TSV storage.
	BackendTSV = "tsv"
	// BackendSQLite selects SQLite-backed storage.
	BackendSQLite = "sqlite"
	// BackendDual selects dual-write TSV+SQLite storage for rollout verification.
	BackendDual = "dual"

	notificationsTSVFileName = "notifications.tsv"
	notificationsDBFileName  = "notifications.db"
	migrationBackupSuffix    = ".sqlite-migration.bak"
)

var _ Storage = (*sqlite.SQLiteStorage)(nil)

var migrateTSVToSQLite = sqlite.MigrateTSVToSQLite
var rollbackTSVMigration = sqlite.RollbackTSVMigration

// NewFromConfig creates a storage backend based on configuration.
func NewFromConfig() (Storage, error) {
	config.Load()
	backend := config.Get("storage_backend", BackendTSV)
	return NewForBackend(backend)
}

// NewForBackend creates a storage backend for the provided backend name.
func NewForBackend(backend string) (Storage, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", BackendTSV:
		return NewFileStorage()
	case BackendSQLite:
		stateDir := GetStateDir()
		dbPath := filepath.Join(stateDir, notificationsDBFileName)
		tsvPath := filepath.Join(stateDir, notificationsTSVFileName)

		if err := maybeMigrateTSVToSQLite(tsvPath, dbPath); err != nil {
			colors.Warning(fmt.Sprintf("sqlite migration failed, falling back to tsv: %v", err))
			return NewFileStorage()
		}

		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		if err != nil {
			colors.Warning(fmt.Sprintf("failed to initialize sqlite backend, falling back to tsv: %v", err))
			return NewFileStorage()
		}
		return sqliteStorage, nil
	case BackendDual:
		stateDir := GetStateDir()
		dbPath := filepath.Join(stateDir, notificationsDBFileName)
		tsvPath := filepath.Join(stateDir, notificationsTSVFileName)

		tsvStorage, err := NewFileStorage()
		if err != nil {
			return nil, err
		}

		if err := maybeMigrateTSVToSQLite(tsvPath, dbPath); err != nil {
			colors.Warning(fmt.Sprintf("sqlite migration failed for dual writer, using tsv only: %v", err))
			return tsvStorage, nil
		}

		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		if err != nil {
			colors.Warning(fmt.Sprintf("failed to initialize sqlite backend for dual writer, using tsv only: %v", err))
			return tsvStorage, nil
		}

		dualWriter, err := NewDualWriter(tsvStorage, sqliteStorage, DualWriterOptions{
			ReadFromSQLite:     true,
			VerifyEveryNWrites: defaultVerifyEveryNWrites,
		})
		if err != nil {
			colors.Warning(fmt.Sprintf("failed to initialize dual writer, using tsv only: %v", err))
			return tsvStorage, nil
		}

		return dualWriter, nil
	default:
		colors.Warning(fmt.Sprintf("unknown storage backend '%s', falling back to tsv", backend))
		return NewFileStorage()
	}
}

func maybeMigrateTSVToSQLite(tsvPath, sqlitePath string) error {
	dbExists, err := pathExists(sqlitePath)
	if err != nil {
		return fmt.Errorf("check sqlite database path: %w", err)
	}
	if dbExists {
		return nil
	}

	hasTSVData, err := fileHasContent(tsvPath)
	if err != nil {
		return fmt.Errorf("check tsv data: %w", err)
	}
	if !hasTSVData {
		return nil
	}

	colors.Info("Detected TSV notifications. Starting SQLite migration...")
	stats, migrateErr := migrateTSVToSQLite(sqlite.MigrationOptions{TSVPath: tsvPath, SQLitePath: sqlitePath})
	if migrateErr != nil {
		backupPath := tsvPath + migrationBackupSuffix
		if rollbackErr := rollbackTSVMigration(tsvPath, sqlitePath, backupPath); rollbackErr != nil {
			return fmt.Errorf("migrate tsv to sqlite: %w (rollback failed: %v)", migrateErr, rollbackErr)
		}
		return fmt.Errorf("migrate tsv to sqlite: %w", migrateErr)
	}

	colors.Success(fmt.Sprintf("SQLite migration complete: %d migrated, %d skipped", stats.MigratedRows, stats.SkippedRows))
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func fileHasContent(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("expected file but found directory: %s", path)
	}
	return info.Size() > 0, nil
}
