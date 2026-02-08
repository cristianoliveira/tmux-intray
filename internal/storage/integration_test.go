package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"database/sql"

	sqlitebackend "github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type parityBackend interface {
	AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error)
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error)
	GetNotificationByID(id string) (string, error)
	DismissNotification(id string) error
	DismissAll() error
	MarkNotificationRead(id string) error
	MarkNotificationUnread(id string) error
	CleanupOldNotifications(daysThreshold int, dryRun bool) error
	GetActiveCount() int
	Close() error
}

type tsvParityBackend struct {
	storage *FileStorage
}

func newTSVParityBackend(t *testing.T) parityBackend {
	t.Helper()

	tmpDir := t.TempDir()
	require.NoError(t, os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0"))
	require.NoError(t, os.Setenv("TMUX_INTRAY_DEBUG", "false"))

	Reset()
	mockClient := new(tmux.MockClient)
	mockClient.On("HasSession").Return(true, nil)
	mockClient.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	SetTmuxClient(mockClient)

	fileStorage, err := NewFileStorage()
	require.NoError(t, err)

	return &tsvParityBackend{storage: fileStorage}
}

func (b *tsvParityBackend) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	return b.storage.AddNotification(message, timestamp, session, window, pane, paneCreated, level)
}

func (b *tsvParityBackend) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
	return b.storage.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
}

func (b *tsvParityBackend) GetNotificationByID(id string) (string, error) {
	return b.storage.GetNotificationByID(id)
}

func (b *tsvParityBackend) DismissNotification(id string) error {
	return b.storage.DismissNotification(id)
}

func (b *tsvParityBackend) DismissAll() error {
	return b.storage.DismissAll()
}

func (b *tsvParityBackend) MarkNotificationRead(id string) error {
	return b.storage.MarkNotificationRead(id)
}

func (b *tsvParityBackend) MarkNotificationUnread(id string) error {
	return b.storage.MarkNotificationUnread(id)
}

func (b *tsvParityBackend) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	return b.storage.CleanupOldNotifications(daysThreshold, dryRun)
}

func (b *tsvParityBackend) GetActiveCount() int {
	return b.storage.GetActiveCount()
}

func (b *tsvParityBackend) Close() error {
	return nil
}

type sqliteParityBackend struct {
	db *sql.DB
	mu sync.Mutex
}

func newSQLiteParityBackend(t *testing.T) parityBackend {
	t.Helper()

	dsn := fmt.Sprintf("file:parity_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)

	schema := `
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY,
    timestamp TEXT NOT NULL,
    state TEXT NOT NULL,
    session TEXT NOT NULL,
    window TEXT NOT NULL,
    pane TEXT NOT NULL,
    message TEXT NOT NULL,
    pane_created TEXT NOT NULL,
    level TEXT NOT NULL,
    read_timestamp TEXT NOT NULL DEFAULT ''
);
`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return &sqliteParityBackend{db: db}
}

func (b *sqliteParityBackend) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	if err := validateNotificationInputs(message, timestamp, session, window, pane, paneCreated, level); err != nil {
		return "", err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	tx, err := b.db.Begin()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	id := 1
	row := tx.QueryRow(`SELECT COALESCE(MAX(id), 0) + 1 FROM notifications`)
	if err := row.Scan(&id); err != nil {
		return "", err
	}

	if timestamp == "" {
		timestamp = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	}

	escapedMessage := escapeMessage(message)

	_, err = tx.Exec(`
INSERT INTO notifications (id, timestamp, state, session, window, pane, message, pane_created, level, read_timestamp)
VALUES (?, ?, 'active', ?, ?, ?, ?, ?, ?, '')
`, id, timestamp, session, window, pane, escapedMessage, paneCreated, level)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return strconv.Itoa(id), nil
}

func (b *sqliteParityBackend) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
	if err := validateListInputs(stateFilter, levelFilter, olderThanCutoff, newerThanCutoff); err != nil {
		return "", err
	}

	lines, err := b.allLines()
	if err != nil {
		return "", err
	}

	filtered := filterNotifications(lines, stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
	return strings.Join(filtered, "\n"), nil
}

func (b *sqliteParityBackend) GetNotificationByID(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("get notification by id: %w", ErrInvalidNotificationID)
	}

	row := b.db.QueryRow(`
SELECT id, timestamp, state, session, window, pane, message, pane_created, level, read_timestamp
FROM notifications
WHERE id = ?
`, id)

	var n normalizedNotification
	if err := row.Scan(&n.ID, &n.Timestamp, &n.State, &n.Session, &n.Window, &n.Pane, &n.Message, &n.PaneCreated, &n.Level, &n.ReadTimestamp); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("get notification by id: %w: id %s", ErrNotificationNotFound, id)
		}
		return "", err
	}

	return n.toTSVLine(), nil
}

