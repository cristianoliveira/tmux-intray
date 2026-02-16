// Package sqlite provides a SQLite-backed storage implementation.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite/sqlcgen"
	_ "modernc.org/sqlite"
)

// SQLiteStorage implements the storage.Storage interface using SQLite.
type SQLiteStorage struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

// NewSQLiteStorage creates a SQLite-backed storage at the provided path.
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("sqlite storage: db path cannot be empty")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("sqlite storage: create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite storage: open db: %w", err)
	}

	storage := &SQLiteStorage{db: db, queries: sqlcgen.New(db)}
	if err := storage.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return storage, nil
}

// Close closes the underlying SQLite connection.
func (s *SQLiteStorage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStorage) init() error {
	if _, err := s.db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return fmt.Errorf("sqlite storage: set busy timeout: %w", err)
	}

	if _, err := s.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("sqlite storage: create schema: %w", err)
	}

	return nil
}

// AddNotification adds a notification and returns its generated ID.
func (s *SQLiteStorage) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	if err := validateNotificationInputs(message, timestamp, session, window, pane, level); err != nil {
		return "", err
	}
	if timestamp == "" {
		timestamp = utcNow()
	}
	id, err := s.nextNotificationID()
	if err != nil {
		return "", err
	}
	escapedMessage := escapeMessage(message)
	envVars := buildNotificationHookEnv(id, level, message, escapedMessage, timestamp, session, window, pane, paneCreated)
	if err := hooks.Run("pre-add", envVars...); err != nil {
		return "", fmt.Errorf("pre-add hook aborted: %w", err)
	}

	now := utcNow()
	err = s.queries.CreateNotification(context.Background(), sqlcgen.CreateNotificationParams{
		ID:          id,
		Timestamp:   timestamp,
		Session:     session,
		Window:      window,
		Pane:        pane,
		Message:     message,
		PaneCreated: paneCreated,
		Level:       level,
		UpdatedAt:   now,
	})
	if err != nil {
		return "", fmt.Errorf("sqlite storage: add notification: %w", err)
	}
	s.syncTmuxStatusOption()
	if err := hooks.Run("post-add", envVars...); err != nil {
		return strconv.FormatInt(id, 10), fmt.Errorf("post-add hook failed: %w", err)
	}

	return strconv.FormatInt(id, 10), nil
}

// ListNotifications returns TSV lines matching all provided filters.
func (s *SQLiteStorage) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	if err := validateListInputs(stateFilter, levelFilter, olderThanCutoff, newerThanCutoff); err != nil {
		return "", err
	}

	rows, err := s.queries.ListNotifications(context.Background(), sqlcgen.ListNotificationsParams{
		StateFilter:     stateFilter,
		LevelFilter:     levelFilter,
		SessionFilter:   sessionFilter,
		WindowFilter:    windowFilter,
		PaneFilter:      paneFilter,
		OlderThanCutoff: olderThanCutoff,
		NewerThanCutoff: newerThanCutoff,
		ReadFilter:      readFilter,
	})
	if err != nil {
		return "", fmt.Errorf("sqlite storage: list notifications: %w", err)
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, formatNotificationLine(
			row.ID,
			row.Timestamp,
			row.State,
			row.Session,
			row.Window,
			row.Pane,
			row.Message,
			row.PaneCreated,
			row.Level,
			row.ReadTimestamp,
		))
	}

	return strings.Join(lines, "\n"), nil
}

// GetNotificationByID retrieves a single notification by ID as TSV.
func (s *SQLiteStorage) GetNotificationByID(id string) (string, error) {
	idInt, err := parseID(id)
	if err != nil {
		return "", err
	}

	row, err := s.queries.GetNotificationLineByID(context.Background(), idInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("sqlite storage: get notification: %w: id %s", ErrNotificationNotFound, id)
		}
		return "", fmt.Errorf("sqlite storage: get notification: %w", err)
	}

	return formatNotificationLine(
		row.ID,
		row.Timestamp,
		row.State,
		row.Session,
		row.Window,
		row.Pane,
		row.Message,
		row.PaneCreated,
		row.Level,
		row.ReadTimestamp,
	), nil
}

// GetActiveCount returns the number of active notifications.
func (s *SQLiteStorage) GetActiveCount() int {
	count, err := s.queries.CountActiveNotifications(context.Background())
	if err != nil {
		return 0
	}
	return int(count)
}

func validateNotificationInputs(message, timestamp, session, window, pane, level string) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("validation error: message cannot be empty")
	}
	if level == "" {
		return fmt.Errorf("validation error: level cannot be empty")
	}
	if !validLevels[level] {
		return fmt.Errorf("validation error: invalid level '%s', must be one of: info, warning, error, critical", level)
	}
	if timestamp != "" {
		if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
			return fmt.Errorf("validation error: invalid timestamp format '%s', expected RFC3339 format", timestamp)
		}
	}
	if session != "" && strings.TrimSpace(session) == "" {
		return fmt.Errorf("validation error: session cannot be whitespace only")
	}
	if window != "" && strings.TrimSpace(window) == "" {
		return fmt.Errorf("validation error: window cannot be whitespace only")
	}
	if pane != "" && strings.TrimSpace(pane) == "" {
		return fmt.Errorf("validation error: pane cannot be whitespace only")
	}

	return nil
}

func validateListInputs(stateFilter, levelFilter, olderThanCutoff, newerThanCutoff string) error {
	if stateFilter != "" && !validStates[stateFilter] {
		return fmt.Errorf("invalid state '%s', must be one of: active, dismissed, all, or empty", stateFilter)
	}
	if levelFilter != "" && !validLevels[levelFilter] {
		return fmt.Errorf("invalid level '%s', must be one of: info, warning, error, critical, or empty", levelFilter)
	}
	if olderThanCutoff != "" {
		if _, err := time.Parse(time.RFC3339, olderThanCutoff); err != nil {
			return fmt.Errorf("invalid olderThanCutoff format '%s', expected RFC3339 format", olderThanCutoff)
		}
	}
	if newerThanCutoff != "" {
		if _, err := time.Parse(time.RFC3339, newerThanCutoff); err != nil {
			return fmt.Errorf("invalid newerThanCutoff format '%s', expected RFC3339 format", newerThanCutoff)
		}
	}
	return nil
}

func formatNotificationLine(id int64, timestamp, state, session, window, pane, message, paneCreated, level, readTimestamp string) string {
	return fmt.Sprintf(
		"%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
		id,
		timestamp,
		state,
		session,
		window,
		pane,
		escapeMessage(message),
		paneCreated,
		level,
		readTimestamp,
	)
}

func parseID(id string) (int64, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return 0, fmt.Errorf("sqlite storage: %w", ErrInvalidNotificationID)
	}
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil || idInt <= 0 {
		return 0, fmt.Errorf("sqlite storage: %w", ErrInvalidNotificationID)
	}
	return idInt, nil
}

func (s *SQLiteStorage) nextNotificationID() (int64, error) {
	id, err := s.queries.NextNotificationID(context.Background())
	if err != nil {
		return 0, fmt.Errorf("sqlite storage: get next id: %w", err)
	}
	return id, nil
}

func escapeMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\\", "\\\\")
	msg = strings.ReplaceAll(msg, "\t", "\\t")
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return msg
}

func utcNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}
