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
	// BackendDual selects dual-write storage (TSV + SQLite).
	BackendDual = "dual"
)

var _ Storage = (*sqlite.SQLiteStorage)(nil)
var _ Storage = (*DualWriter)(nil)

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
		dbPath := filepath.Join(GetStateDir(), "notifications.db")
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		if err != nil {
			colors.Warning(fmt.Sprintf("failed to initialize sqlite backend, falling back to tsv: %v", err))
			return NewFileStorage()
		}
		if err := autoMigrateTSVOnSQLiteOptIn(sqliteStorage, dbPath); err != nil {
			_ = sqliteStorage.Close()
			colors.Warning(fmt.Sprintf("failed to auto-migrate tsv data, falling back to tsv: %v", err))
			return NewFileStorage()
		}
		return sqliteStorage, nil
	case BackendDual:
		opts := DualWriterOptions{
			ReadBackend: config.Get("dual_read_backend", ReadBackendSQLite),
			VerifyOnly:  config.GetBool("dual_verify_only", false),
			SampleSize:  config.GetInt("dual_verify_sample_size", 25),
		}
		dualWriter, err := NewDualWriter(opts)
		if err != nil {
			colors.Warning(fmt.Sprintf("failed to initialize dual backend, falling back to tsv: %v", err))
			return NewFileStorage()
		}
		return dualWriter, nil
	default:
		colors.Warning(fmt.Sprintf("unknown storage backend '%s', falling back to tsv", backend))
		return NewFileStorage()
	}
}

func autoMigrateTSVOnSQLiteOptIn(sqliteStorage *sqlite.SQLiteStorage, dbPath string) error {
	tsvPath := filepath.Join(GetStateDir(), "notifications.tsv")

	hasTSVData, err := fileHasContent(tsvPath)
	if err != nil {
		return fmt.Errorf("inspect tsv data: %w", err)
	}
	if !hasTSVData {
		return nil
	}

	sqliteRows, err := sqliteStorage.ListNotifications("all", "", "", "", "", "", "", "")
	if err != nil {
		return fmt.Errorf("inspect sqlite data: %w", err)
	}
	if strings.TrimSpace(sqliteRows) != "" {
		return nil
	}

	if _, err := sqlite.MigrateTSVToSQLite(sqlite.MigrationOptions{TSVPath: tsvPath, SQLitePath: dbPath}); err != nil {
		return fmt.Errorf("migrate tsv to sqlite: %w", err)
	}

	return nil
}

func fileHasContent(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return strings.TrimSpace(string(data)) != "", nil
}
