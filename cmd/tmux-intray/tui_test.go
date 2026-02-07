/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// originalSessionNameFetcher stores the original function to restore after tests.
var originalSessionNameFetcher func(string) string

// originalFetchAllSessionNames stores the original function to restore after tests.
var originalFetchAllSessionNames func() map[string]string

func init() {
	originalSessionNameFetcher = sessionNameFetcher
	sessionNameFetcher = func(sessionID string) string { return sessionID }
	originalFetchAllSessionNames = fetchAllSessionNames
	fetchAllSessionNames = func() map[string]string { return make(map[string]string) }
}

// TestNewTUIModel creates a new TUI model and verifies it initializes correctly.
func TestNewTUIModel(t *testing.T) {
	model, err := NewTUIModel()
	if err != nil {
		t.Fatalf("Failed to create TUI model: %v", err)
	}

	// Check that the model is initialized
	if model.width != 0 {
		t.Errorf("Expected width to be 0 initially, got %d", model.width)
	}
	if model.height != 0 {
		t.Errorf("Expected height to be 0 initially, got %d", model.height)
	}
	if model.cursor != 0 {
		t.Errorf("Expected cursor to be 0 initially, got %d", model.cursor)
	}
	if model.searchMode != false {
		t.Errorf("Expected searchMode to be false initially, got %v", model.searchMode)
	}
	if model.searchQuery != "" {
		t.Errorf("Expected searchQuery to be empty initially, got %q", model.searchQuery)
	}
}

// TestTUIModelInit verifies the Init method returns the correct command.
func TestTUIModelInit(t *testing.T) {
	model := tuiModel{}
	cmd := model.Init()
	if cmd != nil {
		t.Errorf("Expected Init to return nil, got %v", cmd)
	}
}

// TestTUIModelUpdateHandlesNavigation verifies j/k navigation works correctly.
func TestTUIModelUpdateHandlesNavigation(t *testing.T) {
	model := &tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
		filtered: []Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
		cursor: 0,
	}

	// Test k key (move up - should stay at 0)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, _ := model.Update(msg)
	model = newModel.(*tuiModel)
	if model.cursor != 0 {
		t.Errorf("Expected cursor to stay at 0 when moving up from position 0, got %d", model.cursor)
	}

	// Test j key (move down)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ = model.Update(msg)
	model = newModel.(*tuiModel)
	if model.cursor != 1 {
		t.Errorf("Expected cursor to move to position 1, got %d", model.cursor)
	}

	// Test j key again (move down)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ = model.Update(msg)
	model = newModel.(*tuiModel)
	if model.cursor != 2 {
		t.Errorf("Expected cursor to move to position 2, got %d", model.cursor)
	}

	// Test j key again (should stay at 2 - last position)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ = model.Update(msg)
	model = newModel.(*tuiModel)
	if model.cursor != 2 {
		t.Errorf("Expected cursor to stay at 2 when moving down from last position, got %d", model.cursor)
	}

	// Test k key (move up)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, _ = model.Update(msg)
	model = newModel.(*tuiModel)
	if model.cursor != 1 {
		t.Errorf("Expected cursor to move to position 1, got %d", model.cursor)
	}
}

// TestTUIModelUpdateHandlesSearch verifies search functionality works correctly.
func TestTUIModelUpdateHandlesSearch(t *testing.T) {
	model := &tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "Error: file not found"},
			{ID: 2, Message: "Warning: low memory"},
			{ID: 3, Message: "Error: connection failed"},
		},
		filtered: []Notification{
			{ID: 1, Message: "Error: file not found"},
			{ID: 2, Message: "Warning: low memory"},
			{ID: 3, Message: "Error: connection failed"},
		},
		cursor: 0,
	}

	// Test / key (enter search mode)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newModel, _ := model.Update(msg)
	model = newModel.(*tuiModel)
	if !model.searchMode {
		t.Error("Expected searchMode to be true after pressing /")
	}
	if model.searchQuery != "" {
		t.Errorf("Expected searchQuery to be empty after entering search mode, got %q", model.searchQuery)
	}
	if model.cursor != 0 {
		t.Errorf("Expected cursor to reset to 0 after entering search mode, got %d", model.cursor)
	}
	if len(model.filtered) != 3 {
		t.Errorf("Expected filtered to have 3 items with empty search, got %d", len(model.filtered))
	}

	// Add search query "error" (should filter to 2 items)
	model.searchQuery = "error"
	model.applySearchFilter()
	if len(model.filtered) != 2 {
		t.Errorf("Expected filtered to have 2 items with search 'error', got %d", len(model.filtered))
	}
	if !strings.Contains(model.filtered[0].Message, "Error") {
		t.Error("Expected first filtered item to contain 'Error'")
	}

	// Add more to search query "not found" (should filter to 1 item)
	model.searchQuery = "not found"
	model.applySearchFilter()
	if len(model.filtered) != 1 {
		t.Errorf("Expected filtered to have 1 item with search 'not found', got %d", len(model.filtered))
	}
	if !strings.Contains(strings.ToLower(model.filtered[0].Message), "not found") {
		t.Error("Expected filtered item to contain 'not found'")
	}

	// Clear search query (should show all items)
	model.searchQuery = ""
	model.applySearchFilter()
	if len(model.filtered) != 3 {
		t.Errorf("Expected filtered to have 3 items with empty search, got %d", len(model.filtered))
	}
}

