// Package storage provides storage backend selection and implementations.
package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

const (
	// BackendSQLite selects SQLite-backed storage.
	BackendSQLite = "sqlite"
)

var _ Storage = (*sqlite.SQLiteStorage)(nil)

// NewFromConfig creates a storage backend based on configuration.
func NewFromConfig() (Storage, error) {
	config.Load()
	backend := config.Get("storage_backend", BackendSQLite)

	// Initialize telemetry retention configuration
	retentionDays := config.GetInt("telemetry_retention_days", 90)
	sqlite.SetRetentionDays(retentionDays)

	return NewForBackend(backend)
}

// NewForBackend creates a storage backend for the provided backend name.
func NewForBackend(backend string) (Storage, error) {
	switch backend {
	case BackendSQLite:
		dbPath := filepath.Join(GetStateDir(), "notifications.db")
		sqlite.SetTmuxClient(tmux.NewDefaultClient())
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize sqlite backend: %w", err)
		}

		// Enforce telemetry retention policy on startup
		// This cleanup happens silently to avoid disrupting user operations
		if _, err := sqliteStorage.EnforceRetentionPolicy(); err != nil {
			// Log warning but don't fail initialization
			// Telemetry is not critical to core functionality
			fmt.Fprintf(os.Stderr, "warning: telemetry retention policy enforcement failed: %v\n", err)
		}

		return sqliteStorage, nil
	default:
		return nil, fmt.Errorf("unknown storage backend '%s' (only 'sqlite' is supported)", backend)
	}
}
