// File: dismiss.go
// Purpose: Implements notification dismissal logic with hook integration,
// supporting single dismissals and filtered bulk operations.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite/sqlcgen"
)

// DismissNotification marks a notification as dismissed.
func (s *SQLiteStorage) DismissNotification(id string) error {
	idInt, err := parseID(id)
	if err != nil {
		return err
	}
	notification, err := s.getNotificationForHooks(idInt)
	if err != nil {
		return err
	}
	if notification.state == "dismissed" {
		return fmt.Errorf("sqlite storage: dismiss notification: %w: id %s", ErrNotificationAlreadyDismissed, id)
	}
	if err := s.dismissSingleNotification(notification); err != nil {
		return err
	}
	s.syncTmuxStatusOption()
	return nil
}

// dismissSingleNotification dismisses a single notification with hooks.
func (s *SQLiteStorage) dismissSingleNotification(notification hookNotification) error {
	envVars := buildNotificationHookEnv(
		notification.id,
		notification.level,
		notification.message,
		escapeMessage(notification.message),
		notification.timestamp,
		notification.session,
		notification.window,
		notification.pane,
		notification.paneCreated,
	)
	if err := hooks.Run("pre-dismiss", envVars...); err != nil {
		return err
	}
	if _, err := s.queries.DismissNotificationByID(context.Background(), sqlcgen.DismissNotificationByIDParams{
		UpdatedAt: utcNow(),
		ID:        notification.id,
	}); err != nil {
		return fmt.Errorf("sqlite storage: dismiss notification: %w", err)
	}
	if err := hooks.Run("post-dismiss", envVars...); err != nil {
		return err
	}
	return nil
}

// DismissAll marks all active notifications as dismissed.
func (s *SQLiteStorage) DismissAll() error {
	if err := hooks.Run("pre-clear"); err != nil {
		return err
	}
	activeNotifications, err := s.listActiveNotificationsForHooks()
	if err != nil {
		return err
	}
	for _, notification := range activeNotifications {
		if err := s.dismissSingleNotification(notification); err != nil {
			return err
		}
	}
	s.syncTmuxStatusOption()
	return nil
}

// DismissByFilter marks active notifications matching the provided filters as dismissed.
// Empty string in a field means "match any value".
func (s *SQLiteStorage) DismissByFilter(session, window, pane string) error {
	// Get all notifications matching the filters before dismissal to run hooks
	activeNotifications, err := s.listActiveNotificationsByFilter(session, window, pane)
	if err != nil {
		return err
	}

	if len(activeNotifications) == 0 {
		return nil
	}

	// Run pre-dismiss hooks and dismiss each notification
	for _, notification := range activeNotifications {
		if err := s.dismissSingleNotification(notification); err != nil {
			return err
		}
	}

	s.syncTmuxStatusOption()
	return nil
}

type hookNotification struct {
	id          int64
	timestamp   string
	state       string
	session     string
	window      string
	pane        string
	message     string
	paneCreated string
	level       string
}

func buildNotificationHookEnv(id int64, level, message, escapedMessage, timestamp, session, window, pane, paneCreated string) []string {
	return []string{
		fmt.Sprintf("NOTIFICATION_ID=%d", id),
		fmt.Sprintf("LEVEL=%s", level),
		fmt.Sprintf("MESSAGE=%s", message),
		fmt.Sprintf("ESCAPED_MESSAGE=%s", escapedMessage),
		fmt.Sprintf("TIMESTAMP=%s", timestamp),
		fmt.Sprintf("SESSION=%s", session),
		fmt.Sprintf("WINDOW=%s", window),
		fmt.Sprintf("PANE=%s", pane),
		fmt.Sprintf("PANE_CREATED=%s", paneCreated),
	}
}

func (s *SQLiteStorage) getNotificationForHooks(id int64) (hookNotification, error) {
	row, err := s.queries.GetNotificationForHooksByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return hookNotification{}, fmt.Errorf("sqlite storage: dismiss notification: %w: id %d", ErrNotificationNotFound, id)
		}
		return hookNotification{}, fmt.Errorf("sqlite storage: dismiss notification: %w", err)
	}

	return hookNotification{
		id:          row.ID,
		timestamp:   row.Timestamp,
		state:       row.State,
		session:     row.Session,
		window:      row.Window,
		pane:        row.Pane,
		message:     row.Message,
		paneCreated: row.PaneCreated,
		level:       row.Level,
	}, nil
}

func (s *SQLiteStorage) listActiveNotificationsForHooks() ([]hookNotification, error) {
	rows, err := s.queries.ListActiveNotificationsForHooks(context.Background())
	if err != nil {
		return nil, fmt.Errorf("sqlite storage: list active notifications: %w", err)
	}

	notifications := make([]hookNotification, 0, len(rows))
	for _, row := range rows {
		notifications = append(notifications, hookNotification{
			id:          row.ID,
			timestamp:   row.Timestamp,
			state:       row.State,
			session:     row.Session,
			window:      row.Window,
			pane:        row.Pane,
			message:     row.Message,
			paneCreated: row.PaneCreated,
			level:       row.Level,
		})
	}

	return notifications, nil
}

// TODO: Optimize with a SQL query that filters in database instead of loading all active notifications.
func (s *SQLiteStorage) listActiveNotificationsByFilter(session, window, pane string) ([]hookNotification, error) {
	// First get all active notifications
	allRows, err := s.queries.ListActiveNotificationsForHooks(context.Background())
	if err != nil {
		return nil, fmt.Errorf("sqlite storage: list active notifications: %w", err)
	}

	// Filter by the provided criteria
	notifications := make([]hookNotification, 0, len(allRows))
	for _, row := range allRows {
		// Apply filters - empty string means match any
		if session != "" && row.Session != session {
			continue
		}
		if window != "" && row.Window != window {
			continue
		}
		if pane != "" && row.Pane != pane {
			continue
		}

		notifications = append(notifications, hookNotification{
			id:          row.ID,
			timestamp:   row.Timestamp,
			state:       row.State,
			session:     row.Session,
			window:      row.Window,
			pane:        row.Pane,
			message:     row.Message,
			paneCreated: row.PaneCreated,
			level:       row.Level,
		})
	}

	return notifications, nil
}
