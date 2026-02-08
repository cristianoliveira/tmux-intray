package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

// reset clears the global config state for testing.
func reset() {
	config = nil
	configMap = nil
}

func TestDefaultConfig(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	Load()

	// Check some default values.
	require.Equal(t, "1000", Get("max_notifications", ""))
	require.Equal(t, "30", Get("auto_cleanup_days", ""))
	require.Equal(t, "tsv", Get("storage_backend", ""))
	require.Equal(t, "default", Get("table_format", ""))
	require.Equal(t, "compact", Get("status_format", ""))
	require.Equal(t, "true", Get("status_enabled", ""))
	require.Equal(t, "false", Get("show_levels", ""))
	require.Equal(t, "true", Get("hooks_enabled", ""))
	require.Equal(t, "warn", Get("hooks_failure_mode", ""))
	require.Equal(t, "false", Get("hooks_async", ""))
	require.Equal(t, "30", Get("hooks_async_timeout", ""))
	require.Equal(t, "10", Get("max_hooks", ""))
	require.Equal(t, "tsv", Get("storage_backend", ""))
	// Directories should be non-empty.
	require.NotEmpty(t, Get("state_dir", ""))
	require.NotEmpty(t, Get("config_dir", ""))
	require.NotEmpty(t, Get("hooks_dir", ""))
}

func TestEnvironmentOverrides(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("TMUX_INTRAY_MAX_NOTIFICATIONS", "500")
	t.Setenv("TMUX_INTRAY_STATUS_ENABLED", "0")
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "SQLITE")
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	t.Setenv("TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT", "60")
	t.Setenv("TMUX_INTRAY_MAX_HOOKS", "5")

	Load()

	require.Equal(t, "500", Get("max_notifications", ""))
	require.Equal(t, "false", Get("status_enabled", ""))
	require.Equal(t, "sqlite", Get("storage_backend", ""))
	require.Equal(t, "ignore", Get("hooks_failure_mode", ""))
	require.Equal(t, "60", Get("hooks_async_timeout", ""))
	require.Equal(t, "5", Get("max_hooks", ""))
}

func TestConfigFileTOML(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	data := `
max_notifications = 200
status_enabled = false
storage_backend = "sqlite"
table_format = "minimal"
`
	err := os.WriteFile(configPath, []byte(data), 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	Load()

	require.Equal(t, "200", Get("max_notifications", ""))
	require.Equal(t, "false", Get("status_enabled", ""))
	require.Equal(t, "sqlite", Get("storage_backend", ""))
	require.Equal(t, "minimal", Get("table_format", ""))
}

func TestConfigFileJSON(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := map[string]interface{}{
		"max_notifications": 300,
		"auto_cleanup_days": 7,
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	Load()

	require.Equal(t, "300", Get("max_notifications", ""))
	require.Equal(t, "7", Get("auto_cleanup_days", ""))
}

func TestConfigFileYAML(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	data := `---
max_notifications: 400
status_format: detailed
`
	err := os.WriteFile(configPath, []byte(data), 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	Load()

	require.Equal(t, "400", Get("max_notifications", ""))
	require.Equal(t, "detailed", Get("status_format", ""))
}

func TestConfigFileTypeValidation(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := map[string]interface{}{
		"max_notifications":   []int{1, 2},
		"status_enabled":      map[string]interface{}{"value": true},
		"table_format":        "minimal",
		"hooks_async_timeout": 12,
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	Load()

	require.Equal(t, "1000", Get("max_notifications", ""))
	require.Equal(t, "true", Get("status_enabled", ""))
	require.Equal(t, "minimal", Get("table_format", ""))
	require.Equal(t, "12", Get("hooks_async_timeout", ""))
}

func TestValidation(t *testing.T) {
	// Invalid max_notifications (negative)
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("TMUX_INTRAY_MAX_NOTIFICATIONS", "-5")
	Load()
	// Should be reset to default (1000)
	require.Equal(t, "1000", Get("max_notifications", ""))

	// Invalid table_format
	reset()
	tmpDir2 := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir2)
	t.Setenv("TMUX_INTRAY_TABLE_FORMAT", "invalid")
	Load()
	require.Equal(t, "default", Get("table_format", ""))

	// Invalid hooks_failure_mode
	reset()
	tmpDir3 := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir3)
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "invalid")
	Load()
	require.Equal(t, "warn", Get("hooks_failure_mode", ""))

	// Invalid hooks_async_timeout
	reset()
	tmpDir4 := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir4)
	t.Setenv("TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT", "-10")
	Load()
	require.Equal(t, "30", Get("hooks_async_timeout", ""))

	// Invalid max_hooks
	reset()
	tmpDir5 := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir5)
	t.Setenv("TMUX_INTRAY_MAX_HOOKS", "0")
	Load()
	require.Equal(t, "10", Get("max_hooks", ""))

	// Invalid storage_backend
	reset()
	tmpDir6 := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir6)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "csv")
	Load()
	require.Equal(t, "tsv", Get("storage_backend", ""))
}

