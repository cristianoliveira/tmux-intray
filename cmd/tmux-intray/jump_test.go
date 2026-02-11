package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

func TestJumpSuccess(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	originalValidatePaneExistsFunc := validatePaneExistsFunc
	originalJumpToPaneFunc := jumpToPaneFunc
	originalMarkNotificationReadAfterJumpFunc := markNotificationReadAfterJumpFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
		validatePaneExistsFunc = originalValidatePaneExistsFunc
		jumpToPaneFunc = originalJumpToPaneFunc
		markNotificationReadAfterJumpFunc = originalMarkNotificationReadAfterJumpFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		if id != "42" {
			return "", errors.New("notification with ID 42 not found")
		}
		// Simulate TSV line: ID, timestamp, state, session, window, pane, message, pane_created, level
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return true }
	jumpToPaneFunc = func(session, window, pane string) bool { return true }
	markNotificationReadAfterJumpFunc = func(id string) error { return nil }

	result, err := Jump("42")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result.ID != "42" {
		t.Errorf("Expected ID 42, got %s", result.ID)
	}
	if result.Session != "$0" {
		t.Errorf("Expected session $0, got %s", result.Session)
	}
	if result.Window != "%0" {
		t.Errorf("Expected window %%0, got %s", result.Window)
	}
	if result.Pane != ":0.0" {
		t.Errorf("Expected pane :0.0, got %s", result.Pane)
	}
	if result.State != "active" {
		t.Errorf("Expected state active, got %s", result.State)
	}
	if result.Message != "hello" {
		t.Errorf("Expected message hello, got %s", result.Message)
	}
	if !result.PaneExists {
		t.Error("Expected pane exists true")
	}
}

func TestJumpTmuxNotRunning(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	defer func() { ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc }()
	ensureTmuxRunningFunc = func() bool { return false }

	_, err := Jump("42")
	if err == nil {
		t.Error("Expected error when tmux not running")
	}
	if err.Error() != "tmux not running" {
		t.Errorf("Expected 'tmux not running', got %v", err)
	}
}

func TestJumpNotificationNotFound(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		return "", errors.New("notification with ID 42 not found")
	}

	_, err := Jump("42")
	if err == nil {
		t.Error("Expected error when notification not found")
	}
}

func TestJumpNoPaneAssociation(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		// Missing session/window/pane fields (empty)
		return "42\t2025-02-04T10:00:00Z\tactive\t\t\t\thello\t\tinfo", nil
	}

	_, err := Jump("42")
	if err == nil {
		t.Error("Expected error when no pane association")
	}
	// Updated error message is now more descriptive with which fields are missing
	if !strings.Contains(err.Error(), "missing required fields") {
		t.Errorf("Expected error message about incomplete context, got %v", err)
	}
}

func TestJumpPaneDoesNotExistButWindowSelected(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	originalValidatePaneExistsFunc := validatePaneExistsFunc
	originalJumpToPaneFunc := jumpToPaneFunc
	originalMarkNotificationReadAfterJumpFunc := markNotificationReadAfterJumpFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
		validatePaneExistsFunc = originalValidatePaneExistsFunc
		jumpToPaneFunc = originalJumpToPaneFunc
		markNotificationReadAfterJumpFunc = originalMarkNotificationReadAfterJumpFunc
	}()

	// Tiger Style: Test fallback to window when pane doesn't exist
	// ASSERTION 1: Should return success (true) when window can be selected
	// ASSERTION 2: Should mark PaneExists as false to indicate fallback occurred
	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return false }
	jumpToPaneFunc = func(session, window, pane string) bool { return true }
	markNotificationReadAfterJumpFunc = func(id string) error { return nil }

	result, err := Jump("42")
	if err != nil {
		t.Errorf("Expected no error when pane missing but window selected, got %v", err)
	}
	if result.PaneExists {
		t.Error("Expected pane exists false")
	}
}

func TestJumpWindowDoesNotExist(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	originalValidatePaneExistsFunc := validatePaneExistsFunc
	originalJumpToPaneFunc := jumpToPaneFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
		validatePaneExistsFunc = originalValidatePaneExistsFunc
		jumpToPaneFunc = originalJumpToPaneFunc
	}()

	// Tiger Style: Test that window/pane not existing returns error
	// ASSERTION 1: Should return error (not nil)
	// ASSERTION 2: Error message should clearly indicate what failed
	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return true }
	jumpToPaneFunc = func(session, window, pane string) bool { return false }

	_, err := Jump("42")
	if err == nil {
		t.Error("Expected error when window does not exist")
	}
	if err.Error() != "jump: failed to jump because pane or window does not exist" {
		t.Errorf("Expected 'jump: failed to jump because pane or window does not exist', got %v", err)
	}
}

