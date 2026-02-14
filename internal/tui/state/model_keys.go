package state

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

type errorMsg struct{}

// errorMsgAfter returns a tea.Cmd that sends an errorMsg after the specified duration.
func errorMsgAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return errorMsg{}
	})
}

// handleKeyMsg processes keyboard input for the TUI.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handled, cmd := m.handlePendingKey(msg); handled {
		return m, cmd
	}

	if nextModel, cmd := m.handleKeyType(msg); cmd != nil || nextModel != nil {
		if nextModel == nil {
			nextModel = m
		}
		return nextModel, cmd
	}

	if m.uiState.IsCommandMode() {
		return m, nil
	}

	return m.handleKeyBinding(msg.String())
}

// handleKeyType handles key type-based actions (Ctrl+C, Esc, Enter, etc.).
func (m *Model) handleKeyType(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m.handleCtrlC()
	case tea.KeyEsc:
		return m.handleEsc()
	case tea.KeyEnter:
		return m.handleEnter()
	case tea.KeyRunes:
		m.handleRunes(msg)
		return nil, nil
	case tea.KeyBackspace:
		m.handleBackspace()
		return nil, nil
	case tea.KeyUp, tea.KeyDown:
		return nil, nil
	}
	return nil, nil
}

// canProcessBinding returns true if the current state allows processing mode-restricted bindings.
func (m *Model) canProcessBinding() bool {
	return !m.uiState.IsSearchMode() && !m.uiState.IsCommandMode()
}

// handleKeyBinding handles string-based key bindings.
func (m *Model) handleKeyBinding(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j":
		m.handleMoveDown()
		return m, nil
	case "k":
		m.handleMoveUp()
		return m, nil
	case "G":
		if m.canProcessBinding() {
			m.handleMoveBottom()
		}
		return m, nil
	case "g":
		if m.canProcessBinding() {
			m.uiState.SetPendingKey("g")
		}
		return m, nil
	case "/":
		m.handleSearchMode()
		return m, nil
	case ":":
		if m.canProcessBinding() {
			m.handleCommandMode()
		}
		return m, nil
	case "d":
		return m, m.handleDismiss()
	case "r":
		return m, m.markSelectedRead()
	case "u":
		return m, m.markSelectedUnread()
	case "v":
		if m.canProcessBinding() {
			m.cycleViewMode()
		}
		return m, nil
	case "h":
		m.handleCollapseNode()
		return m, nil
	case "l":
		m.handleExpandNode()
		return m, nil
	case "z":
		if m.canProcessBinding() && m.isGroupedView() {
			m.uiState.SetPendingKey("z")
		}
		return m, nil
	case "i":
		// In search mode, 'i' is handled by KeyRunes
		// This is a no-op but kept for documentation
		return m, nil
	case "q":
		return m.handleQuit()
	}
	return m, nil
}

func (m *Model) handlePendingKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.uiState.IsSearchMode() || m.uiState.IsCommandMode() {
		m.uiState.ClearPendingKey()
	} else if m.uiState.GetPendingKey() != "" {
		if msg.String() == "a" && m.uiState.GetPendingKey() == "z" && m.isGroupedView() {
			m.uiState.ClearPendingKey()
			m.toggleFold()
			return true, nil
		}
		if msg.String() == "g" && m.uiState.GetPendingKey() == "g" {
			m.uiState.ClearPendingKey()
			m.handleMoveTop()
			return true, nil
		}
		if !(m.uiState.GetPendingKey() == "z" && msg.String() == "z") {
			m.uiState.ClearPendingKey()
		}
	}
	return false, nil
}

func (m *Model) handleCtrlC() (tea.Model, tea.Cmd) {
	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		return m, tea.Batch(tea.Quit, errorMsgAfter(errorClearDuration))
	}
	return m, tea.Quit
}

