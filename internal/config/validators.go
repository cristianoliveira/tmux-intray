package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
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

// DurationValidator validates Go-style duration strings (e.g., 30s, 1m, 2h).
// When allowEmpty is true, empty values are preserved (used to disable duration-based behavior).
func DurationValidator(allowEmpty bool) Validator {
	return func(key, value, defaultValue string) (string, error) {
		if value == "" {
			if allowEmpty {
				return value, nil
			}
			return defaultValue, nil
		}
		duration, err := time.ParseDuration(value)
		if err != nil || duration < 0 {
			colors.Warning(fmt.Sprintf("invalid duration for %s: '%s', must be a Go-style duration (e.g. 30s, 5m); using default: %s", key, value, defaultValue))
			return defaultValue, nil
		}
		return duration.String(), nil
	}
}

// initValidators registers all configuration validators.
func initValidators() {
	// Positive integer validators (4 keys)
	positiveIntValidator := PositiveIntValidator()
	RegisterValidator("max_notifications", positiveIntValidator)
	RegisterValidator("auto_cleanup_days", positiveIntValidator)
	RegisterValidator("hooks_async_timeout", positiveIntValidator)
	RegisterValidator("max_hooks", positiveIntValidator)

	// Enum validators (3 keys)
	RegisterValidator("table_format", EnumValidator(map[string]bool{"default": true, "minimal": true, "fancy": true}))
	RegisterValidator("storage_backend", EnumValidator(map[string]bool{"sqlite": true}))
	RegisterValidator("status_format", EnumValidator(map[string]bool{"compact": true, "detailed": true, "count-only": true}))
	RegisterValidator("hooks_failure_mode", EnumValidator(map[string]bool{"ignore": true, "warn": true, "abort": true}))

	// Boolean validators (12 keys) - shared instance
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

	// Logging validators
	RegisterValidator("logging_enabled", boolValidator)
	RegisterValidator("logging_level", EnumValidator(map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}))
	RegisterValidator("logging_max_files", PositiveIntValidator())
	// log_file has no validator (any string)

	registerDedupValidators()
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
