// Package colors provides color output utilities.
package colors

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	structuredMu             sync.Mutex
	structuredLoggingEnabled atomic.Bool
)

func init() {
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

// DisableStructuredLogging disables structured logging output.
// This is useful for commands like TUI where JSON logs interfere with the display.
func DisableStructuredLogging() {
	structuredLoggingEnabled.Store(false)
}

// EnableStructuredLogging enables structured logging output.
func EnableStructuredLogging() {
	structuredLoggingEnabled.Store(true)
}

// StructuredLog writes a structured log entry to stderr.
// Redaction of sensitive fields should be applied before calling this function.
func StructuredLog(level StructuredLogLevel, component, action, status string, err error, id string, fields map[string]interface{}) {
	// Only output debug logs when debug mode is enabled
	if !debugEnabled {
		return
	}
	// Skip all structured logs if disabled
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
		// Fallback to simple log
		errorFallback(fmt.Sprintf("failed to marshal structured log: %v", marshalErr))
		return
	}

	structuredMu.Lock()
	defer structuredMu.Unlock()
	_, writeErr := fmt.Fprintf(os.Stderr, "%s\n", data)
	if writeErr != nil {
		errorFallback(fmt.Sprintf("failed to write structured log: %v", writeErr))
	}
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