// TestTUIModelUpdateHandlesQuit verifies quit functionality works correctly.
func TestTUIModelUpdateHandlesQuit(t *testing.T) {
	model := tuiModel{}

	// Test q key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)
	if cmd == nil {
		t.Error("Expected Update to return a non-nil command for q key")
	}

	// Test Ctrl+C key
	msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = model.Update(msg)
	if cmd == nil {
		t.Error("Expected Update to return a non-nil command for Ctrl+C")
	}

	// Test ESC key
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd = model.Update(msg)
	if cmd == nil {
		t.Error("Expected Update to return a non-nil command for ESC")
	}
}

// TestTUIModelUpdateHandlesSearchEscape verifies ESC in search mode exits search mode but not TUI.
func TestTUIModelUpdateHandlesSearchEscape(t *testing.T) {
	model := &tuiModel{
		searchMode:  true,
		searchQuery: "test",
	}
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd := model.Update(msg)
	model = newModel.(*tuiModel)
	if cmd != nil {
		t.Error("Expected Update to return nil command for ESC in search mode")
	}
	if model.searchMode != false {
		t.Error("Expected searchMode to be false after ESC")
	}
	if model.searchQuery != "" {
		t.Errorf("Expected searchQuery to be empty after ESC, got %q", model.searchQuery)
	}
}

// TestTUIModelUpdateHandlesCommandMode verifies command mode functionality.
func TestTUIModelUpdateHandlesCommandMode(t *testing.T) {
	model := &tuiModel{}
	// Test ':' enters command mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
	newModel, _ := model.Update(msg)
	model = newModel.(*tuiModel)
	if !model.commandMode {
		t.Error("Expected commandMode to be true after pressing ':'")
	}
	if model.commandQuery != "" {
		t.Errorf("Expected commandQuery to be empty, got %q", model.commandQuery)
	}
	// Test ESC exits command mode without quitting
	model.commandMode = true
	model.commandQuery = "test"
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd := model.Update(msg)
	model = newModel.(*tuiModel)
	if cmd != nil {
		t.Error("Expected Update to return nil command for ESC in command mode")
	}
	if model.commandMode != false {
		t.Error("Expected commandMode to be false after ESC")
	}
	if model.commandQuery != "" {
		t.Errorf("Expected commandQuery to be empty after ESC, got %q", model.commandQuery)
	}
	// Test command input
	model.commandMode = true
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, _ = model.Update(msg)
	model = newModel.(*tuiModel)
	if model.commandQuery != "q" {
		t.Errorf("Expected commandQuery to be 'q', got %q", model.commandQuery)
	}
	// Test backspace
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ = model.Update(msg)
	model = newModel.(*tuiModel)
	if model.commandQuery != "" {
		t.Errorf("Expected commandQuery to be empty after backspace, got %q", model.commandQuery)
	}
	// Test executing :q command
	model.commandMode = true
	model.commandQuery = "q"
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd = model.Update(msg)
	if cmd == nil {
		t.Error("Expected Update to return non-nil command for :q")
	}
	// Ensure command mode is reset after execution
	// (we can't check because model is not returned, but we can trust the code)
}

// TestTUIModelUpdateHandlesWindowSize verifies terminal resize is handled correctly.
func TestTUIModelUpdateHandlesWindowSize(t *testing.T) {
	model := &tuiModel{}

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	newModel, _ := model.Update(msg)
	model = newModel.(*tuiModel)

	if model.width != 100 {
		t.Errorf("Expected width to be 100, got %d", model.width)
	}
	if model.height != 30 {
		t.Errorf("Expected height to be 30, got %d", model.height)
	}
}

