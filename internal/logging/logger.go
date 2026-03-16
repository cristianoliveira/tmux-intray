// Package logging provides structured logging with file output, rotation, and redaction.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

var (
	// logger is the global structured logger instance
	logger *log.Logger

	// mu protects access to logger initialization
	mu sync.RWMutex

	// config holds the current logging configuration
	config *LoggingConfig

	// configFileWriter is the file writer for logs
	configFileWriter *os.File

	// redactionPattern matches sensitive field names
	redactionPattern = regexp.MustCompile(`(?i)(secret|password|token|key|auth|credential)`)
)

// LoggingConfig holds the configuration for structured logging.
type LoggingConfig struct {
	Enabled  bool
	Level    string
	MaxFiles int
	LogFile  string
	StateDir string
}

// Init initializes the structured logging system.
// This should be called early in the application startup.
func Init(cfg *LoggingConfig) error {
	mu.Lock()
	defer mu.Unlock()

	config = cfg

	// If logging is disabled, don't initialize anything
	if !cfg.Enabled {
		return nil
	}

	// Parse log level
	logLevel, err := parseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
	}

	// Determine log file path
	logFilePath, err := determineLogPath(cfg)
	if err != nil {
		return fmt.Errorf("failed to determine log path: %w", err)
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Create log file
	configFileWriter, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
	}

	// Configure charmbracelet/log with JSON handler for file output
	logger = log.New(configFileWriter)
	logger.SetLevel(logLevel)
	logger.SetFormatter(log.JSONFormatter)
	logger.SetReportTimestamp(true)

	// Perform log rotation
	if err := rotateLogs(logDir, cfg.MaxFiles); err != nil {
		// Log rotation failure shouldn't prevent startup
		fmt.Fprintf(os.Stderr, "Warning: failed to rotate logs: %v\n", err)
	}

	// Print log file path to console (once per run)
	if _, err := fmt.Fprintf(os.Stdout, "Logging enabled: %s\n", logFilePath); err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}

// Close closes the log file and releases resources.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if configFileWriter != nil {
		if err := configFileWriter.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
		configFileWriter = nil
	}
	return nil
}

// GetLogger returns the global structured logger instance.
// Returns nil if logging is not enabled.
func GetLogger() *log.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return logger
}

// IsEnabled reports whether structured logging is enabled.
func IsEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return logger != nil
}

// parseLevel converts a string log level to log.Level.
func parseLevel(levelStr string) (log.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return log.DebugLevel, nil
	case "info":
		return log.InfoLevel, nil
	case "warn", "warning":
		return log.WarnLevel, nil
	case "error":
		return log.ErrorLevel, nil
	case "fatal":
		return log.FatalLevel, nil
	default:
		return log.InfoLevel, fmt.Errorf("unknown level: %s", levelStr)
	}
}

// determineLogPath determines the log file path based on configuration.
func determineLogPath(cfg *LoggingConfig) (string, error) {
	// If explicit log file is specified, use it
	if cfg.LogFile != "" {
		return cfg.LogFile, nil
	}

	// Generate per-run log file name
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	pid := os.Getpid()
	command := getCommandName()
	logFileName := fmt.Sprintf("tmux-intray_%s_PID%d_%s.log", timestamp, pid, command)

	// Use state_dir/logs as primary location
	logDir := filepath.Join(cfg.StateDir, "logs")

	// Ensure state_dir exists
	if err := os.MkdirAll(cfg.StateDir, 0755); err != nil {
		// Fallback to temp directory
		logDir = filepath.Join(os.TempDir(), "tmux-intray", "logs")
	}

	return filepath.Join(logDir, logFileName), nil
}

// getCommandName returns the name of the command being executed.
func getCommandName() string {
	if len(os.Args) > 1 {
		// Return first argument (command name)
		cmd := os.Args[1]
		// Sanitize command name for file system
		cmd = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
				return r
			}
			return '_'
		}, cmd)
		return cmd
	}
	return "cli"
}

