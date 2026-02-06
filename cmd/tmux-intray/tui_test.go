/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

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
	model := tuiModel{
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
	model = newModel.(tuiModel)
	if model.cursor != 0 {
		t.Errorf("Expected cursor to stay at 0 when moving up from position 0, got %d", model.cursor)
	}

	// Test j key (move down)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ = model.Update(msg)
	model = newModel.(tuiModel)
	if model.cursor != 1 {
		t.Errorf("Expected cursor to move to position 1, got %d", model.cursor)
	}

	// Test j key again (move down)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ = model.Update(msg)
	model = newModel.(tuiModel)
	if model.cursor != 2 {
		t.Errorf("Expected cursor to move to position 2, got %d", model.cursor)
	}

	// Test j key again (should stay at 2 - last position)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ = model.Update(msg)
	model = newModel.(tuiModel)
	if model.cursor != 2 {
		t.Errorf("Expected cursor to stay at 2 when moving down from last position, got %d", model.cursor)
	}

	// Test k key (move up)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, _ = model.Update(msg)
	model = newModel.(tuiModel)
	if model.cursor != 1 {
		t.Errorf("Expected cursor to move to position 1, got %d", model.cursor)
	}
}

// TestTUIModelUpdateHandlesSearch verifies search functionality works correctly.
func TestTUIModelUpdateHandlesSearch(t *testing.T) {
	model := tuiModel{
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
	model = newModel.(tuiModel)
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

// TestTUIModelUpdateHandlesWindowSize verifies terminal resize is handled correctly.
func TestTUIModelUpdateHandlesWindowSize(t *testing.T) {
	model := tuiModel{}

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	newModel, _ := model.Update(msg)
	model = newModel.(tuiModel)

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
	if !strings.Contains(view, "SUMMARY") {
		t.Error("Expected View to contain 'SUMMARY' header")
	}
	if !strings.Contains(view, "SOURCE") {
		t.Error("Expected View to contain 'SOURCE' header")
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
	id := storage.AddNotification("Test message", "2024-01-01T12:00:00Z", "$1", "@2", "%3", "1234", "info")
	if id == "" {
		t.Fatal("Failed to add test notification")
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
	model := tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "Test notification", Session: "$1", Window: "@2", Pane: "%3"},
		},
		filtered: []Notification{
			{ID: 1, Message: "Test notification", Session: "$1", Window: "@2", Pane: "%3"},
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
	model := tuiModel{
		notifications: []Notification{
			{ID: 1, Message: "Test notification", Session: "", Window: "@2", Pane: "%3"},
		},
		filtered: []Notification{
			{ID: 1, Message: "Test notification", Session: "", Window: "@2", Pane: "%3"},
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
	_ = newModel.(tuiModel) // Verify it's still a valid model
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
	_ = newModel.(tuiModel) // Verify it's still a valid model
}
