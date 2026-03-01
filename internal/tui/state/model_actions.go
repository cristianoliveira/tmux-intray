package state

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

type jumpTarget struct {
	Session string
	Window  string
	Pane    string
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
	if err := m.ensureInteractionController().DismissNotification(id); err != nil {
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
	if err := m.ensureInteractionController().DismissByFilter(session, window, pane); err != nil {
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
	return errorMsgAfter(errorClearDuration)
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
	if err := m.ensureInteractionController().MarkNotificationRead(id); err != nil {
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
	if err := m.ensureInteractionController().MarkNotificationUnread(id); err != nil {
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

	target, ok := m.resolveJumpTarget()
	if !ok {
		m.errorHandler.Error("jump: unable to resolve target from current selection")
		return errorMsgAfter(errorClearDuration)
	}

	// Check if target has valid session and window.
	if target.Session == "" || target.Window == "" {
		m.errorHandler.Error("jump: notification missing session or window information")
		return errorMsgAfter(errorClearDuration)
	}

	// Ensure tmux is running
	if !m.ensureInteractionController().EnsureTmuxRunning() {
		m.errorHandler.Error("tmux not running")
		return errorMsgAfter(errorClearDuration)
	}

	ctrl := m.ensureInteractionController()
	jumped := false
	if target.Pane == "" {
		jumped = ctrl.JumpToWindow(target.Session, target.Window)
	} else {
		// The error handler (set in NewModel) will capture and display errors in the TUI footer.
		jumped = ctrl.JumpToPane(target.Session, target.Window, target.Pane)
	}

	if !jumped {
		// Error was already handled by m.errorHandler, just return error clear command
		return errorMsgAfter(errorClearDuration)
	}

	selected, selectedOK := m.selectedNotification()
	if selectedOK {
		id := strconv.Itoa(selected.ID)
		if err := ctrl.MarkNotificationRead(id); err != nil {
			m.errorHandler.Warning(fmt.Sprintf("jump: jumped, but failed to mark notification as read: %v", err))
		}
	}

	return tea.Quit
}

func (m *Model) resolveJumpTarget() (jumpTarget, bool) {
	if !m.isGroupedView() {
		selected, ok := m.selectedNotification()
		if !ok {
			return jumpTarget{}, false
		}
		return jumpTarget{
			Session: selected.Session,
			Window:  selected.Window,
			Pane:    selected.Pane,
		}, true
	}

	node := m.selectedVisibleNode()
	if node == nil {
		return jumpTarget{}, false
	}

	if node.Notification != nil {
		return jumpTarget{
			Session: node.Notification.Session,
			Window:  node.Notification.Window,
			Pane:    node.Notification.Pane,
		}, true
	}

	target, ok := jumpTargetFromLatestOrSource(node)
	if !ok {
		return jumpTarget{}, false
	}

	if node.Kind == model.NodeKindWindow {
		target.Pane = ""
	}

	return target, true
}

func jumpTargetFromLatestOrSource(node *model.TreeNode) (jumpTarget, bool) {
	if node != nil && node.LatestEvent != nil {
		return jumpTarget{
			Session: node.LatestEvent.Session,
			Window:  node.LatestEvent.Window,
			Pane:    node.LatestEvent.Pane,
		}, true
	}

	if node == nil || len(node.Sources) == 0 {
		return jumpTarget{}, false
	}

	for _, source := range node.Sources {
		return jumpTarget{Session: source.Session, Window: source.Window, Pane: source.Pane}, true
	}

	return jumpTarget{}, false
}

// selectedNotification returns the currently selected notification.
func (m *Model) selectedNotification() (notification.Notification, bool) {
	cursor := m.uiState.GetCursor()
	if m.isGroupedView() {
		return m.selectedGroupedNotification(cursor)
	}

	if cursor < 0 || cursor >= len(m.filtered) {
		return notification.Notification{}, false
	}
	return m.filtered[cursor], true
}

func (m *Model) selectedGroupedNotification(cursor int) (notification.Notification, bool) {
	visibleNodes := m.treeService.GetVisibleNodes()
	if cursor < 0 || cursor >= len(visibleNodes) {
		return notification.Notification{}, false
	}

	node := visibleNodes[cursor]
	if node == nil {
		return notification.Notification{}, false
	}

	if node.Notification != nil {
		return *node.Notification, true
	}

	if node.Kind != model.NodeKindMessage || string(m.uiState.GetGroupBy()) != settings.GroupByPaneMessage {
		return notification.Notification{}, false
	}

	if node.LatestEvent == nil {
		return notification.Notification{}, false
	}

	return *node.LatestEvent, true
}
