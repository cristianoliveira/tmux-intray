// Package storage provides storage backend selection and implementations.
package storage

import (
	"fmt"
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
