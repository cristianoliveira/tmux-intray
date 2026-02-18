// Package core provides core tmux interaction and tray management.
package core

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/logging"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

// JumpService handles jump-to-pane operations with validation and error handling.
type JumpService struct {
	tmuxClient tmux.TmuxClient
	storage    storage.Storage
}

// JumpResult contains the result of a jump operation.
type JumpResult struct {
	Success      bool   // Overall success
	JumpedToPane bool   // true = jumped to pane, false = fell back to window
	Session      string // Target session ID
	Window       string // Target window ID
	Pane         string // Target pane ID (empty if fell back)
	Message      string // User-friendly message
}

// NewJumpService creates a new JumpService with default dependencies.
func NewJumpService() *JumpService {
	return NewJumpServiceWithDeps(tmux.NewDefaultClient(), nil)
}

// NewJumpServiceWithDeps creates a JumpService with custom dependencies (for testing).
// Panics if storage initialization fails, which is safer than continuing with nil storage.
func NewJumpServiceWithDeps(tmuxClient tmux.TmuxClient, stor storage.Storage) *JumpService {
	if tmuxClient == nil {
		tmuxClient = tmux.NewDefaultClient()
	}
	if stor == nil {
		fileStor, err := storage.NewFromConfig()
		if err != nil {
			panic(fmt.Sprintf("failed to initialize storage: %v", err))
		}
		stor = fileStor
	}
	return &JumpService{
		tmuxClient: tmuxClient,
		storage:    stor,
	}
}

// JumpToNotification jumps to the pane/window of a notification.
func (s *JumpService) JumpToNotification(notificationID string) (*JumpResult, error) {
	// 1. Get notification from storage
	line, err := s.storage.GetNotificationByID(notificationID)
	if err != nil {
		return nil, fmt.Errorf("get notification: %w", err)
	}

	// 2. Parse notification
	notif, err := notification.ParseNotification(line)
	if err != nil {
		return nil, fmt.Errorf("parse notification: %w", err)
	}

	return s.jumpToNotificationInternal(&notif)
}

// JumpToNotificationParsed jumps to the pane/window of a parsed notification.
// This is useful when you already have a parsed notification.
func (s *JumpService) JumpToNotificationParsed(notif *notification.Notification) (*JumpResult, error) {
	return s.jumpToNotificationInternal(notif)
}

// jumpToNotificationInternal contains the core logic for jumping to a notification.
func (s *JumpService) jumpToNotificationInternal(notif *notification.Notification) (*JumpResult, error) {
	result := s.validateNotificationContext(notif)
	if result != nil {
		return result, nil
	}

	result, err := s.checkTmuxRunningForJump()
	if result != nil || err != nil {
		return result, err
	}

	paneExists, err := s.validateNotificationPaneExists(notif)
	if err != nil {
		return nil, err
	}

	if paneExists {
		return s.jumpToNotificationPane(notif)
	}

	return s.jumpToNotificationWindow(notif)
}

func (s *JumpService) validateNotificationContext(notif *notification.Notification) *JumpResult {
	logging.StructuredDebug("core/jump", "jump_to_notification", "started", nil, "", map[string]interface{}{
		"session": notif.Session,
		"window":  notif.Window,
		"pane":    notif.Pane,
		"level":   notif.Level,
	})
	if notif.Session == "" {
		return &JumpResult{
			Success: false,
			Message: "notification has no tmux session context",
		}
	}
	return nil
}

func (s *JumpService) checkTmuxRunningForJump() (*JumpResult, error) {
	logging.StructuredDebug("core/jump", "check_tmux_running", "started", nil, "", nil)
	running, err := s.tmuxClient.HasSession()
	if err != nil {
		logging.StructuredError("core/jump", "check_tmux_running", "error", err, "", nil)
		return nil, fmt.Errorf("check tmux running: %w", err)
	}
	if !running {
		logging.StructuredDebug("core/jump", "check_tmux_running", "tmux_not_running", nil, "", nil)
		return &JumpResult{
			Success: false,
			Message: "tmux not running",
		}, nil
	}
	return nil, nil
}

func (s *JumpService) validateNotificationPaneExists(notif *notification.Notification) (bool, error) {
	logging.StructuredDebug("core/jump", "validate_pane_exists", "started", nil, "", map[string]interface{}{
		"session": notif.Session,
		"window":  notif.Window,
		"pane":    notif.Pane,
	})
	paneExists, err := s.tmuxClient.ValidatePaneExists(notif.Session, notif.Window, notif.Pane)
	if err != nil {
		logging.StructuredError("core/jump", "validate_pane_exists", "error", err, "", nil)
		return false, fmt.Errorf("validate pane exists: %w", err)
	}
	logging.StructuredDebug("core/jump", "validate_pane_exists", "result", nil, "", map[string]interface{}{
		"pane_exists": paneExists,
	})
	return paneExists, nil
}

