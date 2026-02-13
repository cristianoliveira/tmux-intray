// Package storage provides storage backend selection and implementations.
package storage

import (
	"fmt"
	"path/filepath"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
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
	return NewForBackend(backend)
}

// NewForBackend creates a storage backend for the provided backend name.
func NewForBackend(backend string) (Storage, error) {
	switch backend {
	case BackendSQLite:
		dbPath := filepath.Join(GetStateDir(), "notifications.db")
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize sqlite backend: %w", err)
		}
		return sqliteStorage, nil
	default:
		return nil, fmt.Errorf("unknown storage backend '%s' (only 'sqlite' is supported)", backend)
	}
}