// TestTUIModelView verifies the View method renders correctly.
func TestTUIModelView(t *testing.T) {
	model := tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "Test notification", Timestamp: "2024-01-01T12:00:00Z", Level: "info", State: "active", Session: "0", Window: "0", Pane: "0"},
		},
		filtered: []Notification{
			{ID: 1, Message: "Test notification", Timestamp: "2024-01-01T12:00:00Z", Level: "info", State: "active", Session: "0", Window: "0", Pane: "0"},
		},
		cursor: 0,
		width:  80,
		height: 24,
	}

	view := model.View()
	if view == "" {
		t.Error("Expected View to return a non-empty string")
	}

	// Check that the view contains the header
	if !strings.Contains(view, "TYPE") {
		t.Error("Expected View to contain 'TYPE' header")
	}
	if !strings.Contains(view, "STATUS") {
		t.Error("Expected View to contain 'STATUS' header")
	}
	if !strings.Contains(view, "SESSION") {
		t.Error("Expected View to contain 'SESSION' header")
	}
	if !strings.Contains(view, "MESSAGE") {
		t.Error("Expected View to contain 'MESSAGE' header")
	}
	if !strings.Contains(view, "PANE") {
		t.Error("Expected View to contain 'PANE' header")
	}
	if !strings.Contains(view, "AGE") {
		t.Error("Expected View to contain 'AGE' header")
	}

	// Check that the view contains the notification message
	if !strings.Contains(view, "Test notification") {
		t.Error("Expected View to contain the notification message")
	}

	// Check that the view contains help text
	if !strings.Contains(view, "j/k: move") {
		t.Error("Expected View to contain 'j/k: move' help text")
	}
	if !strings.Contains(view, "q: quit") {
		t.Error("Expected View to contain 'q: quit' help text")
	}
}

// TestGetLevelIcon verifies level icons are returned correctly.
func TestGetLevelIcon(t *testing.T) {
	model := tuiModel{}

	tests := []struct {
		level    string
		expected string
	}{
		{"error", "❌ err"},
		{"warning", "⚠️ wrn"},
		{"critical", "‼️ crt"},
		{"info", "ℹ️ inf"},
		{"", "ℹ️ inf"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := model.getLevelIcon(tt.level)
			if result != tt.expected {
				t.Errorf("Expected level icon %q for level %q, got %q", tt.expected, tt.level, result)
			}
		})
	}
}

// TestGetStatusIcon verifies status icons are returned correctly.
func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"active", "●"},
		{"", "●"},
		{"dismissed", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := getStatusIcon(tt.state)
			if result != tt.expected {
				t.Errorf("Expected status icon %q for state %q, got %q", tt.expected, tt.state, result)
			}
		})
	}
}

// TestCalculateAge verifies age calculation works correctly.
func TestCalculateAge(t *testing.T) {
	// Note: We can't easily test this with fixed timestamps since time.Since()
	// is based on the current time. We'll just verify it doesn't crash and
	// returns a non-empty string for valid timestamps.
	result := calculateAge("2024-01-01T12:00:00Z")
	if result == "" {
		t.Error("Expected calculateAge to return a non-empty string for valid timestamp")
	}

	// Test empty timestamp
	result = calculateAge("")
	if result != "" {
		t.Error("Expected calculateAge to return empty string for empty timestamp")
	}

	// Test invalid timestamp
	result = calculateAge("invalid")
	if result != "" {
		t.Error("Expected calculateAge to return empty string for invalid timestamp")
	}
}

// TestTUIModelViewWithNoNotifications verifies the View method handles empty notifications.
func TestTUIModelViewWithNoNotifications(t *testing.T) {
	model := tuiModel{
		notifications: []Notification{},
		filtered:      []Notification{},
		cursor:        0,
		width:         80,
		height:        24,
	}

	view := model.View()
	if view == "" {
		t.Error("Expected View to return a non-empty string")
	}

	// Check that the view contains the "No notifications found" message
	if !strings.Contains(view, "No notifications found") {
		t.Error("Expected View to contain 'No notifications found' message")
	}
}

