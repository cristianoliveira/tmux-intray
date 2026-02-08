// Package sqlite provides a SQLite-backed storage implementation.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
	_ "modernc.org/sqlite"
)

var (
	// ErrInvalidNotificationID indicates an empty or malformed notification ID.
	ErrInvalidNotificationID = errors.New("invalid notification ID")
	// ErrNotificationNotFound indicates that a notification cannot be found.
	ErrNotificationNotFound = errors.New("notification not found")
	// ErrNotificationAlreadyDismissed indicates the notification is already dismissed.
	ErrNotificationAlreadyDismissed = errors.New("notification already dismissed")
)

var validLevels = map[string]bool{
	"info":     true,
	"warning":  true,
	"error":    true,
	"critical": true,
}

var validStates = map[string]bool{
	"active":    true,
	"dismissed": true,
	"all":       true,
}

// SQLiteStorage implements the storage.Storage interface using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

var _ storage.Storage = (*SQLiteStorage)(nil)

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

	storage := &SQLiteStorage{db: db}
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

	const schema = `
CREATE TABLE IF NOT EXISTS notifications (
	id INTEGER PRIMARY KEY,
	timestamp TEXT NOT NULL CHECK (strftime('%s', timestamp) IS NOT NULL),
	state TEXT NOT NULL CHECK (state IN ('active', 'dismissed')),
	session TEXT NOT NULL DEFAULT '',
	window TEXT NOT NULL DEFAULT '',
	pane TEXT NOT NULL DEFAULT '',
	message TEXT NOT NULL,
	pane_created TEXT NOT NULL DEFAULT '' CHECK (pane_created = '' OR strftime('%s', pane_created) IS NOT NULL),
	level TEXT NOT NULL CHECK (level IN ('info', 'warning', 'error', 'critical')),
	read_timestamp TEXT NOT NULL DEFAULT '' CHECK (read_timestamp = '' OR strftime('%s', read_timestamp) IS NOT NULL),
	updated_at TEXT NOT NULL CHECK (strftime('%s', updated_at) IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_notifications_state ON notifications(state);
CREATE INDEX IF NOT EXISTS idx_notifications_level ON notifications(level);
CREATE INDEX IF NOT EXISTS idx_notifications_session ON notifications(session);
CREATE INDEX IF NOT EXISTS idx_notifications_timestamp ON notifications(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_state_timestamp ON notifications(state, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_session_state_timestamp ON notifications(session, state, timestamp DESC);
`

	if _, err := s.db.Exec(schema); err != nil {
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

	now := utcNow()
	res, err := s.db.Exec(
		`INSERT INTO notifications (timestamp, state, session, window, pane, message, pane_created, level, read_timestamp, updated_at)
		 VALUES (?, 'active', ?, ?, ?, ?, ?, ?, '', ?)`,
		timestamp, session, window, pane, message, paneCreated, level, now,
	)
	if err != nil {
		return "", fmt.Errorf("sqlite storage: add notification: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return "", fmt.Errorf("sqlite storage: get inserted id: %w", err)
	}

	return strconv.FormatInt(id, 10), nil
}

// ListNotifications returns TSV lines matching all provided filters.
func (s *SQLiteStorage) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
	if err := validateListInputs(stateFilter, levelFilter, olderThanCutoff, newerThanCutoff); err != nil {
		return "", err
	}

	query := `SELECT id, timestamp, state, session, window, pane, message, pane_created, level, read_timestamp
		FROM notifications WHERE 1=1`
	args := []any{}

	if stateFilter != "" && stateFilter != "all" {
		query += " AND state = ?"
		args = append(args, stateFilter)
	}
	if levelFilter != "" {
		query += " AND level = ?"
		args = append(args, levelFilter)
	}
	if sessionFilter != "" {
		query += " AND session = ?"
		args = append(args, sessionFilter)
	}
	if windowFilter != "" {
		query += " AND window = ?"
		args = append(args, windowFilter)
	}
	if paneFilter != "" {
		query += " AND pane = ?"
		args = append(args, paneFilter)
	}
	if olderThanCutoff != "" {
		query += " AND timestamp < ?"
		args = append(args, olderThanCutoff)
	}
	if newerThanCutoff != "" {
		query += " AND timestamp > ?"
		args = append(args, newerThanCutoff)
	}

	query += " ORDER BY id ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return "", fmt.Errorf("sqlite storage: list notifications: %w", err)
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		line, err := scanNotificationLine(rows)
		if err != nil {
			return "", err
		}
		lines = append(lines, line)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("sqlite storage: iterate notifications: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}

// GetNotificationByID retrieves a single notification by ID as TSV.
func (s *SQLiteStorage) GetNotificationByID(id string) (string, error) {
	idInt, err := parseID(id)
	if err != nil {
		return "", err
	}

	row := s.db.QueryRow(
		`SELECT id, timestamp, state, session, window, pane, message, pane_created, level, read_timestamp
		 FROM notifications WHERE id = ?`,
		idInt,
	)

	line, err := scanNotificationLine(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("sqlite storage: get notification: %w: id %s", ErrNotificationNotFound, id)
		}
		return "", err
	}

	return line, nil
}

// DismissNotification marks a notification as dismissed.
func (s *SQLiteStorage) DismissNotification(id string) error {
	idInt, err := parseID(id)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("sqlite storage: begin dismiss transaction: %w", err)
	}
	defer tx.Rollback()

	var state string
	err = tx.QueryRow("SELECT state FROM notifications WHERE id = ?", idInt).Scan(&state)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("sqlite storage: dismiss notification: %w: id %s", ErrNotificationNotFound, id)
		}
		return fmt.Errorf("sqlite storage: dismiss notification: %w", err)
	}
	if state == "dismissed" {
		return fmt.Errorf("sqlite storage: dismiss notification: %w: id %s", ErrNotificationAlreadyDismissed, id)
	}

	if _, err := tx.Exec(
		"UPDATE notifications SET state = 'dismissed', updated_at = ? WHERE id = ?",
		utcNow(),
		idInt,
	); err != nil {
		return fmt.Errorf("sqlite storage: update dismissed state: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite storage: commit dismiss transaction: %w", err)
	}

	return nil
}

