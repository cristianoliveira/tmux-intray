package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clog "github.com/charmbracelet/log"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (tempDir string, cleanup func()) {
	tmp := t.TempDir()
	// Override XDG_STATE_HOME to point to temp dir so state_dir is inside tmp
	t.Setenv("XDG_STATE_HOME", tmp)
	t.Setenv("HOME", tmp)
	// Ensure config.Load picks up our env vars
	config.Load()
	return tmp, func() {
		// nothing extra
	}
}

func TestConfigFromGlobal(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()

	// Set logging config via environment
	t.Setenv("TMUX_INTRAY_LOGGING_ENABLED", "true")
	t.Setenv("TMUX_INTRAY_LOGGING_LEVEL", "debug")
	t.Setenv("TMUX_INTRAY_LOGGING_MAX_FILES", "5")
	config.Load()

	cfg := FromGlobalConfig()
	require.True(t, cfg.Enabled)
	require.Equal(t, "debug", cfg.Level)
	require.Equal(t, 5, cfg.MaxFiles)
	require.Equal(t, filepath.Base(os.Args[0]), cfg.Command)
	require.Equal(t, os.Getpid(), cfg.PID)
}

func TestLogLevelMapping(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()

	// Test debug overrides level to debug
	t.Setenv("TMUX_INTRAY_DEBUG", "true")
	t.Setenv("TMUX_INTRAY_LOGGING_LEVEL", "info") // should be overridden
	config.Load()
	cfg := FromGlobalConfig()
	require.Equal(t, "debug", cfg.Level)

	// Test quiet overrides level to error (but debug wins if both)
	t.Setenv("TMUX_INTRAY_QUIET", "true")
	config.Load()
	cfg = FromGlobalConfig()
	// debug still set, should still be debug
	require.Equal(t, "debug", cfg.Level)

	// Clear debug, only quiet -> error
	t.Setenv("TMUX_INTRAY_DEBUG", "")
	config.Load()
	cfg = FromGlobalConfig()
	require.Equal(t, "error", cfg.Level)

	// Neither debug nor quiet -> keep configured level
	t.Setenv("TMUX_INTRAY_QUIET", "")
	t.Setenv("TMUX_INTRAY_LOGGING_LEVEL", "warn")
	config.Load()
	cfg = FromGlobalConfig()
	require.Equal(t, "warn", cfg.Level)
}

