// Package state provides BubbleTea messages for inter-component communication.
package state

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// TreeUpdatedMsg is sent when the tree structure changes (nodes added/removed/restructured).
type TreeUpdatedMsg struct {
	Root *model.TreeNode
}

// NodeExpandedMsg is sent when a tree node is expanded.
type NodeExpandedMsg struct {
	NodeID string
	Node   *model.TreeNode
}

// NodeCollapsedMsg is sent when a tree node is collapsed.
type NodeCollapsedMsg struct {
	NodeID string
	Node   *model.TreeNode
}

// NotificationsLoadedMsg is sent when notifications are loaded from storage.
type NotificationsLoadedMsg struct {
	Notifications []notification.Notification
}

// NotificationsFilteredMsg is sent when notifications are filtered (search, filters).
type NotificationsFilteredMsg struct {
	Filtered []notification.Notification
	TreeRoot *model.TreeNode
}

// NotificationDismissedMsg is sent when a notification is dismissed.
type NotificationDismissedMsg struct {
	ID int
}

// NotificationReadMsg is sent when a notification is marked as read.
type NotificationReadMsg struct {
	ID int
}

// NotificationUnreadMsg is sent when a notification is marked as unread.
type NotificationUnreadMsg struct {
	ID int
}

// ViewportUpdatedMsg is sent when the viewport content is updated.
type ViewportUpdatedMsg struct {
	Width  int
	Height int
}

// CursorMovedMsg is sent when the cursor is moved.
type CursorMovedMsg struct {
	Position int
	ListLen  int
}

// SearchModeChangedMsg is sent when search mode is toggled.
type SearchModeChangedMsg struct {
	Active bool
}

// SearchQueryChangedMsg is sent when the search query is updated.
type SearchQueryChangedMsg struct {
	Query string
}

// CommandModeChangedMsg is sent when command mode is toggled.
type CommandModeChangedMsg struct {
	Active bool
}

// CommandQueryChangedMsg is sent when the command query is updated.
type CommandQueryChangedMsg struct {
	Query string
}

// CommandExecutedMsg is sent when a command is executed.
type CommandExecutedMsg struct {
	Result *model.CommandResult
}

// SettingsChangedMsg is sent when TUI settings change.
type SettingsChangedMsg struct {
	ViewMode    string
	GroupBy     string
	ExpandLevel int
}

// TmuxJumpMsg is sent when jumping to a pane.
type TmuxJumpMsg struct {
	SessionID string
	WindowID  string
	PaneID    string
}

// TmuxContextChangedMsg is sent when tmux context changes.
type TmuxContextChangedMsg struct {
	Context *model.TmuxContext
}

// ErrorMsg is sent when an error occurs.
type ErrorMsg struct {
	Error error
}

// RefreshNamesMsg is sent when tmux name caches need refresh.
type RefreshNamesMsg struct{}

// JumpCompletedMsg is sent after a successful jump to a pane.
type JumpCompletedMsg struct{}

// JumpFailedMsg is sent when a jump operation fails.
type JumpFailedMsg struct {
	Error error
}

// saveSettingsSuccessMsg is sent when settings are saved successfully.
type saveSettingsSuccessMsg struct{}

// SaveSettingsSuccessMsg is exported version of saveSettingsSuccessMsg.
type SaveSettingsSuccessMsg struct {
	saveSettingsSuccessMsg
}

// saveSettingsFailedMsg is sent when settings save fails.
type saveSettingsFailedMsg struct {
	err error
}

// SaveSettingsFailedMsg is exported version of saveSettingsFailedMsg.
type SaveSettingsFailedMsg struct {
	saveSettingsFailedMsg
}

// SaveSettingsCmd returns a command to save settings.
func SaveSettingsCmd(saveFn func() error) tea.Cmd {
	return func() tea.Msg {
		if err := saveFn(); err != nil {
			return saveSettingsFailedMsg{err: err}
		}
		return saveSettingsSuccessMsg{}
	}
}
