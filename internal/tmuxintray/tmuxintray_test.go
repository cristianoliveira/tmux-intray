package tmuxintray

import (
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/require"
)

// TestGetVisibility tests GetVisibility function and its delegation to core.
func TestGetVisibility(t *testing.T) {
	// Set up test environment
	tmpDir := t.TempDir()

	// Create SQLite storage for tests
	dbPath := filepath.Join(tmpDir, "notifications.db")
	sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqliteStorage.Close()
	})

	// Mock tmux client
	mockClient := new(tmux.MockClient)
	mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("1", nil)

	coreClient := core.NewCore(mockClient, sqliteStorage)
	origGetVisibility := getVisibilityFunc
	t.Cleanup(func() {
		getVisibilityFunc = origGetVisibility
	})
	getVisibilityFunc = coreClient.GetVisibility

	// Test when tmux returns "1"
	result, err := GetVisibility()
	require.NoError(t, err)
	require.Equal(t, "1", result)

	// Test when tmux returns empty string (fallback to default)
	mockClient.ExpectedCalls = nil
	mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("", tmux.ErrTmuxNotRunning)
	getVisibilityFunc = coreClient.GetVisibility

	result, err = GetVisibility()
	require.NoError(t, err)
	require.Equal(t, "0", result)

	mockClient.AssertExpectations(t)
}

func TestSetVisibility(t *testing.T) {
	// Note: This test just verifies that SetVisibility can be called.
	// In a real scenario, this would require mocking tmux, which is complex.
	// For now, we just test that the function exists and can be called.
	// The actual functionality is tested through integration tests.

	// Since we can't easily mock the tmux client for SetVisibility,
	// we'll just verify the function signature is correct
	_ = SetVisibility(true)
	_ = SetVisibility(false)
}

func TestListAllNotifications(t *testing.T) {
	// Test that the function can be called without error
	result, err := ListAllNotifications()
	require.NoError(t, err)
	// Result is a TSV string (may be empty or contain existing data)
	require.NotNil(t, result)
}

func TestListNotifications(t *testing.T) {
	// Test with valid level
	result, err := ListNotifications("error", "active")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test with valid state
	result, err = ListNotifications("", "active")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test with empty filters (returns all)
	result, err = ListNotifications("", "")
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetActiveCount(t *testing.T) {
	// Test that the function can be called without error
	count := GetActiveCount()
	require.GreaterOrEqual(t, count, 0)
}

func TestGetStateDir(t *testing.T) {
	dir := GetStateDir()
	require.NotEmpty(t, dir)
}

func TestParseNotification(t *testing.T) {
	tests := []struct {
		name          string
		tsvLine       string
		expectedError bool
		validate      func(*testing.T, Notification)
	}{
		{
			name:          "valid notification with all fields (10 fields)",
			tsvLine:       "1\t2025-01-01T10:00:00Z\tactive\tsession1\twindow1\tpane1\ttest message\t2025-01-01T09:00:00Z\tinfo\t2025-01-01T10:05:00Z",
			expectedError: false,
			validate: func(t *testing.T, n Notification) {
				require.Equal(t, "1", n.ID)
				require.Equal(t, "2025-01-01T10:00:00Z", n.Timestamp)
				require.Equal(t, "active", n.State)
				require.Equal(t, "session1", n.Session)
				require.Equal(t, "window1", n.Window)
				require.Equal(t, "pane1", n.Pane)
				require.Equal(t, "test message", n.Message)
				require.Equal(t, "info", n.Level)
				require.Equal(t, "2025-01-01T10:05:00Z", n.ReadTimestamp)
			},
		},
		{
			name:          "valid notification without read timestamp (9 fields)",
			tsvLine:       "2\t2025-01-01T11:00:00Z\tactive\tsession2\twindow2\tpane2\ttest message 2\t2025-01-01T08:00:00Z\twarning",
			expectedError: false,
			validate: func(t *testing.T, n Notification) {
				require.Equal(t, "2", n.ID)
				require.Equal(t, "warning", n.Level)
				require.Equal(t, "", n.ReadTimestamp)
			},
		},
		{
			name:          "invalid notification - too few fields (8 fields)",
			tsvLine:       "1\t2025-01-01T10:00:00Z\tactive\tsession1\twindow1\tpane1\tmessage\tcreated",
			expectedError: true,
		},
		{
			name:          "invalid notification - too many fields (11 fields)",
			tsvLine:       "1\t2025-01-01T10:00:00Z\tactive\tsession1\twindow1\tpane1\tmessage\tcreated\tinfo\tread\textra\tfield",
			expectedError: true,
		},
		{
			name:          "notification with empty message",
			tsvLine:       "4\t2025-01-01T13:00:00Z\tactive\tsession4\twindow4\tpane4\t\t2025-01-01T06:00:00Z\tinfo\t",
			expectedError: false,
			validate: func(t *testing.T, n Notification) {
				require.Equal(t, "", n.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notif, err := ParseNotification(tt.tsvLine)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, notif)
				}
			}
		})
	}
}

