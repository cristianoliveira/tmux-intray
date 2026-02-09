package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/require"
)

// TestGroupCountsUpdateOnReadUnreadChange verifies that unread counts
// in the grouped tree update correctly when notifications are marked
// as read or unread.
func TestGroupCountsUpdateOnReadUnreadChange(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:            1,
			Session:       "$1",
			Window:        "@1",
			Pane:          "%1",
			Message:       "Unread message 1",
			Timestamp:     "2024-01-01T10:00:00Z",
			ReadTimestamp: "", // Unread
		},
		{
			ID:            2,
			Session:       "$1",
			Window:        "@1",
			Pane:          "%1",
			Message:       "Unread message 2",
			Timestamp:     "2024-01-02T10:00:00Z",
			ReadTimestamp: "", // Unread
		},
		{
			ID:            3,
			Session:       "$2",
			Window:        "@2",
			Pane:          "%2",
			Message:       "Unread message 3",
			Timestamp:     "2024-01-03T10:00:00Z",
			ReadTimestamp: "", // Unread
		},
	}

	root := BuildTree(notifications, settings.GroupByPane)

	// Initially, all notifications are unread
	require.Equal(t, 3, root.UnreadCount)
	require.Equal(t, 2, root.Children[0].UnreadCount) // Session $1 has 2 unread
	require.Equal(t, 1, root.Children[1].UnreadCount) // Session $2 has 1 unread

	// Mark one notification as read
	notifications[1] = notifications[1].MarkRead()

	// Rebuild tree with updated notification
	root = BuildTree(notifications, settings.GroupByPane)

	// Verify unread counts updated
	require.Equal(t, 2, root.UnreadCount)             // One less unread
	require.Equal(t, 1, root.Children[0].UnreadCount) // Session $1 now has 1 unread
	require.Equal(t, 1, root.Children[1].UnreadCount) // Session $2 still has 1 unread

	// Mark another notification as read
	notifications[0] = notifications[0].MarkRead()

	// Rebuild tree with updated notification
	root = BuildTree(notifications, settings.GroupByPane)

	// Verify unread counts updated again
	require.Equal(t, 1, root.UnreadCount)             // One less unread
	require.Equal(t, 0, root.Children[0].UnreadCount) // Session $1 now has 0 unread
	require.Equal(t, 1, root.Children[1].UnreadCount) // Session $2 still has 1 unread

	// Mark the last unread notification as read
	notifications[2] = notifications[2].MarkRead()

	// Rebuild tree with updated notification
	root = BuildTree(notifications, settings.GroupByPane)

	// Verify all notifications are now read
	require.Equal(t, 0, root.UnreadCount)
	require.Equal(t, 0, root.Children[0].UnreadCount) // Session $1 has 0 unread
	require.Equal(t, 0, root.Children[1].UnreadCount) // Session $2 has 0 unread
}
