// Package config provides configuration loading.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/pelletier/go-toml/v2"
)

// File permission constants
const (
	// FileModeDir is the permission for directories (rwxr-xr-x)
	// Owner: read/write/execute, Group/others: read/execute
	FileModeDir os.FileMode = 0755
	// FileModeFile is the permission for data files (rw-r--r--)
	// Owner: read/write, Group/others: read only
	FileModeFile os.FileMode = 0644

	// File extension constants for configuration files
	// FileExtTOML is the file extension for TOML configuration files (primary format).
	FileExtTOML = ".toml"
)

var (
	config    map[string]string
	configMap map[string]string
	mu        sync.RWMutex
)

func init() {
	initValidators()
}

// Load initializes configuration.
func Load() {
	mu.Lock()
	defer mu.Unlock()

	// Reset to defaults
	config = make(map[string]string)
	configMap = make(map[string]string)

	// Set default values
	setDefaults()
	// Apply environment variable overrides
	loadFromEnv()
	// Load from configuration file
	loadFromFile()
	// Re-apply environment variable overrides so env wins
	loadFromEnv()
	// Validate and normalize values
	validate()
	// Compute derived directories
	computeDirs()
	// Create sample config if none exists
	createSampleConfig()
}

// setDefaults populates config with default values.
func setDefaults() {
	// Compute XDG directories
	home, _ := os.UserHomeDir()
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(home, ".config")
	}
	xdgStateHome := os.Getenv("XDG_STATE_HOME")
	if xdgStateHome == "" {
		xdgStateHome = filepath.Join(home, ".local", "state")
	}

	configDir := filepath.Join(xdgConfigHome, "tmux-intray")
	stateDir := filepath.Join(xdgStateHome, "tmux-intray")
	hooksDir := filepath.Join(configDir, "hooks")

	// Set defaults
	setDefault("config_dir", configDir)
	setDefault("state_dir", stateDir)
	setDefault("storage_backend", "sqlite")
	setDefault("hooks_dir", hooksDir)
	setDefault("max_notifications", "1000")
	setDefault("auto_cleanup_days", "30")
	setDefault("date_format", "%Y-%m-%d %H:%M:%S")
	setDefault("table_format", "default")
	setDefault("status_enabled", "true")
	setDefault("status_format", "compact")
	setDefault("show_levels", "false")
	setDefault("level_colors", "info:green,warning:yellow,error:red,critical:magenta")
	setDefault("hooks_enabled", "true")
	setDefault("hooks_failure_mode", "warn")
	setDefault("hooks_async", "false")
	setDefault("hooks_async_timeout", "30")
	setDefault("max_hooks", "10")
	// Optional per-hook keys default to "true"
	setDefault("hooks_enabled_pre_add", "true")
	setDefault("hooks_enabled_post_add", "true")
	setDefault("hooks_enabled_pre_dismiss", "true")
	setDefault("hooks_enabled_post_dismiss", "true")
	setDefault("hooks_enabled_cleanup", "true")
	setDefault("hooks_enabled_post_cleanup", "true")
	setDefault("debug", "false")
	setDefault("quiet", "false")
	setDedupDefaults()
}

func setDefault(key, value string) {
	config[key] = value
	configMap[key] = value
}

// loadFromFile reads configuration from a file.
func loadFromFile() {
	configPath := os.Getenv("TMUX_INTRAY_CONFIG_PATH")
	if configPath == "" {
		// Try default location
		if configDir, ok := config["config_dir"]; ok {
			configPath = filepath.Join(configDir, "config"+FileExtTOML)
			if _, err := os.Stat(configPath); err != nil {
				// TOML file doesn't exist, no configuration to load
				configPath = ""
			}
		}
	}
	if configPath == "" {
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		colors.Debug(fmt.Sprintf("unable to read config file %s: %v", configPath, err))
		return
	}

	var raw map[string]interface{}
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case FileExtTOML:
		err = toml.Unmarshal(data, &raw)
	default:
		return
	}
	if err != nil {
		colors.Warning(fmt.Sprintf("unable to parse config file %s: %v", configPath, err))
		return
	}

	flat := flattenConfigMap(raw)
	for key, value := range flat {
		config[key] = value
	}
}

// flattenConfigMap flattens nested TOML structures into dot-delimited keys.
func flattenConfigMap(raw map[string]interface{}) map[string]string {
	result := make(map[string]string)
	flattenConfigMapInto(result, "", raw)
	return result
}

func flattenConfigMapInto(result map[string]string, prefix string, raw map[string]interface{}) {
	if raw == nil {
		return
	}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := raw[key]
		lowerKey := strings.ToLower(key)
		if prefix != "" {
			lowerKey = prefix + "." + lowerKey
		}
		switch typed := value.(type) {
		case map[string]interface{}:
			flattenConfigMapInto(result, lowerKey, typed)
		default:
			converted, ok := coerceConfigValue(value)
			if !ok {
				colors.Warning(fmt.Sprintf("unsupported config value type for %s: %T", lowerKey, value))
				continue
			}
			result[lowerKey] = converted
		}
	}
}

