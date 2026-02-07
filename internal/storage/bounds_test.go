package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBoundsChecking tests that all storage functions handle out-of-bounds scenarios gracefully.
func TestBoundsChecking(t *testing.T) {
	tmpDir := t.TempDir()
	notifFile := filepath.Join(tmpDir, "notifications.tsv")

	// Write malformed TSV data with various field counts
	malformedData := "" +
		// Line 1: Only 1 field (ID only)
		"1\n" +
		// Line 2: Only 2 fields (ID, timestamp)
		"2\t2025-01-01T12:00:00Z\n" +
		// Line 3: 3 fields (ID, timestamp, state)
		"3\t2025-01-01T12:00:00Z\tactive\n" +
		// Line 4: 4 fields (up to session)
		"4\t2025-01-01T12:00:00Z\tactive\tsess1\n" +
		// Line 5: 5 fields (up to window)
		"5\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\n" +
		// Line 6: 6 fields (up to pane)
		"6\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\n" +
		// Line 7: 7 fields (up to message)
		"7\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\tmessage7\n" +
		// Line 8: 8 fields (up to paneCreated)
		"8\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\tmessage8\tcreated8\n" +
		// Line 9: Complete 9 fields (valid)
		"9\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\tmessage9\tcreated9\tinfo\n" +
		// Line 10: Extra fields (10 fields - should be handled gracefully)
		"10\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\tmessage10\tcreated10\tinfo\textra\n"

	err := os.WriteFile(notifFile, []byte(malformedData), 0644)
	require.NoError(t, err)

	// Reset and reinitialize to load the malformed data
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	Reset()
	require.NoError(t, Init())

	t.Run("GetLatestNotificationsDoesNotPanic", func(t *testing.T) {
		// Should not panic on malformed data
		latest, err := getLatestNotifications()
		require.NoError(t, err)
		// Should return valid entries that can be parsed
		require.NotEmpty(t, latest)
	})

	t.Run("ListNotificationsDoesNotPanic", func(t *testing.T) {
		// Should not panic on malformed data
		list, listErr := ListNotifications("all", "", "", "", "", "", "")
		require.NoError(t, listErr)
		require.NotEmpty(t, list)
		// Should contain valid notification
		require.Contains(t, list, "9")
		require.Contains(t, list, "message9")
	})

	t.Run("GetActiveCountDoesNotPanic", func(t *testing.T) {
		// Should not panic on malformed data
		count := GetActiveCount()
		// At least notification 9 should be active
		require.GreaterOrEqual(t, count, 1)
	})

	t.Run("GetNotificationByIDInvalid", func(t *testing.T) {
		// GetNotificationByID returns the line even if malformed (it just finds the line by ID)
		// The validation happens when you try to use the notification (e.g., dismiss it)

		// Try to get notification with only 1 field
		notif, err := GetNotificationByID("1")
		require.NoError(t, err)
		require.Contains(t, notif, "1")

		// Try to get notification with only 2 fields
		notif, err = GetNotificationByID("2")
		require.NoError(t, err)
		require.Contains(t, notif, "2")

		// Try to get notification with only 3 fields
		notif, err = GetNotificationByID("3")
		require.NoError(t, err)
		require.Contains(t, notif, "3")

		// Valid notification should also work
		notif, err = GetNotificationByID("9")
		require.NoError(t, err)
		require.Contains(t, notif, "9")
	})

	t.Run("DismissNotificationInvalid", func(t *testing.T) {
		// Try to dismiss notifications with insufficient fields
		err := DismissNotification("1") // Only 1 field
		require.Error(t, err)

		err = DismissNotification("2") // Only 2 fields
		require.Error(t, err)

		err = DismissNotification("3") // Only 3 fields - has state but missing other fields
		require.Error(t, err)

		// Valid notification should be dismissible
		err = DismissNotification("9")
		require.NoError(t, err)
	})

	t.Run("DismissAllDoesNotPanic", func(t *testing.T) {
		// Should not panic on malformed data
		err := DismissAll()
		require.NoError(t, err)
	})

	t.Run("FilterNotificationsDoesNotPanic", func(t *testing.T) {
		// Manually call filterNotifications with malformed data
		lines := []string{
			"1",                               // 1 field
			"2\t2025-01-01T12:00:00Z",         // 2 fields
			"3\t2025-01-01T12:00:00Z\tactive", // 3 fields
			"9\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\tmessage9\tcreated9\tinfo", // 9 fields
		}

		// Should not panic
		filtered := filterNotifications(lines, "all", "", "", "", "", "", "")
		require.NotEmpty(t, filtered)
	})

	t.Run("CleanupOldDoesNotPanic", func(t *testing.T) {
		// Should not panic on malformed data
		err := CleanupOldNotifications(0, true) // Dry run
		require.NoError(t, err)
	})

	t.Run("AddNotificationAfterMalformedData", func(t *testing.T) {
		// Should be able to add valid notification after malformed data
		id, err := AddNotification("valid after malformed", "", "sess2", "win2", "pane2", "", "info")
		require.NoError(t, err)
		require.NotEmpty(t, id)

		// Verify it was added
		list, listErr := ListNotifications("all", "", "", "", "", "", "")
		require.NoError(t, listErr)
		require.Contains(t, list, id)
		require.Contains(t, list, "valid after malformed")
	})
}

