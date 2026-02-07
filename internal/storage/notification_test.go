package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetNotificationByID(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tmux-intray-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases
	testCases := []struct {
		name      string
		setupFunc func()
		id        string
		wantLine  string
		wantError bool
	}{
		{
			name: "notification exists and is active",
			setupFunc: func() {
				// Reset and init storage for each test case
				Reset()
				os.Setenv("TMUX_INTRAY_STATE_DIR", tempDir)
				require.NoError(t, Init())
				_, err := AddNotification("test message", "2025-02-04T10:00:00Z", "session1", "window1", "pane1", "123456", "info")
				require.NoError(t, err)
			},
			id:        "1",
			wantLine:  "1\t2025-02-04T10:00:00Z\tactive\tsession1\twindow1\tpane1\ttest message\t123456\tinfo",
			wantError: false,
		},
		{
			name: "notification exists and is dismissed",
			setupFunc: func() {
				// Reset and init storage for each test case
				Reset()
				os.Setenv("TMUX_INTRAY_STATE_DIR", tempDir)
				require.NoError(t, Init())
				_, err := AddNotification("test message", "2025-02-04T10:00:00Z", "session1", "window1", "pane1", "123456", "info")
				require.NoError(t, err)
				DismissNotification("1")
			},
			id:        "1",
			wantLine:  "1\t2025-02-04T10:00:00Z\tdismissed\tsession1\twindow1\tpane1\ttest message\t123456\tinfo",
			wantError: false,
		},
		{
			name: "notification does not exist",
			setupFunc: func() {
				// Reset and init storage for each test case
				Reset()
				os.Setenv("TMUX_INTRAY_STATE_DIR", tempDir)
				require.NoError(t, Init())
				// No notifications added
			},
			id:        "999",
			wantError: true,
		},
		{
			name: "empty ID",
			setupFunc: func() {
				// Reset and init storage for each test case
				Reset()
				os.Setenv("TMUX_INTRAY_STATE_DIR", tempDir)
				require.NoError(t, Init())
				_, err := AddNotification("test message", "2025-02-04T10:00:00Z", "session1", "window1", "pane1", "123456", "info")
				require.NoError(t, err)
			},
			id:        "",
			wantError: true,
		},
		{
			name: "multiple notifications, get latest",
			setupFunc: func() {
				// Reset and init storage for each test case
				Reset()
				os.Setenv("TMUX_INTRAY_STATE_DIR", tempDir)
				require.NoError(t, Init())
				// Add first notification
				_, _ = AddNotification("test message", "2025-02-04T10:00:00Z", "session1", "window1", "pane1", "123456", "info")
				// Add another notification
				_, _ = AddNotification("test message 2", "2025-02-04T10:01:00Z", "session2", "window2", "pane2", "123457", "info")
				// Update first notification (dismiss it)
				DismissNotification("1")
			},
			id:        "1",
			wantLine:  "1\t2025-02-04T10:00:00Z\tdismissed\tsession1\twindow1\tpane1\ttest message\t123456\tinfo",
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test data
			tc.setupFunc()

			// Call GetNotificationByID
			line, err := GetNotificationByID(tc.id)

			// Check expectations
			if tc.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if line != tc.wantLine {
				t.Errorf("Got line %q, want %q", line, tc.wantLine)
			}
		})
	}
}

func TestGetNotificationByIDWithLock(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tmux-intray-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Reset storage state
	Reset()

	// Set up test environment
	os.Setenv("TMUX_INTRAY_STATE_DIR", tempDir)
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Add a notification
	_, err = AddNotification("test message", "2025-02-04T10:00:00Z", "session1", "window1", "pane1", "123456", "info")
	if err != nil {
		t.Fatalf("Failed to add notification: %v", err)
	}

	// Test that the function properly acquires and releases the lock
	// This is more of a smoke test - the actual lock testing would be complex
	line, err := GetNotificationByID("1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	expected := "1\t2025-02-04T10:00:00Z\tactive\tsession1\twindow1\tpane1\ttest message\t123456\tinfo"
	if line != expected {
		t.Errorf("Got line %q, want %q", line, expected)
	}
}

func TestGetNotificationByIDNotInitialized(t *testing.T) {
	// Reset storage state
	Reset()

	// Try to get notification without initializing
	_, err := GetNotificationByID("1")

	if err == nil {
		t.Error("Expected error when storage not initialized")
	}

	// The error message will be "notification with ID 1 not found" if the state_dir is not set
	// or "storage not initialized" if the state_dir is set but not initialized
	// We'll check that we get an error in either case
}
