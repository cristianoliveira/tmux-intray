package state

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

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

// handleDismissGroup handles the dismiss group action.
// Shows confirmation dialog if current selection is a group node in grouped view.
func (m *Model) handleDismissGroup() tea.Cmd {
	// Only available in grouped view
	if !m.isGroupedView() {
		return nil
	}

	if m.currentListLen() == 0 {
		return nil
	}

	// Get the selected node
	node := m.selectedVisibleNode()
	if node == nil {
		return nil
	}

	// Only work on group nodes (not notification nodes)
	if !m.isGroupNode(node) {
		return nil
	}
	// Only session, window, and pane groups can be dismissed
	// Only session, window, and pane groups can be dismissed
	if node.Kind != model.NodeKindSession && node.Kind != model.NodeKindWindow && node.Kind != model.NodeKindPane {
		return nil
	}
	// Collect session, window, pane filters and count
	session, window, pane, count := m.collectNotificationsInGroup(node)
	if count == 0 {
		return nil
	}

	// Set up confirmation
	action := PendingAction{
		Type:     ActionDismissGroup,
		Message:  fmt.Sprintf("Dismiss %d notifications in this %s?", count, getGroupTypeLabel(node.Kind)),
		Session:  session,
		Window:   window,
		Pane:     pane,
		Count:    count,
		NodeKind: node.Kind,
	}
	m.uiState.SetPendingAction(action)
	m.uiState.SetConfirmationMode(true)

	return nil
}

// handleDismissByFilter dismisses notifications matching the provided filters.
func (m *Model) handleDismissByFilter(session, window, pane string) tea.Cmd {
	// Dismiss using storage
	if err := storage.DismissByFilter(session, window, pane); err != nil {
		m.errorHandler.Error(fmt.Sprintf("Failed to dismiss notifications: %v", err))
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

	// Show success message
	m.errorHandler.Success("Notifications dismissed")

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