// DismissAll marks all active notifications as dismissed.
func (s *SQLiteStorage) DismissAll() error {
	if _, err := s.db.Exec(
		"UPDATE notifications SET state = 'dismissed', updated_at = ? WHERE state = 'active'",
		utcNow(),
	); err != nil {
		return fmt.Errorf("sqlite storage: dismiss all notifications: %w", err)
	}
	return nil
}

// MarkNotificationRead sets read_timestamp to current UTC time.
func (s *SQLiteStorage) MarkNotificationRead(id string) error {
	return s.markNotificationReadState(id, utcNow())
}

// MarkNotificationUnread clears read_timestamp.
func (s *SQLiteStorage) MarkNotificationUnread(id string) error {
	return s.markNotificationReadState(id, "")
}

func (s *SQLiteStorage) markNotificationReadState(id, readTimestamp string) error {
	idInt, err := parseID(id)
	if err != nil {
		return err
	}

	res, err := s.db.Exec(
		"UPDATE notifications SET read_timestamp = ?, updated_at = ? WHERE id = ?",
		readTimestamp,
		utcNow(),
		idInt,
	)
	if err != nil {
		return fmt.Errorf("sqlite storage: update read state: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite storage: read rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("sqlite storage: mark read state: %w: id %s", ErrNotificationNotFound, id)
	}

	return nil
}

// CleanupOldNotifications removes dismissed notifications older than threshold days.
func (s *SQLiteStorage) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	if daysThreshold < 0 {
		return fmt.Errorf("sqlite storage: days threshold must be >= 0")
	}

	if dryRun {
		return nil
	}

	if daysThreshold == 0 {
		_, err := s.db.Exec("DELETE FROM notifications WHERE state = 'dismissed'")
		if err != nil {
			return fmt.Errorf("sqlite storage: cleanup dismissed notifications: %w", err)
		}
		return nil
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -daysThreshold).Format("2006-01-02T15:04:05Z")
	_, err := s.db.Exec(
		"DELETE FROM notifications WHERE state = 'dismissed' AND timestamp < ?",
		cutoff,
	)
	if err != nil {
		return fmt.Errorf("sqlite storage: cleanup old notifications: %w", err)
	}

	return nil
}

// GetActiveCount returns the number of active notifications.
func (s *SQLiteStorage) GetActiveCount() int {
	var count int
	err := s.db.QueryRow("SELECT COUNT(1) FROM notifications WHERE state = 'active'").Scan(&count)
	if err != nil {
		return 0
	}
	return count
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

type scanner interface {
	Scan(dest ...any) error
}

func scanNotificationLine(s scanner) (string, error) {
	var id int64
	var timestamp, state, session, window, pane, message, paneCreated, level, readTimestamp string
	if err := s.Scan(&id, &timestamp, &state, &session, &window, &pane, &message, &paneCreated, &level, &readTimestamp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		return "", fmt.Errorf("sqlite storage: scan notification: %w", err)
	}

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
	), nil
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

func escapeMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\\", "\\\\")
	msg = strings.ReplaceAll(msg, "\t", "\\t")
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return msg
}

func utcNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}