// coerceConfigValue converts a configuration value to its string representation.
// Supported types are string, int, int64, float64, and bool.
// Returns the string representation and true if conversion succeeded,
// otherwise returns empty string and false.
func coerceConfigValue(value interface{}) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case int:
		return strconv.Itoa(typed), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(typed), true
	default:
		return "", false
	}
}

// loadFromEnv applies environment variable overrides.
func loadFromEnv() {
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "TMUX_INTRAY_") {
			continue
		}
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimPrefix(parts[0], "TMUX_INTRAY_")
		key = strings.ReplaceAll(key, "__", ".")
		key = strings.ToLower(key)
		config[key] = parts[1]
	}
}

// validate checks and normalizes configuration values using registered validators.
func validate() {
	for key, value := range config {
		validator := getValidator(key)
		if validator == nil {
			continue // No validator for this key
		}
		defaultValue := configMap[key]
		normalizedValue, err := validator(key, value, defaultValue)
		if err != nil {
			// Validators should handle errors themselves and log warnings,
			// but if one returns an error, we log it and use default
			colors.Warning(fmt.Sprintf("validation error for %s: %v, using default: %s", key, err, defaultValue))
			config[key] = defaultValue
		} else {
			config[key] = normalizedValue
		}
	}
}

// valueToInterface converts a configuration value to appropriate type for TOML.
func valueToInterface(key, val string) interface{} {
	// Try to parse as integer first
	if n, err := strconv.Atoi(val); err == nil {
		return n
	}
	// Try to parse as boolean
	if b, err := strconv.ParseBool(val); err == nil {
		return b
	}
	// default string
	return val
}

// buildSampleConfigMap converts the flat defaults map into a nested map suitable for TOML marshaling.
func buildSampleConfigMap(flat map[string]string) map[string]interface{} {
	root := make(map[string]interface{})
	keys := make([]string, 0, len(flat))
	for key := range flat {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.Split(key, ".")
		current := root
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = valueToInterface(key, flat[key])
				continue
			}
			next, ok := current[part]
			if !ok {
				nested := make(map[string]interface{})
				current[part] = nested
				current = nested
				continue
			}
			nested, ok := next.(map[string]interface{})
			if !ok {
				nested = make(map[string]interface{})
				current[part] = nested
			}
			current = nested
		}
	}
	return root
}

// computeDirs recomputes directory paths after config is loaded.
func computeDirs() {
	// config_dir may have been overridden by environment
	configDir := config["config_dir"]
	if configDir == "" {
		return
	}
	// hooks_dir defaults to config_dir/hooks unless explicitly set
	if _, set := config["hooks_dir"]; !set {
		config["hooks_dir"] = filepath.Join(configDir, "hooks")
	}
	// Ensure state_dir and config_dir exist? Not yet.
}

// createSampleConfig creates a sample configuration file if none exists.
func createSampleConfig() {
	configDir := config["config_dir"]
	if configDir == "" {
		return
	}
	samplePath := filepath.Join(configDir, "config"+FileExtTOML)
	if _, err := os.Stat(samplePath); err == nil {
		return // file exists
	}
	// Ensure directory exists
	if err := os.MkdirAll(configDir, FileModeDir); err != nil {
		colors.Warning(fmt.Sprintf("unable to create config directory %s: %v", configDir, err))
		return
	}

	// Build typed map from configMap (defaults)
	typed := buildSampleConfigMap(configMap)

	data, err := toml.Marshal(typed)
	if err != nil {
		colors.Warning(fmt.Sprintf("unable to marshal sample config: %v", err))
		return
	}
	// Add a header comment
	header := "# tmux-intray configuration\n# This file is in TOML format.\n# Uncomment and edit values as needed.\n\n"
	if err := os.WriteFile(samplePath, append([]byte(header), data...), 0644); err != nil {
		colors.Warning(fmt.Sprintf("unable to write sample config to %s: %v", samplePath, err))
	}
}

// Get returns a configuration value or default.
func Get(key, defaultValue string) string {
	mu.RLock()
	defer mu.RUnlock()
	if val, ok := config[key]; ok {
		return val
	}
	return defaultValue
}

// GetInt returns a configuration value as integer, or default.
func GetInt(key string, defaultValue int) int {
	mu.RLock()
	defer mu.RUnlock()
	val, ok := config[key]
	if !ok {
		return defaultValue
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return n
}

// GetBool returns a configuration value as boolean, or default.
func GetBool(key string, defaultValue bool) bool {
	mu.RLock()
	defer mu.RUnlock()
	val, ok := config[key]
	if !ok {
		return defaultValue
	}
	switch strings.ToLower(val) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// GetDuration returns a configuration value parsed as time.Duration, or defaultValue when missing/invalid.
func GetDuration(key string, defaultValue time.Duration) time.Duration {
	mu.RLock()
	defer mu.RUnlock()
	val, ok := config[key]
	if !ok || val == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultValue
	}
	return d
}
