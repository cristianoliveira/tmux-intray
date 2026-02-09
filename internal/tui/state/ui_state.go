package state

import (
	"github.com/charmbracelet/bubbles/viewport"
)

// UIState manages all UI-specific state for the TUI.
// This includes viewport management, cursor position, search and command modes,
// and other UI-related state that should be separated from business logic.
type UIState struct {
	// Viewport management
	viewport viewport.Model
	width    int
	height   int

	// Cursor and navigation
	cursor int

	// Search state
	searchMode  bool
	searchQuery string

	// Command state
	commandMode  bool
	commandQuery string

	// Input handling state
	pendingKey string
}

// NewUIState creates a new UIState instance with default values.
func NewUIState() *UIState {
	return &UIState{
		viewport: viewport.New(defaultViewportWidth, defaultViewportHeight),
		width:    defaultViewportWidth,
		height:   defaultViewportHeight,
		cursor:   0,
	}
}

// GetViewport returns the current viewport model.
func (u *UIState) GetViewport() *viewport.Model {
	return &u.viewport
}

// SetViewport updates the viewport model.
func (u *UIState) SetViewport(v viewport.Model) {
	u.viewport = v
}

// GetWidth returns the current width of the UI.
func (u *UIState) GetWidth() int {
	return u.width
}

// SetWidth updates the width of the UI and adjusts the viewport.
func (u *UIState) SetWidth(width int) {
	u.width = width
	if width <= 0 {
		u.width = defaultViewportWidth
	}
}

// GetHeight returns the current height of the UI.
func (u *UIState) GetHeight() int {
	return u.height
}

// SetHeight updates the height of the UI and adjusts the viewport.
func (u *UIState) SetHeight(height int) {
	u.height = height
	if height <= 0 {
		u.height = defaultViewportHeight
	}
}

// UpdateViewportSize updates the viewport dimensions based on the current width and height.
func (u *UIState) UpdateViewportSize() {
	viewportHeight := u.height - headerFooterLines
	u.viewport = viewport.New(u.width, viewportHeight)
}

// GetCursor returns the current cursor position.
func (u *UIState) GetCursor() int {
	return u.cursor
}

// SetCursor updates the cursor position.
func (u *UIState) SetCursor(cursor int) {
	u.cursor = cursor
	if u.cursor < 0 {
		u.cursor = 0
	}
}

// IsSearchMode returns whether search mode is active.
func (u *UIState) IsSearchMode() bool {
	return u.searchMode
}

// SetSearchMode activates or deactivates search mode.
func (u *UIState) SetSearchMode(active bool) {
	u.searchMode = active
	if !active {
		u.searchQuery = ""
	}
}

// GetSearchQuery returns the current search query.
func (u *UIState) GetSearchQuery() string {
	return u.searchQuery
}

// SetSearchQuery updates the search query.
func (u *UIState) SetSearchQuery(query string) {
	u.searchQuery = query
}

// AppendToSearchQuery appends a rune to the search query.
func (u *UIState) AppendToSearchQuery(r rune) {
	u.searchQuery += string(r)
}

// BackspaceSearchQuery removes the last character from the search query.
func (u *UIState) BackspaceSearchQuery() {
	if len(u.searchQuery) > 0 {
		u.searchQuery = u.searchQuery[:len(u.searchQuery)-1]
	}
}

// IsCommandMode returns whether command mode is active.
func (u *UIState) IsCommandMode() bool {
	return u.commandMode
}

// SetCommandMode activates or deactivates command mode.
func (u *UIState) SetCommandMode(active bool) {
	u.commandMode = active
	if !active {
		u.commandQuery = ""
	}
}

// GetCommandQuery returns the current command query.
func (u *UIState) GetCommandQuery() string {
	return u.commandQuery
}

// SetCommandQuery updates the command query.
func (u *UIState) SetCommandQuery(query string) {
	u.commandQuery = query
}

// AppendToCommandQuery appends a rune to the command query.
func (u *UIState) AppendToCommandQuery(r rune) {
	u.commandQuery += string(r)
}

// BackspaceCommandQuery removes the last character from the command query.
func (u *UIState) BackspaceCommandQuery() {
	if len(u.commandQuery) > 0 {
		u.commandQuery = u.commandQuery[:len(u.commandQuery)-1]
	}
}

// GetPendingKey returns the current pending key.
func (u *UIState) GetPendingKey() string {
	return u.pendingKey
}

// SetPendingKey updates the pending key.
func (u *UIState) SetPendingKey(key string) {
	u.pendingKey = key
}

// ClearPendingKey clears the pending key.
func (u *UIState) ClearPendingKey() {
	u.pendingKey = ""
}

// MoveCursorUp moves the cursor up one position if possible.
func (u *UIState) MoveCursorUp(listLen int) {
	if u.cursor > 0 {
		u.cursor--
	}
}

// MoveCursorDown moves the cursor down one position if possible.
func (u *UIState) MoveCursorDown(listLen int) {
	if u.cursor < listLen-1 {
		u.cursor++
	}
}

// EnsureCursorVisible adjusts the viewport to ensure the cursor is visible.
func (u *UIState) EnsureCursorVisible(listLen int) {
	if listLen == 0 {
		return
	}

	// Get the current viewport line offset
	lineOffset := u.viewport.YOffset

	// Calculate the viewport height
	viewportHeight := u.viewport.Height

	// If cursor is above viewport, scroll up
	if u.cursor < lineOffset {
		u.viewport.LineUp(lineOffset - u.cursor)
	}

	// If cursor is below viewport, scroll down
	if u.cursor >= lineOffset+viewportHeight {
		u.viewport.LineDown(u.cursor - (lineOffset + viewportHeight) + 1)
	}
}

// AdjustCursorBounds ensures the cursor is within valid bounds.
func (u *UIState) AdjustCursorBounds(listLen int) {
	if listLen == 0 {
		u.cursor = 0
		return
	}
	if u.cursor >= listLen {
		u.cursor = listLen - 1
	}
	if u.cursor < 0 {
		u.cursor = 0
	}
}

// ResetCursor resets the cursor to the first item.
func (u *UIState) ResetCursor() {
	u.cursor = 0
}
