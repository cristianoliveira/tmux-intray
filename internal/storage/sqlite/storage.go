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

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite/sqlcgen"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
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

var tmuxClient tmux.TmuxClient = tmux.NewDefaultClient()

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

// MarkNotificationRead sets read_timestamp to current UTC time.
func (s *SQLiteStorage) MarkNotificationRead(id string) error {
	return s.markNotificationReadState(id, utcNow())
}

// MarkNotificationUnread clears read_timestamp.
func (s *SQLiteStorage) MarkNotificationUnread(id string) error {
	return s.markNotificationReadState(id, "")
}

// MarkNotificationReadWithTimestamp sets read_timestamp to the provided timestamp.
func (s *SQLiteStorage) MarkNotificationReadWithTimestamp(id, timestamp string) error {
	return s.markNotificationReadState(id, timestamp)
}

// MarkNotificationUnreadWithTimestamp clears read_timestamp (timestamp parameter is ignored, kept for consistency).
func (s *SQLiteStorage) MarkNotificationUnreadWithTimestamp(id, timestamp string) error {
	return s.markNotificationReadState(id, timestamp)
}

func (s *SQLiteStorage) markNotificationReadState(id, readTimestamp string) error {
	idInt, err := parseID(id)
	if err != nil {
		return err
	}

	res, err := s.queries.UpdateReadTimestampByID(context.Background(), sqlcgen.UpdateReadTimestampByIDParams{
		ReadTimestamp: readTimestamp,
		UpdatedAt:     utcNow(),
		ID:            idInt,
	})
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
	cutoff := time.Now().UTC().AddDate(0, 0, -daysThreshold).Format("2006-01-02T15:04:05Z")
	envVars := []string{
		fmt.Sprintf("CLEANUP_DAYS=%d", daysThreshold),
		fmt.Sprintf("CUTOFF_TIMESTAMP=%s", cutoff),
		fmt.Sprintf("DRY_RUN=%t", dryRun),
	}
	if err := hooks.Run("cleanup", envVars...); err != nil {
		return fmt.Errorf("pre-cleanup hook failed: %w", err)
	}

	countCutoff := cutoff
	if daysThreshold == 0 {
		countCutoff = ""
	}

	deletedCount, err := s.queries.CountDismissedForCleanup(context.Background(), countCutoff)
	if err != nil {
		return fmt.Errorf("sqlite storage: count notifications for cleanup: %w", err)
	}
	if deletedCount == 0 {
		postEnv := append(envVars, "DELETED_COUNT=0")
		if err := hooks.Run("post-cleanup", postEnv...); err != nil {
			return fmt.Errorf("post-cleanup hook failed: %w", err)
		}
		return nil
	}

	if dryRun {
		postEnv := append(envVars, fmt.Sprintf("DELETED_COUNT=%d", deletedCount))
		if err := hooks.Run("post-cleanup", postEnv...); err != nil {
			return fmt.Errorf("post-cleanup hook failed: %w", err)
		}
		return nil
	}

	deleteCutoff := cutoff
	if daysThreshold == 0 {
		deleteCutoff = ""
	}

	if err := s.queries.DeleteDismissedForCleanup(context.Background(), deleteCutoff); err != nil {
		return fmt.Errorf("sqlite storage: cleanup old notifications: %w", err)
	}
	postEnv := append(envVars, fmt.Sprintf("DELETED_COUNT=%d", deletedCount))
	if err := hooks.Run("post-cleanup", postEnv...); err != nil {
		return fmt.Errorf("post-cleanup hook failed: %w", err)
	}

	return nil
}

// SetTmuxClient sets the tmux client used for status updates.
func SetTmuxClient(client tmux.TmuxClient) {
	if client == nil {
		return
	}
	tmuxClient = client
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

func (s *SQLiteStorage) nextNotificationID() (int64, error) {
	id, err := s.queries.NextNotificationID(context.Background())
	if err != nil {
		return 0, fmt.Errorf("sqlite storage: get next id: %w", err)
	}
	return id, nil
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

func (s *SQLiteStorage) syncTmuxStatusOption() {
	if err := s.updateTmuxStatusOption(s.GetActiveCount()); err != nil {
		colors.Error(fmt.Sprintf("failed to update tmux status: %v", err))
	}
}

func (s *SQLiteStorage) updateTmuxStatusOption(count int) error {
	running, err := tmuxClient.HasSession()
	if err != nil {
		return fmt.Errorf("updateTmuxStatusOption: tmux not available: %w", err)
	}
	if !running {
		return fmt.Errorf("updateTmuxStatusOption: tmux not running")
	}
	if err := tmuxClient.SetStatusOption("@tmux_intray_active_count", fmt.Sprintf("%d", count)); err != nil {
		return fmt.Errorf("updateTmuxStatusOption: failed to set @tmux_intray_active_count to %d: %w", count, err)
	}
	return nil
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