func TestGetIntGetBool(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("TMUX_INTRAY_MAX_NOTIFICATIONS", "123")
	t.Setenv("TMUX_INTRAY_STATUS_ENABLED", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT", "45")
	t.Setenv("TMUX_INTRAY_MAX_HOOKS", "7")
	Load()

	require.Equal(t, 123, GetInt("max_notifications", 0))
	require.Equal(t, true, GetBool("status_enabled", false))
	require.Equal(t, 45, GetInt("hooks_async_timeout", 0))
	require.Equal(t, 7, GetInt("max_hooks", 0))
	// Missing key returns default.
	require.Equal(t, 999, GetInt("missing_key", 999))
	require.Equal(t, true, GetBool("missing_key", true))
}

func TestSampleConfigCreation(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	// Ensure no config file exists.
	Load()
	// Should create sample config.
	samplePath := filepath.Join(tmpDir, "tmux-intray", "config.toml")
	require.FileExists(t, samplePath)
	// Load it and verify it's valid TOML.
	data, err := os.ReadFile(samplePath)
	require.NoError(t, err)
	var cfg map[string]interface{}
	err = toml.Unmarshal(data, &cfg)
	require.NoError(t, err)
	// Should contain expected keys.
	require.Contains(t, cfg, "max_notifications")
	require.Contains(t, cfg, "state_dir")
	require.Contains(t, cfg, "storage_backend")
}

func TestLoadWithoutConfigFile(t *testing.T) {
	reset()
	// Set config dir to empty temp dir so no config file exists.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	Load()
	// Should not crash, defaults should be present.
	require.Equal(t, "1000", Get("max_notifications", ""))
}

func TestGetWithMissingKey(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	Load()
	// Unknown key returns default.
	require.Equal(t, "mydefault", Get("nonexistent_key", "mydefault"))
}

// Test that environment overrides take precedence over config file.
func TestPriority(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	data := `max_notifications = 800`
	err := os.WriteFile(configPath, []byte(data), 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	t.Setenv("TMUX_INTRAY_MAX_NOTIFICATIONS", "500") // should override config file
	Load()

	require.Equal(t, "500", Get("max_notifications", ""))
}

// Test XDG directory defaults.
func TestXdgDefaults(t *testing.T) {
	reset()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// Unset XDG_* env vars.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	Load()

	expectedConfigDir := filepath.Join(tmpHome, ".config", "tmux-intray")
	expectedStateDir := filepath.Join(tmpHome, ".local", "state", "tmux-intray")
	require.Equal(t, expectedConfigDir, Get("config_dir", ""))
	require.Equal(t, expectedStateDir, Get("state_dir", ""))
	// hooks_dir should be config_dir/hooks
	require.Equal(t, filepath.Join(expectedConfigDir, "hooks"), Get("hooks_dir", ""))
}

// Test that a non-existent config file is handled gracefully and logged at debug level.
func TestConfigFileNotFound(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "does_not_exist.toml")

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", nonExistentPath)
	// Enable debug output to capture debug messages.
	colors.SetDebug(true)
	defer colors.SetDebug(false)

	// Capture stderr to verify debug logging.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Load()

	// Close writer and restore stderr.
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain debug message about file not found.
	require.Contains(t, output, "Debug:")
	require.Contains(t, output, "unable to read config file")
	require.Contains(t, output, nonExistentPath)

	// Defaults should still be loaded.
	require.Equal(t, "1000", Get("max_notifications", ""))
}

// Test that a malformed config file is handled gracefully and logged at warning level.
func TestConfigFileMalformed(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Write malformed TOML.
	err := os.WriteFile(configPath, []byte("invalid toml content [unclosed"), 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)

	// Capture stderr to verify warning logging.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Load()

	// Close writer and restore stderr.
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain warning message about parse error.
	require.Contains(t, output, "Warning:")
	require.Contains(t, output, "unable to parse config file")
	require.Contains(t, output, configPath)

	// Defaults should still be loaded.
	require.Equal(t, "1000", Get("max_notifications", ""))
}

// Test that malformed JSON is logged at warning level.
func TestConfigFileMalformedJSON(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write malformed JSON.
	err := os.WriteFile(configPath, []byte(`{"invalid": json}`), 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)

	// Capture stderr to verify warning logging.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Load()

	// Close writer and restore stderr.
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain warning message about parse error.
	require.Contains(t, output, "Warning:")
	require.Contains(t, output, "unable to parse config file")
	require.Contains(t, output, configPath)
}

// Test that malformed YAML is logged at warning level.
func TestConfigFileMalformedYAML(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write malformed YAML (invalid indentation).
	err := os.WriteFile(configPath, []byte("invalid:\n  - item1\n item2"), 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)

	// Capture stderr to verify warning logging.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Load()

	// Close writer and restore stderr.
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain warning message about parse error.
	require.Contains(t, output, "Warning:")
	require.Contains(t, output, "unable to parse config file")
	require.Contains(t, output, configPath)
}

// Test that read errors (permission denied) are logged at debug level.
func TestConfigFileReadError(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create file with no read permissions.
	err := os.WriteFile(configPath, []byte("max_notifications = 200"), 0000)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	// Enable debug output to capture debug messages.
	colors.SetDebug(true)
	defer colors.SetDebug(false)

	// Capture stderr to verify debug logging.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Load()

	// Close writer and restore stderr.
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain debug message about read error.
	require.Contains(t, output, "Debug:")
	require.Contains(t, output, "unable to read config file")
	require.Contains(t, output, configPath)

	// Defaults should still be loaded.
	require.Equal(t, "1000", Get("max_notifications", ""))
}

// Test that debug messages are not shown when TMUX_INTRAY_DEBUG is not set.
func TestConfigFileNotFoundNoDebug(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "does_not_exist.toml")

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", nonExistentPath)
	// Ensure debug is not enabled.
	t.Setenv("TMUX_INTRAY_DEBUG", "")

	// Capture stderr to verify no debug logging.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Load()

	// Close writer and restore stderr.
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should NOT contain debug message.
	require.NotContains(t, output, "Debug:")
	require.NotContains(t, output, "unable to read config file")

	// Defaults should still be loaded.
	require.Equal(t, "1000", Get("max_notifications", ""))
}
