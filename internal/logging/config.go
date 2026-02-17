// Package logging provides structured file logging for tmux-intray.
package logging

import (
	"os"
	"path/filepath"

	"github.com/cristianoliveira/tmux-intray/internal/config"
)

// Config holds logging configuration.
type Config struct {
	// Enabled determines whether logging is active.
	Enabled bool
	// Level is the minimum log level to record.
	Level string
	// MaxFiles is the maximum number of log files to retain.
	MaxFiles int
	// Command is the name of the command being executed.
	Command string
	// PID is the process ID.
	PID int
}

// ConfigLogging is a temporary alias for backward compatibility.
// Deprecated: use Config instead.
type ConfigLogging = Config

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		Enabled:  false,
		Level:    "info",
		MaxFiles: 10,
		Command:  filepath.Base(os.Args[0]),
		PID:      os.Getpid(),
	}
}

// FromGlobalConfig creates a logging Config from the global configuration.
func FromGlobalConfig() Config {
	cfg := DefaultConfig()
	cfg.Enabled = config.GetBool("logging_enabled", false)
	cfg.Level = config.Get("logging_level", "info")
	cfg.MaxFiles = config.GetInt("logging_max_files", 10)
	return cfg
}

// LogDir returns the directory where log files should be stored.
// It uses the following priority:
// 1. {state_dir}/logs (if state_dir is accessible and writable)
// 2. {os.TempDir()}/tmux-intray/logs (fallback)
func LogDir() (string, error) {
	stateDir := config.Get("state_dir", "")
	if stateDir != "" {
		logDir := filepath.Join(stateDir, "logs")
		// Try to create directory with 0700 permissions
		if err := os.MkdirAll(logDir, 0700); err == nil {
			// Verify we can write to it
			if testFileWrite(logDir) {
				return logDir, nil
			}
		}
	}
	// Fallback to temporary directory
	tempBase := filepath.Join(os.TempDir(), "tmux-intray", "logs")
	if err := os.MkdirAll(tempBase, 0700); err != nil {
		return "", err
	}
	return tempBase, nil
}

// testFileWrite attempts to create a temporary file in dir to verify write permissions.
func testFileWrite(dir string) bool {
	tmp := filepath.Join(dir, ".write_test")
	f, err := os.Create(tmp)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(tmp)
	return true
}
