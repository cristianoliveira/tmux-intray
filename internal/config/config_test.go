package config

import (
	"bytes"
	"fmt"
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
	require.Equal(t, "sqlite", Get("storage_backend", ""))
	require.Equal(t, "default", Get("table_format", ""))
	require.Equal(t, "compact", Get("status_format", ""))
	require.Equal(t, "true", Get("status_enabled", ""))
	require.Equal(t, "false", Get("show_levels", ""))
	require.Equal(t, "true", Get("hooks_enabled", ""))
	require.Equal(t, "warn", Get("hooks_failure_mode", ""))
	require.Equal(t, "false", Get("hooks_async", ""))
	require.Equal(t, "30", Get("hooks_async_timeout", ""))
	require.Equal(t, "10", Get("max_hooks", ""))
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

func TestConfigFileTypeValidation(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "config.toml")
	data := `
max_notifications = [1, 2]
status_enabled = {value = true}
table_format = "minimal"
hooks_async_timeout = 12
`
	require.NoError(t, os.WriteFile(configPath, []byte(data), 0644))

	t.Setenv("TMUX_INTRAY_CONFIG_PATH", configPath)
	Load()

	// Unsupported types should be skipped, using defaults
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
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "unknown")
	Load()
	require.Equal(t, "sqlite", Get("storage_backend", ""))
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
	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
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
	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain warning message about parse error.
	require.Contains(t, output, "Warning:")
	require.Contains(t, output, "unable to parse config file")
	require.Contains(t, output, configPath)

	// Defaults should still be loaded.
	require.Equal(t, "1000", Get("max_notifications", ""))
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
	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
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
	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should NOT contain debug message.
	require.NotContains(t, output, "Debug:")
	require.NotContains(t, output, "unable to read config file")

	// Defaults should still be loaded.
	require.Equal(t, "1000", Get("max_notifications", ""))
}

// TestRegisterValidatorPanic tests that registering a duplicate validator panics.
func TestRegisterValidatorPanic(t *testing.T) {
	reset()
	// Register a test validator
	RegisterValidator("test_key", func(key, value, defaultValue string) (string, error) {
		return value, nil
	})

	// Attempting to register the same key again should panic
	require.Panics(t, func() {
		RegisterValidator("test_key", func(key, value, defaultValue string) (string, error) {
			return value, nil
		})
	})
}

