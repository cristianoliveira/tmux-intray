// Package storage provides the storage interface for tmux-intray.
package storage

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
)

// File permission constants
const (
	// FileModeDir is the permission for directories (rwxr-xr-x)
	// Owner: read/write/execute, Group/others: read/execute
	FileModeDir os.FileMode = 0755
	// FileModeFile is the permission for data files (rw-r--r--)
	// Owner: read/write, Group/others: read only
	FileModeFile os.FileMode = 0644
)

var (
	stateDir    string
	initOnce    = &sync.Once{}
	initMu      sync.RWMutex
	initErr     error
	initialized bool
)

// Init initializes storage directories.
// Returns an error if initialization fails. Safe for concurrent calls.
func Init() error {
	start := time.Now()
	colors.StructuredDebug(\"storage\", \"init\", \"started\", nil, \"\", nil)
	var err error
	initOnce.Do(func() {
		// Load configuration
		config.Load()

		// Prefer environment variable directly (should match config.Load but ensure it works)
		stateDir = os.Getenv("TMUX_INTRAY_STATE_DIR")
		if stateDir == "" {
			stateDir = config.Get("state_dir", "")
		}
		colors.Debug("state_dir: " + stateDir)
		if stateDir == "" {
			err = fmt.Errorf("storage initialization failed: TMUX_INTRAY_STATE_DIR not configured")
			return
		}

		// Ensure directories exist
		if err = os.MkdirAll(stateDir, FileModeDir); err != nil {
			err = fmt.Errorf("failed to create state directory: %w", err)
			return
		}

		// Mark initialized only if all steps succeeded
		initMu.Lock()
		initialized = true
		initErr = nil
		initMu.Unlock()

		colors.Debug("storage initialized")
	})

	// Return any initialization error from first call
	if err != nil {
		colors.StructuredError("storage", "init", "failed", err, "", map[string]interface{}{"duration_seconds": time.Since(start).Seconds()})
		return err
	}

	// Check if there was an error from a previous initialization attempt
	initMu.RLock()
	err = initErr
	initMu.RUnlock()
	if err != nil {
		colors.StructuredError("storage", "init", "failed", err, "", map[string]interface{}{"duration_seconds": time.Since(start).Seconds()})
		return err
	}
	colors.StructuredDebug(\"storage\", \"init\", \"completed\", nil, \"\", map[string]interface{}{\"duration_seconds\": time.Since(start).Seconds()})
	return err
}

// GetStateDir returns the state directory path.
func GetStateDir() string {
	if stateDir != "" {
		return stateDir
	}
	if dir := os.Getenv("TMUX_INTRAY_STATE_DIR"); dir != "" {
		return dir
	}
	config.Load()
	return config.Get("state_dir", "")
}

// Reset resets the storage package state for testing.
func Reset() {
	initMu.Lock()
	defer initMu.Unlock()

	stateDir = ""
	initialized = false
	initErr = nil

	// Reset sync.Once by creating a new one
	// This is safe because Reset() should only be called in tests
	initOnce = &sync.Once{}

	// Also reset the default storage from storage.go
	defaultStorage = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}