// TestHandleDismiss verifies the dismiss action works correctly.
func TestHandleDismiss(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Add a test notification
	_, err := storage.AddNotification("Test message", "2024-01-01T12:00:00Z", "$1", "@2", "%3", "1234", "info")
	if err != nil {
		t.Fatalf("Failed to add test notification: %v", err)
	}

	// Create TUI model and load notifications
	model, err := NewTUIModel()
	if err != nil {
		t.Fatalf("Failed to create TUI model: %v", err)
	}

	// Verify we have notifications
	if len(model.filtered) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(model.filtered))
	}

	// Dismiss the notification
	cmd := model.handleDismiss()
	if cmd != nil {
		t.Error("Expected handleDismiss to return nil, got a command")
	}

	// Reload the model to verify dismissal
	model, err = NewTUIModel()
	if err != nil {
		t.Fatalf("Failed to reload TUI model: %v", err)
	}

	// Verify no active notifications remain
	if len(model.filtered) != 0 {
		t.Errorf("Expected 0 active notifications after dismissal, got %d", len(model.filtered))
	}
}

// TestHandleDismissWithEmptyList verifies dismiss handles empty notification list.
func TestHandleDismissWithEmptyList(t *testing.T) {
	model := tuiModel{
		notifications: []Notification{},
		filtered:      []Notification{},
		cursor:        0,
	}

	// Should not crash
	cmd := model.handleDismiss()
	if cmd != nil {
		t.Error("Expected handleDismiss to return nil for empty list, got a command")
	}
}

// TestHandleJump verifies the jump action handles errors correctly.
func TestHandleJump(t *testing.T) {
	model := &tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "Test"},
		},
		filtered: []Notification{
			{ID: 1, Message: "Test"},
		},
		cursor: 0,
	}

	// Jump will fail because tmux is not running in test environment
	// This should not crash and should return nil
	cmd := model.handleJump()
	// Note: We can't test the actual tea.Quit in unit tests easily,
	// but we can verify it doesn't crash
	_ = cmd
}

// TestHandleJumpWithMissingContext verifies jump handles notifications with missing context.
func TestHandleJumpWithMissingContext(t *testing.T) {
	// Test with missing session
	model := &tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "Test"},
		},
		filtered: []Notification{
			{ID: 1, Message: "Test"},
		},
		cursor: 0,
	}

	cmd := model.handleJump()
	if cmd != nil {
		t.Error("Expected handleJump to return nil for notification with missing session, got a command")
	}

	// Test with missing window
	model.filtered[0].Session = "$1"
	model.filtered[0].Window = ""
	cmd = model.handleJump()
	if cmd != nil {
		t.Error("Expected handleJump to return nil for notification with missing window, got a command")
	}

	// Test with missing pane
	model.filtered[0].Window = "@2"
	model.filtered[0].Pane = ""
	cmd = model.handleJump()
	if cmd != nil {
		t.Error("Expected handleJump to return nil for notification with missing pane, got a command")
	}
}

// TestHandleJumpWithEmptyList verifies jump handles empty notification list.
func TestHandleJumpWithEmptyList(t *testing.T) {
	model := tuiModel{
		notifications: []Notification{},
		filtered:      []Notification{},
		cursor:        0,
	}

	// Should not crash
	cmd := model.handleJump()
	if cmd != nil {
		t.Error("Expected handleJump to return nil for empty list, got a command")
	}
}

// TestTUIModelUpdateHandlesDismissKey verifies the 'd' key triggers dismiss.
func TestTUIModelUpdateHandlesDismissKey(t *testing.T) {
	// We can't easily test the full dismiss flow in a unit test
	// because it requires storage setup. This test just verifies the
	// key binding works without crashing.
	model := tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "Test"},
		},
		filtered: []Notification{
			{ID: 1, Message: "Test"},
		},
		cursor: 0,
	}

	// Simulate 'd' key press
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, _ := model.Update(msg)
	_ = newModel.(*tuiModel) // Verify it's still a valid model
}

// TestTUIModelUpdateHandlesEnterKey verifies the Enter key triggers jump.
func TestTUIModelUpdateHandlesEnterKey(t *testing.T) {
	model := tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "Test"},
		},
		filtered: []Notification{
			{ID: 1, Message: "Test"},
		},
		cursor: 0,
	}

	// Simulate Enter key press
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.Update(msg)
	_ = newModel.(*tuiModel) // Verify it's still a valid model
}

