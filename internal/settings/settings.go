// Package settings provides TUI user preferences persistence.
package settings

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

// Default column values.
const (
	ColumnID          = "id"
	ColumnTimestamp   = "timestamp"
	ColumnState       = "state"
	ColumnSession     = "session"
	ColumnWindow      = "window"
	ColumnPane        = "pane"
	ColumnMessage     = "message"
	ColumnPaneCreated = "pane_created"
	ColumnLevel       = "level"
)

// Default column order for TUI display.
var DefaultColumns = []string{
	ColumnID,
	ColumnTimestamp,
	ColumnState,
	ColumnLevel,
	ColumnSession,
	ColumnWindow,
	ColumnPane,
	ColumnMessage,
}

// Sort direction constants.
const (
	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// Sort by constants.
const (
	SortByID        = "id"
	SortByTimestamp = "timestamp"
	SortByState     = "state"
	SortByLevel     = "level"
	SortBySession   = "session"
)

// View mode constants.
const (
	ViewModeCompact  = "compact"
	ViewModeDetailed = "detailed"
)

// State filter constants.
const (
	StateFilterActive    = "active"
	StateFilterDismissed = "dismissed"
)

// Level filter constants.
const (
	LevelFilterInfo     = "info"
	LevelFilterWarning  = "warning"
	LevelFilterError    = "error"
	LevelFilterCritical = "critical"
)

// Filter defines active filter criteria for notification display.
type Filter struct {
	// Level filters notifications by severity level.
	// Empty string means no filter (show all levels).
	// Valid values: "info", "warning", "error", "critical", "".
	Level string `json:"level"`

	// State filters notifications by state.
	// Empty string means no filter (show all states).
	// Valid values: "active", "dismissed", "".
	State string `json:"state"`

	// Session filters notifications by tmux session name.
	// Empty string means no filter (show all sessions).
	Session string `json:"session"`

	// Window filters notifications by tmux window ID.
	// Empty string means no filter (show all windows).
	Window string `json:"window"`

	// Pane filters notifications by tmux pane ID.
	// Empty string means no filter (show all panes).
	Pane string `json:"pane"`
}

// Settings holds TUI user preferences persisted to disk.
//
// JSON Schema:
//
//	{
//	  "columns": ["id", "timestamp", "state", "level", "session", "window", "pane", "message"],
//	  "sortBy": "timestamp",
//	  "sortOrder": "desc",
//	  "filters": {
//	    "level": "",
//	    "state": "",
//	    "session": "",
//	    "window": "",
//	    "pane": ""
//	  },
//	  "viewMode": "compact"
//	}
//
// Settings are stored at ~/.config/tmux-intray/settings.json
type Settings struct {
	// Columns defines which columns are displayed and their order.
	// Empty slice means use default column order.
	// Valid column names: "id", "timestamp", "state", "session", "window", "pane", "message", "pane_created", "level".
	Columns []string `json:"columns"`

	// SortBy specifies which column to sort by.
	// Empty string means use default sort (timestamp).
	// Valid values: "id", "timestamp", "state", "level", "session".
	SortBy string `json:"sortBy"`

	// SortOrder specifies sort direction: "asc" or "desc".
	// Empty string means use default sort order (desc).
	SortOrder string `json:"sortOrder"`

	// Filters contains active filter criteria.
	Filters Filter `json:"filters"`

	// ViewMode specifies the display layout: "compact" or "detailed".
	// Empty string means use default view mode (compact).
	ViewMode string `json:"viewMode"`
}

// DefaultSettings returns settings with all default values.
func DefaultSettings() *Settings {
	return &Settings{
		Columns:   DefaultColumns,
		SortBy:    SortByTimestamp,
		SortOrder: SortOrderDesc,
		Filters: Filter{
			Level:   "",
			State:   "",
			Session: "",
			Window:  "",
			Pane:    "",
		},
		ViewMode: ViewModeCompact,
	}
}

// Load reads settings from the config directory.
// If the settings file does not exist, returns default settings.
// If the settings file is corrupted, returns default settings with a warning.
func Load() (*Settings, error) {
	config.Load()
	settingsPath := getSettingsPath()

	colors.Debug("Loading settings from:", settingsPath)

	// If file doesn't exist, return defaults
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		colors.Debug("Settings file does not exist, using defaults")
		return DefaultSettings(), nil
	}

	var settings *Settings
	var loadErr error

	// Use file locking to prevent concurrent access
	// Lock the directory containing the settings file, not the file itself
	settingsDir := filepath.Dir(settingsPath)
	err := storage.WithLock(settingsDir+".lock", func() error {
		// Read and parse settings file
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			loadErr = fmt.Errorf("failed to read settings file: %w", err)
			return loadErr
		}

		settings = DefaultSettings()
		if err := json.Unmarshal(data, settings); err != nil {
			// Handle corrupted JSON gracefully - return defaults with warning
			colors.Warning("Failed to parse settings file:", err.Error(), "- using defaults")
			colors.Debug("JSON parse error:", err.Error())
			loadErr = nil // Don't return an error, just use defaults
			settings = DefaultSettings()
			return nil
		}

		// Validate settings
		if err := validate(settings); err != nil {
			loadErr = fmt.Errorf("invalid settings: %w", err)
			return loadErr
		}

		colors.Debug("Settings loaded successfully")
		return nil
	})

	if err != nil {
		return nil, err
	}

	return settings, loadErr
}

