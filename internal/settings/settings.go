// Package settings provides TUI user preferences persistence.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cristianoliveira/tmux-intray/internal/config"
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
func Load() (*Settings, error) {
	config.Load()
	settingsPath := getSettingsPath()

	// If file doesn't exist, return defaults
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return DefaultSettings(), nil
	}

	// Read and parse settings file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	settings := DefaultSettings()
	if err := json.Unmarshal(data, settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	// Validate settings
	if err := validate(settings); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	return settings, nil
}

// Save writes settings to the config directory.
// Creates the config directory if it doesn't exist.
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

	// Write to settings file
	settingsPath := getSettingsPath()
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// validate checks that settings values are valid.
func validate(settings *Settings) error {
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
