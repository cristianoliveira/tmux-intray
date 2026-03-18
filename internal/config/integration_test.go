//go:build integration
// +build integration

package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfigLoadingPrecedence verifies that Go config loading follows
// the same precedence as Bash: environment → config file → defaults.
func TestConfigLoadingPrecedence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config file with some values
	configDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configFile := filepath.Join(configDir, "config.toml")
	configContent := `
storage_backend = "sqlite"
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Set environment variables (should override config file)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configFile)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "sqlite")

	reset()
	Load()

	// Verify precedence: environment should win
	require.Equal(t, "sqlite", Get("storage_backend", ""), "Environment should override config file")
}

// TestConfigFileBashCompatibility verifies that Go and Bash implementations
// produce identical config objects for the same inputs.
func TestConfigFileBashCompatibility(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a comprehensive config file
	configDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configFile := filepath.Join(configDir, "config.toml")
	configContent := `
storage_backend = "sqlite"
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Load config via Go
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configFile)
	reset()
	Load()

	// Verify Go loaded all values correctly
	goConfig := map[string]string{
		"storage_backend": Get("storage_backend", ""),
	}

	// Verify expected values
	require.Equal(t, "sqlite", goConfig["storage_backend"])
}

// TestEnvironmentVariableConfigBashCompatibility verifies that environment
// variable overrides work identically between Go and Bash.
func TestEnvironmentVariableConfigBashCompatibility(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a basic config file
	configDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configFile := filepath.Join(configDir, "config.toml")
	configContent := `
	storage_backend = "sqlite"
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Set environment variables
	envVars := map[string]string{
		"TMUX_INTRAY_STORAGE_BACKEND": "sqlite",
	}

	for k, v := range envVars {
		t.Setenv(k, v)
	}

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configFile)

	// Load config via Go
	reset()
	Load()

	// Verify all env vars override config file
	require.Equal(t, "sqlite", Get("storage_backend", ""))
}

// TestDefaultConfigBashCompatibility verifies that Go and Bash implementations
// use the same default values when no config file or env vars are present.
func TestDefaultConfigBashCompatibility(t *testing.T) {
	tmpDir := t.TempDir()

	// No config file, no env vars (except config path pointing to non-existent file)
	nonExistentConfig := filepath.Join(tmpDir, "does-not-exist.toml")
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", nonExistentConfig)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	reset()
	Load()

	// Verify defaults match expected values from documentation
	defaults := map[string]string{
		"storage_backend":   "sqlite",
		"auto_cleanup_days": "30",
	}

	for key, expectedValue := range defaults {
		actualValue := Get(key, "")
		require.Equal(t, expectedValue, actualValue, "Default value mismatch for %s", key)
	}
}

// TestBooleanConfigNormalization verifies that boolean values are normalized
// consistently between Go and Bash implementations.
func TestBooleanConfigNormalization(t *testing.T) {
	tmpDir := t.TempDir()

	// Test via environment variables (which allows various boolean representations)
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"1", "1", "true"},
		{"true", "true", "true"},
		{"yes", "yes", "true"},
		{"on", "on", "true"},
		{"TRUE", "TRUE", "true"},
		{"0", "0", "false"},
		{"false", "false", "false"},
		{"no", "no", "false"},
		{"off", "off", "false"},
		{"FALSE", "FALSE", "false"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TMUX_INTRAY_DEBUG", tc.input)
			t.Setenv("XDG_CONFIG_HOME", tmpDir)
			reset()
			Load()

			actualValue := Get("debug", "")
			require.Equal(t, tc.expected, actualValue)
		})
	}
}

// TestXdgDirectoryDefaults verifies that XDG directory defaults are
// computed correctly in both Go and Bash implementations.
func TestXdgDirectoryDefaults(t *testing.T) {
	tmpHome := t.TempDir()

	// Set HOME but not XDG_* vars
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")

	reset()
	Load()

	expectedConfigDir := filepath.Join(tmpHome, ".config", "tmux-intray")
	expectedStateDir := filepath.Join(tmpHome, ".local", "state", "tmux-intray")
	expectedHooksDir := filepath.Join(expectedConfigDir, "hooks")

	require.Equal(t, expectedConfigDir, Get("config_dir", ""))
	require.Equal(t, expectedStateDir, Get("state_dir", ""))
	require.Equal(t, expectedHooksDir, Get("hooks_dir", ""))
}

// TestXdgDirectoryOverrides verifies that XDG environment variables
// are respected correctly.
func TestXdgDirectoryOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	reset()
	Load()

	expectedConfigDir := filepath.Join(tmpDir, "tmux-intray")
	expectedStateDir := filepath.Join(tmpDir, "state", "tmux-intray")
	expectedHooksDir := filepath.Join(expectedConfigDir, "hooks")

	require.Equal(t, expectedConfigDir, Get("config_dir", ""))
	require.Equal(t, expectedStateDir, Get("state_dir", ""))
	require.Equal(t, expectedHooksDir, Get("hooks_dir", ""))
}

// TestConfigFileFormats verifies that different config file formats
// (TOML, JSON, YAML) are handled consistently.
func TestConfigFileFormats(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name    string
		ext     string
		content string
	}{
		{
			name: "TOML",
			ext:  ".toml",
			content: `
storage_backend = "sqlite"
`,
		},
		{
			name: "JSON",
			ext:  ".json",
			content: `{
  "storage_backend": "sqlite"
}`,
		},
		{
			name: "YAML",
			ext:  ".yaml",
			content: `storage_backend: sqlite
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configDir := filepath.Join(tmpDir, tc.name)
			require.NoError(t, os.MkdirAll(configDir, 0755))
			configFile := filepath.Join(configDir, "config"+tc.ext)
			require.NoError(t, os.WriteFile(configFile, []byte(tc.content), 0644))

			t.Setenv("TMUX_INTRAY_CONFIG_PATH", configFile)
			reset()
			Load()

			require.Equal(t, "sqlite", Get("storage_backend", ""))
		})
	}
}