// Save writes settings to the config directory.
// Creates the config directory if it doesn't exist.
// Uses atomic writes to prevent corruption.
func Save(settings *Settings) error {
	// Load config to ensure config_dir is set
	config.Load()

	// Validate settings before saving
	if err := validate(settings); err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}

	// Create config directory if needed
	configDir := config.Get("config_dir", "")
	if configDir == "" {
		return fmt.Errorf("config_dir not configured")
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal settings to JSON with indentation for readability
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	settingsPath := getSettingsPath()
	colors.Debug("Saving settings to:", settingsPath)

	// Use file locking to prevent concurrent access
	// Lock the directory containing the settings file, not the file itself
	settingsDir := filepath.Dir(settingsPath)
	return storage.WithLock(settingsDir+".lock", func() error {
		// Write to temporary file first for atomic operation
		tempPath := settingsPath + ".tmp." + strconv.Itoa(rand.Intn(1000000))
		if err := os.WriteFile(tempPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write temporary settings file: %w", err)
		}

		// Atomically rename temp file to final destination
		if err := os.Rename(tempPath, settingsPath); err != nil {
			// Clean up temp file if rename fails
			_ = os.Remove(tempPath)
			return fmt.Errorf("failed to rename settings file: %w", err)
		}

		colors.Debug("Settings saved successfully")
		return nil
	})
}

// Init initializes the settings package.
// Creates the settings directory if needed and loads default settings.
// Returns the loaded settings.
func Init() (*Settings, error) {
	config.Load()
	colors.Debug("Initializing settings package")

	// Ensure config directory exists
	configDir := config.Get("config_dir", "")
	if configDir == "" {
		// Use default path
		home, _ := os.UserHomeDir()
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(xdgConfigHome, "tmux-intray")
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load settings (will use defaults if file doesn't exist)
	settings, err := Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	colors.Debug("Settings package initialized successfully")
	return settings, nil
}

// Reset resets settings to defaults by deleting the settings file.
// Returns default settings after deletion.
func Reset() (*Settings, error) {
	config.Load()
	settingsPath := getSettingsPath()

	colors.Debug("Resetting settings to defaults")

	// Use file locking to prevent concurrent access
	// Lock the directory containing the settings file, not the file itself
	settingsDir := filepath.Dir(settingsPath)
	err := storage.WithLock(settingsDir+".lock", func() error {
		// Check if file exists
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
			// File doesn't exist, nothing to do
			colors.Debug("Settings file does not exist, nothing to reset")
			return nil
		}

		// Delete settings file
		if err := os.Remove(settingsPath); err != nil {
			return fmt.Errorf("failed to delete settings file: %w", err)
		}

		colors.Debug("Settings file deleted successfully")
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Return default settings
	defaults := DefaultSettings()
	colors.Debug("Settings reset to defaults")
	return defaults, nil
}

// Validate checks that settings values are valid.
func Validate(settings *Settings) error {
	// Validate columns
	validColumns := map[string]bool{
		ColumnID: true, ColumnTimestamp: true, ColumnState: true,
		ColumnSession: true, ColumnWindow: true, ColumnPane: true,
		ColumnMessage: true, ColumnPaneCreated: true, ColumnLevel: true,
	}
	if len(settings.Columns) > 0 {
		for _, col := range settings.Columns {
			if !validColumns[col] {
				return fmt.Errorf("invalid column name: %s", col)
			}
		}
	}

	// Validate sortBy
	validSortBy := map[string]bool{
		SortByID: true, SortByTimestamp: true, SortByState: true,
		SortByLevel: true, SortBySession: true,
	}
	if settings.SortBy != "" && !validSortBy[settings.SortBy] {
		return fmt.Errorf("invalid sortBy value: %s", settings.SortBy)
	}

	// Validate sortOrder
	if settings.SortOrder != "" && settings.SortOrder != SortOrderAsc && settings.SortOrder != SortOrderDesc {
		return fmt.Errorf("invalid sortOrder value: %s", settings.SortOrder)
	}

	// Validate viewMode
	if settings.ViewMode != "" && settings.ViewMode != ViewModeCompact && settings.ViewMode != ViewModeDetailed {
		return fmt.Errorf("invalid viewMode value: %s", settings.ViewMode)
	}

	// Validate filters
	validLevels := map[string]bool{
		"": true, LevelFilterInfo: true, LevelFilterWarning: true,
		LevelFilterError: true, LevelFilterCritical: true,
	}
	if !validLevels[settings.Filters.Level] {
		return fmt.Errorf("invalid filter level: %s", settings.Filters.Level)
	}

	validStates := map[string]bool{
		"": true, StateFilterActive: true, StateFilterDismissed: true,
	}
	if !validStates[settings.Filters.State] {
		return fmt.Errorf("invalid filter state: %s", settings.Filters.State)
	}

	return nil
}

// validate is an alias for Validate for internal use.
func validate(settings *Settings) error {
	return Validate(settings)
}

// getSettingsPath returns the path to the settings.json file.
func getSettingsPath() string {
	configDir := config.Get("config_dir", "")
	if configDir == "" {
		// Fallback to XDG_CONFIG_HOME default
		home, _ := os.UserHomeDir()
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(xdgConfigHome, "tmux-intray")
	}
	return filepath.Join(configDir, "settings.json")
}
