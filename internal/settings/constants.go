// Package settings provides TUI user preferences persistence.
package settings

import "os"

// File permission constants
const (
	// FileModeDir is the permission for directories (rwxr-xr-x)
	// Owner: read/write/execute, Group/others: read/execute
	FileModeDir os.FileMode = 0755
	// FileModeFile is the permission for data files (rw-r--r--)
	// Owner: read/write, Group/others: read only
	FileModeFile os.FileMode = 0644

	// FileExtTOML is the file extension for TOML files.
	// Used for user settings persistence.
	FileExtTOML = ".toml"
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
	ViewModeGrouped  = "grouped"
	ViewModeSearch   = "search"
)

// Group by constants.
const (
	GroupByNone        = "none"
	GroupBySession     = "session"
	GroupByWindow      = "window"
	GroupByPane        = "pane"
	GroupByMessage     = "message"
	GroupByPaneMessage = "pane_message"
)

// Expansion level limits.
const (
	MinExpandLevel = 0
	MaxExpandLevel = 3
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

// Read filter constants.
const (
	ReadFilterRead   = "read"
	ReadFilterUnread = "unread"
)
