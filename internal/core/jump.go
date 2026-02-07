// Package core provides core tmux interaction and tray management.
package core

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

// JumpService handles jump-to-pane operations with validation and error handling.
type JumpService struct {
	tmuxClient tmux.TmuxClient
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
	return &JumpService{
		tmuxClient: tmux.NewDefaultClient(),
	}
}

// NewJumpServiceWithDeps creates a JumpService with custom dependencies (for testing).
func NewJumpServiceWithDeps(tmuxClient tmux.TmuxClient) *JumpService {
	return &JumpService{
		tmuxClient: tmuxClient,
	}
}

// JumpToNotification jumps to the pane/window of a notification.
func (s *JumpService) JumpToNotification(notificationID string) (*JumpResult, error) {
	// 1. Get notification from storage
	line, err := storage.GetNotificationByID(notificationID)
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
	// 1. Validate tmux context fields
	if notif.Session == "" {
		return &JumpResult{
			Success: false,
			Message: "Notification has no tmux session context",
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
			Message: "tmux is not running",
		}, nil
	}

	// 3. Validate pane exists
	paneExists, err := s.tmuxClient.ValidatePaneExists(notif.Session, notif.Window, notif.Pane)
	if err != nil {
		return nil, fmt.Errorf("validate pane exists: %w", err)
	}

	if paneExists {
		// 4a. Jump to pane
		success, err := s.tmuxClient.JumpToPane(notif.Session, notif.Window, notif.Pane)
		if err != nil {
			return nil, fmt.Errorf("jump to pane: %w", err)
		}
		return &JumpResult{
			Success:      success,
			JumpedToPane: true,
			Session:      notif.Session,
			Window:       notif.Window,
			Pane:         notif.Pane,
			Message:      fmt.Sprintf("Jumped to %s:%s.%s", notif.Session, notif.Window, notif.Pane),
		}, nil
	}

	// 4b. Fallback: jump to window only
	success, err := s.tmuxClient.JumpToPane(notif.Session, notif.Window, "")
	if err != nil {
		return nil, fmt.Errorf("jump to window: %w", err)
	}
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
			Message: "Session ID cannot be empty",
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
			Message: "tmux is not running",
		}, nil
	}

	// 3. Validate pane exists (if paneID is provided)
	var paneExists bool
	if paneID != "" {
		paneExists, err = s.tmuxClient.ValidatePaneExists(sessionID, windowID, paneID)
		if err != nil {
			return nil, fmt.Errorf("validate pane exists: %w", err)
		}
	}

	if paneExists && paneID != "" {
		// 4a. Jump to pane
		success, err := s.tmuxClient.JumpToPane(sessionID, windowID, paneID)
		if err != nil {
			return nil, fmt.Errorf("jump to pane: %w", err)
		}
		return &JumpResult{
			Success:      success,
			JumpedToPane: true,
			Session:      sessionID,
			Window:       windowID,
			Pane:         paneID,
			Message:      fmt.Sprintf("Jumped to %s:%s.%s", sessionID, windowID, paneID),
		}, nil
	}

	// 4b. Fallback: jump to window only (or if no pane provided)
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
