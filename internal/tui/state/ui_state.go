package state

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// UIState manages all UI-specific state for the TUI.
// This includes viewport management, cursor position, search mode,
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

	// Error state
	errorMessage string

	// Input handling state
	pendingKey string

	// View mode management
	viewMode model.ViewMode

	// Group by configuration
	groupBy model.GroupBy

	// Expansion state
	expandLevel    int
	expansionState map[string]bool
}

// NewUIState creates a new UIState instance with default values.
func NewUIState() *UIState {
	return &UIState{
		viewport:       viewport.New(defaultViewportWidth, defaultViewportHeight),
		width:          defaultViewportWidth,
		height:         defaultViewportHeight,
		cursor:         0,
		viewMode:       model.ViewModeDetailed,
		groupBy:        model.GroupByNone,
		expandLevel:    1, // Default expand level
		expansionState: make(map[string]bool),
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

// GetError returns the current error message.
func (u *UIState) GetError() string {
	return u.errorMessage
}

// SetError sets the error message.
func (u *UIState) SetError(msg string) {
	u.errorMessage = msg
}

// ClearError clears the error message.
func (u *UIState) ClearError() {
	u.errorMessage = ""
}

// HasError returns whether there is an active error.
func (u *UIState) HasError() bool {
	return u.errorMessage != ""
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
		u.viewport.ScrollUp(lineOffset - u.cursor)
	}

	// If cursor is below viewport, scroll down
	if u.cursor >= lineOffset+viewportHeight {
		u.viewport.ScrollDown(u.cursor - (lineOffset + viewportHeight) + 1)
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

// GetViewMode returns the current view mode.
func (u *UIState) GetViewMode() model.ViewMode {
	return u.viewMode
}

// SetViewMode sets the current view mode.
func (u *UIState) SetViewMode(mode model.ViewMode) {
	u.viewMode = mode
}

// CycleViewMode cycles to the next available view mode.
func (u *UIState) CycleViewMode() {
	// Cycle through modes: compact -> detailed -> grouped -> compact
	switch u.viewMode {
	case model.ViewModeCompact:
		u.viewMode = model.ViewModeDetailed
	case model.ViewModeDetailed:
		u.viewMode = model.ViewModeGrouped
	case model.ViewModeGrouped:
		u.viewMode = model.ViewModeCompact
	default:
		u.viewMode = model.ViewModeDetailed
	}
}

// GetGroupBy returns the current grouping mode.
func (u *UIState) GetGroupBy() model.GroupBy {
	return u.groupBy
}

// SetGroupBy sets the grouping mode.
func (u *UIState) SetGroupBy(groupBy model.GroupBy) {
	u.groupBy = groupBy
}

// GetExpandLevel returns the default expansion level for tree nodes.
func (u *UIState) GetExpandLevel() int {
	return u.expandLevel
}

// SetExpandLevel sets the default expansion level for tree nodes.
func (u *UIState) SetExpandLevel(level int) {
	u.expandLevel = level
}

// IsGroupedView returns true if the current view mode is grouped.
func (u *UIState) IsGroupedView() bool {
	return u.viewMode == model.ViewModeGrouped
}

// GetExpansionState returns the saved expansion state for tree nodes.
func (u *UIState) GetExpansionState() map[string]bool {
	return u.expansionState
}

// SetExpansionState sets the expansion state for tree nodes.
func (u *UIState) SetExpansionState(state map[string]bool) {
	u.expansionState = state
}

// UpdateExpansionState updates the expansion state for a specific node.
func (u *UIState) UpdateExpansionState(nodeIdentifier string, expanded bool) {
	if u.expansionState == nil {
		u.expansionState = make(map[string]bool)
	}
	u.expansionState[nodeIdentifier] = expanded
}

// GetSelectedNotification returns the notification at the current cursor position.
func (u *UIState) GetSelectedNotification(notifications []notification.Notification, visibleNodes []*model.TreeNode) (notification.Notification, bool) {
	if u.isGroupedView() {
		// In grouped view, get the notification from the visible nodes
		if u.cursor < 0 || u.cursor >= len(visibleNodes) {
			return notification.Notification{}, false
		}
		node := visibleNodes[u.cursor]
		if node == nil || node.Notification == nil {
			return notification.Notification{}, false
		}
		return *node.Notification, true
	}

	// In flat view, get from the notifications list directly
	if u.cursor < 0 || u.cursor >= len(notifications) {
		return notification.Notification{}, false
	}
	return notifications[u.cursor], true
}

// GetSelectedNode returns the tree node at the current cursor position (in grouped view).
func (u *UIState) GetSelectedNode(visibleNodes []*model.TreeNode) *model.TreeNode {
	if !u.isGroupedView() {
		return nil
	}
	if u.cursor < 0 || u.cursor >= len(visibleNodes) {
		return nil
	}
	return visibleNodes[u.cursor]
}

// GetViewportDimensions returns the current viewport width and height.
func (u *UIState) GetViewportDimensions() (width, height int) {
	return u.width, u.viewport.Height
}

// SetViewportDimensions sets the viewport dimensions.
func (u *UIState) SetViewportDimensions(width, height int) {
	u.width = width
	u.height = height
	if width <= 0 {
		u.width = defaultViewportWidth
	}
	if height <= 0 {
		u.height = defaultViewportHeight
	}
	// Update viewport size
	viewportHeight := u.height - headerFooterLines
	u.viewport = viewport.New(u.width, viewportHeight)
}

// GetDimensions returns the total terminal width and height.
func (u *UIState) GetDimensions() (width, height int) {
	return u.width, u.height
}

// SetDimensions sets the total terminal dimensions.
func (u *UIState) SetDimensions(width, height int) {
	u.width = width
	u.height = height
	if width <= 0 {
		u.width = defaultViewportWidth
	}
	if height <= 0 {
		u.height = defaultViewportHeight
	}
}

// Save saves the current UI state to persistent storage.
func (u *UIState) Save() error {
	// UI state is saved through the Model's saveSettings() method
	// This is a placeholder for future direct UI state persistence
	return nil
}

// Load loads UI state from persistent storage.
func (u *UIState) Load() error {
	// UI state is loaded through the Model's FromState() method
	// This is a placeholder for future direct UI state persistence
	return nil
}

// ToDTO converts the UI state to a data transfer object for persistence.
func (u *UIState) ToDTO() model.UIDTO {
	return model.UIDTO{
		ViewMode:       u.viewMode,
		GroupBy:        u.groupBy,
		ExpandLevel:    u.expandLevel,
		ExpansionState: u.expansionState,
	}
}

// FromDTO applies UI state from a data transfer object.
func (u *UIState) FromDTO(dto model.UIDTO) error {
	// Preserve current values for unset fields
	if dto.ViewMode != "" {
		u.viewMode = dto.ViewMode
	}
	if dto.GroupBy != "" {
		u.groupBy = dto.GroupBy
	}
	// ExpandLevel is int, so 0 could be valid value
	// Only update if explicitly set
	if dto.ExpandLevelSet {
		u.expandLevel = dto.ExpandLevel
	}
	if dto.ExpansionState != nil {
		u.expansionState = dto.ExpansionState
	}
	return nil
}

// isGroupedView is a helper method to check if view mode is grouped.
func (u *UIState) isGroupedView() bool {
	return u.viewMode == model.ViewModeGrouped
}
