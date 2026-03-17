// Package logging provides structured logging helpers.
package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
)

var (
	debugEnabled             atomic.Bool
	structuredMu             sync.Mutex
	structuredLoggingEnabled atomic.Bool
)

func init() {
	if val := os.Getenv("TMUX_INTRAY_DEBUG"); val == "true" || val == "1" {
		debugEnabled.Store(true)
	}
	structuredLoggingEnabled.Store(true)
}

// SetDebug enables or disables debug output.
func SetDebug(enabled bool) {
	debugEnabled.Store(enabled)
}

// IsDebugEnabled reports whether debug output is enabled.
func IsDebugEnabled() bool {
	return debugEnabled.Load()
}

// DisableStructuredLogging disables structured logging output.
// This is useful for commands like TUI where JSON logs interfere with the display.
func DisableStructuredLogging() {
	structuredLoggingEnabled.Store(false)
}

// EnableStructuredLogging enables structured logging output.
func EnableStructuredLogging() {
	structuredLoggingEnabled.Store(true)
}

// StructuredLogLevel represents log level for structured logs.
type StructuredLogLevel string

const (
	LevelDebug StructuredLogLevel = "debug"
	LevelInfo  StructuredLogLevel = "info"
	LevelWarn  StructuredLogLevel = "warn"
	LevelError StructuredLogLevel = "error"
)

// StructuredLogEntry represents a structured log entry.
type StructuredLogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     StructuredLogLevel     `json:"level"`
	Component string                 `json:"component"`
	Action    string                 `json:"action"`
	Status    string                 `json:"status"`
	Error     string                 `json:"error,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// StructuredLog writes a structured log entry to stderr and optionally to file.
// Redaction of sensitive fields should be applied before calling this function.
func StructuredLog(level StructuredLogLevel, component, action, status string, err error, id string, fields map[string]interface{}) {
	if !IsDebugEnabled() {
		return
	}
	if !structuredLoggingEnabled.Load() {
		return
	}

	entry := StructuredLogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Component: component,
		Action:    action,
		Status:    status,
		ID:        id,
		Fields:    fields,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	data, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		errorFallback(fmt.Sprintf("failed to marshal structured log: %v", marshalErr))
		return
	}

	structuredMu.Lock()
	defer structuredMu.Unlock()

	// Always write to stderr (existing behavior)
	_, writeErr := fmt.Fprintf(os.Stderr, "%s\n", data)
	if writeErr != nil {
		errorFallback(fmt.Sprintf("failed to write structured log to stderr: %v", writeErr))
	}

	// Also write to file logger if enabled
	if IsEnabled() && GetLogger() != nil {
		// Convert StructuredLogLevel to charmbracelet/log.Level
		logLevel := structuredToCharmLevel(level)
		logger := GetLogger()
		logger.Log(logLevel, fmt.Sprintf("%s.%s", component, action), convertFields(fields))
	}
}

// structuredToCharmLevel converts StructuredLogLevel to charmbracelet/log.Level.
func structuredToCharmLevel(level StructuredLogLevel) log.Level {
	switch level {
	case LevelDebug:
		return log.DebugLevel
	case LevelInfo:
		return log.InfoLevel
	case LevelWarn:
		return log.WarnLevel
	case LevelError:
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

// convertFields converts map[string]interface{} to map[string]any.
func convertFields(fields map[string]interface{}) map[string]any {
	if fields == nil {
		return nil
	}
	result := make(map[string]any, len(fields))
	for k, v := range fields {
		result[k] = v
	}
	return result
}

// StructuredDebug logs a structured debug entry.
func StructuredDebug(component, action, status string, err error, id string, fields map[string]interface{}) {
	StructuredLog(LevelDebug, component, action, status, err, id, fields)
}

// StructuredInfo logs a structured info entry.
func StructuredInfo(component, action, status string, err error, id string, fields map[string]interface{}) {
	StructuredLog(LevelInfo, component, action, status, err, id, fields)
}

// StructuredWarn logs a structured warning entry.
func StructuredWarn(component, action, status string, err error, id string, fields map[string]interface{}) {
	StructuredLog(LevelWarn, component, action, status, err, id, fields)
}

// StructuredError logs a structured error entry.
func StructuredError(component, action, status string, err error, id string, fields map[string]interface{}) {
	StructuredLog(LevelError, component, action, status, err, id, fields)
}

// TODO: consider sharing errorFallback with colors package
func errorFallback(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}
