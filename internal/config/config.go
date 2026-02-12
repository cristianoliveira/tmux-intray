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

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/pelletier/go-toml/v2"
)

// Validator validates and normalizes a configuration value.
// Returns the normalized value and an error if validation fails.
type Validator func(key, value, defaultValue string) (normalized string, err error)

// validatorRegistry manages the set of registered validators.
type validatorRegistry struct {
	mu         sync.RWMutex
	validators map[string]Validator
}

// registry is the global validator registry.
var registry = &validatorRegistry{
	validators: make(map[string]Validator),
}

// RegisterValidator registers a validator for a configuration key.
// Panics if a validator is already registered for the key.
func RegisterValidator(key string, validator Validator) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.validators[key]; exists {
		panic(fmt.Sprintf("validator already registered for key: %s", key))
	}
	registry.validators[key] = validator
}

// getValidator returns the validator for a key, or nil if not registered.
func getValidator(key string) Validator {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	return registry.validators[key]
}

// PositiveIntValidator returns a validator that ensures a value is a positive integer.
func PositiveIntValidator() Validator {
	return func(key, value, defaultValue string) (string, error) {
		if value == "" {
			return defaultValue, nil
		}
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			colors.Warning(fmt.Sprintf("invalid %s value '%s': must be a positive integer, using default: %s", key, value, defaultValue))
			return defaultValue, nil
		}
		return value, nil
	}
}

// EnumValidator returns a validator that ensures a value is one of the allowed enum values.
func EnumValidator(allowed map[string]bool) Validator {
	return func(key, value, defaultValue string) (string, error) {
		if value == "" {
			return defaultValue, nil
		}
		valueLower := strings.ToLower(value)
		if !allowed[valueLower] {
			colors.Warning(fmt.Sprintf("invalid %s value '%s': must be one of: %s; using default: %s", key, value, allowedValues(allowed), defaultValue))
			return defaultValue, nil
		}
		return valueLower, nil
	}
}

// BoolValidator returns a validator that normalizes and validates boolean values.
// Returns a shared validator instance for all boolean keys.
func BoolValidator() Validator {
	return func(key, value, defaultValue string) (string, error) {
		if value == "" {
			return defaultValue, nil
		}
		normalized := normalizeBool(value)
		if normalized != "true" && normalized != "false" {
			colors.Warning(fmt.Sprintf("invalid boolean value for %s: '%s', must be one of: 1, true, yes, on, 0, false, no, off; using default: %s", key, value, defaultValue))
			return defaultValue, nil
		}
		return normalized, nil
	}
}

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
	setDefault("dual_read_backend", "sqlite")
	setDefault("dual_verify_only", "false")
	setDefault("dual_verify_sample_size", "25")
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

// normalizeBool converts various boolean representations to "true"/"false".
func normalizeBool(val string) string {
	switch strings.ToLower(val) {
	case "1", "true", "yes", "on":
		return "true"
	case "0", "false", "no", "off":
		return "false"
	default:
		// If invalid, return as-is; validation will fix it.
		return val
	}
}

// initValidators registers all configuration validators.
func initValidators() {
	// Positive integer validators (5 keys)
	positiveIntValidator := PositiveIntValidator()
	RegisterValidator("max_notifications", positiveIntValidator)
	RegisterValidator("auto_cleanup_days", positiveIntValidator)
	RegisterValidator("hooks_async_timeout", positiveIntValidator)
	RegisterValidator("max_hooks", positiveIntValidator)
	RegisterValidator("dual_verify_sample_size", positiveIntValidator)

	// Enum validators (5 keys)
	RegisterValidator("table_format", EnumValidator(map[string]bool{"default": true, "minimal": true, "fancy": true}))
	RegisterValidator("storage_backend", EnumValidator(map[string]bool{"tsv": true, "sqlite": true, "dual": true}))
	RegisterValidator("status_format", EnumValidator(map[string]bool{"compact": true, "detailed": true, "count-only": true}))
	RegisterValidator("hooks_failure_mode", EnumValidator(map[string]bool{"ignore": true, "warn": true, "abort": true}))
	RegisterValidator("dual_read_backend", EnumValidator(map[string]bool{"tsv": true, "sqlite": true}))

	// Boolean validators (13 keys) - shared instance
	boolValidator := BoolValidator()
	RegisterValidator("status_enabled", boolValidator)
	RegisterValidator("show_levels", boolValidator)
	RegisterValidator("hooks_enabled", boolValidator)
	RegisterValidator("hooks_async", boolValidator)
	RegisterValidator("debug", boolValidator)
	RegisterValidator("quiet", boolValidator)
	RegisterValidator("hooks_enabled_pre_add", boolValidator)
	RegisterValidator("hooks_enabled_post_add", boolValidator)
	RegisterValidator("hooks_enabled_pre_dismiss", boolValidator)
	RegisterValidator("hooks_enabled_post_dismiss", boolValidator)
	RegisterValidator("hooks_enabled_cleanup", boolValidator)
	RegisterValidator("hooks_enabled_post_cleanup", boolValidator)
	RegisterValidator("dual_verify_only", boolValidator)
}

// allowedValues returns a comma-separated string of allowed values.
func allowedValues(allowed map[string]bool) string {
	values := make([]string, 0, len(allowed))
	for k := range allowed {
		values = append(values, k)
	}
	// Sort for consistent output
	sort.Strings(values)
	return strings.Join(values, ", ")
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