func (b *sqliteParityBackend) DismissNotification(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	tx, err := b.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var state string
	row := tx.QueryRow(`SELECT state FROM notifications WHERE id = ?`, id)
	if err := row.Scan(&state); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("dismiss notification: %w: id %s", ErrNotificationNotFound, id)
		}
		return err
	}
	if state == "dismissed" {
		return fmt.Errorf("dismiss notification: %w: id %s", ErrNotificationAlreadyDismissed, id)
	}

	_, err = tx.Exec(`UPDATE notifications SET state = 'dismissed' WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (b *sqliteParityBackend) DismissAll() error {
	_, err := b.db.Exec(`UPDATE notifications SET state = 'dismissed' WHERE state = 'active'`)
	return err
}

func (b *sqliteParityBackend) MarkNotificationRead(id string) error {
	return b.markReadState(id, time.Now().UTC().Format(time.RFC3339))
}

func (b *sqliteParityBackend) MarkNotificationUnread(id string) error {
	return b.markReadState(id, "")
}

func (b *sqliteParityBackend) markReadState(id, readTimestamp string) error {
	res, err := b.db.Exec(`UPDATE notifications SET read_timestamp = ? WHERE id = ?`, readTimestamp, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("markNotificationReadState: %w: id %s", ErrNotificationNotFound, id)
	}
	return nil
}

func (b *sqliteParityBackend) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -daysThreshold).Format("2006-01-02T15:04:05Z")

	query := `DELETE FROM notifications WHERE state = 'dismissed'`
	args := []any{}
	if daysThreshold > 0 {
		query += ` AND timestamp < ?`
		args = append(args, cutoff)
	}
	if dryRun {
		query = strings.Replace(query, "DELETE", "SELECT id", 1)
		rows, err := b.db.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		return rows.Err()
	}

	_, err := b.db.Exec(query, args...)
	return err
}

func (b *sqliteParityBackend) GetActiveCount() int {
	row := b.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE state = 'active'`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return count
}

func (b *sqliteParityBackend) Close() error {
	return b.db.Close()
}

