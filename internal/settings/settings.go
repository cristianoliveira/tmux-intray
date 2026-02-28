// Package settings provides TUI user preferences persistence.
package settings

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/pelletier/go-toml/v2"
)

// convertCamelToSnake replaces known camelCase keys with snake_case in TOML data.
func convertCamelToSnake(data []byte) []byte {
	replacements := map[string]string{
		"sortBy":                "sort_by",
		"sortOrder":             "sort_order",
		"viewMode":              "view_mode",
		"groupBy":               "group_by",
		"defaultExpandLevel":    "default_expand_level",
		"autoExpandUnread":      "auto_expand_unread",
		"expansionState":        "expansion_state",
		"groupHeader":           "group_header",
		"showTimeRange":         "show_time_range",
		"showLevelBadges":       "show_level_badges",
		"showSourceAggregation": "show_source_aggregation",
		"badgeColors":           "badge_colors",
		"activeTab":             "active_tab",
	}
	result := string(data)
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old+" =", new+" =")
		result = strings.ReplaceAll(result, "["+old+"]", "["+new+"]")
	}
	return []byte(result)
}

// Filter defines active filter criteria for notification display.
type Filter struct {
	// Level filters notifications by severity level.
	// Empty string means no filter (show all levels).
	// Valid values: "info", "warning", "error", "critical", "".
	Level string `toml:"level"`

	// State filters notifications by state.
	// Empty string means no filter (show all states).
	// Valid values: "active", "dismissed", "".
	State string `toml:"state"`

	// Read filters notifications by read status.
	// Empty string means no filter (show all notifications).
	// Valid values: "read", "unread", "".
	Read string `toml:"read"`

	// Session filters notifications by tmux session name.
	// Empty string means no filter (show all sessions).
	Session string `toml:"session"`

	// Window filters notifications by tmux window ID.
	// Empty string means no filter (show all windows).
	Window string `toml:"window"`

	// Pane filters notifications by tmux pane ID.
	// Empty string means no filter (show all panes).
	Pane string `toml:"pane"`
}

// GroupHeaderOptions controls how group headers render additional context.
type GroupHeaderOptions struct {
	// ShowTimeRange toggles whether grouped nodes display earliest/latest ages.
	ShowTimeRange bool `toml:"show_time_range"`

	// ShowLevelBadges toggles whether grouped nodes display level badges.
	ShowLevelBadges bool `toml:"show_level_badges"`

	// ShowSourceAggregation toggles whether grouped nodes display source info.
	ShowSourceAggregation bool `toml:"show_source_aggregation"`

	// BadgeColors defines ANSI color codes per level key.
	// Keys: info, warning, error, critical.
	BadgeColors map[string]string `toml:"badge_colors"`
}

// DefaultGroupHeaderOptions returns default rendering options for group headers.
func DefaultGroupHeaderOptions() GroupHeaderOptions {
	return GroupHeaderOptions{
		ShowTimeRange:         true,
		ShowLevelBadges:       true,
		ShowSourceAggregation: false,
		BadgeColors:           defaultBadgeColors(),
	}
}

// Clone returns a copy of the options with a deep copy of BadgeColors.
func (o GroupHeaderOptions) Clone() GroupHeaderOptions {
	clone := GroupHeaderOptions{
		ShowTimeRange:         o.ShowTimeRange,
		ShowLevelBadges:       o.ShowLevelBadges,
		ShowSourceAggregation: o.ShowSourceAggregation,
		BadgeColors:           make(map[string]string, len(o.BadgeColors)),
	}
	for level, color := range o.BadgeColors {
		clone.BadgeColors[level] = color
	}
	clone.normalize()
	return clone
}

func defaultBadgeColors() map[string]string {
	return map[string]string{
		LevelFilterInfo:     colors.Blue,
		LevelFilterWarning:  colors.Yellow,
		LevelFilterError:    colors.Red,
		LevelFilterCritical: colors.Red,
	}
}

func (o *GroupHeaderOptions) normalize() {
	if o == nil {
		return
	}
	if o.BadgeColors == nil {
		o.BadgeColors = make(map[string]string)
	}
	for level, color := range defaultBadgeColors() {
		if o.BadgeColors[level] == "" {
			o.BadgeColors[level] = color
		}
	}
}

// Validate ensures the options structure is well-formed.
func (o GroupHeaderOptions) Validate() error {
	requiredLevels := []string{LevelFilterInfo, LevelFilterWarning, LevelFilterError, LevelFilterCritical}
	for _, level := range requiredLevels {
		if o.BadgeColors[level] == "" {
			return fmt.Errorf("missing badge color for level: %s", level)
		}
	}
	return nil
}

