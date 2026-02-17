// Package colors provides color output utilities.
package colors

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Color constants
const (
	Red    = "\033[0;31m"
	Green  = "\033[0;32m"
	Yellow = "\033[1;33m"
	Blue   = "\033[0;34m"
	Cyan   = "\033[0;36m"
	Reset  = "\033[0m"
)

const checkmark = "✓"

// Logger defines the interface for structured logging.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

var (
	debugEnabled    = false
	inErrorHandling = false
	errorMutex      sync.RWMutex
	logger          Logger
	loggerMu        sync.RWMutex
)

func init() {
	if val := os.Getenv("TMUX_INTRAY_DEBUG"); val == "true" || val == "1" {
		debugEnabled = true
	}
}

// SetDebug enables or disables debug output.
func SetDebug(enabled bool) {
	debugEnabled = enabled
}

// SetLogger sets the structured logger to mirror console output.
func SetLogger(l Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = l
}

// errorFallback logs an error message without using colors to avoid recursion.
func errorFallback(msg string) {
	// Direct write to stderr, ignore errors
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}

// Error outputs an error message to stderr.
func Error(msgs ...string) {
	msg := strings.Join(msgs, " ")
	// Mirror to structured logger if set
	loggerMu.RLock()
	l := logger
	loggerMu.RUnlock()
	if l != nil {
		l.Error(msg)
	}
	_, err := fmt.Fprintf(os.Stderr, "%sError:%s %s%s\n", Red, Reset, msg, Reset)
	if err != nil {
		errorMutex.RLock()
		alreadyHandling := inErrorHandling
		errorMutex.RUnlock()

		if !alreadyHandling {
			errorMutex.Lock()
			inErrorHandling = true
			errorMutex.Unlock()

			defer func() {
				errorMutex.Lock()
				inErrorHandling = false
				errorMutex.Unlock()
			}()
			Warning("failed to print error message: " + err.Error())
		} else {
			errorFallback("Error: failed to print error message: " + err.Error())
		}
	}
}

// Success outputs a success message to stdout.
func Success(msgs ...string) {
	msg := strings.Join(msgs, " ")
	// Mirror to structured logger if set
	loggerMu.RLock()
	l := logger
	loggerMu.RUnlock()
	if l != nil {
		l.Info(msg, "type", "success")
	}
	_, err := fmt.Fprintf(os.Stdout, "%s%s%s %s%s\n", Green, checkmark, Reset, msg, Reset)
	if err != nil {
		errorMutex.RLock()
		alreadyHandling := inErrorHandling
		errorMutex.RUnlock()

		if !alreadyHandling {
			errorMutex.Lock()
			inErrorHandling = true
			errorMutex.Unlock()

			defer func() {
				errorMutex.Lock()
				inErrorHandling = false
				errorMutex.Unlock()
			}()
			Warning("failed to print success message: " + err.Error())
		} else {
			errorFallback("Warning: failed to print success message: " + err.Error())
		}
	}
}

// Warning outputs a warning message to stderr.
func Warning(msgs ...string) {
	msg := strings.Join(msgs, " ")
	// Mirror to structured logger if set
	loggerMu.RLock()
	l := logger
	loggerMu.RUnlock()
	if l != nil {
		l.Warn(msg)
	}
	_, err := fmt.Fprintf(os.Stderr, "%sWarning:%s %s%s\n", Yellow, Reset, msg, Reset)
	if err != nil {
		errorMutex.RLock()
		alreadyHandling := inErrorHandling
		errorMutex.RUnlock()

		if !alreadyHandling {
			errorMutex.Lock()
			inErrorHandling = true
			errorMutex.Unlock()

			defer func() {
				errorMutex.Lock()
				inErrorHandling = false
				errorMutex.Unlock()
			}()
			Error("failed to print warning message: " + err.Error())
		} else {
			errorFallback("Error: failed to print warning message: " + err.Error())
		}
	}
}

// Info outputs an informational message to stdout.
func Info(msgs ...string) {
	msg := strings.Join(msgs, " ")
	// Mirror to structured logger if set
	loggerMu.RLock()
	l := logger
	loggerMu.RUnlock()
	if l != nil {
		l.Info(msg)
	}
	_, err := fmt.Fprintf(os.Stdout, "%s%s%s\n", Blue, msg, Reset)
	if err != nil {
		errorMutex.RLock()
		alreadyHandling := inErrorHandling
		errorMutex.RUnlock()

		if !alreadyHandling {
			errorMutex.Lock()
			inErrorHandling = true
			errorMutex.Unlock()

			defer func() {
				errorMutex.Lock()
				inErrorHandling = false
				errorMutex.Unlock()
			}()
			Warning("failed to print info message: " + err.Error())
		} else {
			errorFallback("Warning: failed to print info message: " + err.Error())
		}
	}
}

// LogInfo outputs a log informational message to stderr.
func LogInfo(msgs ...string) {
	msg := strings.Join(msgs, " ")
	// Mirror to structured logger if set
	loggerMu.RLock()
	l := logger
	loggerMu.RUnlock()
	if l != nil {
		l.Info(msg)
	}
	_, err := fmt.Fprintf(os.Stderr, "%s%s%s\n", Blue, msg, Reset)
	if err != nil {
		errorMutex.RLock()
		alreadyHandling := inErrorHandling
		errorMutex.RUnlock()

		if !alreadyHandling {
			errorMutex.Lock()
			inErrorHandling = true
			errorMutex.Unlock()

			defer func() {
				errorMutex.Lock()
				inErrorHandling = false
				errorMutex.Unlock()
			}()
			Warning("failed to print log info message: " + err.Error())
		} else {
			errorFallback("Warning: failed to print log info message: " + err.Error())
		}
	}
}

// Debug outputs a debug message to stderr if debug is enabled.
func Debug(msgs ...string) {
	if !debugEnabled {
		return
	}
	msg := strings.Join(msgs, " ")
	// Mirror to structured logger if set
	loggerMu.RLock()
	l := logger
	loggerMu.RUnlock()
	if l != nil {
		l.Debug(msg)
	}
	_, err := fmt.Fprintf(os.Stderr, "%sDebug:%s %s%s\n", Cyan, Reset, msg, Reset)
	if err != nil {
		errorMutex.RLock()
		alreadyHandling := inErrorHandling
		errorMutex.RUnlock()

		if !alreadyHandling {
			errorMutex.Lock()
			inErrorHandling = true
			errorMutex.Unlock()

			defer func() {
				errorMutex.Lock()
				inErrorHandling = false
				errorMutex.Unlock()
			}()
			Warning("failed to print debug message: " + err.Error())
		} else {
			errorFallback("Warning: failed to print debug message: " + err.Error())
		}
	}
}
