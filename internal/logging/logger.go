// Package logging provides structured file logging for tmux-intray.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	clog "github.com/charmbracelet/log"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

// Logger is the structured logging interface.
type Logger interface {
	// Debug logs a debug message.
	Debug(msg string, args ...any)
	// Info logs an informational message.
	Info(msg string, args ...any)
	// Warn logs a warning message.
	Warn(msg string, args ...any)
	// Error logs an error message.
	Error(msg string, args ...any)
	// With returns a new logger with additional key-value pairs.
	With(args ...any) Logger
	// Shutdown flushes any buffered logs and releases resources.
	Shutdown() error
}

// loggerImpl is the charmbracelet/log based implementation.
type loggerImpl struct {
	mu       sync.RWMutex
	clogger  *clog.Logger
	file     *os.File
	config   Config
	redactor *redactor
	fields   map[string]any // base fields added via With
	path     string         // full path to the log file
}

// Init initializes a new Logger with the given configuration.
// If config.Enabled is false, returns a no-op logger.
// It creates the log directory, applies file rotation, opens the log file,
// and configures the underlying logger with JSON formatting.
func Init(cfg Config) (Logger, error) {
	if !cfg.Enabled {
		return noopLogger{}, nil
	}
	logDir, err := LogDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine log directory: %w", err)
	}
	// Perform rotation before creating new log file
	if err := rotate(logDir, cfg.MaxFiles); err != nil {
		// Non-fatal; log to stderr but continue
		fmt.Fprintf(os.Stderr, "log rotation failed: %v\n", err)
	}
	// Generate log file name
	fname := fmt.Sprintf("tmux-intray_%s_PID%d_%s.log",
		time.Now().Format("20060102_150405"),
		cfg.PID,
		strings.ReplaceAll(cfg.Command, " ", "_"))
	path := filepath.Join(logDir, fname)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	// Create charmbracelet logger with JSON formatter
	clogger := clog.NewWithOptions(f, clog.Options{
		ReportTimestamp: true,
		ReportCaller:    false, // we can enable if needed
		TimeFormat:      time.RFC3339Nano,
		Level:           parseLevel(cfg.Level),
	})
	clogger.SetFormatter(clog.JSONFormatter)
	// Add standard fields
	clogger = clogger.With("pid", cfg.PID, "command", cfg.Command)
	l := &loggerImpl{
		clogger:  clogger,
		file:     f,
		config:   cfg,
		redactor: newRedactor(),
		fields:   make(map[string]any),
		path:     path,
	}
	return l, nil
}

// parseLevel converts a string level to clog.Level.
func parseLevel(level string) clog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return clog.DebugLevel
	case "info":
		return clog.InfoLevel
	case "warn", "warning":
		return clog.WarnLevel
	case "error":
		return clog.ErrorLevel
	default:
		return clog.InfoLevel
	}
}

func (l *loggerImpl) Debug(msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.log(clog.DebugLevel, msg, args)
}

func (l *loggerImpl) Info(msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.log(clog.InfoLevel, msg, args)
}

func (l *loggerImpl) Warn(msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.log(clog.WarnLevel, msg, args)
}

func (l *loggerImpl) Error(msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.log(clog.ErrorLevel, msg, args)
}

// log writes a log entry with redaction applied to the key-value pairs.
func (l *loggerImpl) log(level clog.Level, msg string, args []any) {
	// Combine base fields with args
	allArgs := make([]any, 0, len(l.fields)*2+len(args))
	for k, v := range l.fields {
		allArgs = append(allArgs, k, v)
	}
	allArgs = append(allArgs, args...)
	// Redact sensitive values
	redacted := l.redactor.redact(allArgs)
	// Call underlying logger
	l.clogger.Log(level, msg, redacted...)
}

func (l *loggerImpl) With(args ...any) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Merge new fields into a copy of fields
	newFields := make(map[string]any, len(l.fields)+len(args)/2)
	for k, v := range l.fields {
		newFields[k] = v
	}
	// Process args in key-value pairs (expect even number)
	for i := 0; i+1 < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			newFields[key] = args[i+1]
		}
	}
	// Create a new loggerImpl that shares the same file and clogger but with merged fields
	return &loggerImpl{
		clogger:  l.clogger,
		file:     l.file,
		config:   l.config,
		redactor: l.redactor,
		fields:   newFields,
		path:     l.path,
	}
}

func (l *loggerImpl) Shutdown() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// filePath returns the full path to the log file.
func (l *loggerImpl) filePath() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.path
}

// noopLogger is a logger that discards all output.
type noopLogger struct{}

func (n noopLogger) Debug(msg string, args ...any) {}
func (n noopLogger) Info(msg string, args ...any)  {}
func (n noopLogger) Warn(msg string, args ...any)  {}
func (n noopLogger) Error(msg string, args ...any) {}
func (n noopLogger) With(args ...any) Logger       { return n }
func (n noopLogger) Shutdown() error               { return nil }

// Global logger instance (optional, for convenience)
var (
	globalLogger     Logger
	globalLoggerOnce sync.Once
	globalLoggerMu   sync.RWMutex
)

// InitGlobal initializes the global logger using configuration from the global config.
// It is safe to call multiple times; only the first call initializes the logger.
func InitGlobal() error {
	var err error
	globalLoggerOnce.Do(func() {
		cfg := FromGlobalConfig()
		globalLogger, err = Init(cfg)
	})
	if err == nil && globalLogger != nil {
		colors.SetLogger(globalLogger)
		if path := CurrentLogFile(); path != "" {
			colors.Info("Logging to file:", path)
		}
	}
	return err
}

// GetGlobal returns the global logger, or a no-op logger if not initialized.
func GetGlobal() Logger {
	globalLoggerMu.RLock()
	defer globalLoggerMu.RUnlock()
	if globalLogger == nil {
		return noopLogger{}
	}
	return globalLogger
}

// Debug logs a debug message using the global logger.
func Debug(msg string, args ...any) {
	GetGlobal().Debug(msg, args...)
}

// Info logs an info message using the global logger.
func Info(msg string, args ...any) {
	GetGlobal().Info(msg, args...)
}

// Warn logs a warning message using the global logger.
func Warn(msg string, args ...any) {
	GetGlobal().Warn(msg, args...)
}

// Error logs an error message using the global logger.
func Error(msg string, args ...any) {
	GetGlobal().Error(msg, args...)
}

// With returns a new global logger with additional key-value pairs.
func With(args ...any) Logger {
	return GetGlobal().With(args...)
}

// ShutdownGlobal shuts down the global logger.
func ShutdownGlobal() error {
	globalLoggerMu.Lock()
	defer globalLoggerMu.Unlock()
	if globalLogger != nil {
		return globalLogger.Shutdown()
	}
	return nil
}

// CurrentLogFile returns the path to the current log file if logging is enabled and a file logger is active.
// Returns empty string if logging is disabled or no file is being written.
func CurrentLogFile() string {
	globalLoggerMu.RLock()
	defer globalLoggerMu.RUnlock()
	if globalLogger == nil {
		return ""
	}
	if impl, ok := globalLogger.(*loggerImpl); ok {
		return impl.filePath()
	}
	return ""
}