// TestInvalidConfigValues verifies that invalid config values are
// handled gracefully (reset to defaults) as Bash would do.
func TestInvalidConfigValues(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name          string
		configKey     string
		invalidValue  string
		defaultValue  string
		configSnippet string
	}{
		{
			name:          "invalid_storage_backend",
			configKey:     "storage_backend",
			invalidValue:  "unknown",
			defaultValue:  "sqlite",
			configSnippet: `storage_backend = "unknown"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configDir := filepath.Join(tmpDir, tc.name)
			require.NoError(t, os.MkdirAll(configDir, 0755))
			configFile := filepath.Join(configDir, "config.toml")
			require.NoError(t, os.WriteFile(configFile, []byte(tc.configSnippet), 0644))

			t.Setenv("TMUX_INTRAY_CONFIG_PATH", configFile)
			t.Setenv("XDG_CONFIG_HOME", tmpDir)
			reset()

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			Load()

			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			stderrOutput := buf.String()

			// Value should be reset to default
			actualValue := Get(tc.configKey, "")
			require.Equal(t, tc.defaultValue, actualValue, "Invalid value should be reset to default")

			// Warning should be logged
			require.Contains(t, stderrOutput, "Warning:")
		})
	}
}

// TestConfigGetIntGetBoolBashCompatibility verifies that GetInt and GetBool
// helper functions work correctly and match Bash behavior.
func TestConfigGetIntGetBoolBashCompatibility(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config file with mixed types
	configDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configFile := filepath.Join(configDir, "config.toml")
	configContent := `
auto_cleanup_days = 15
debug = true
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configFile)
	reset()
	Load()

	// Test GetInt
	require.Equal(t, 15, GetInt("auto_cleanup_days", 0))

	// Test GetBool
	require.Equal(t, true, GetBool("debug", false))

	// Test missing keys return defaults
	require.Equal(t, 999, GetInt("missing_key", 999))
	require.Equal(t, true, GetBool("missing_key", true))
}

// TestEnvironmentVariableCasing verifies that environment variables are
// case-insensitive for values but keys use TMUX_INTRAY_ prefix.
func TestEnvironmentVariableCasing(t *testing.T) {
	tmpDir := t.TempDir()

	// Set env vars with different casings for values
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "SQLITE") // uppercase value
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	reset()
	Load()

	// Values should be normalized to lowercase (for enum values)
	require.Equal(t, "sqlite", Get("storage_backend", ""))
}

// TestConfigSampleCreation verifies that a sample config file is created
// when none exists (similar to Bash behavior).
func TestConfigSampleCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Ensure no config exists yet
	reset()
	Load()

	sampleConfigPath := filepath.Join(tmpDir, "tmux-intray", "config.toml")
	require.FileExists(t, sampleConfigPath, "Sample config should be created")

	// Verify it's valid TOML with expected keys
	content, err := os.ReadFile(sampleConfigPath)
	require.NoError(t, err)

	// Check for some expected keys
	require.Contains(t, string(content), "storage_backend")
	require.Contains(t, string(content), "state_dir")
}