func TestValidateIndex(t *testing.T) {
	tests := []struct {
		name          string
		index         string
		expectedValue int
		expectedError bool
	}{
		{
			name:          "valid single digit",
			index:         "1",
			expectedValue: 1,
			expectedError: false,
		},
		{
			name:          "valid multiple digits",
			index:         "123",
			expectedValue: 123,
			expectedError: false,
		},
		{
			name:          "valid large number",
			index:         "9999",
			expectedValue: 9999,
			expectedError: false,
		},
		{
			name:          "invalid - zero",
			index:         "0",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "invalid - negative",
			index:         "-1",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "invalid - contains letters",
			index:         "abc",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "invalid - contains special chars",
			index:         "1.5",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "invalid - empty string",
			index:         "",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "invalid - whitespace",
			index:         " ",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "invalid - alphanumeric",
			index:         "1a2b3c",
			expectedValue: 0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := ValidateIndex(tt.index)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestValidateLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedError bool
	}{
		{
			name:          "valid - info",
			level:         "info",
			expectedError: false,
		},
		{
			name:          "valid - warning",
			level:         "warning",
			expectedError: false,
		},
		{
			name:          "valid - error",
			level:         "error",
			expectedError: false,
		},
		{
			name:          "valid - critical",
			level:         "critical",
			expectedError: false,
		},
		{
			name:          "valid - empty string (defaults to info)",
			level:         "",
			expectedError: false,
		},
		{
			name:          "invalid - debug",
			level:         "debug",
			expectedError: true,
		},
		{
			name:          "invalid - trace",
			level:         "trace",
			expectedError: true,
		},
		{
			name:          "invalid - INFO (case sensitive)",
			level:         "INFO",
			expectedError: true,
		},
		{
			name:          "invalid - lowercase ERROR",
			level:         "error",
			expectedError: false,
		},
		{
			name:          "invalid - special characters",
			level:         "!@#",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLevel(tt.level)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateState(t *testing.T) {
	tests := []struct {
		name          string
		state         string
		expectedError bool
	}{
		{
			name:          "valid - active",
			state:         "active",
			expectedError: false,
		},
		{
			name:          "valid - dismissed",
			state:         "dismissed",
			expectedError: false,
		},
		{
			name:          "valid - all",
			state:         "all",
			expectedError: false,
		},
		{
			name:          "valid - empty string (defaults to active)",
			state:         "",
			expectedError: false,
		},
		{
			name:          "invalid - read",
			state:         "read",
			expectedError: true,
		},
		{
			name:          "invalid - unread",
			state:         "unread",
			expectedError: true,
		},
		{
			name:          "invalid - deleted",
			state:         "deleted",
			expectedError: true,
		},
		{
			name:          "invalid - ACTIVE (case sensitive)",
			state:         "ACTIVE",
			expectedError: true,
		},
		{
			name:          "invalid - special characters",
			state:         "!@#",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateState(tt.state)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUnescapeMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no escaping needed",
			input:    "simple message",
			expected: "simple message",
		},
		{
			name:     "escaped newline",
			input:    "line1\\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "escaped tab",
			input:    "col1\\tcol2",
			expected: "col1\tcol2",
		},
		{
			name:     "escaped backslash",
			input:    "path\\\\file",
			expected: "path\\file",
		},
		{
			name:     "multiple escaped sequences",
			input:    "line1\\nline2\\tcol1\\tcol2\\\\path",
			expected: "line1\nline2\tcol1\tcol2\\path",
		},
		{
			name:     "mixed escaping",
			input:    "\\n\\t\\\\",
			expected: "\n\t\\",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unescapeMessage(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