// rotateLogs rotates log files to keep only the most recent N files.
func rotateLogs(logDir string, maxFiles int) error {
	if maxFiles <= 0 {
		return nil // No rotation
	}

	// Read all log files in the directory
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist yet, nothing to rotate
		}
		return err
	}

	// Filter and collect log files with their modification times
	var logFiles []logFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), "tmux-intray_") || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't get info for
		}

		logFiles = append(logFiles, logFileInfo{
			name:    entry.Name(),
			path:    filepath.Join(logDir, entry.Name()),
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.After(logFiles[j].modTime)
	})

	// Delete files beyond maxFiles limit (excluding the newest)
	for i := maxFiles; i < len(logFiles); i++ {
		// Skip the current log file (the one we just created/opened)
		if configFileWriter != nil {
			currentPath := configFileWriter.Name()
			if logFiles[i].path == currentPath {
				continue
			}
		}

		if err := os.Remove(logFiles[i].path); err != nil {
			// Log but don't fail rotation for a single file
			fmt.Fprintf(os.Stderr, "Warning: failed to delete old log file %s: %v\n", logFiles[i].path, err)
		}
	}

	return nil
}

type logFileInfo struct {
	name    string
	path    string
	modTime time.Time
}

// RedactFields redacts sensitive data from a map of fields.
func RedactFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}

	redacted := make(map[string]any, len(fields))
	for k, v := range fields {
		redacted[k] = redactValue(k, v)
	}
	return redacted
}

// redactValue redacts a value if its key contains sensitive keywords.
func redactValue(key string, value any) any {
	if redactionPattern.MatchString(key) {
		return "[REDACTED]"
	}

	// Recursively redact nested maps
	if m, ok := value.(map[string]any); ok {
		return RedactFields(m)
	}

	// Recursively redact values in slices
	if s, ok := value.([]any); ok {
		redactedSlice := make([]any, len(s))
		for i, item := range s {
			redactedSlice[i] = redactValue(key, item)
		}
		return redactedSlice
	}

	return value
}

// RedactEnv redacts sensitive environment variables from os.Environ().
func RedactEnv() []string {
	env := os.Environ()
	redacted := make([]string, 0, len(env))

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			redacted = append(redacted, e)
			continue
		}

		key := parts[0]

		if redactionPattern.MatchString(key) {
			redacted = append(redacted, fmt.Sprintf("%s=[REDACTED]", key))
		} else {
			redacted = append(redacted, e)
		}
	}

	return redacted
}

// RedactArgs redacts sensitive command-line arguments.
func RedactArgs() []string {
	args := os.Args
	redacted := make([]string, len(args))

	copy(redacted, args)

	// Check for flags that might have sensitive values
	hasEqualsValue := make(map[int]bool) // Track flags with equals syntax
	for i, arg := range redacted {
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 && redactionPattern.MatchString(parts[0]) {
				redacted[i] = fmt.Sprintf("%s=[REDACTED]", parts[0])
				hasEqualsValue[i] = true
			}
		}
	}

	// Check for flag-value pairs (e.g., --password secret)
	// Skip if previous flag already had equals syntax
	for i := 1; i < len(redacted); i++ {
		prevArg := redacted[i-1]
		if !hasEqualsValue[i-1] && strings.HasPrefix(prevArg, "-") && redactionPattern.MatchString(prevArg) {
			redacted[i] = "[REDACTED]"
		}
	}

	return redacted
}

// GetLogFilePath returns the current log file path, or empty string if logging is not enabled.
func GetLogFilePath() string {
	mu.RLock()
	defer mu.RUnlock()
	if configFileWriter != nil {
		return configFileWriter.Name()
	}
	return ""
}

// LogStartup logs startup information including environment and arguments (with redaction).
func LogStartup() {
	if !IsEnabled() {
		return
	}

	l := GetLogger()
	if l == nil {
		return
	}

	fields := map[string]any{
		"pid":        os.Getpid(),
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"args":       RedactArgs(),
		"env_count":  len(os.Environ()),
	}

	l.Info("startup", fields)
}

func getGoVersion() string {
	return runtime.Version()
}