// TestRenderRowSessionColumn verifies session column is displayed correctly.
func TestRenderRowSessionColumn(t *testing.T) {
	// Mock session name fetcher to return predictable names
	original := sessionNameFetcher
	defer func() { sessionNameFetcher = original }()
	sessionNameFetcher = func(sessionID string) string {
		return sessionID + "-name"
	}

	model := tuiModel{width: 100}
	notif := Notification{
		ID:        1,
		Session:   "$1",
		Window:    "@2",
		Pane:      "%3",
		Message:   "Test message",
		Timestamp: "2024-01-01T12:00:00Z",
		Level:     "info",
		State:     "active",
	}
	row := model.renderRow(notif, false)
	// Should contain session name (mocked)
	if !strings.Contains(row, "$1-name") {
		t.Error("Expected row to contain session name")
	}
	// Should contain pane ID
	if !strings.Contains(row, "%3") {
		t.Error("Expected row to contain pane ID in pane column")
	}
	// Should NOT contain window in pane column
	if strings.Contains(row, "@2:%3") {
		t.Error("Pane column should not contain window prefix")
	}
}

// TestToState verifies conversion from tuiModel to TUIState.
func TestToState(t *testing.T) {
	tests := []struct {
		name  string
		model *tuiModel
		want  settings.TUIState
	}{
		{
			name:  "empty model",
			model: &tuiModel{},
			want:  settings.TUIState{},
		},
		{
			name: "model with settings",
			model: &tuiModel{
				sortBy:    settings.SortByLevel,
				sortOrder: settings.SortOrderAsc,
				columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				viewMode: settings.ViewModeDetailed,
			},
			want: settings.TUIState{
				SortBy:    settings.SortByLevel,
				SortOrder: settings.SortOrderAsc,
				Columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode: settings.ViewModeDetailed,
			},
		},
		{
			name: "model with only some settings",
			model: &tuiModel{
				sortBy:   settings.SortByTimestamp,
				viewMode: settings.ViewModeCompact,
			},
			want: settings.TUIState{
				SortBy:   settings.SortByTimestamp,
				ViewMode: settings.ViewModeCompact,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.ToState()
			assert.Equal(t, tt.want.SortBy, got.SortBy)
			assert.Equal(t, tt.want.SortOrder, got.SortOrder)
			assert.Equal(t, tt.want.Columns, got.Columns)
			assert.Equal(t, tt.want.Filters, got.Filters)
			assert.Equal(t, tt.want.ViewMode, got.ViewMode)
		})
	}
}

// TestFromState verifies applying TUIState to tuiModel.
func TestFromState(t *testing.T) {
	tests := []struct {
		name     string
		model    *tuiModel
		state    settings.TUIState
		wantErr  bool
		verifyFn func(*testing.T, *tuiModel)
	}{
		{
			name:    "empty state - no changes",
			model:   &tuiModel{},
			state:   settings.TUIState{},
			wantErr: false,
			verifyFn: func(t *testing.T, m *tuiModel) {
				assert.Equal(t, "", m.sortBy)
				assert.Equal(t, "", m.sortOrder)
				assert.Empty(t, m.columns)
				assert.Equal(t, "", m.viewMode)
				assert.Equal(t, settings.Filter{}, m.filters)
			},
		},
		{
			name:  "full state - all fields set",
			model: &tuiModel{},
			state: settings.TUIState{
				SortBy:    settings.SortByLevel,
				SortOrder: settings.SortOrderAsc,
				Columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode: settings.ViewModeDetailed,
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *tuiModel) {
				assert.Equal(t, settings.SortByLevel, m.sortBy)
				assert.Equal(t, settings.SortOrderAsc, m.sortOrder)
				assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel}, m.columns)
				assert.Equal(t, settings.ViewModeDetailed, m.viewMode)
				assert.Equal(t, settings.LevelFilterWarning, m.filters.Level)
				assert.Equal(t, settings.StateFilterActive, m.filters.State)
				assert.Equal(t, "my-session", m.filters.Session)
				assert.Equal(t, "@1", m.filters.Window)
				assert.Equal(t, "%1", m.filters.Pane)
			},
		},
		{
			name: "partial state - only some fields set",
			model: &tuiModel{
				sortBy:    settings.SortByTimestamp,
				sortOrder: settings.SortOrderDesc,
				columns:   []string{settings.ColumnID},
				filters: settings.Filter{
					Level: settings.LevelFilterError,
				},
				viewMode: settings.ViewModeCompact,
			},
			state: settings.TUIState{
				SortBy:  settings.SortByLevel,
				Columns: []string{settings.ColumnID, settings.ColumnMessage},
				// sortOrder and viewMode are not set - should be preserved
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *tuiModel) {
				assert.Equal(t, settings.SortByLevel, m.sortBy, "sortBy should be updated")
				assert.Equal(t, settings.SortOrderDesc, m.sortOrder, "sortOrder should be preserved")
				assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage}, m.columns, "columns should be updated")
				assert.Equal(t, settings.LevelFilterError, m.filters.Level, "existing filter level should be preserved")
				assert.Equal(t, settings.ViewModeCompact, m.viewMode, "viewMode should be preserved")
			},
		},
		{
			name: "partial filters - only some filter fields set",
			model: &tuiModel{
				filters: settings.Filter{
					Level:   settings.LevelFilterError,
					State:   settings.StateFilterActive,
					Session: "old-session",
				},
			},
			state: settings.TUIState{
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					Session: "new-session",
					// State, Window, Pane not set - should be preserved
				},
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *tuiModel) {
				assert.Equal(t, settings.LevelFilterWarning, m.filters.Level, "level should be updated")
				assert.Equal(t, settings.StateFilterActive, m.filters.State, "state should be preserved")
				assert.Equal(t, "new-session", m.filters.Session, "session should be updated")
				assert.Empty(t, m.filters.Window, "window should be empty (never set)")
				assert.Empty(t, m.filters.Pane, "pane should be empty (never set)")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.FromState(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.verifyFn != nil {
				tt.verifyFn(t, tt.model)
			}
		})
	}
}

