package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
	require.Equal(t, "default", Get("table_format", ""))
	require.Equal(t, "compact", Get("status_format", ""))
	require.Equal(t, "true", Get("status_enabled", ""))
	require.Equal(t, "false", Get("show_levels", ""))
	require.Equal(t, "true", Get("hooks_enabled", ""))
	require.Equal(t, "warn", Get("hooks_failure_mode", ""))
	require.Equal(t, "false", Get("hooks_async", ""))
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
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	Load()

	require.Equal(t, "500", Get("max_notifications", ""))
	require.Equal(t, "false", Get("status_enabled", ""))
	require.Equal(t, "ignore", Get("hooks_failure_mode", ""))
}

func TestConfigFileTOML(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	data := `
max_notifications = 200
status_enabled = false
table_format = "minimal"
`
	err := os.WriteFile(configPath, []byte(data), 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	Load()

	require.Equal(t, "200", Get("max_notifications", ""))
	require.Equal(t, "false", Get("status_enabled", ""))
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
}

func TestGetIntGetBool(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("TMUX_INTRAY_MAX_NOTIFICATIONS", "123")
	t.Setenv("TMUX_INTRAY_STATUS_ENABLED", "1")
	Load()

	require.Equal(t, 123, GetInt("max_notifications", 0))
	require.Equal(t, true, GetBool("status_enabled", false))
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

// Test that environment overrides are overridden by config file.
func TestPriority(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	data := `max_notifications = 800`
	err := os.WriteFile(configPath, []byte(data), 0644)
	require.NoError(t, err)

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	t.Setenv("TMUX_INTRAY_MAX_NOTIFICATIONS", "500") // should be ignored because config file overrides
	Load()

	require.Equal(t, "800", Get("max_notifications", ""))
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