// Settings holds TUI user preferences persisted to disk.
//
// TOML Schema:
//
//	{
//	  "columns": ["id", "timestamp", "state", "level", "session", "window", "pane", "message"],
//	  "sortBy": "timestamp",
//	  "sortOrder": "desc",
//	  "unreadFirst": true,
//	  "filters": {
//	    "level": "",
//	    "state": "",
//	    "read": "",
//	    "session": "",
//	    "window": "",
//	    "pane": ""
//	  },
//	  "viewMode": "grouped",
//	  "groupBy": "none",
//	  "defaultExpandLevel": 1,
//	  "expansionState": {}
//	}
//
// Valid viewMode values: "compact", "detailed", "grouped".
//
// Settings are stored at ~/.config/tmux-intray/tui.toml by default.
type Settings struct {
	// Columns defines which columns are displayed and their order.
	// Empty slice means use default column order.
	// Valid column names: "id", "timestamp", "state", "session", "window", "pane", "message", "pane_created", "level".
	Columns []string `toml:"columns"`

	// SortBy specifies which column to sort by.
	// Empty string means use default sort (timestamp).
	// Valid values: "id", "timestamp", "state", "level", "session".
	SortBy string `toml:"sort_by"`

	// SortOrder specifies sort direction: "asc" or "desc".
	// Empty string means use default sort order (desc).
	SortOrder string `toml:"sort_order"`

	// UnreadFirst controls whether unread notifications are sorted first.
	// When true, unread notifications appear before read notifications,
	// and then the configured sort_by and sort_order are applied within each group.
	// Defaults to true for backward compatibility.
	UnreadFirst bool `toml:"unread_first"`

	// Filters contains active filter criteria.
	Filters Filter `toml:"filters"`

	// ViewMode specifies the display layout: "compact", "detailed", or "grouped".
	// Empty string means use default view mode (grouped).
	ViewMode string `toml:"view_mode"`

	// GroupBy specifies the grouping mode: "none", "session", "window", "pane", "message", or "pane_message".
	// Empty string means use default grouping (none).
	GroupBy string `toml:"group_by"`

	// DefaultExpandLevel controls the default grouping expansion level (0-3).
	// Use 0 to collapse all groups by default.
	DefaultExpandLevel int `toml:"default_expand_level"`

	// AutoExpandUnread controls whether groups with unread notifications are auto-expanded.
	AutoExpandUnread bool `toml:"auto_expand_unread"`

	// ExpansionState stores explicit expansion overrides by node path.
	ExpansionState map[string]bool `toml:"expansion_state"`

	// GroupHeader configures group header rendering.
	GroupHeader GroupHeaderOptions `toml:"group_header"`

	// ShowHelp controls whether to show help text in the footer.
	// Defaults to true for backward compatibility.
	ShowHelp bool `toml:"show_help"`

	// ActiveTab is the selected tab contract for notifications lanes.
	// Valid values: "recents", "all".
	ActiveTab Tab `toml:"active_tab"`
}

// DefaultSettings returns settings with all default values.
func DefaultSettings() *Settings {
	return &Settings{
		Columns:     DefaultColumns,
		SortBy:      SortByTimestamp,
		SortOrder:   SortOrderDesc,
		UnreadFirst: true, // Default to true to maintain current behavior (unread first)
		Filters: Filter{
			Level:   "",
			State:   "",
			Read:    "",
			Session: "",
			Window:  "",
			Pane:    "",
		},
		ViewMode:           ViewModeGrouped,
		GroupBy:            GroupByNone,
		DefaultExpandLevel: 1,
		AutoExpandUnread:   false, // Default to false to avoid unexpected behavior
		ExpansionState:     map[string]bool{},
		GroupHeader:        DefaultGroupHeaderOptions(),
		ShowHelp:           true,
		ActiveTab:          DefaultTab(),
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
		// Convert camelCase keys to snake_case for backward compatibility
		data = convertCamelToSnake(data)

		settings = DefaultSettings()
		if err := toml.Unmarshal(data, settings); err != nil {
			// Handle corrupted TOML gracefully - return defaults with warning
			colors.Warning("Failed to parse settings file:", err.Error(), "- using defaults")
			colors.Debug("TOML parse error:", err.Error())
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
// Preconditions: settings must be non-nil and valid.
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
	if err := os.MkdirAll(configDir, FileModeDir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal settings to TOML
	data, err := toml.Marshal(settings)
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
		if err := os.WriteFile(tempPath, data, FileModeFile); err != nil {
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

	configDir := resolveConfigDir()
	if err := os.MkdirAll(configDir, FileModeDir); err != nil {
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

// getSettingsPath is implemented in path.go.