// TestRegisterValidatorThreadSafety tests that RegisterValidator is thread-safe.
func TestRegisterValidatorThreadSafety(t *testing.T) {
	reset()
	done := make(chan bool)

	// Register multiple validators concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := fmt.Sprintf("concurrent_key_%d", idx)
			RegisterValidator(key, func(key, value, defaultValue string) (string, error) {
				return value, nil
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all validators were registered
	require.NotNil(t, getValidator("concurrent_key_0"))
	require.NotNil(t, getValidator("concurrent_key_5"))
	require.NotNil(t, getValidator("concurrent_key_9"))
}

// TestGetValidatorThreadSafety tests that getValidator is thread-safe.
func TestGetValidatorThreadSafety(t *testing.T) {
	reset()
	// Register a validator
	RegisterValidator("thread_test_key", func(key, value, defaultValue string) (string, error) {
		return value, nil
	})

	done := make(chan bool)

	// Read validators concurrently
	for i := 0; i < 10; i++ {
		go func() {
			v := getValidator("thread_test_key")
			require.NotNil(t, v)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestGetValidatorNonExistent tests that getValidator returns nil for unregistered keys.
func TestGetValidatorNonExistent(t *testing.T) {
	reset()
	v := getValidator("non_existent_key")
	require.Nil(t, v)
}

// TestInitValidators verifies all validators are registered.
func TestInitValidators(t *testing.T) {
	reset()
	// Reinitialize validators (init() already ran, but we can test the registry)
	require.NotNil(t, getValidator("max_notifications"))
	require.NotNil(t, getValidator("auto_cleanup_days"))
	require.NotNil(t, getValidator("hooks_async_timeout"))
	require.NotNil(t, getValidator("max_hooks"))

	// Enum validators (3 keys)
	require.NotNil(t, getValidator("table_format"))
	require.NotNil(t, getValidator("storage_backend"))
	require.NotNil(t, getValidator("status_format"))
	require.NotNil(t, getValidator("hooks_failure_mode"))

	// Boolean validators (12 keys)
	require.NotNil(t, getValidator("status_enabled"))
	require.NotNil(t, getValidator("show_levels"))
	require.NotNil(t, getValidator("hooks_enabled"))
	require.NotNil(t, getValidator("hooks_async"))
	require.NotNil(t, getValidator("debug"))
	require.NotNil(t, getValidator("quiet"))
	require.NotNil(t, getValidator("hooks_enabled_pre_add"))
	require.NotNil(t, getValidator("hooks_enabled_post_add"))
	require.NotNil(t, getValidator("hooks_enabled_pre_dismiss"))
	require.NotNil(t, getValidator("hooks_enabled_post_dismiss"))
	require.NotNil(t, getValidator("hooks_enabled_cleanup"))
	require.NotNil(t, getValidator("hooks_enabled_post_cleanup"))
}

// TestPositiveIntValidatorEmpty tests that empty value returns default.
func TestPositiveIntValidatorEmpty(t *testing.T) {
	validator := PositiveIntValidator()
	result, err := validator("test_key", "", "100")
	require.NoError(t, err)
	require.Equal(t, "100", result)
}

// TestPositiveIntValidatorValid tests valid positive integers.
func TestPositiveIntValidatorValid(t *testing.T) {
	validator := PositiveIntValidator()

	for _, val := range []string{"1", "42", "9999"} {
		result, err := validator("test_key", val, "100")
		require.NoError(t, err)
		require.Equal(t, val, result)
	}
}

// TestPositiveIntValidatorZero tests that zero is rejected and default is returned.
func TestPositiveIntValidatorZero(t *testing.T) {
	validator := PositiveIntValidator()
	result, err := validator("test_key", "0", "100")
	require.NoError(t, err)
	require.Equal(t, "100", result)
}

// TestPositiveIntValidatorNegative tests that negative numbers are rejected.
func TestPositiveIntValidatorNegative(t *testing.T) {
	validator := PositiveIntValidator()
	result, err := validator("test_key", "-5", "100")
	require.NoError(t, err)
	require.Equal(t, "100", result)
}

// TestPositiveIntValidatorNonInteger tests that non-integers are rejected.
func TestPositiveIntValidatorNonInteger(t *testing.T) {
	validator := PositiveIntValidator()
	result, err := validator("test_key", "abc", "100")
	require.NoError(t, err)
	require.Equal(t, "100", result)
}

// TestPositiveIntValidatorFloat tests that floats are rejected.
func TestPositiveIntValidatorFloat(t *testing.T) {
	validator := PositiveIntValidator()
	result, err := validator("test_key", "3.14", "100")
	require.NoError(t, err)
	require.Equal(t, "100", result)
}

// TestEnumValidatorEmpty tests that empty value returns default.
func TestEnumValidatorEmpty(t *testing.T) {
	validator := EnumValidator(map[string]bool{"option1": true, "option2": true})
	result, err := validator("test_key", "", "option1")
	require.NoError(t, err)
	require.Equal(t, "option1", result)
}

// TestEnumValidatorValid tests valid enum values.
func TestEnumValidatorValid(t *testing.T) {
	validator := EnumValidator(map[string]bool{"red": true, "green": true, "blue": true})

	testCases := []struct {
		input    string
		expected string
	}{
		{"red", "red"},
		{"RED", "red"},     // Case insensitive
		{"Green", "green"}, // Case insensitive
		{"BLUE", "blue"},   // Case insensitive
	}

	for _, tc := range testCases {
		result, err := validator("test_key", tc.input, "red")
		require.NoError(t, err)
		require.Equal(t, tc.expected, result)
	}
}

// TestEnumValidatorInvalid tests that invalid enum values return default.
func TestEnumValidatorInvalid(t *testing.T) {
	validator := EnumValidator(map[string]bool{"red": true, "green": true, "blue": true})
	result, err := validator("test_key", "yellow", "red")
	require.NoError(t, err)
	require.Equal(t, "red", result)
}

// TestEnumValidatorEmptyAllowed tests enum validator with empty allowed map.
func TestEnumValidatorEmptyAllowed(t *testing.T) {
	validator := EnumValidator(map[string]bool{})
	result, err := validator("test_key", "anything", "default")
	require.NoError(t, err)
	require.Equal(t, "default", result)
}

// TestBoolValidatorEmpty tests that empty value returns default.
func TestBoolValidatorEmpty(t *testing.T) {
	validator := BoolValidator()
	result, err := validator("test_key", "", "true")
	require.NoError(t, err)
	require.Equal(t, "true", result)
}

// TestBoolValidatorTrueValues tests all variations that normalize to "true".
func TestBoolValidatorTrueValues(t *testing.T) {
	validator := BoolValidator()

	for _, val := range []string{"1", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On"} {
		result, err := validator("test_key", val, "false")
		require.NoError(t, err)
		require.Equal(t, "true", result, "value %s should normalize to true", val)
	}
}

// TestBoolValidatorFalseValues tests all variations that normalize to "false".
func TestBoolValidatorFalseValues(t *testing.T) {
	validator := BoolValidator()

	for _, val := range []string{"0", "false", "FALSE", "False", "no", "NO", "No", "off", "OFF", "Off"} {
		result, err := validator("test_key", val, "true")
		require.NoError(t, err)
		require.Equal(t, "false", result, "value %s should normalize to false", val)
	}
}

// TestBoolValidatorInvalid tests completely invalid boolean values.
func TestBoolValidatorInvalid(t *testing.T) {
	validator := BoolValidator()

	for _, val := range []string{"maybe", "2", "invalid", "t", "f", "y", "n"} {
		result, err := validator("test_key", val, "default")
		require.NoError(t, err)
		require.Equal(t, "default", result, "value %s should return default", val)
	}
}

// TestNormalizeBool tests the normalizeBool helper function.
func TestNormalizeBool(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"1", "true"},
		{"true", "true"},
		{"TRUE", "true"},
		{"True", "true"},
		{"yes", "true"},
		{"YES", "true"},
		{"on", "true"},
		{"ON", "true"},
		{"0", "false"},
		{"false", "false"},
		{"FALSE", "false"},
		{"no", "false"},
		{"NO", "false"},
		{"off", "false"},
		{"OFF", "false"},
		{"maybe", "maybe"},     // Invalid returns as-is
		{"invalid", "invalid"}, // Invalid returns as-is
	}

	for _, tc := range testCases {
		result := normalizeBool(tc.input)
		require.Equal(t, tc.expected, result, "normalizeBool(%q)", tc.input)
	}
}

// TestAllowedValues tests the allowedValues helper function.
func TestAllowedValues(t *testing.T) {
	testCases := []struct {
		allowed  map[string]bool
		expected string
	}{
		{map[string]bool{"a": true, "b": true, "c": true}, "a, b, c"},
		{map[string]bool{"zebra": true, "apple": true}, "apple, zebra"},
		{map[string]bool{"single": true}, "single"},
		{map[string]bool{}, ""},
	}

	for _, tc := range testCases {
		result := allowedValues(tc.allowed)
		require.Equal(t, tc.expected, result)
	}
}

// TestValidatorReturningError tests the validate function with a validator that returns an error.
func TestValidatorReturningError(t *testing.T) {
	reset()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Register a validator that returns an error
	RegisterValidator("error_test_key", func(key, value, defaultValue string) (string, error) {
		if value == "error" {
			return "", fmt.Errorf("simulated error")
		}
		return value, nil
	})

	t.Setenv("TMUX_INTRAY_ERROR_TEST_KEY", "error")
	Load()

	// Should use default value when validator returns error
	require.Equal(t, "", Get("error_test_key", ""))
}
