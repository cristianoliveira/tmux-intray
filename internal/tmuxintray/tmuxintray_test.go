package tmuxintray

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/require"
)

// TestGetVisibility tests the GetVisibility function and its delegation to core.
func TestGetVisibility(t *testing.T) {
	// Set up test environment
	colors.SetDebug(true)

	storage.Init()
	fileStorage, err := storage.NewFileStorage()
	require.NoError(t, err)

	// Mock tmux client
	mockClient := new(tmux.MockClient)
	mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("1", nil)

	coreClient := core.NewCore(mockClient, fileStorage)
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