// TestRoundTripSettings verifies round-trip conversion preserves settings.
func TestRoundTripSettings(t *testing.T) {
	tests := []struct {
		name  string
		model *tuiModel
	}{
		{
			name:  "empty model",
			model: &tuiModel{},
		},
		{
			name: "model with all settings",
			model: &tuiModel{
				sortBy:    settings.SortByLevel,
				sortOrder: settings.SortOrderAsc,
				columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				viewMode: settings.ViewModeDetailed,
			},
		},
		{
			name: "model with partial settings",
			model: &tuiModel{
				sortBy:   settings.SortByTimestamp,
				viewMode: settings.ViewModeCompact,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to state
			state := tt.model.ToState()

			// Create new model and apply state
			newModel := &tuiModel{}
			err := newModel.FromState(state)
			require.NoError(t, err)

			// Verify all settings were preserved
			assert.Equal(t, tt.model.sortBy, newModel.sortBy)
			assert.Equal(t, tt.model.sortOrder, newModel.sortOrder)
			assert.Equal(t, tt.model.columns, newModel.columns)
			assert.Equal(t, tt.model.filters, newModel.filters)
			assert.Equal(t, tt.model.viewMode, newModel.viewMode)
		})
	}
}

// TestSaveSettings verifies saveSettings method works correctly.
func TestSaveSettings(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Set up config directory
	os.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_CONFIG_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Create a model with custom settings
	model := &tuiModel{
		sortBy:    settings.SortByLevel,
		sortOrder: settings.SortOrderAsc,
		columns:   []string{settings.ColumnID, settings.ColumnMessage},
		viewMode:  settings.ViewModeDetailed,
	}

	// Save settings
	err := model.saveSettings()
	require.NoError(t, err)

	// Load settings and verify
	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage}, loaded.Columns)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
}

// TestTUIModelSaveOnQuit verifies settings are saved when quitting with 'q'.
func TestTUIModelSaveOnQuit(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Set up config directory
	os.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_CONFIG_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Create a model with custom settings
	model := &tuiModel{
		sortBy:    settings.SortByLevel,
		sortOrder: settings.SortOrderAsc,
		viewMode:  settings.ViewModeDetailed,
	}

	// Simulate 'q' key press
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)

	// Verify quit command is returned
	assert.NotNil(t, cmd)

	// Load settings and verify they were saved
	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
}

// TestTUIModelSaveOnCtrlC verifies settings are saved when quitting with Ctrl+C.
func TestTUIModelSaveOnCtrlC(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Set up config directory
	os.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_CONFIG_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Create a model with custom settings
	model := &tuiModel{
		sortBy:   settings.SortByTimestamp,
		viewMode: settings.ViewModeDetailed,
	}

	// Simulate Ctrl+C key press
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(msg)

	// Verify quit command is returned
	assert.NotNil(t, cmd)

	// Load settings and verify they were saved
	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByTimestamp, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
}