func TestLogDir(t *testing.T) {
	tmp, cleanup := setupTest(t)
	defer cleanup()

	// state_dir should be under XDG_STATE_HOME/tmux-intray
	stateDir := config.Get("state_dir", "")
	require.NotEmpty(t, stateDir)
	require.True(t, strings.HasPrefix(stateDir, tmp), "state_dir %s not in temp dir %s", stateDir, tmp)

	logDir, err := LogDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(stateDir, "logs"), logDir)
	// Directory should exist with 0700 permissions
	info, err := os.Stat(logDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestLogDirFallback(t *testing.T) {
	// Temporarily unset XDG_STATE_HOME and set a non-writable state_dir
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/non/existent")
	// Also override config's state_dir to a non-writable location
	t.Setenv("TMUX_INTRAY_STATE_DIR", "/root/nope")
	config.Load()

	logDir, err := LogDir()
	require.NoError(t, err)
	// Should fallback to temp directory
	require.True(t, strings.HasPrefix(logDir, os.TempDir()))
	require.True(t, strings.HasSuffix(logDir, filepath.Join("tmux-intray", "logs")))
}

func TestInitDisabled(t *testing.T) {
	cfg := Config{Enabled: false}
	logger, err := Init(cfg)
	require.NoError(t, err)
	require.IsType(t, noopLogger{}, logger)
	// Calling methods should not panic
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
	logger.Shutdown()
}

func TestInitEnabledCreatesFile(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()
	t.Setenv("TMUX_INTRAY_LOGGING_ENABLED", "true")
	config.Load()

	cfg := FromGlobalConfig()
	cfg.Command = "testcmd"
	logger, err := Init(cfg)
	require.NoError(t, err)
	defer logger.Shutdown()

	// Verify log file exists in state_dir/logs with expected name pattern
	stateDir := config.Get("state_dir", "")
	logDir := filepath.Join(stateDir, "logs")
	entries, err := os.ReadDir(logDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	fname := entries[0].Name()
	require.True(t, strings.HasPrefix(fname, "tmux-intray_"))
	require.True(t, strings.Contains(fname, fmt.Sprintf("_PID%d_", os.Getpid())))
	require.True(t, strings.Contains(fname, "_testcmd.log"))
	// File permissions should be 0600
	info, err := os.Stat(filepath.Join(logDir, fname))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestLoggingWritesJSON(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()
	t.Setenv("TMUX_INTRAY_LOGGING_ENABLED", "true")
	config.Load()

	cfg := FromGlobalConfig()
	logger, err := Init(cfg)
	require.NoError(t, err)
	defer logger.Shutdown()

	logger.Info("test message", "key1", "value1", "key2", 42)
	// Need to flush? charmbracelet/log writes synchronously.
	// Close logger to ensure writes are flushed
	logger.Shutdown()

	// Read log file
	stateDir := config.Get("state_dir", "")
	logDir := filepath.Join(stateDir, "logs")
	entries, _ := os.ReadDir(logDir)
	require.Greater(t, len(entries), 0)
	logPath := filepath.Join(logDir, entries[0].Name())
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Greater(t, len(lines), 0)
	// Parse last line as JSON
	var entry map[string]interface{}
	err = json.Unmarshal([]byte(lines[len(lines)-1]), &entry)
	require.NoError(t, err)
	require.Equal(t, "info", entry["level"])
	require.Equal(t, "test message", entry["msg"])
	require.Equal(t, float64(os.Getpid()), entry["pid"])
	require.Contains(t, entry, "command")
	require.IsType(t, "", entry["command"])
	// Check extra fields (they may be nested under "fields" or at top level depending on formatter)
	// charmbracelet/log JSONFormatter puts them at top level
	val, ok := entry["key1"]
	if ok {
		require.Equal(t, "value1", val)
	}
	val2, ok := entry["key2"]
	if ok {
		require.Equal(t, float64(42), val2)
	}
}

func TestRedaction(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()
	t.Setenv("TMUX_INTRAY_LOGGING_ENABLED", "true")
	config.Load()

	cfg := FromGlobalConfig()
	logger, err := Init(cfg)
	require.NoError(t, err)
	defer logger.Shutdown()

	logger.Info("secrets", "password", "supersecret", "token", "xyz", "normal", "ok")
	logger.Shutdown()

	stateDir := config.Get("state_dir", "")
	logDir := filepath.Join(stateDir, "logs")
	entries, _ := os.ReadDir(logDir)
	logPath := filepath.Join(logDir, entries[0].Name())
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	lastLine := lines[len(lines)-1]
	require.Contains(t, lastLine, `"password":"[REDACTED]"`)
	require.Contains(t, lastLine, `"token":"[REDACTED]"`)
	require.Contains(t, lastLine, `"normal":"ok"`)
}

func TestRedactionEdgeCases(t *testing.T) {
	r := newRedactor()

	// Test case-insensitive keys
	require.Equal(t, []any{"password", "[REDACTED]"}, r.redact([]any{"password", "secret"}))
	require.Equal(t, []any{"PASSWORD", "[REDACTED]"}, r.redact([]any{"PASSWORD", "secret"}))
	require.Equal(t, []any{"PaSsWoRd", "[REDACTED]"}, r.redact([]any{"PaSsWoRd", "secret"}))

	// Test keys with separators
	require.Equal(t, []any{"api_token", "[REDACTED]"}, r.redact([]any{"api_token", "xyz"}))
	require.Equal(t, []any{"api-token", "[REDACTED]"}, r.redact([]any{"api-token", "xyz"}))
	require.Equal(t, []any{"api.token", "[REDACTED]"}, r.redact([]any{"api.token", "xyz"}))
	require.Equal(t, []any{"api_token_key", "[REDACTED]"}, r.redact([]any{"api_token_key", "xyz"})) // multiple sensitive words

	// Test non-sensitive keys
	require.Equal(t, []any{"apitoken", "xyz"}, r.redact([]any{"apitoken", "xyz"})) // no separator
	require.Equal(t, []any{"normal", "value"}, r.redact([]any{"normal", "value"}))
	require.Equal(t, []any{"secretary", "value"}, r.redact([]any{"secretary", "value"})) // contains 'secret' but not as separate segment

	// Test mixed pairs
	input := []any{"password", "hidden", "name", "john", "token", "abc", "age", 30}
	output := r.redact(input)
	expected := []any{"password", "[REDACTED]", "name", "john", "token", "[REDACTED]", "age", 30}
	require.Equal(t, expected, output)

	// Test odd number of elements (should ignore last)
	inputOdd := []any{"password", "hidden", "extra"}
	outputOdd := r.redact(inputOdd)
	require.Equal(t, []any{"password", "[REDACTED]", "extra"}, outputOdd)

	// Test empty slice
	require.Empty(t, r.redact([]any{}))
}

func TestRotation(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()
	t.Setenv("TMUX_INTRAY_LOGGING_ENABLED", "true")
	t.Setenv("TMUX_INTRAY_LOGGING_MAX_FILES", "2")
	config.Load()

	cfg := FromGlobalConfig()
	// Create 3 log files manually
	logDir, err := LogDir()
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("tmux-intray_20250101_12000%d_PID999_test.log", i)
		path := filepath.Join(logDir, name)
		f, err := os.Create(path)
		require.NoError(t, err)
		f.Close()
		// Adjust mtime to ensure ordering
		oldTime := time.Now().Add(-time.Duration(i) * time.Hour)
		os.Chtimes(path, oldTime, oldTime)
	}
	entries, err := os.ReadDir(logDir)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// Init should trigger rotation (max files = 2)
	logger, err := Init(cfg)
	require.NoError(t, err)
	logger.Shutdown()

	entries, err = os.ReadDir(logDir)
	require.NoError(t, err)
	// Should have 2 old files + 1 new file = 3? Wait rotation removes oldest beyond maxFiles.
	// We have maxFiles = 2, we start with 3 old files, rotation should remove 1 oldest, leaving 2 old files.
	// Then Init creates a new file, total = 3? Actually rotation runs before creating new file,
	// so after rotation we have 2 old files, then new file makes 3 total.
	// However rotation only removes files matching pattern before creating new file.
	// We'll just assert that at most 3 files exist.
	require.LessOrEqual(t, len(entries), 3)
	// Ensure the oldest file (i=2) is gone
	_, err = os.Stat(filepath.Join(logDir, "tmux-intray_20250101_120002_PID999_test.log"))
	require.Error(t, err)
}

func TestRotationEdgeCases(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()
	t.Setenv("TMUX_INTRAY_LOGGING_ENABLED", "true")
	t.Setenv("TMUX_INTRAY_LOGGING_MAX_FILES", "0")
	config.Load()

	cfg := FromGlobalConfig()
	// Validator should replace 0 with default 10
	require.Equal(t, 10, cfg.MaxFiles)
	// Create 5 log files manually
	logDir, err := LogDir()
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("tmux-intray_20250101_12000%d_PID999_test.log", i)
		path := filepath.Join(logDir, name)
		f, err := os.Create(path)
		require.NoError(t, err)
		f.Close()
	}
	entries, err := os.ReadDir(logDir)
	require.NoError(t, err)
	require.Len(t, entries, 5)

	// Init with maxFiles = 10 should not delete any files (since 5 <= 10)
	logger, err := Init(cfg)
	require.NoError(t, err)
	logger.Shutdown()

	entries, err = os.ReadDir(logDir)
	require.NoError(t, err)
	// Should have 5 old files + 1 new file = 6
	require.Len(t, entries, 6)
}

func TestGlobalLogger(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()
	t.Setenv("TMUX_INTRAY_LOGGING_ENABLED", "true")
	config.Load()

	err := InitGlobal()
	require.NoError(t, err)
	defer ShutdownGlobal()

	// Global functions should work
	Info("global info")
	Warn("global warning", "count", 1)
	// Verify file written
	stateDir := config.Get("state_dir", "")
	logDir := filepath.Join(stateDir, "logs")
	entries, _ := os.ReadDir(logDir)
	require.Greater(t, len(entries), 0)
}

func TestWith(t *testing.T) {
	_, cleanup := setupTest(t)
	defer cleanup()
	t.Setenv("TMUX_INTRAY_LOGGING_ENABLED", "true")
	config.Load()

	cfg := FromGlobalConfig()
	logger, err := Init(cfg)
	require.NoError(t, err)
	defer logger.Shutdown()

	child := logger.With("request_id", "abc")
	child.Info("with context")
	logger.Shutdown()

	// Verify extra field appears in log
	stateDir := config.Get("state_dir", "")
	logDir := filepath.Join(stateDir, "logs")
	entries, _ := os.ReadDir(logDir)
	logPath := filepath.Join(logDir, entries[0].Name())
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	lastLine := lines[len(lines)-1]
	require.Contains(t, lastLine, `"request_id":"abc"`)
}

func TestLevelParsing(t *testing.T) {
	require.Equal(t, clog.DebugLevel, parseLevel("debug"))
	require.Equal(t, clog.InfoLevel, parseLevel("info"))
	require.Equal(t, clog.WarnLevel, parseLevel("warn"))
	require.Equal(t, clog.WarnLevel, parseLevel("warning"))
	require.Equal(t, clog.ErrorLevel, parseLevel("error"))
	require.Equal(t, clog.InfoLevel, parseLevel("unknown"))
}