func (m *Model) handleEsc() (tea.Model, tea.Cmd) {
	if m.uiState.IsSearchMode() {
		m.uiState.SetSearchMode(false)
		m.applySearchFilter()
		m.uiState.ResetCursor()
	} else if m.uiState.IsCommandMode() {
		m.uiState.SetCommandMode(false)
	} else {
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.uiState.IsSearchMode() {
		m.uiState.SetSearchMode(false)
		return m, nil
	}
	if m.uiState.IsCommandMode() {
		cmd := m.executeCommandViaService()
		m.uiState.SetCommandMode(false)
		return m, cmd
	}
	if m.isGroupedView() && m.toggleNodeExpansion() {
		return m, nil
	}
	return m, m.handleJump()
}

func (m *Model) handleRunes(msg tea.KeyMsg) {
	if m.uiState.IsSearchMode() {
		// In search mode, append runes to search query
		for _, r := range msg.Runes {
			m.uiState.AppendToSearchQuery(r)
		}
		m.applySearchFilter()
		m.uiState.ResetCursor()
	} else if m.uiState.IsCommandMode() {
		// In command mode, append runes to command query
		for _, r := range msg.Runes {
			m.uiState.AppendToCommandQuery(r)
		}
	}
}

func (m *Model) handleBackspace() {
	if m.uiState.IsSearchMode() {
		if len(m.uiState.GetSearchQuery()) > 0 {
			m.uiState.BackspaceSearchQuery()
			m.applySearchFilter()
			m.uiState.ResetCursor()
		}
	} else if m.uiState.IsCommandMode() {
		if len(m.uiState.GetCommandQuery()) > 0 {
			m.uiState.BackspaceCommandQuery()
		}
	}
}

func (m *Model) handleMoveDown() {
	listLen := m.currentListLen()
	m.uiState.MoveCursorDown(listLen)
	m.updateViewportContent()
	m.uiState.EnsureCursorVisible(listLen)
}

func (m *Model) handleMoveUp() {
	listLen := m.currentListLen()
	m.uiState.MoveCursorUp(listLen)
	m.updateViewportContent()
	m.uiState.EnsureCursorVisible(listLen)
}

func (m *Model) handleMoveTop() {
	listLen := m.currentListLen()
	if listLen == 0 {
		return
	}
	m.uiState.SetCursor(0)
	m.updateViewportContent()
	m.uiState.EnsureCursorVisible(listLen)
}

func (m *Model) handleMoveBottom() {
	listLen := m.currentListLen()
	if listLen == 0 {
		return
	}
	m.uiState.SetCursor(listLen - 1)
	m.updateViewportContent()
	m.uiState.EnsureCursorVisible(listLen)
}

func (m *Model) handleSearchMode() {
	m.uiState.SetSearchMode(true)
	m.applySearchFilter()
	m.uiState.ResetCursor()
}

func (m *Model) handleCommandMode() {
	if !m.uiState.IsSearchMode() && !m.uiState.IsCommandMode() {
		m.uiState.SetCommandMode(true)
	}
}

func (m *Model) handleCollapseNode() {
	node := m.selectedVisibleNode()
	if node != nil {
		m.treeService.CollapseNode(node)
		m.invalidateCache()
		m.updateViewportContent()
	}
}

func (m *Model) handleExpandNode() {
	node := m.selectedVisibleNode()
	if node != nil {
		m.treeService.ExpandNode(node)
		m.invalidateCache()
		m.updateViewportContent()
	}
}

func (m *Model) handleQuit() (tea.Model, tea.Cmd) {
	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		return m, tea.Batch(tea.Quit, errorMsgAfter(errorClearDuration))
	}
	// Quit
	return m, tea.Quit
}

func (m *Model) handleSaveSettingsSuccess(msg saveSettingsSuccessMsg) (tea.Model, tea.Cmd) {
	// Settings saved successfully - already displayed info message in saveSettings
	return m, nil
}

func (m *Model) handleSaveSettingsFailed(msg saveSettingsFailedMsg) (tea.Model, tea.Cmd) {
	// Settings save failed - already displayed warning message in saveSettings
	return m, nil
}

func (m *Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.uiState.SetWidth(msg.Width)
	m.uiState.SetHeight(msg.Height)
	// Initialize or update viewport dimensions
	m.uiState.UpdateViewportSize()
	// Update viewport content
	m.updateViewportContent()
	return m, nil
}

// handleDismiss handles the dismiss action for the selected notification.
func (m *Model) handleDismiss() tea.Cmd {
	if m.currentListLen() == 0 {
		return nil
	}

	// Get the selected notification
	selected, ok := m.selectedNotification()
	if !ok {
		return nil
	}

	// Dismiss the notification using storage
	id := strconv.Itoa(selected.ID)
	if err := storage.DismissNotification(id); err != nil {
		m.errorHandler.Error(fmt.Sprintf("Failed to dismiss notification: %v", err))
		return errorMsgAfter(errorClearDuration)
	}

	// Save the current cursor position before reload
	oldCursor := m.uiState.GetCursor()

	// Reload notifications to get updated state (preserve cursor)
	if err := m.loadNotifications(true); err != nil {
		m.errorHandler.Error(fmt.Sprintf("Failed to reload notifications: %v", err))
		return errorMsgAfter(errorClearDuration)
	}

	// Restore cursor to the saved position, adjusting for bounds
	listLen := m.currentListLen()
	if listLen == 0 {
		m.uiState.SetCursor(0)
	} else {
		m.uiState.SetCursor(oldCursor)
		// Ensure cursor is within bounds
		m.adjustCursorBounds()
	}

	// Update viewport content
	m.updateViewportContent()

	return nil
}

// markSelectedRead marks the selected notification as read.
func (m *Model) markSelectedRead() tea.Cmd {
	if m.currentListLen() == 0 {
		return nil
	}

	selected, ok := m.selectedNotification()
	if !ok {
		return nil
	}

	// Save the notification ID to restore cursor later
	selectedID := selected.ID

	id := strconv.Itoa(selected.ID)
	if err := storage.MarkNotificationRead(id); err != nil {
		m.errorHandler.Error(fmt.Sprintf("Failed to mark notification read: %v", err))
		return errorMsgAfter(errorClearDuration)
	}

	if err := m.loadNotifications(true); err != nil {
		m.errorHandler.Error(fmt.Sprintf("Failed to reload notifications: %v", err))
		return errorMsgAfter(errorClearDuration)
	}

	// Restore cursor to the selected notification
	identifier := fmt.Sprintf("notif:%d", selectedID)
	m.restoreCursor(identifier)

	m.updateViewportContent()
	return nil
}

