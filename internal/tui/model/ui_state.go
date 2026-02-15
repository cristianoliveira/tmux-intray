// Package model provides interface contracts for TUI components.
// These interfaces define the contracts between different parts of the TUI system.
package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// UIState defines the interface for view state management.
// It manages the interactive state of the TUI including cursor position, modes, and settings.
type UIState interface {
	// GetCursor returns the current cursor position in the visible list.
	GetCursor() int

	// SetCursor sets the cursor position.
	// Bounds checking should be performed to ensure valid position.
	SetCursor(pos int)

	// ResetCursor resets the cursor to the first item.
	ResetCursor()

	// AdjustCursorBounds ensures the cursor is within valid range.
	AdjustCursorBounds(listLength int)

	// GetSearchMode returns true if search mode is active.
	GetSearchMode() bool

	// SetSearchMode enables or disables search mode.
	SetSearchMode(enabled bool)

	// GetSearchQuery returns the current search query string.
	GetSearchQuery() string

	// SetSearchQuery sets the search query string.
	SetSearchQuery(query string)

	// GetCommandMode returns true if command mode is active.
	GetCommandMode() bool

	// SetCommandMode enables or disables command mode.
	SetCommandMode(enabled bool)

	// GetCommandQuery returns the current command query string.
	GetCommandQuery() string

	// SetCommandQuery sets the command query string.
	SetCommandQuery(query string)

	// GetViewMode returns the current view mode (compact, detailed, grouped).
	GetViewMode() ViewMode

	// SetViewMode sets the current view mode.
	SetViewMode(mode ViewMode)

	// CycleViewMode cycles to the next available view mode.
	CycleViewMode()

	// GetGroupBy returns the current grouping mode (none, session, window, pane).
	GetGroupBy() GroupBy

	// SetGroupBy sets the grouping mode.
	SetGroupBy(groupBy GroupBy)

	// GetExpandLevel returns the default expansion level for tree nodes.
	GetExpandLevel() int

	// SetExpandLevel sets the default expansion level for tree nodes.
	SetExpandLevel(level int)

	// IsGroupedView returns true if the current view mode is grouped.
	IsGroupedView() bool

	// GetExpansionState returns the saved expansion state for tree nodes.
	// Keys are node identifiers, values indicate expanded state.
	GetExpansionState() map[string]bool

	// SetExpansionState sets the expansion state for tree nodes.
	SetExpansionState(state map[string]bool)

	// UpdateExpansionState updates the expansion state for a specific node.
	UpdateExpansionState(nodeIdentifier string, expanded bool)

	// GetSelectedNotification returns the notification at the current cursor position.
	// Returns the notification and true if found, or empty notification and false if not.
	GetSelectedNotification(notifications []notification.Notification, visibleNodes []*TreeNode) (notification.Notification, bool)

	// GetSelectedNode returns the tree node at the current cursor position (in grouped view).
	// Returns nil if not in grouped view or cursor is out of bounds.
	GetSelectedNode(visibleNodes []*TreeNode) *TreeNode

	// GetViewportDimensions returns the current viewport width and height.
	GetViewportDimensions() (width, height int)

	// SetViewportDimensions sets the viewport dimensions.
	SetViewportDimensions(width, height int)

	// GetDimensions returns the total terminal width and height.
	GetDimensions() (width, height int)

	// SetDimensions sets the total terminal dimensions.
	SetDimensions(width, height int)

	// Save saves the current UI state to persistent storage.
	Save() error

	// Load loads UI state from persistent storage.
	Load() error

	// ToDTO converts the UI state to a data transfer object for persistence.
	ToDTO() UIDTO

	// FromDTO applies UI state from a data transfer object.
	FromDTO(dto UIDTO) error
}

// ViewMode represents the display mode for notifications.
type ViewMode string

const (
	// ViewModeCompact shows minimal information per notification.
	ViewModeCompact ViewMode = "compact"

	// ViewModeDetailed shows full information per notification.
	ViewModeDetailed ViewMode = "detailed"

	// ViewModeGrouped shows notifications in a hierarchical tree.
	ViewModeGrouped ViewMode = "grouped"
)

// GroupBy represents the grouping mode for notifications.
type GroupBy string

const (
	// GroupByNone shows notifications in a flat list.
	GroupByNone GroupBy = "none"

	// GroupBySession groups notifications by session.
	GroupBySession GroupBy = "session"

	// GroupByWindow groups notifications by session and window.
	GroupByWindow GroupBy = "window"

	// GroupByPane groups notifications by session, window, and pane.
	GroupByPane GroupBy = "pane"

	// GroupByMessage groups notifications by message text.
	GroupByMessage GroupBy = "message"
)

// UIDTO is a data transfer object for UI state persistence.
type UIDTO struct {
	// ViewMode is the current view mode.
	ViewMode ViewMode

	// GroupBy is the current grouping mode.
	GroupBy GroupBy

	// ExpandLevel is the default expansion level for tree nodes.
	ExpandLevel int

	// ExpandLevelSet indicates whether ExpandLevel was explicitly set.
	ExpandLevelSet bool

	// ExpansionState maps node identifiers to their expanded state.
	ExpansionState map[string]bool

	// Columns are the columns to display in detailed view.
	Columns []string

	// SortBy is the field to sort notifications by.
	SortBy string

	// SortOrder is the sort order (asc or desc).
	SortOrder string

	// Filters are the active notification filters.
	Filters Filters
}

// Filters represents notification filters.
type Filters struct {
	// Level filters by notification level (info, warning, error).
	Level string

	// State filters by notification state (active, dismissed).
	State string

	// Session filters by session ID.
	Session string

	// Window filters by window ID.
	Window string

	// Pane filters by pane ID.
	Pane string
}

// CommandResult represents the result of executing a command.
type CommandResult struct {
	// Message is a user-facing message describing the result.
	Message string

	// Quit indicates whether the TUI should exit after this command.
	Quit bool

	// Cmd is a bubbletea command to execute (optional).
	Cmd tea.Cmd

	// Error indicates whether the command failed.
	Error bool
}
