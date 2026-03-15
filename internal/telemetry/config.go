// Package telemetry - config instrumentation
// This file provides helper functions for logging config read operations.
package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	// FeatureCategoryConfig categorizes config reads under CLI category
	// since config is typically accessed during CLI startup and initialization.
	FeatureCategoryConfig = FeatureCategoryCLI
)

// LogConfigRead logs a configuration read operation.
// This is a no-op if telemetry is disabled.
// Values that should be tracked include:
// - view_mode, table_format (view/display settings)
// - active_tab, sort_by, sort_order (UI/tab settings)
// - filter settings (level, state, read, session, window, pane)
// - dedup settings (dedup.criteria, dedup.window)
// - debug, quiet, telemetry_enabled
func LogConfigRead(key string, value interface{}) {
	if !IsEnabled() {
		return
	}

	// Filter config keys to avoid logging noise from rarely-accessed settings
	if !shouldTrackConfigKey(key) {
		return
	}

	context := map[string]interface{}{
		"key":   key,
		"value": value,
	}

	LogFeature("config_read", FeatureCategoryConfig, context)
}

// LogConfigLoad logs when configuration is fully loaded/reloaded.
// This is a no-op if telemetry is disabled.
// Includes snapshot of active configuration values.
func LogConfigLoad(configSnapshot map[string]string) {
	if !IsEnabled() {
		return
	}

	// Filter to only include tracked config keys
	trackedConfig := make(map[string]string)
	for key, value := range configSnapshot {
		if shouldTrackConfigKey(key) {
			trackedConfig[key] = value
		}
	}

	context := map[string]interface{}{
		"config": trackedConfig,
	}

	LogFeature("config_load", FeatureCategoryConfig, context)
}

// LogSettingsLoad logs when TUI settings are loaded.
// This is a no-op if telemetry is disabled.
// Tracks view_mode, active_tab, filters, and other UI preferences.
func LogSettingsLoad(settings map[string]interface{}) {
	if !IsEnabled() {
		return
	}

	context := map[string]interface{}{
		"settings": settings,
	}

	LogFeature("settings_load", FeatureCategoryConfig, context)
}

// LogSettingChange logs when a TUI setting is modified.
// This is a no-op if telemetry is disabled.
func LogSettingChange(key string, oldValue, newValue interface{}) {
	if !IsEnabled() {
		return
	}

	context := map[string]interface{}{
		"key":       key,
		"old_value": oldValue,
		"new_value": newValue,
	}

	LogFeature("setting_change", FeatureCategoryConfig, context)
}

// shouldTrackConfigKey returns true if the given config key should be tracked.
// This helps avoid logging noise from rarely-accessed settings.
func shouldTrackConfigKey(key string) bool {
	// Config keys to track - these are settings that indicate user preferences
	trackedKeys := map[string]bool{
		// View/Display settings
		"view_mode":      true,
		"table_format":   true,
		"status_enabled": true,
		"status_format":  true,
		"show_levels":    true,
		// Filter settings (from TUI settings, but also in config)
		"filter_level":   true,
		"filter_state":   true,
		"filter_read":    true,
		"filter_session": true,
		// Dedup settings
		"dedup.criteria": true,
		"dedup.window":   true,
		// Feature flags
		"hooks_enabled":     true,
		"telemetry_enabled": true,
		// Debug flags
		"debug":           true,
		"quiet":           true,
		"logging_enabled": true,
		"logging_level":   true,
	}

	return trackedKeys[key]
}

// MergeContextData merges context data into JSON.
// This is a utility for combining multiple context fields.
func MergeContextData(contexts ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, ctx := range contexts {
		if ctx != nil {
			for k, v := range ctx {
				result[k] = v
			}
		}
	}
	return result
}

// ContextToJSON converts context map to JSON string for storage.
// Returns empty object "{}" on error.
func ContextToJSON(context map[string]interface{}) string {
	if context == nil {
		return "{}"
	}
	data, err := json.Marshal(context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "telemetry: failed to marshal context: %v\n", err)
		return "{}"
	}
	return string(data)
}

// ConfigGetWithTelemetry is a wrapper around config.Get that logs to telemetry.
// This function avoids circular imports by accepting the config value directly.
// Usage: value := telemetry.ConfigGetWithTelemetry("view_mode", result)
func ConfigGetWithTelemetry(key, value string) string {
	LogConfigRead(key, value)
	return value
}

// ConfigGetBoolWithTelemetry is a wrapper around config.GetBool that logs to telemetry.
func ConfigGetBoolWithTelemetry(key string, value bool) bool {
	LogConfigRead(key, value)
	return value
}

// ConfigGetIntWithTelemetry is a wrapper around config.GetInt that logs to telemetry.
func ConfigGetIntWithTelemetry(key string, value int) int {
	LogConfigRead(key, value)
	return value
}

// ConfigGetDurationWithTelemetry is a wrapper around config.GetDuration that logs to telemetry.
func ConfigGetDurationWithTelemetry(key string, value interface{}) interface{} {
	LogConfigRead(key, value)
	return value
}
