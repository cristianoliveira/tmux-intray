package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUIState(t *testing.T) {
	uiState := NewUIState()

	// Test initial values
	assert.Equal(t, 0, uiState.GetCursor())
	assert.Equal(t, "", uiState.GetSearchQuery())
	assert.False(t, uiState.IsSearchMode())
	assert.Equal(t, defaultViewportWidth, uiState.GetWidth())
	assert.Equal(t, defaultViewportHeight, uiState.GetHeight())

	// Test cursor management
	uiState.SetCursor(5)
	assert.Equal(t, 5, uiState.GetCursor())

	uiState.MoveCursorUp(10)
	assert.Equal(t, 4, uiState.GetCursor())

	uiState.MoveCursorDown(10)
	assert.Equal(t, 5, uiState.GetCursor())

	// Test search mode
	uiState.SetSearchMode(true)
	assert.True(t, uiState.IsSearchMode())
	assert.Equal(t, "", uiState.GetSearchQuery())

	uiState.SetSearchQuery("test")
	assert.Equal(t, "test", uiState.GetSearchQuery())

	uiState.AppendToSearchQuery('i')
	uiState.AppendToSearchQuery('n')
	uiState.AppendToSearchQuery('g')
	assert.Equal(t, "testing", uiState.GetSearchQuery())

	uiState.BackspaceSearchQuery()
	assert.Equal(t, "testin", uiState.GetSearchQuery())

	uiState.SetSearchMode(false)
	assert.False(t, uiState.IsSearchMode())
	assert.Equal(t, "", uiState.GetSearchQuery())

	// Test viewport
	uiState.SetWidth(100)
	assert.Equal(t, 100, uiState.GetWidth())

	uiState.SetHeight(30)
	assert.Equal(t, 30, uiState.GetHeight())

	uiState.UpdateViewportSize()
	viewport := uiState.GetViewport()
	assert.NotNil(t, viewport)
	assert.Equal(t, 100, viewport.Width)
	assert.Equal(t, 27, viewport.Height) // height - headerFooterLines

	// Test pending key
	uiState.SetPendingKey("z")
	assert.Equal(t, "z", uiState.GetPendingKey())

	uiState.ClearPendingKey()
	assert.Equal(t, "", uiState.GetPendingKey())
}

func TestUIStateWithModel(t *testing.T) {
	// Create a Model with UIState
	model := &Model{
		uiState: NewUIState(),
	}

	require.NotNil(t, model.uiState)

	// Test that the Model's methods work with UIState
	model.uiState.SetCursor(2)
	assert.Equal(t, 2, model.uiState.GetCursor())

	model.uiState.SetSearchMode(true)
	model.uiState.SetSearchQuery("test")
	assert.True(t, model.uiState.IsSearchMode())
	assert.Equal(t, "test", model.uiState.GetSearchQuery())
}

func TestSetConfirmationModeResetsState(t *testing.T) {
	uiState := NewUIState()

	uiState.SetConfirmationMode(true)
	assert.True(t, uiState.IsConfirmationMode())

	uiState.SetPendingAction(PendingAction{Type: ActionDismissGroup})
	uiState.SetConfirmationMode(false)
	assert.False(t, uiState.IsConfirmationMode())
	assert.Equal(t, PendingAction{}, uiState.GetPendingAction())
}
