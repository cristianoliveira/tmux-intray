package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_Disabled(t *testing.T) {
	cfg := &LoggingConfig{
		Enabled: false,
	}

	err := Init(cfg)
	require.NoError(t, err)
	assert.False(t, IsEnabled())
}

func TestInit_Enabled(t *testing.T) {
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "state")
	logFile := filepath.Join(tempDir, "test.log")

	cfg := &LoggingConfig{
		Enabled:  true,
		Level:    "info",
		MaxFiles: 5,
		LogFile:  logFile,
		StateDir: stateDir,
	}

	err := Init(cfg)
	require.NoError(t, err)
	assert.True(t, IsEnabled())
	assert.NotNil(t, GetLogger())

	// Check log file exists
	_, err = os.Stat(logFile)
	require.NoError(t, err)

	// Cleanup
	err = Close()
	require.NoError(t, err)
}

func TestInit_InvalidLevel(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &LoggingConfig{
		Enabled:  true,
		Level:    "invalid",
		MaxFiles: 5,
		StateDir: tempDir,
	}

	err := Init(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestInit_LogFilePath(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &LoggingConfig{
		Enabled:  true,
		Level:    "info",
		MaxFiles: 5,
		StateDir: tempDir,
	}

	// Mock os.Args for testing command name
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"tmux-intray", "test-cmd"}

	err := Init(cfg)
	require.NoError(t, err)

	logPath := GetLogFilePath()
	assert.Contains(t, logPath, "tmux-intray_")
	assert.Contains(t, logPath, "test-cmd")
	assert.True(t, strings.HasSuffix(logPath, ".log"))

	// Cleanup
	Close()
}

func TestRedactFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]any
		expected map[string]any
	}{
		{
			name: "no sensitive fields",
			fields: map[string]any{
				"user":      "john",
				"timestamp": "2024-01-01",
			},
			expected: map[string]any{
				"user":      "john",
				"timestamp": "2024-01-01",
			},
		},
		{
			name: "sensitive fields",
			fields: map[string]any{
				"user":     "john",
				"password": "secret123",
				"api_key":  "abc123",
			},
			expected: map[string]any{
				"user":     "john",
				"password": "[REDACTED]",
				"api_key":  "[REDACTED]",
			},
		},
		{
			name: "case insensitive",
			fields: map[string]any{
				"user":      "john",
				"PASSWORD":  "secret123",
				"ApiSecret": "xyz",
			},
			expected: map[string]any{
				"user":      "john",
				"PASSWORD":  "[REDACTED]",
				"ApiSecret": "[REDACTED]",
			},
		},
		{
			name: "nested maps",
			fields: map[string]any{
				"user": "john",
				"config": map[string]any{
					"db_host":     "localhost",
					"db_password": "secret",
				},
			},
			expected: map[string]any{
				"user": "john",
				"config": map[string]any{
					"db_host":     "localhost",
					"db_password": "[REDACTED]",
				},
			},
		},
		{
			name: "slices",
			fields: map[string]any{
				"tags": []any{"tag1", "tag2"},
			},
			expected: map[string]any{
				"tags": []any{"tag1", "tag2"},
			},
		},
		{
			name: "nil fields",
			fields: map[string]any{
				"user": nil,
			},
			expected: map[string]any{
				"user": nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactFields(tt.fields)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactFields_NilInput(t *testing.T) {
	result := RedactFields(nil)
	assert.Nil(t, result)
}

func TestRedactEnv(t *testing.T) {
	// Set up test environment variables
	oldEnv := os.Environ()
	defer func() {
		// Restore original environment
		os.Clearenv()
		for _, e := range oldEnv {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	os.Setenv("USER", "john")
	os.Setenv("HOME", "/home/john")
	os.Setenv("API_SECRET", "supersecret")
	os.Setenv("DATABASE_PASSWORD", "dbpass")

	redacted := RedactEnv()

	// Find and check each variable
	var user, home, apiSecret, dbPassword string
	for _, e := range redacted {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}

		switch parts[0] {
		case "USER":
			user = parts[1]
		case "HOME":
			home = parts[1]
		case "API_SECRET":
			apiSecret = parts[1]
		case "DATABASE_PASSWORD":
			dbPassword = parts[1]
		}
	}

	assert.Equal(t, "john", user)
	assert.Equal(t, "/home/john", home)
	assert.Equal(t, "[REDACTED]", apiSecret)
	assert.Equal(t, "[REDACTED]", dbPassword)
}

func TestRedactArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "no sensitive args",
			args:     []string{"tmux-intray", "list", "--verbose"},
			expected: []string{"tmux-intray", "list", "--verbose"},
		},
		{
			name:     "flag with equals",
			args:     []string{"tmux-intray", "--password=secret123", "list"},
			expected: []string{"tmux-intray", "--password=[REDACTED]", "list"},
		},
		{
			name:     "flag-value pairs",
			args:     []string{"tmux-intray", "--token", "abc123", "--verbose"},
			expected: []string{"tmux-intray", "--token", "[REDACTED]", "--verbose"},
		},
		{
			name:     "mixed sensitive and non-sensitive",
			args:     []string{"app", "--user", "john", "--secret", "pass", "--debug"},
			expected: []string{"app", "--user", "john", "--secret", "[REDACTED]", "--debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			result := RedactArgs()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRotateLogs(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "logs")

	// Create some test log files
	now := time.Now()
	for i := 0; i < 15; i++ {
		fileName := filepath.Join(logDir, fmt.Sprintf("tmux-intray_%s.log", now.Add(time.Duration(i)*-time.Minute).Format("2006-01-02T15-04")))
		err := os.MkdirAll(filepath.Dir(fileName), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fileName, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Create a non-log file to ensure it's not deleted
	nonLogFile := filepath.Join(tempDir, "other.txt")
	err := os.WriteFile(nonLogFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Run rotation with maxFiles = 10
	err = rotateLogs(logDir, 10)
	require.NoError(t, err)

	// Count remaining log files
	entries, err := os.ReadDir(logDir)
	require.NoError(t, err)

	logCount := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "tmux-intray_") && strings.HasSuffix(entry.Name(), ".log") {
			logCount++
		}
	}

	assert.LessOrEqual(t, logCount, 10, "should have at most 10 log files after rotation")

	// Non-log file should still exist
	_, err = os.Stat(nonLogFile)
	require.NoError(t, err, "non-log file should not be deleted")
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		level   string
		wantErr bool
	}{
		{"debug", false},
		{"info", false},
		{"warn", false},
		{"warning", false},
		{"error", false},
		{"fatal", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			level, err := parseLevel(tt.level)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, level, -1)
			}
		})
	}
}

func TestDetermineLogPath_ExplicitFile(t *testing.T) {
	tempDir := t.TempDir()
	explicitFile := filepath.Join(tempDir, "custom.log")

	cfg := &LoggingConfig{
		LogFile:  explicitFile,
		StateDir: "/tmp",
	}

	path, err := determineLogPath(cfg)
	require.NoError(t, err)
	assert.Equal(t, explicitFile, path)
}

func TestDetermineLogPath_GeneratedFile(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &LoggingConfig{
		StateDir: tempDir,
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"tmux-intray", "test"}

	path, err := determineLogPath(cfg)
	require.NoError(t, err)

	assert.Contains(t, path, tempDir)
	assert.Contains(t, path, "tmux-intray_")
	assert.Contains(t, path, "test")
	assert.True(t, strings.HasSuffix(path, ".log"))
}

func TestLogging_Write(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	cfg := &LoggingConfig{
		Enabled:  true,
		Level:    "info",
		MaxFiles: 5,
		LogFile:  logFile,
		StateDir: tempDir,
	}

	err := Init(cfg)
	require.NoError(t, err)
	defer Close()

	// Get logger and write some logs
	logger := GetLogger()
	require.NotNil(t, logger)

	logger.Info("test", "action", "logging")
	logger.Debug("this should not appear", "reason", "debug level")

	// Read log file
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	// Check that info log is present
	assert.Contains(t, string(content), "test")
	assert.Contains(t, string(content), "logging")

	// Check that debug log is not present (level is info)
	assert.NotContains(t, string(content), "this should not appear")
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	cfg := &LoggingConfig{
		Enabled:  true,
		Level:    "info",
		MaxFiles: 5,
		LogFile:  logFile,
		StateDir: tempDir,
	}

	err := Init(cfg)
	require.NoError(t, err)

	err = Close()
	require.NoError(t, err)

	// After close, logger should still be marked as enabled but GetLogger might return nil
	// This is expected behavior - we don't reinitialize after close
	assert.True(t, IsEnabled())
}

func TestGetCommandName(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "with command",
			args:     []string{"tmux-intray", "list"},
			expected: "list",
		},
		{
			name:     "with complex command",
			args:     []string{"tmux-intray", "add", "--verbose"},
			expected: "add",
		},
		{
			name:     "without command",
			args:     []string{"tmux-intray"},
			expected: "cli",
		},
		{
			name:     "special characters",
			args:     []string{"tmux-intray", "test-cmd@123"},
			expected: "test-cmd_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			result := getCommandName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogStartup(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	cfg := &LoggingConfig{
		Enabled:  true,
		Level:    "info",
		MaxFiles: 5,
		LogFile:  logFile,
		StateDir: tempDir,
	}

	err := Init(cfg)
	require.NoError(t, err)
	defer Close()

	LogStartup()

	// Read log file
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	// Check startup info is logged
	logContent := string(content)
	assert.Contains(t, logContent, "startup")
	assert.Contains(t, logContent, "pid")
	assert.Contains(t, logContent, "version")
	assert.Contains(t, logContent, "os")
	assert.Contains(t, logContent, "arch")
	assert.Contains(t, logContent, "args")
}
