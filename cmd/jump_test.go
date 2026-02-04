package cmd

import (
	"errors"
	"testing"
)

func TestJumpSuccess(t *testing.T) {
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

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		if id != "42" {
			return "", errors.New("not found")
		}
		// Simulate TSV line: ID, timestamp, state, session, window, pane, message, pane_created, level
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return true }
	jumpToPaneFunc = func(session, window, pane string) bool { return true }

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
	if err.Error() != "tmux is not running" {
		t.Errorf("Expected 'tmux is not running', got %v", err)
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
	if err.Error() != "notification 42 has no pane association" {
		t.Errorf("Expected 'notification 42 has no pane association', got %v", err)
	}
}

func TestJumpPaneDoesNotExistButWindowSelected(t *testing.T) {
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

	ensureTmuxRunningFunc = func() bool { return true }
	getNotificationLineFunc = func(id string) (string, error) {
		return "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo", nil
	}
	validatePaneExistsFunc = func(session, window, pane string) bool { return false }
	jumpToPaneFunc = func(session, window, pane string) bool { return true }

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
	if err.Error() != "failed to jump to pane (maybe window no longer exists)" {
		t.Errorf("Expected 'failed to jump to pane (maybe window no longer exists)', got %v", err)
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