// markSelectedUnread marks the selected notification as unread.
func (m *Model) markSelectedUnread() tea.Cmd {
	if m.currentListLen() == 0 {
		return nil
	}

	selected, ok := m.selectedNotification()
	if !ok {
		return nil
	}

	// Save the notification ID to restore cursor later
	selectedID := selected.ID

	id := strconv.Itoa(selected.ID)
	if err := storage.MarkNotificationUnread(id); err != nil {
		m.errorHandler.Error(fmt.Sprintf("tui: failed to mark notification unread: %v", err))
		return errorMsgAfter(errorClearDuration)
	}

	if err := m.loadNotifications(true); err != nil {
		m.errorHandler.Error(fmt.Sprintf("tui: failed to reload notifications: %v", err))
		return errorMsgAfter(errorClearDuration)
	}

	// Restore cursor to the selected notification
	identifier := fmt.Sprintf("notif:%d", selectedID)
	m.restoreCursor(identifier)

	m.updateViewportContent()
	return nil
}

// handleJump handles the jump action for the selected notification.
func (m *Model) handleJump() tea.Cmd {
	if m.currentListLen() == 0 {
		return nil
	}

	// Get the selected notification
	selected, ok := m.selectedNotification()
	if !ok {
		return nil
	}

	// Check if notification has valid session, window, pane
	if selected.Session == "" || selected.Window == "" || selected.Pane == "" {
		m.errorHandler.Error("jump: notification missing session, window, or pane information")
		return errorMsgAfter(errorClearDuration)
	}

	// Ensure tmux is running
	if !m.runtimeCoordinator.EnsureTmuxRunning() {
		m.errorHandler.Error("tmux not running")
		return errorMsgAfter(errorClearDuration)
	}

	// Jump to the pane using RuntimeCoordinator
	// The error handler (set in NewModel) will capture and display errors in the TUI footer
	if !m.runtimeCoordinator.JumpToPane(selected.Session, selected.Window, selected.Pane) {
		// Error was already handled by m.errorHandler, just return error clear command
		return errorMsgAfter(errorClearDuration)
	}

	id := strconv.Itoa(selected.ID)
	if err := storage.MarkNotificationRead(id); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("jump: jumped, but failed to mark notification as read: %v", err))
	}

	return tea.Quit
}

// saveSettings extracts current settings from model and saves to disk.
func (m *Model) saveSettings() error {
	// Extract current settings state
	state := m.ToState()
	colors.Debug("Saving settings from TUI state")
	if err := m.ensureSettingsService().save(state); err != nil {
		return err
	}
	m.loadedSettings = state.ToSettings()
	return nil
}

// SaveSettings is the public version of saveSettings.
func (m *Model) SaveSettings() error {
	return m.saveSettings()
}

// selectedNotification returns the currently selected notification.
func (m *Model) selectedNotification() (notification.Notification, bool) {
	cursor := m.uiState.GetCursor()
	if m.isGroupedView() {
		visibleNodes := m.treeService.GetVisibleNodes()
		if cursor < 0 || cursor >= len(visibleNodes) {
			return notification.Notification{}, false
		}
		node := visibleNodes[cursor]
		if node == nil || node.Notification == nil {
			return notification.Notification{}, false
		}
		return *node.Notification, true
	}

	if cursor < 0 || cursor >= len(m.filtered) {
		return notification.Notification{}, false
	}
	return m.filtered[cursor], true
}

// GetGroupBy returns the current group-by setting.
func (m *Model) GetGroupBy() string {
	return string(m.uiState.GetGroupBy())
}

// SetGroupBy sets the group-by setting.
func (m *Model) SetGroupBy(groupBy string) error {
	if !settings.IsValidGroupBy(groupBy) {
		return fmt.Errorf("invalid group-by value: %s", groupBy)
	}

	if m.GetGroupBy() == groupBy {
		return nil // Already set
	}

	m.uiState.SetGroupBy(model.GroupBy(groupBy))
	return nil
}

// GetExpandLevel returns the current expand level setting.
func (m *Model) GetExpandLevel() int {
	return m.uiState.GetExpandLevel()
}

// SetExpandLevel sets the expand level setting.
func (m *Model) SetExpandLevel(level int) error {
	if level < settings.MinExpandLevel || level > settings.MaxExpandLevel {
		return fmt.Errorf("invalid expand level value: %d (expected %d-%d)", level, settings.MinExpandLevel, settings.MaxExpandLevel)
	}

	if m.uiState.GetExpandLevel() == level {
		return nil // Already set
	}

	m.uiState.SetExpandLevel(level)
	return nil
}
