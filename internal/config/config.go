// Package config provides configuration loading.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
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
	// FileExtJSON is the file extension for JSON configuration files.
	FileExtJSON = ".json"
	// FileExtYAML is the file extension for YAML configuration files.
	FileExtYAML = ".yaml"
	// FileExtYML is an alternative file extension for YAML configuration files.
	FileExtYML = ".yml"
)

var (
	config    map[string]string
	configMap map[string]string
	mu        sync.RWMutex
)

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
	setDefault("storage_backend", "tsv")
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
	setDefault("storage_backend", "tsv")
	setDefault("debug", "false")
	setDefault("quiet", "false")
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
				// Try other extensions
				for _, ext := range []string{FileExtJSON, FileExtYAML, FileExtYML} {
					path := filepath.Join(configDir, "config"+ext)
					if _, err := os.Stat(path); err == nil {
						configPath = path
						break
					}
				}
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
	case FileExtJSON:
		err = json.Unmarshal(data, &raw)
	case FileExtYAML, FileExtYML:
		err = yaml.Unmarshal(data, &raw)
	default:
		return
	}
	if err != nil {
		colors.Warning(fmt.Sprintf("unable to parse config file %s: %v", configPath, err))
		return
	}

	// Merge into config, converting values to strings
	for k, v := range raw {
		key := strings.ToLower(k)
		converted, ok := coerceConfigValue(v)
		if !ok {
			colors.Warning(fmt.Sprintf("unsupported config value type for %s: %T", key, v))
			continue
		}
		config[key] = converted
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
		key = strings.ToLower(key)
		config[key] = parts[1]
	}
}