// TestEmptyFieldsHandling tests that empty fields are handled correctly.
func TestEmptyFieldsHandling(t *testing.T) {
	tmpDir := t.TempDir()
	notifFile := filepath.Join(tmpDir, "notifications.tsv")

	// Write data with empty fields
	dataWithEmptyFields := "" +
		"1\t2025-01-01T12:00:00Z\tactive\t\t\t\t\tinfo\n" + // Empty session, window, pane, message, paneCreated
		"2\t\tactive\tsess2\twin2\tpane2\tmessage2\t\tinfo\n" + // Empty timestamp, paneCreated
		"3\t2025-01-01T12:00:00Z\t\t\t\t\tmessage3\tcreated3\tinfo\n" // Empty state, session, window, pane

	err := os.WriteFile(notifFile, []byte(dataWithEmptyFields), 0644)
	require.NoError(t, err)

	// Reset and reinitialize
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	Reset()
	require.NoError(t, Init())

	t.Run("ListNotificationsWithEmptyFields", func(t *testing.T) {
		list, listErr := ListNotifications("all", "", "", "", "", "", "")
		require.NoError(t, listErr)
		require.NotEmpty(t, list)
	})

	t.Run("GetNotificationByIDWithEmptyFields", func(t *testing.T) {
		notif, err := GetNotificationByID("1")
		require.NoError(t, err)
		require.NotEmpty(t, notif)
	})

	t.Run("DismissNotificationWithEmptyFields", func(t *testing.T) {
		// Dismiss notification 2
		err := DismissNotification("2")
		require.NoError(t, err)

		// Verify it was dismissed
		list, listErr := ListNotifications("dismissed", "", "", "", "", "", "")
		require.NoError(t, listErr)
		require.Contains(t, list, "2")
	})
}

// TestGetNextIDWithMalformedData tests getNextID handles malformed data.
func TestGetNextIDWithMalformedData(t *testing.T) {
	tmpDir := t.TempDir()
	notifFile := filepath.Join(tmpDir, "notifications.tsv")

	// Write data with malformed IDs
	malformedIDs := "" +
		"abc\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\tmsg1\tcreated1\tinfo\n" + // Non-numeric ID
		"5\t2025-01-01T12:00:00Z\tactive\tsess2\twin2\tpane2\tmsg2\tcreated2\tinfo\n" + // Valid ID 5
		"xyz\t2025-01-01T12:00:00Z\tactive\tsess3\twin3\tpane3\tmsg3\tcreated3\tinfo\n" // Non-numeric ID

	err := os.WriteFile(notifFile, []byte(malformedIDs), 0644)
	require.NoError(t, err)

	// Reset and reinitialize
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	Reset()
	require.NoError(t, Init())

	// Next ID should be 6 (max valid ID + 1)
	id, err := getNextID()
	require.NoError(t, err)
	require.Equal(t, 6, id)
}

// TestDismissByIDInternal tests the internal dismissByID function with malformed data.
func TestDismissByIDInternal(t *testing.T) {
	tmpDir := t.TempDir()
	notifFile := filepath.Join(tmpDir, "notifications.tsv")

	// Write valid notification
	validData := "1\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\tmessage1\tcreated1\tinfo\n"
	err := os.WriteFile(notifFile, []byte(validData), 0644)
	require.NoError(t, err)

	// Reset and reinitialize
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	Reset()
	require.NoError(t, Init())

	// Should be able to dismiss
	err = dismissByID("1")
	require.NoError(t, err)

	// Verify dismissed
	list, listErr := ListNotifications("dismissed", "", "", "", "", "", "")
	require.NoError(t, listErr)
	require.Contains(t, list, "1")
}

// TestDismissAllActiveInternal tests the internal dismissAllActive function with malformed data.
func TestDismissAllActiveInternal(t *testing.T) {
	tmpDir := t.TempDir()
	notifFile := filepath.Join(tmpDir, "notifications.tsv")

	// Write mixed valid and malformed data
	mixedData := "" +
		"1\t2025-01-01T12:00:00Z\tactive\tsess1\twin1\tpane1\tmsg1\tcreated1\tinfo\n" + // Valid
		"2\t\t\t\t\t\t\t\t\n" + // Empty fields (state is empty, not active)
		"3\t2025-01-01T12:00:00Z\tactive\t\t\t\t\t\tinfo\n" + // Some empty fields but valid
		"4\t2025-01-01T12:00:00Z\tactive\tsess4\twin4\tpane4\tmsg4\tcreated4\tinfo\n" // Valid

	err := os.WriteFile(notifFile, []byte(mixedData), 0644)
	require.NoError(t, err)

	// Reset and reinitialize
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	Reset()
	require.NoError(t, Init())

	// Should not panic
	err = dismissAllActive()
	require.NoError(t, err)

	// Verify that the two valid active notifications (1, 3, 4) were dismissed
	// Notification 2 has empty state, so it's not active
	count := GetActiveCount()
	require.Equal(t, 0, count)
}