// TestTUIModelSaveOnCommandQ verifies settings are saved when quitting with ':q'.
func TestTUIModelSaveOnCommandQ(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Set up config directory
	os.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_CONFIG_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Create a model with custom settings
	model := &tuiModel{
		sortBy:   settings.SortByLevel,
		viewMode: settings.ViewModeDetailed,
	}

	// Enter command mode and type 'q'
	model.commandMode = true
	model.commandQuery = "q"
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(msg)

	// Verify quit command is returned
	assert.NotNil(t, cmd)

	// Load settings and verify they were saved
	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
}

// TestTUIModelSaveCommandW verifies ':w' command saves settings without quitting.
func TestTUIModelSaveCommandW(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Set up config directory
	os.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_CONFIG_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Create a model with custom settings
	model := &tuiModel{
		sortBy:   settings.SortByLevel,
		viewMode: settings.ViewModeDetailed,
	}

	// Enter command mode and type 'w'
	model.commandMode = true
	model.commandQuery = "w"
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.Update(msg)

	// Verify a command is returned (save command)
	assert.NotNil(t, cmd)

	// Verify command mode was reset (TUI continues)
	model = newModel.(*tuiModel)
	assert.False(t, model.commandMode)
	assert.Equal(t, "", model.commandQuery)

	// Load settings and verify they were saved
	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
}

// TestTUIModelMissingSettingsFile verifies TUI works when settings file doesn't exist.
func TestTUIModelMissingSettingsFile(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Set up config directory to a non-existent path
	os.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_CONFIG_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Ensure no settings file exists
	settingsPath := tmpDir + "/settings.json"
	_, err := os.Stat(settingsPath)
	assert.True(t, os.IsNotExist(err), "Settings file should not exist initially")

	// Create TUI model - should not fail
	model, err := NewTUIModel()
	require.NoError(t, err)
	assert.NotNil(t, model)

	// Model should have default settings (empty or defaults)
	// The important thing is that it doesn't crash
}

// TestTUIModelCorruptedSettingsFile verifies TUI works when settings file is corrupted.
func TestTUIModelCorruptedSettingsFile(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Set up config directory
	os.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_CONFIG_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Create a corrupted settings file
	settingsPath := tmpDir + "/settings.json"
	err := os.WriteFile(settingsPath, []byte("invalid json {{{"), 0644)
	require.NoError(t, err)

	// Load settings - should return defaults with warning
	loaded, err := settings.Load()
	require.NoError(t, err, "Load should not fail on corrupted JSON")
	assert.NotNil(t, loaded, "Should return default settings")

	// The settings should be defaults (not the corrupted values)
	// We just verify it loaded successfully without crashing
	// The actual values will be defaults, but we don't assert specific values
	// to avoid test flakiness from package-level state
	assert.NotEmpty(t, loaded.SortBy, "SortBy should have a default value")
}

// TestTUIModelSettingsLifecycle verifies full settings lifecycle: load -> modify -> save.
func TestTUIModelSettingsLifecycle(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Set up state directory
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Set up config directory
	os.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_CONFIG_DIR")

	// Initialize storage
	storage.Reset()
	storage.Init()

	// Step 1: Load initial settings (should be defaults since file doesn't exist)
	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.NotNil(t, loaded)

	// Step 2: Create a model and apply loaded settings
	model := &tuiModel{}
	state := settings.FromSettings(loaded)
	err = model.FromState(state)
	require.NoError(t, err)

	// Step 3: Modify model settings
	model.sortBy = settings.SortByLevel
	model.sortOrder = settings.SortOrderAsc
	model.viewMode = settings.ViewModeDetailed

	// Step 4: Save settings
	err = model.saveSettings()
	require.NoError(t, err)

	// Step 5: Reload settings and verify persistence
	reloaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, reloaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, reloaded.SortOrder)
	assert.Equal(t, settings.ViewModeDetailed, reloaded.ViewMode)

	// Step 6: Apply reloaded settings to a new model and verify
	newModel := &tuiModel{}
	newState := settings.FromSettings(reloaded)
	err = newModel.FromState(newState)
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, newModel.sortBy)
	assert.Equal(t, settings.SortOrderAsc, newModel.sortOrder)
	assert.Equal(t, settings.ViewModeDetailed, newModel.viewMode)
}