func (s *JumpService) jumpToNotificationPane(notif *notification.Notification) (*JumpResult, error) {
	logging.StructuredDebug("core/jump", "jump_to_pane", "started", nil, "", map[string]interface{}{
		"session": notif.Session,
		"window":  notif.Window,
		"pane":    notif.Pane,
	})
	success, err := s.tmuxClient.JumpToPane(notif.Session, notif.Window, notif.Pane)
	if err != nil {
		logging.StructuredError("core/jump", "jump_to_pane", "error", err, "", nil)
		return nil, fmt.Errorf("jump to pane: %w", err)
	}
	logging.StructuredInfo("core/jump", "jump_to_pane", "success", nil, "", map[string]interface{}{
		"success": success,
	})
	return &JumpResult{
		Success:      success,
		JumpedToPane: true,
		Session:      notif.Session,
		Window:       notif.Window,
		Pane:         notif.Pane,
		Message:      fmt.Sprintf("Jumped to %s:%s.%s", notif.Session, notif.Window, notif.Pane),
	}, nil
}

func (s *JumpService) jumpToNotificationWindow(notif *notification.Notification) (*JumpResult, error) {
	logging.StructuredDebug("core/jump", "jump_to_window", "started", nil, "", map[string]interface{}{
		"session": notif.Session,
		"window":  notif.Window,
	})
	success, err := s.tmuxClient.JumpToPane(notif.Session, notif.Window, "")
	if err != nil {
		logging.StructuredError("core/jump", "jump_to_window", "error", err, "", nil)
		return nil, fmt.Errorf("jump to window: %w", err)
	}
	logging.StructuredInfo("core/jump", "jump_to_window", "success", nil, "", map[string]interface{}{
		"success": success,
	})
	return &JumpResult{
		Success:      success,
		JumpedToPane: false,
		Session:      notif.Session,
		Window:       notif.Window,
		Pane:         "",
		Message:      fmt.Sprintf("Jumped to %s:%s (pane not found)", notif.Session, notif.Window),
	}, nil
}

// JumpToContext jumps directly to a tmux context.
func (s *JumpService) JumpToContext(sessionID, windowID, paneID string) (*JumpResult, error) {
	// 1. Validate session context
	if sessionID == "" {
		return &JumpResult{
			Success: false,
			Message: "session id cannot be empty",
		}, nil
	}

	// 2. Check tmux is running
	running, err := s.tmuxClient.HasSession()
	if err != nil {
		return nil, fmt.Errorf("check tmux running: %w", err)
	}
	if !running {
		return &JumpResult{
			Success: false,
			Message: "tmux not running",
		}, nil
	}

	// 3. Validate pane exists (if paneID is provided)
	paneExists := false
	if paneID != "" {
		paneExists, err = s.tmuxClient.ValidatePaneExists(sessionID, windowID, paneID)
		if err != nil {
			return nil, fmt.Errorf("validate pane exists: %w", err)
		}
	}

	if paneExists && paneID != "" {
		return s.jumpToPane(sessionID, windowID, paneID, true)
	}

	// Fallback: jump to window only (or if no pane provided)
	return s.jumpToWindow(sessionID, windowID, paneID)
}

// jumpToPane jumps to a specific pane.
func (s *JumpService) jumpToPane(sessionID, windowID, paneID string, jumpedToPane bool) (*JumpResult, error) {
	success, err := s.tmuxClient.JumpToPane(sessionID, windowID, paneID)
	if err != nil {
		return nil, fmt.Errorf("jump to pane: %w", err)
	}
	return &JumpResult{
		Success:      success,
		JumpedToPane: jumpedToPane,
		Session:      sessionID,
		Window:       windowID,
		Pane:         paneID,
		Message:      fmt.Sprintf("Jumped to %s:%s.%s", sessionID, windowID, paneID),
	}, nil
}

// jumpToWindow jumps to a specific window.
func (s *JumpService) jumpToWindow(sessionID, windowID, paneID string) (*JumpResult, error) {
	success, err := s.tmuxClient.JumpToPane(sessionID, windowID, "")
	if err != nil {
		return nil, fmt.Errorf("jump to window: %w", err)
	}

	if paneID != "" {
		return &JumpResult{
			Success:      success,
			JumpedToPane: false,
			Session:      sessionID,
			Window:       windowID,
			Pane:         "",
			Message:      fmt.Sprintf("Jumped to %s:%s (pane not found)", sessionID, windowID),
		}, nil
	}

	return &JumpResult{
		Success:      success,
		JumpedToPane: false,
		Session:      sessionID,
		Window:       windowID,
		Pane:         "",
		Message:      fmt.Sprintf("Jumped to %s:%s", sessionID, windowID),
	}, nil
}