func TestJumpInvalidLineFormat(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		// Too few fields
		return "42\tactive", nil
	}

	_, err := Jump("42")
	if err == nil {
		t.Error("Expected error when line format invalid")
	}
}

func TestJumpOptimizedRetrieval(t *testing.T) {
	// Test that the jump command uses the optimized GetNotificationByID function
	// This test verifies the integration between cmd/jump.go and internal/storage
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalValidatePaneExistsFunc := validatePaneExistsFunc
	originalJumpToPaneFunc := jumpToPaneFunc
	originalMarkNotificationReadAfterJumpFunc := markNotificationReadAfterJumpFunc
	originalFileStorage := fileStorage
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		validatePaneExistsFunc = originalValidatePaneExistsFunc
		jumpToPaneFunc = originalJumpToPaneFunc
		markNotificationReadAfterJumpFunc = originalMarkNotificationReadAfterJumpFunc
		fileStorage = originalFileStorage
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	validatePaneExistsFunc = func(session, window, pane string) bool { return true }
	jumpToPaneFunc = func(session, window, pane string) bool { return true }
	markNotificationReadAfterJumpFunc = func(id string) error {
		return fileStorage.MarkNotificationRead(id)
	}

	tempDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", tempDir)

	// Reset and recreate backend storage in test state directory.
	storage.Reset()
	t.Cleanup(storage.Reset)

	var err error
	fileStorage, err = storage.NewFromConfig()
	require.NoError(t, err)

	// Add a test notification
	id, err := fileStorage.AddNotification("test message", "2025-02-04T10:00:00Z", "session1", "window1", "pane1", "123456", "info")
	if err != nil {
		t.Fatalf("Failed to add notification: %v", err)
	}

	// Jump to the notification
	result, err := Jump(id)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result.ID != id {
		t.Errorf("Expected ID %s, got %s", id, result.ID)
	}
	if result.Session != "session1" {
		t.Errorf("Expected session session1, got %s", result.Session)
	}
	if result.Window != "window1" {
		t.Errorf("Expected window window1, got %s", result.Window)
	}
	if result.Pane != "pane1" {
		t.Errorf("Expected pane pane1, got %s", result.Pane)
	}
	if result.State != "active" {
		t.Errorf("Expected state active, got %s", result.State)
	}
	if result.Message != "test message" {
		t.Errorf("Expected message 'test message', got %s", result.Message)
	}
	if !result.PaneExists {
		t.Error("Expected pane exists true")
	}

	line, err := fileStorage.GetNotificationByID(id)
	require.NoError(t, err)
	loaded, err := notification.ParseNotification(line)
	require.NoError(t, err)
	assert.NotEmpty(t, loaded.ReadTimestamp)
}

func TestJumpToPaneReturnsCorrectBooleans(t *testing.T) {
	// Tiger Style: Test that JumpToPane returns correct boolean values
	// ASSERTION 1: Returns true when jump succeeds to pane
	// ASSERTION 2: Returns true when jump falls back to window (pane missing)
	// ASSERTION 3: Returns false when window doesn't exist
	// ASSERTION 4: Returns false when pane selection fails despite window existing
	originalJumpToPaneFunc := jumpToPaneFunc
	originalMarkNotificationReadAfterJumpFunc := markNotificationReadAfterJumpFunc
	defer func() { jumpToPaneFunc = originalJumpToPaneFunc }()
	defer func() { markNotificationReadAfterJumpFunc = originalMarkNotificationReadAfterJumpFunc }()

	// Scenario 1: Successful pane jump
	jumpToPaneFunc = func(session, window, pane string) bool { return true }
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	originalValidatePaneExistsFunc := validatePaneExistsFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
		validatePaneExistsFunc = originalValidatePaneExistsFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		return "1\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t%1\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return true }
	markNotificationReadAfterJumpFunc = func(id string) error { return nil }

	result, err := Jump("1")
	require.NoError(t, err, "Jump should succeed when JumpToPane returns true")
	assert.NotNil(t, result, "Result should not be nil on success")
	assert.True(t, result.PaneExists, "PaneExists should be true when pane exists")

	// Scenario 2: Fall back to window (pane missing but window exists)
	validatePaneExistsFunc = func(session, window, pane string) bool { return false }
	jumpToPaneFunc = func(session, window, pane string) bool { return true }

	result, err = Jump("1")
	require.NoError(t, err, "Jump should succeed even if pane doesn't exist (window fallback)")
	assert.False(t, result.PaneExists, "PaneExists should be false when pane missing")

	// Scenario 3: Window doesn't exist
	jumpToPaneFunc = func(session, window, pane string) bool { return false }
	result, err = Jump("1")
	assert.Error(t, err, "Jump should fail when window doesn't exist")
	assert.Nil(t, result, "Result should be nil on failure")
}