// validate checks and normalizes configuration values.
func validate() {
	// max_notifications must be positive integer
	if val, ok := config["max_notifications"]; ok {
		if n, err := strconv.Atoi(val); err != nil || n <= 0 {
			colors.Warning(fmt.Sprintf("invalid max_notifications value '%s': must be a positive integer, using default: %s", val, configMap["max_notifications"]))
			config["max_notifications"] = configMap["max_notifications"]
		}
	}

	// auto_cleanup_days must be positive integer
	if val, ok := config["auto_cleanup_days"]; ok {
		if n, err := strconv.Atoi(val); err != nil || n <= 0 {
			colors.Warning(fmt.Sprintf("invalid auto_cleanup_days value '%s': must be a positive integer, using default: %s", val, configMap["auto_cleanup_days"]))
			config["auto_cleanup_days"] = configMap["auto_cleanup_days"]
		}
	}

	// hooks_async_timeout must be positive integer
	if val, ok := config["hooks_async_timeout"]; ok {
		if n, err := strconv.Atoi(val); err != nil || n <= 0 {
			colors.Warning(fmt.Sprintf("invalid hooks_async_timeout value '%s': must be a positive integer, using default: %s", val, configMap["hooks_async_timeout"]))
			config["hooks_async_timeout"] = configMap["hooks_async_timeout"]
		}
	}

	// max_hooks must be positive integer
	if val, ok := config["max_hooks"]; ok {
		if n, err := strconv.Atoi(val); err != nil || n <= 0 {
			colors.Warning(fmt.Sprintf("invalid max_hooks value '%s': must be a positive integer, using default: %s", val, configMap["max_hooks"]))
			config["max_hooks"] = configMap["max_hooks"]
		}
	}

	// table_format must be one of allowed values
	if val, ok := config["table_format"]; ok {
		valLower := strings.ToLower(val)
		allowed := map[string]bool{"default": true, "minimal": true, "fancy": true}
		if !allowed[valLower] {
			colors.Warning(fmt.Sprintf("invalid table_format value '%s': must be one of: default, minimal, fancy; using default: %s", val, configMap["table_format"]))
			config["table_format"] = configMap["table_format"]
		} else if valLower != val {
			config["table_format"] = valLower
		}
	}

	// storage_backend must be one of allowed values
	if val, ok := config["storage_backend"]; ok {
		valLower := strings.ToLower(val)
		allowed := map[string]bool{"tsv": true, "sqlite": true}
		if !allowed[valLower] {
			colors.Warning(fmt.Sprintf("invalid storage_backend value '%s': must be one of: tsv, sqlite; using default: %s", val, configMap["storage_backend"]))
			config["storage_backend"] = configMap["storage_backend"]
		} else if valLower != val {
			config["storage_backend"] = valLower
		}
	}

	// status_format validation
	if val, ok := config["status_format"]; ok {
		valLower := strings.ToLower(val)
		allowed := map[string]bool{"compact": true, "detailed": true, "count-only": true}
		if !allowed[valLower] {
			colors.Warning(fmt.Sprintf("invalid status_format value '%s': must be one of: compact, detailed, count-only; using default: %s", val, configMap["status_format"]))
			config["status_format"] = configMap["status_format"]
		} else if valLower != val {
			config["status_format"] = valLower
		}
	}

	// hooks_failure_mode validation
	if val, ok := config["hooks_failure_mode"]; ok {
		valLower := strings.ToLower(val)
		allowed := map[string]bool{"ignore": true, "warn": true, "abort": true}
		if !allowed[valLower] {
			colors.Warning(fmt.Sprintf("invalid hooks_failure_mode value '%s': must be one of: ignore, warn, abort; using default: %s", val, configMap["hooks_failure_mode"]))
			config["hooks_failure_mode"] = configMap["hooks_failure_mode"]
		} else if valLower != val {
			config["hooks_failure_mode"] = valLower
		}
	}

	// storage_backend validation
	if val, ok := config["storage_backend"]; ok {
		valLower := strings.ToLower(val)
		allowed := map[string]bool{"tsv": true, "sqlite": true}
		if !allowed[valLower] {
			colors.Warning(fmt.Sprintf("invalid storage_backend value '%s': must be one of: tsv, sqlite; using default: %s", val, configMap["storage_backend"]))
			config["storage_backend"] = configMap["storage_backend"]
		} else if valLower != val {
			config["storage_backend"] = valLower
		}
	}

	// Normalize and validate boolean values
	for key, val := range config {
		if isBoolKey(key) {
			normalized := normalizeBool(val)
			config[key] = normalized
			if normalized != "true" && normalized != "false" {
				// Invalid boolean, revert to default
				if def, ok := configMap[key]; ok {
					colors.Warning(fmt.Sprintf("invalid boolean value for %s: '%s', must be one of: 1, true, yes, on, 0, false, no, off; using default: %s", key, val, def))
					config[key] = def
				}
			}
		}
	}
}

// isBoolKey returns true if the key expects a boolean value.
func isBoolKey(key string) bool {
	boolKeys := []string{
		"status_enabled", "show_levels", "hooks_enabled", "hooks_async", "debug", "quiet",
		"hooks_enabled_pre_add", "hooks_enabled_post_add", "hooks_enabled_pre_dismiss",
		"hooks_enabled_post_dismiss", "hooks_enabled_cleanup", "hooks_enabled_post_cleanup",
	}
	for _, k := range boolKeys {
		if key == k {
			return true
		}
	}
	return false
}

// normalizeBool converts various boolean representations to "true"/"false".
func normalizeBool(val string) string {
	switch strings.ToLower(val) {
	case "1", "true", "yes", "on":
		return "true"
	case "0", "false", "no", "off":
		return "false"
	default:
		// If invalid, return default false? We'll keep as is and validation will fix later.
		return val
	}
}

// valueToInterface converts a configuration value to appropriate type for TOML.
func valueToInterface(key, val string) interface{} {
	// integer keys
	if key == "max_notifications" || key == "auto_cleanup_days" || key == "hooks_async_timeout" || key == "max_hooks" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	// boolean keys
	if isBoolKey(key) {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	// default string
	return val
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
	os.MkdirAll(configDir, FileModeDir)

	// Build typed map from configMap (defaults)
	typed := make(map[string]interface{})
	for k, v := range configMap {
		typed[k] = valueToInterface(k, v)
	}

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