func (b *sqliteParityBackend) allLines() ([]string, error) {
	rows, err := b.db.Query(`
SELECT id, timestamp, state, session, window, pane, message, pane_created, level, read_timestamp
FROM notifications
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var n normalizedNotification
		if err := rows.Scan(&n.ID, &n.Timestamp, &n.State, &n.Session, &n.Window, &n.Pane, &n.Message, &n.PaneCreated, &n.Level, &n.ReadTimestamp); err != nil {
			return nil, err
		}
		lines = append(lines, n.toTSVLine())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

type normalizedNotification struct {
	ID            string
	Timestamp     string
	State         string
	Session       string
	Window        string
	Pane          string
	Message       string
	PaneCreated   string
	Level         string
	ReadTimestamp string
}

func (n normalizedNotification) toTSVLine() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
		n.ID,
		n.Timestamp,
		n.State,
		n.Session,
		n.Window,
		n.Pane,
		n.Message,
		n.PaneCreated,
		n.Level,
		n.ReadTimestamp,
	)
}

func parseTSVLines(t *testing.T, raw string) []normalizedNotification {
	t.Helper()

	if strings.TrimSpace(raw) == "" {
		return []normalizedNotification{}
	}

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	out := make([]normalizedNotification, 0, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		for len(fields) < numFields {
			fields = append(fields, "")
		}
		out = append(out, normalizedNotification{
			ID:            fields[fieldID],
			Timestamp:     fields[fieldTimestamp],
			State:         fields[fieldState],
			Session:       fields[fieldSession],
			Window:        fields[fieldWindow],
			Pane:          fields[fieldPane],
			Message:       fields[fieldMessage],
			PaneCreated:   fields[fieldPaneCreated],
			Level:         fields[fieldLevel],
			ReadTimestamp: normalizeReadTimestamp(t, fields[fieldReadTimestamp]),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := strconv.Atoi(out[i].ID)
		right, _ := strconv.Atoi(out[j].ID)
		return left < right
	})
	return out
}

func normalizeReadTimestamp(t *testing.T, value string) string {
	t.Helper()
	if value == "" {
		return ""
	}
	_, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return "<set>"
}

func assertErrorParity(t *testing.T, errA, errB error) {
	t.Helper()
	require.Equal(t, errA != nil, errB != nil)
}

func assertListParity(t *testing.T, tsv parityBackend, sqlite parityBackend, stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) []normalizedNotification {
	t.Helper()

	listTSV, errTSV := tsv.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
	listSQLite, errSQLite := sqlite.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)

	assertErrorParity(t, errTSV, errSQLite)
	require.NoError(t, errTSV)
	require.NoError(t, errSQLite)

	parsedTSV := parseTSVLines(t, listTSV)
	parsedSQLite := parseTSVLines(t, listSQLite)
	require.Equal(t, parsedTSV, parsedSQLite)

	return parsedTSV
}

func assertFullStateParity(t *testing.T, tsv parityBackend, sqlite parityBackend) []normalizedNotification {
	t.Helper()
	all := assertListParity(t, tsv, sqlite, "all", "", "", "", "", "", "")
	require.Equal(t, tsv.GetActiveCount(), sqlite.GetActiveCount())
	return all
}

func addParity(t *testing.T, tsv parityBackend, sqlite parityBackend, message, timestamp, session, window, pane, paneCreated, level string) string {
	t.Helper()

	idTSV, errTSV := tsv.AddNotification(message, timestamp, session, window, pane, paneCreated, level)
	idSQLite, errSQLite := sqlite.AddNotification(message, timestamp, session, window, pane, paneCreated, level)

	assertErrorParity(t, errTSV, errSQLite)
	require.NoError(t, errTSV)
	require.NoError(t, errSQLite)
	require.Equal(t, idTSV, idSQLite)

	return idTSV
}

func TestStorageBackendParityIntegration(t *testing.T) {
	mockClient := new(tmux.MockClient)
	mockClient.On("HasSession").Return(true, nil)
	mockClient.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	SetTmuxClient(mockClient)

	scenarios := []struct {
		name string
		run  func(t *testing.T, tsv parityBackend, sqlite parityBackend)
	}{
		{
			name: "crud_parity",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				id := addParity(t, tsv, sqlite, "crud-message", "2025-01-01T00:00:00Z", "s1", "w1", "p1", "101", "info")

				gotTSV, errTSV := tsv.GetNotificationByID(id)
				gotSQLite, errSQLite := sqlite.GetNotificationByID(id)
				assertErrorParity(t, errTSV, errSQLite)
				require.NoError(t, errTSV)
				require.Equal(t, parseTSVLines(t, gotTSV), parseTSVLines(t, gotSQLite))

				errTSV = tsv.DismissNotification(id)
				errSQLite = sqlite.DismissNotification(id)
				assertErrorParity(t, errTSV, errSQLite)
				require.NoError(t, errTSV)

				assertFullStateParity(t, tsv, sqlite)
			},
		},
		{
			name: "filter_parity",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				addParity(t, tsv, sqlite, "m1", "2025-01-01T10:00:00Z", "alpha", "w1", "p1", "11", "info")
				addParity(t, tsv, sqlite, "m2", "2025-01-02T10:00:00Z", "beta", "w2", "p2", "12", "warning")
				id3 := addParity(t, tsv, sqlite, "m3", "2025-01-03T10:00:00Z", "alpha", "w2", "p3", "13", "error")

				require.NoError(t, tsv.DismissNotification(id3))
				require.NoError(t, sqlite.DismissNotification(id3))

				assertListParity(t, tsv, sqlite, "active", "", "", "", "", "", "")
				assertListParity(t, tsv, sqlite, "dismissed", "", "", "", "", "", "")
				assertListParity(t, tsv, sqlite, "all", "warning", "", "", "", "", "")
				assertListParity(t, tsv, sqlite, "all", "", "alpha", "", "", "", "")
				assertListParity(t, tsv, sqlite, "all", "", "", "w2", "", "", "")
				assertListParity(t, tsv, sqlite, "all", "", "", "", "p1", "", "")
				assertListParity(t, tsv, sqlite, "all", "", "", "", "", "2025-01-03T00:00:00Z", "")
				assertListParity(t, tsv, sqlite, "all", "", "", "", "", "", "2025-01-01T23:59:59Z")
			},
		},
		{
			name: "edge_case_empty_dataset",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				records := assertFullStateParity(t, tsv, sqlite)
				require.Len(t, records, 0)
			},
		},
		{
			name: "edge_case_single_record",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				addParity(t, tsv, sqlite, "single", "2025-01-10T10:10:10Z", "only", "w", "p", "5", "critical")
				records := assertFullStateParity(t, tsv, sqlite)
				require.Len(t, records, 1)
			},
		},
		{
			name: "edge_case_large_dataset_1200",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				fixture := make([]struct {
					message, timestamp, session, window, pane, paneCreated, level string
				}, 0, 1200)

				base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				levels := []string{"info", "warning", "error", "critical"}

				for i := 0; i < 1200; i++ {
					fixture = append(fixture, struct {
						message, timestamp, session, window, pane, paneCreated, level string
					}{
						message:     fmt.Sprintf("bulk-%04d", i),
						timestamp:   base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
						session:     fmt.Sprintf("s-%d", i%9),
						window:      fmt.Sprintf("w-%d", i%7),
						pane:        fmt.Sprintf("p-%d", i%5),
						paneCreated: fmt.Sprintf("%d", 1000+i),
						level:       levels[i%len(levels)],
					})
				}

				for _, row := range fixture {
					addParity(t, tsv, sqlite, row.message, row.timestamp, row.session, row.window, row.pane, row.paneCreated, row.level)
				}

				all := assertFullStateParity(t, tsv, sqlite)
				require.Len(t, all, 1200)

				warnings := assertListParity(t, tsv, sqlite, "all", "warning", "", "", "", "", "")
				require.Len(t, warnings, 300)
			},
		},
		{
			name: "concurrent_access_behavior",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				const workers = 64
				ids := make([]string, 0, workers)
				for i := 0; i < workers; i++ {
					ids = append(ids, addParity(
						t,
						tsv,
						sqlite,
						fmt.Sprintf("concurrent-%03d", i),
						"2025-01-01T00:00:00Z",
						"session-concurrent",
						"window-concurrent",
						fmt.Sprintf("pane-%03d", i),
						strconv.Itoa(i),
						"info",
					))
				}

				runConcurrentReadMark := func(t *testing.T, backend parityBackend, ids []string) []normalizedNotification {
					t.Helper()
					var wg sync.WaitGroup
					errCh := make(chan error, len(ids))
					wg.Add(len(ids))

					for _, id := range ids {
						go func(id string) {
							defer wg.Done()
							err := backend.MarkNotificationRead(id)
							errCh <- err
						}(id)
					}
					wg.Wait()
					close(errCh)

					for err := range errCh {
						require.NoError(t, err)
					}

					list, err := backend.ListNotifications("all", "", "", "", "", "", "")
					require.NoError(t, err)
					out := parseTSVLines(t, list)
					require.Len(t, out, workers)
					return out
				}

				tsvRecords := runConcurrentReadMark(t, tsv, ids)
				sqliteRecords := runConcurrentReadMark(t, sqlite, ids)

				require.Len(t, tsvRecords, len(sqliteRecords))
				require.Equal(t, tsv.GetActiveCount(), sqlite.GetActiveCount())

				countReadMarks := func(records []normalizedNotification) int {
					total := 0
					for _, rec := range records {
						if rec.ReadTimestamp == "<set>" {
							total++
						}
					}
					return total
				}

				require.Equal(t, workers, countReadMarks(tsvRecords))
				require.Equal(t, workers, countReadMarks(sqliteRecords))
			},
		},
		{
			name: "error_handling_consistency",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				_, errTSV := tsv.AddNotification("", "", "", "", "", "", "info")
				_, errSQLite := sqlite.AddNotification("", "", "", "", "", "", "info")
				assertErrorParity(t, errTSV, errSQLite)

				_, errTSV = tsv.AddNotification("ok", "bad-ts", "", "", "", "", "info")
				_, errSQLite = sqlite.AddNotification("ok", "bad-ts", "", "", "", "", "info")
				assertErrorParity(t, errTSV, errSQLite)

				_, errTSV = tsv.ListNotifications("bad-state", "", "", "", "", "", "")
				_, errSQLite = sqlite.ListNotifications("bad-state", "", "", "", "", "", "")
				assertErrorParity(t, errTSV, errSQLite)

				_, errTSV = tsv.GetNotificationByID("9999")
				_, errSQLite = sqlite.GetNotificationByID("9999")
				assertErrorParity(t, errTSV, errSQLite)
			},
		},
		{
			name: "failure_consistency_no_partial_mutation",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				id := addParity(t, tsv, sqlite, "rollback", "2025-02-01T00:00:00Z", "s", "w", "p", "1", "info")

				errTSV := tsv.DismissNotification("404")
				errSQLite := sqlite.DismissNotification("404")
				assertErrorParity(t, errTSV, errSQLite)

				errTSV = tsv.MarkNotificationRead("404")
				errSQLite = sqlite.MarkNotificationRead("404")
				assertErrorParity(t, errTSV, errSQLite)

				lineTSV, errTSV := tsv.GetNotificationByID(id)
				lineSQLite, errSQLite := sqlite.GetNotificationByID(id)
				assertErrorParity(t, errTSV, errSQLite)
				require.Equal(t, parseTSVLines(t, lineTSV), parseTSVLines(t, lineSQLite))
				require.Equal(t, "active", parseTSVLines(t, lineTSV)[0].State)
			},
		},
		{
			name: "special_characters_quotes_tabs_newlines",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				original := "quote=\"x\" tab=\t newline=\n slash=\\"
				id := addParity(t, tsv, sqlite, original, "2025-03-01T12:00:00Z", "s", "w", "p", "20", "warning")

				lineTSV, errTSV := tsv.GetNotificationByID(id)
				lineSQLite, errSQLite := sqlite.GetNotificationByID(id)
				assertErrorParity(t, errTSV, errSQLite)

				parsedTSV := parseTSVLines(t, lineTSV)
				parsedSQLite := parseTSVLines(t, lineSQLite)
				require.Equal(t, parsedTSV, parsedSQLite)
				require.Equal(t, original, unescapeMessage(parsedTSV[0].Message))
			},
		},
		{
			name: "unicode_non_ascii",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				id := addParity(t, tsv, sqlite, "olá café こんにちは مرحبا 🚀", "2025-03-02T12:00:00Z", "sessão", "窗", "панель", "30", "error")

				lineTSV, errTSV := tsv.GetNotificationByID(id)
				lineSQLite, errSQLite := sqlite.GetNotificationByID(id)
				assertErrorParity(t, errTSV, errSQLite)

				parsedTSV := parseTSVLines(t, lineTSV)
				parsedSQLite := parseTSVLines(t, lineSQLite)
				require.Equal(t, parsedTSV, parsedSQLite)
				require.Equal(t, "olá café こんにちは مرحبا 🚀", unescapeMessage(parsedTSV[0].Message))
			},
		},
		{
			name: "timestamp_edges_and_timezones",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				addParity(t, tsv, sqlite, "epoch", "1970-01-01T00:00:00Z", "s", "w", "p", "1", "info")
				addParity(t, tsv, sqlite, "leap-ish", "2024-02-29T23:59:59Z", "s", "w", "p", "2", "warning")
				addParity(t, tsv, sqlite, "offset", "2025-01-01T10:00:00+02:00", "s", "w", "p", "3", "error")
				addParity(t, tsv, sqlite, "fractional", "2025-01-01T10:00:00.123Z", "s", "w", "p", "4", "critical")

				assertFullStateParity(t, tsv, sqlite)
				assertListParity(t, tsv, sqlite, "all", "", "", "", "", "2025-01-01T00:00:00Z", "")
				assertListParity(t, tsv, sqlite, "all", "", "", "", "", "", "1970-01-01T00:00:00Z")
			},
		},
		{
			name: "read_unread_transitions",
			run: func(t *testing.T, tsv parityBackend, sqlite parityBackend) {
				id := addParity(t, tsv, sqlite, "read me", "2025-04-01T00:00:00Z", "s", "w", "p", "40", "info")

				errTSV := tsv.MarkNotificationRead(id)
				errSQLite := sqlite.MarkNotificationRead(id)
				assertErrorParity(t, errTSV, errSQLite)
				require.NoError(t, errTSV)

				lineTSV, errTSV := tsv.GetNotificationByID(id)
				lineSQLite, errSQLite := sqlite.GetNotificationByID(id)
				assertErrorParity(t, errTSV, errSQLite)
				parsedTSV := parseTSVLines(t, lineTSV)
				parsedSQLite := parseTSVLines(t, lineSQLite)
				require.Equal(t, parsedTSV, parsedSQLite)
				require.Equal(t, "<set>", parsedTSV[0].ReadTimestamp)

				errTSV = tsv.MarkNotificationUnread(id)
				errSQLite = sqlite.MarkNotificationUnread(id)
				assertErrorParity(t, errTSV, errSQLite)
				require.NoError(t, errTSV)

				assertFullStateParity(t, tsv, sqlite)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tsv := newTSVParityBackend(t)
			sqlite := newSQLiteParityBackend(t)

			t.Cleanup(func() {
				require.NoError(t, sqlite.Close())
				require.NoError(t, tsv.Close())
			})

			scenario.run(t, tsv, sqlite)
		})
	}
}

func TestSQLiteStorageOperationParityAgainstTSV(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	tsvStore, tsvMock := newTSVBackendForIntegration(t)
	sqliteStore, sqliteMock := newSQLiteBackendForIntegration(t)

	idsTSV := addFixtureNotifications(t, tsvStore)
	idsSQLite := addFixtureNotifications(t, sqliteStore)
	require.Equal(t, idsTSV, idsSQLite)

	listTSV, err := tsvStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	listSQLite, err := sqliteStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(listTSV), normalizeTSVOutput(listSQLite))

	getTSV, err := tsvStore.GetNotificationByID(idsTSV[0])
	require.NoError(t, err)
	getSQLite, err := sqliteStore.GetNotificationByID(idsSQLite[0])
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(getTSV), normalizeTSVOutput(getSQLite))

	require.NoError(t, tsvStore.MarkNotificationRead(idsTSV[1]))
	require.NoError(t, sqliteStore.MarkNotificationRead(idsSQLite[1]))
	readTSV, err := tsvStore.GetNotificationByID(idsTSV[1])
	require.NoError(t, err)
	readSQLite, err := sqliteStore.GetNotificationByID(idsSQLite[1])
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(readTSV), normalizeTSVOutput(readSQLite))

	require.NoError(t, tsvStore.MarkNotificationUnread(idsTSV[1]))
	require.NoError(t, sqliteStore.MarkNotificationUnread(idsSQLite[1]))
	unreadTSV, err := tsvStore.GetNotificationByID(idsTSV[1])
	require.NoError(t, err)
	unreadSQLite, err := sqliteStore.GetNotificationByID(idsSQLite[1])
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(unreadTSV), normalizeTSVOutput(unreadSQLite))

	require.NoError(t, tsvStore.DismissNotification(idsTSV[0]))
	require.NoError(t, sqliteStore.DismissNotification(idsSQLite[0]))
	require.Equal(t, tsvStore.GetActiveCount(), sqliteStore.GetActiveCount())

	require.NoError(t, tsvStore.DismissAll())
	require.NoError(t, sqliteStore.DismissAll())
	require.Equal(t, 0, tsvStore.GetActiveCount())
	require.Equal(t, 0, sqliteStore.GetActiveCount())

	require.NoError(t, tsvStore.CleanupOldNotifications(1, true))
	require.NoError(t, sqliteStore.CleanupOldNotifications(1, true))

	preCleanupTSV, err := tsvStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	preCleanupSQLite, err := sqliteStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(preCleanupTSV), normalizeTSVOutput(preCleanupSQLite))

	require.NoError(t, tsvStore.CleanupOldNotifications(1, false))
	require.NoError(t, sqliteStore.CleanupOldNotifications(1, false))

	postCleanupTSV, err := tsvStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	postCleanupSQLite, err := sqliteStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(postCleanupTSV), normalizeTSVOutput(postCleanupSQLite))
	require.Contains(t, postCleanupSQLite, "2099-01-01T00:00:00Z")
	require.NotContains(t, postCleanupSQLite, "2000-01-01T00:00:00Z")

	tsvMock.AssertNumberOfCalls(t, "SetStatusOption", 5)
	sqliteMock.AssertNumberOfCalls(t, "SetStatusOption", 5)
}

func TestSQLiteStorageMigrationIntegration(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	baseDir := t.TempDir()
	tsvPath := filepath.Join(baseDir, "notifications.tsv")
	sqlitePath := filepath.Join(baseDir, "notifications.db")

	data := strings.Join([]string{
		"1\t2026-01-01T00:00:00Z\tactive\tsession-a\twindow-a\tpane-a\tfirst\\nline\t\tinfo\t",
		"1\t2026-01-02T00:00:00Z\tdismissed\tsession-a\twindow-a\tpane-a\tfirst updated\t\terror\t2026-01-02T00:00:01Z",
		"2\t2026-01-03T00:00:00Z\tactive\tsession-b\twindow-b\tpane-b\tsecond\t\twarning\t",
		"invalid\t2026-01-03T00:00:00Z\tactive\tsession\twindow\tpane\tbad row\t\tinfo\t",
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(data), 0o644))

	statsDryRun, err := sqlitebackend.MigrateTSVToSQLite(sqlitebackend.MigrationOptions{
		TSVPath:    tsvPath,
		SQLitePath: sqlitePath,
		DryRun:     true,
	})
	require.NoError(t, err)
	require.Equal(t, 4, statsDryRun.TotalRows)
	require.Equal(t, 1, statsDryRun.SkippedRows)
	require.Equal(t, 1, statsDryRun.DuplicateRows)
	require.Equal(t, 2, statsDryRun.MigratedRows)

	stats, err := sqlitebackend.MigrateTSVToSQLite(sqlitebackend.MigrationOptions{
		TSVPath:    tsvPath,
		SQLitePath: sqlitePath,
	})
	require.NoError(t, err)
	require.True(t, stats.BackupCreated)
	require.FileExists(t, stats.BackupPath)

	store, err := sqlitebackend.NewSQLiteStorage(sqlitePath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	lineOne, err := store.GetNotificationByID("1")
	require.NoError(t, err)
	require.Contains(t, lineOne, "\tdismissed\t")
	require.Contains(t, lineOne, "first updated")
	require.Contains(t, lineOne, "\terror\t")

	lineTwo, err := store.GetNotificationByID("2")
	require.NoError(t, err)
	require.Contains(t, lineTwo, "\tactive\t")
	require.Contains(t, lineTwo, "\twarning\t")
	count := store.GetActiveCount()
	require.Equal(t, 1, count)
}

func TestSQLiteStorageIntegrationHooksAndTmux(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "abort")
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")

	hooksDir := filepath.Join(t.TempDir(), "hooks")
	hookLog := filepath.Join(t.TempDir(), "hooks.log")
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)
	t.Setenv("HOOK_LOG", hookLog)

	hookBody := "#!/bin/sh\necho \"$HOOK_POINT:$NOTIFICATION_ID:$DELETED_COUNT\" >> \"$HOOK_LOG\"\n"
	writeIntegrationHook(t, hooksDir, "pre-add", "pre-add.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "post-add", "post-add.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "pre-dismiss", "pre-dismiss.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "post-dismiss", "post-dismiss.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "pre-clear", "pre-clear.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "cleanup", "cleanup.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "post-cleanup", "post-cleanup.sh", hookBody)

	store, tmuxMock := newSQLiteBackendForIntegration(t)

	id1, err := store.AddNotification("old", "2000-01-01T00:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)
	_, err = store.AddNotification("new", "2099-01-01T00:00:00Z", "", "", "", "", "warning")
	require.NoError(t, err)
	require.NoError(t, store.DismissNotification(id1))
	require.NoError(t, store.DismissAll())
	require.NoError(t, store.CleanupOldNotifications(1, false))

	content, err := os.ReadFile(hookLog)
	require.NoError(t, err)
	logOutput := string(content)
	require.Contains(t, logOutput, "pre-add:1:")
	require.Contains(t, logOutput, "post-add:1:")
	require.Contains(t, logOutput, "pre-dismiss:1:")
	require.Contains(t, logOutput, "post-dismiss:1:")
	require.Contains(t, logOutput, "pre-clear::")
	require.Contains(t, logOutput, "cleanup::")
	require.Contains(t, logOutput, "post-cleanup::1")

	tmuxMock.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "1")
	tmuxMock.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "2")
	tmuxMock.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "1")
	tmuxMock.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "0")
}

func TestSQLiteStorageLargeDatasetIntegration(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	store, _ := newSQLiteBackendForIntegration(t)

	for i := 0; i < 1000; i++ {
		_, err := store.AddNotification(
			fmt.Sprintf("message-%d", i),
			fmt.Sprintf("2026-01-01T00:00:%02dZ", i%60),
			"load-session",
			"load-window",
			"load-pane",
			"",
			"info",
		)
		require.NoError(t, err)
	}

	require.Equal(t, 1000, store.GetActiveCount())

	list, err := store.ListNotifications("active", "info", "load-session", "", "", "", "")
	require.NoError(t, err)
	require.Len(t, strings.Split(strings.TrimSpace(list), "\n"), 1000)
}

func addFixtureNotifications(t *testing.T, s Storage) []string {
	t.Helper()

	fixtures := []struct {
		message     string
		timestamp   string
		session     string
		window      string
		pane        string
		paneCreated string
		level       string
	}{
		{"first line\nwith tab\tvalue", "2000-01-01T00:00:00Z", "session-a", "window-1", "pane-1", "", "info"},
		{"second", "2026-01-03T04:05:06Z", "session-b", "window-2", "pane-2", "", "warning"},
		{"third", "2099-01-01T00:00:00Z", "session-c", "window-3", "pane-3", "", "error"},
	}

	ids := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		id, err := s.AddNotification(
			fixture.message,
			fixture.timestamp,
			fixture.session,
			fixture.window,
			fixture.pane,
			fixture.paneCreated,
			fixture.level,
		)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	return ids
}

func newTSVBackendForIntegration(t *testing.T) (Storage, *tmux.MockClient) {
	t.Helper()

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	Reset()
	t.Cleanup(Reset)

	tmuxMock := new(tmux.MockClient)
	tmuxMock.On("HasSession").Return(true, nil)
	tmuxMock.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	SetTmuxClient(tmuxMock)

	store, err := NewFileStorage()
	require.NoError(t, err)
	return store, tmuxMock
}

func newSQLiteBackendForIntegration(t *testing.T) (Storage, *tmux.MockClient) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "notifications.db")
	store, err := sqlitebackend.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	tmuxMock := new(tmux.MockClient)
	tmuxMock.On("HasSession").Return(true, nil)
	tmuxMock.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	sqlitebackend.SetTmuxClient(tmuxMock)
	t.Cleanup(func() {
		sqlitebackend.SetTmuxClient(tmux.NewDefaultClient())
	})

	return store, tmuxMock
}

func normalizeTSVOutput(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(content), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		for len(parts) < 10 {
			parts = append(parts, "")
		}
		if parts[9] != "" {
			parts[9] = "<read_timestamp>"
		}
		normalized = append(normalized, strings.Join(parts, "\t"))
	}
	return strings.Join(normalized, "\n")
}

func writeIntegrationHook(t *testing.T, hooksDir, hookPoint, name, body string) {
	t.Helper()
	hookPointDir := filepath.Join(hooksDir, hookPoint)
	require.NoError(t, os.MkdirAll(hookPointDir, 0o755))
	hookPath := filepath.Join(hookPointDir, name)
	require.NoError(t, os.WriteFile(hookPath, []byte(body), 0o755))
}