func TestJumpMarksNotificationReadOnSuccess(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	originalValidatePaneExistsFunc := validatePaneExistsFunc
	originalJumpToPaneFunc := jumpToPaneFunc
	originalMarkNotificationReadAfterJumpFunc := markNotificationReadAfterJumpFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
		validatePaneExistsFunc = originalValidatePaneExistsFunc
		jumpToPaneFunc = originalJumpToPaneFunc
		markNotificationReadAfterJumpFunc = originalMarkNotificationReadAfterJumpFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return true }
	jumpToPaneFunc = func(session, window, pane string) bool { return true }

	called := false
	markedID := ""
	markNotificationReadAfterJumpFunc = func(id string) error {
		called = true
		markedID = id
		return nil
	}

	_, err := Jump("42")
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "42", markedID)
}

func TestJumpDoesNotMarkReadWhenJumpFails(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	originalValidatePaneExistsFunc := validatePaneExistsFunc
	originalJumpToPaneFunc := jumpToPaneFunc
	originalMarkNotificationReadAfterJumpFunc := markNotificationReadAfterJumpFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
		validatePaneExistsFunc = originalValidatePaneExistsFunc
		jumpToPaneFunc = originalJumpToPaneFunc
		markNotificationReadAfterJumpFunc = originalMarkNotificationReadAfterJumpFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return true }
	jumpToPaneFunc = func(session, window, pane string) bool { return false }

	called := false
	markNotificationReadAfterJumpFunc = func(id string) error {
		called = true
		return nil
	}

	_, err := Jump("42")
	require.Error(t, err)
	assert.False(t, called)
}

func TestJumpWithMarkReadFalseSkipsMarkReadAfterSuccessfulJump(t *testing.T) {
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	originalValidatePaneExistsFunc := validatePaneExistsFunc
	originalJumpToPaneFunc := jumpToPaneFunc
	originalMarkNotificationReadAfterJumpFunc := markNotificationReadAfterJumpFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
		validatePaneExistsFunc = originalValidatePaneExistsFunc
		jumpToPaneFunc = originalJumpToPaneFunc
		markNotificationReadAfterJumpFunc = originalMarkNotificationReadAfterJumpFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return true }
	jumpToPaneFunc = func(session, window, pane string) bool { return true }

	called := false
	markNotificationReadAfterJumpFunc = func(id string) error {
		called = true
		return nil
	}

	_, err := JumpWithMarkRead("42", false)
	require.NoError(t, err)
	assert.False(t, called)
}

func TestJumpInvalidFieldData(t *testing.T) {
	// Tiger Style: Test that Jump validates field data properly
	// ASSERTION: Should return error for missing session/window/pane
	originalEnsureTmuxRunningFunc := ensureTmuxRunningFunc
	originalGetNotificationLineFunc := getNotificationLineFunc
	defer func() {
		ensureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		getNotificationLineFunc = originalGetNotificationLineFunc
	}()

	ensureTmuxRunningFunc = func() bool { return true }

	// Test missing session
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t\t%0\t%1\thello\t1234567890\tinfo", nil
	}
	_, err := Jump("42")
	assert.Error(t, err, "Should error when session is missing")
	assert.Contains(t, err.Error(), "missing required fields", "Error message should indicate incomplete context")

	// Test missing window
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t\t%1\thello\t1234567890\tinfo", nil
	}
	_, err = Jump("42")
	assert.Error(t, err, "Should error when window is missing")

	// Test missing pane
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t\thello\t1234567890\tinfo", nil
	}
	_, err = Jump("42")
	assert.Error(t, err, "Should error when pane is missing")
}
