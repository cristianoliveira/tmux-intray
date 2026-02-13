package tmuxintray

import (
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/require"
)

// TestGetVisibility tests the GetVisibility function and its delegation to core.
func TestGetVisibility(t *testing.T) {
	// Set up test environment
	tmpDir := t.TempDir()
	colors.SetDebug(true)

	// Create SQLite storage for tests
	dbPath := filepath.Join(tmpDir, "notifications.db")
	sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqliteStorage.Close()
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
