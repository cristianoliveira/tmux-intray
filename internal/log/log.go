// Package log provides simple logging with configurable levels.
package log

import (
	"log"
	"os"
	"strings"
)

// Level represents a logging level.
type Level int

const (
	// LevelDebug is for debug messages.
	LevelDebug Level = iota
	// LevelInfo is for informational messages.
	LevelInfo
	// LevelWarn is for warning messages.
	LevelWarn
	// LevelError is for error messages.
	LevelError
	// LevelOff disables all logging.
	LevelOff
)

var (
	currentLevel Level = LevelInfo
	logger             = log.New(os.Stderr, "", log.LstdFlags)
)

// Init initializes logging with the specified level.
// Can be called with a level string from environment: debug, info, warn, error, off.
// Default is "info".
func Init(levelStr string) {
	if levelStr == "" {
		levelStr = os.Getenv("TMUX_INTRAY_LOG_LEVEL")
	}
	if levelStr == "" {
		levelStr = "info"
	}

	switch strings.ToLower(levelStr) {
	case "debug":
		currentLevel = LevelDebug
	case "info":
		currentLevel = LevelInfo
	case "warn":
		currentLevel = LevelWarn
	case "error":
		currentLevel = LevelError
	case "off":
		currentLevel = LevelOff
	default:
		currentLevel = LevelInfo
	}
}

// SetLevel sets the logging level programmatically.
func SetLevel(level Level) {
	currentLevel = level
}

// Debug logs a debug message.
func Debug(format string, args ...interface{}) {
	if currentLevel <= LevelDebug {
		logger.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs an info message.
func Info(format string, args ...interface{}) {
	if currentLevel <= LevelInfo {
		logger.Printf("[INFO] "+format, args...)
	}
}

// Warn logs a warning message.
func Warn(format string, args ...interface{}) {
	if currentLevel <= LevelWarn {
		logger.Printf("[WARN] "+format, args...)
	}
}

// Error logs an error message.
func Error(format string, args ...interface{}) {
	if currentLevel <= LevelError {
		logger.Printf("[ERROR] "+format, args...)
	}
}

// Errorf is an alias for Error (for compatibility).
func Errorf(format string, args ...interface{}) {
	Error(format, args...)
}
