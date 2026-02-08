package sqlite

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite/sqlcgen"
)

const (
	migrationFieldID = iota
	migrationFieldTimestamp
	migrationFieldState
	migrationFieldSession
	migrationFieldWindow
	migrationFieldPane
	migrationFieldMessage
	migrationFieldPaneCreated
	migrationFieldLevel
	migrationFieldReadTimestamp
	migrationNumFields
	migrationMinFields = migrationNumFields - 1
)

// MigrationOptions configures TSV to SQLite migration behavior.
type MigrationOptions struct {
	TSVPath    string
	SQLitePath string
	BackupPath string
	DryRun     bool
}

// MigrationStats summarizes a migration run.
type MigrationStats struct {
	TotalRows     int
	MigratedRows  int
	SkippedRows   int
	FailedRows    int
	DuplicateRows int
	BackupCreated bool
	BackupPath    string
	Warnings      []string
}

type migrationRow struct {
	id            int64
	timestamp     string
	state         string
	session       string
	window        string
	pane          string
	message       string
	paneCreated   string
	level         string
	readTimestamp string
	updatedAt     string
}

// MigrateTSVToSQLite validates TSV rows and imports the latest valid row per notification ID.
//
// Safety behavior:
//   - In non-dry-run mode, a TSV backup is created before any SQLite writes.
//   - Inserts happen inside a single transaction.
//   - Malformed rows are skipped with warnings instead of aborting the full migration.
//   - Import is idempotent by using UPSERT on notification ID.
func MigrateTSVToSQLite(opts MigrationOptions) (MigrationStats, error) {
	stats := MigrationStats{}

	if strings.TrimSpace(opts.TSVPath) == "" {
		return stats, fmt.Errorf("migration: tsv path cannot be empty")
	}
	if strings.TrimSpace(opts.SQLitePath) == "" {
		return stats, fmt.Errorf("migration: sqlite path cannot be empty")
	}

	latestRows, stats, err := parseLatestRows(opts.TSVPath)
	if err != nil {
		return stats, err
	}

	if opts.DryRun {
		stats.MigratedRows = len(latestRows)
		return stats, nil
	}

	backupPath := strings.TrimSpace(opts.BackupPath)
	if backupPath == "" {
		backupPath = defaultBackupPath(opts.TSVPath)
	}
	if err := createMigrationBackup(opts.TSVPath, backupPath); err != nil {
		return stats, err
	}
	stats.BackupCreated = true
	stats.BackupPath = backupPath

	s, err := NewSQLiteStorage(opts.SQLitePath)
	if err != nil {
		return stats, fmt.Errorf("migration: open sqlite storage: %w", err)
	}
	defer s.Close()

	if err := upsertRows(s, latestRows); err != nil {
		stats.FailedRows = len(latestRows)
		return stats, err
	}

	stats.MigratedRows = len(latestRows)
	return stats, nil
}

// RollbackTSVMigration restores TSV from backup and removes the SQLite database file.
func RollbackTSVMigration(tsvPath, sqlitePath, backupPath string) error {
	if strings.TrimSpace(tsvPath) == "" {
		return fmt.Errorf("rollback: tsv path cannot be empty")
	}
	if strings.TrimSpace(sqlitePath) == "" {
		return fmt.Errorf("rollback: sqlite path cannot be empty")
	}
	if strings.TrimSpace(backupPath) == "" {
		backupPath = defaultBackupPath(tsvPath)
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("rollback: read backup: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(tsvPath), 0o755); err != nil {
		return fmt.Errorf("rollback: ensure tsv directory: %w", err)
	}
	if err := os.WriteFile(tsvPath, backupData, 0o644); err != nil {
		return fmt.Errorf("rollback: restore tsv: %w", err)
	}

	if err := os.Remove(sqlitePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rollback: remove sqlite db: %w", err)
	}

	return nil
}

func parseLatestRows(tsvPath string) (map[int64]migrationRow, MigrationStats, error) {
	stats := MigrationStats{}

	f, err := os.Open(tsvPath)
	if err != nil {
		return nil, stats, fmt.Errorf("migration: open tsv: %w", err)
	}
	defer f.Close()

	latestByID := make(map[int64]migrationRow)
	scanner := bufio.NewScanner(f)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		stats.TotalRows++
		row, warning, err := parseTSVLine(line)
		if err != nil {
			stats.SkippedRows++
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("line %d: %s", lineNumber, warning))
			continue
		}

		if _, exists := latestByID[row.id]; exists {
			stats.DuplicateRows++
		}
		latestByID[row.id] = row
	}

	if err := scanner.Err(); err != nil {
		return nil, stats, fmt.Errorf("migration: read tsv: %w", err)
	}

	return latestByID, stats, nil
}

func parseTSVLine(line string) (migrationRow, string, error) {
	fields := strings.Split(line, "\t")
	if len(fields) < migrationMinFields {
		return migrationRow{}, fmt.Sprintf("invalid field count: got %d, need at least %d", len(fields), migrationMinFields), fmt.Errorf("invalid field count")
	}
	if len(fields) > migrationNumFields {
		return migrationRow{}, fmt.Sprintf("invalid field count: got %d, max %d", len(fields), migrationNumFields), fmt.Errorf("invalid field count")
	}
	for len(fields) < migrationNumFields {
		fields = append(fields, "")
	}

	id, err := strconv.ParseInt(fields[migrationFieldID], 10, 64)
	if err != nil || id <= 0 {
		return migrationRow{}, "invalid notification id", fmt.Errorf("invalid id")
	}

	timestamp := fields[migrationFieldTimestamp]
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		return migrationRow{}, "invalid timestamp format", fmt.Errorf("invalid timestamp")
	}

	state := fields[migrationFieldState]
	if !validStates[state] || state == "all" {
		return migrationRow{}, fmt.Sprintf("invalid state '%s'", state), fmt.Errorf("invalid state")
	}

	level := fields[migrationFieldLevel]
	if !validLevels[level] {
		return migrationRow{}, fmt.Sprintf("invalid level '%s'", level), fmt.Errorf("invalid level")
	}

	paneCreated := fields[migrationFieldPaneCreated]
	if paneCreated != "" {
		if _, err := time.Parse(time.RFC3339, paneCreated); err != nil {
			return migrationRow{}, "invalid pane_created timestamp format", fmt.Errorf("invalid pane_created")
		}
	}

	readTimestamp := fields[migrationFieldReadTimestamp]
	if readTimestamp != "" {
		if _, err := time.Parse(time.RFC3339, readTimestamp); err != nil {
			return migrationRow{}, "invalid read_timestamp format", fmt.Errorf("invalid read_timestamp")
		}
	}

	return migrationRow{
		id:            id,
		timestamp:     timestamp,
		state:         state,
		session:       fields[migrationFieldSession],
		window:        fields[migrationFieldWindow],
		pane:          fields[migrationFieldPane],
		message:       unescapeTSVMessage(fields[migrationFieldMessage]),
		paneCreated:   paneCreated,
		level:         level,
		readTimestamp: readTimestamp,
		updatedAt:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}, "", nil
}

func upsertRows(s *SQLiteStorage, rowsByID map[int64]migrationRow) error {
	ctx := context.Background()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("migration: begin transaction: %w", err)
	}
	qtx := s.queries.WithTx(tx)

	ids := make([]int64, 0, len(rowsByID))
	for id := range rowsByID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		row := rowsByID[id]
		if err := qtx.UpsertNotification(ctx, sqlcgen.UpsertNotificationParams{
			ID:            row.id,
			Timestamp:     row.timestamp,
			State:         row.state,
			Session:       row.session,
			Window:        row.window,
			Pane:          row.pane,
			Message:       row.message,
			PaneCreated:   row.paneCreated,
			Level:         row.level,
			ReadTimestamp: row.readTimestamp,
			UpdatedAt:     row.updatedAt,
		}); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration: upsert id %d: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration: commit transaction: %w", err)
	}

	return nil
}

func createMigrationBackup(tsvPath, backupPath string) error {
	if _, err := os.Stat(backupPath); err == nil {
		return fmt.Errorf("migration: backup already exists at %s", backupPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("migration: stat backup path: %w", err)
	}

	data, err := os.ReadFile(tsvPath)
	if err != nil {
		return fmt.Errorf("migration: read tsv for backup: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return fmt.Errorf("migration: create backup directory: %w", err)
	}
	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return fmt.Errorf("migration: write backup: %w", err)
	}

	return nil
}

func defaultBackupPath(tsvPath string) string {
	return tsvPath + ".sqlite-migration.bak"
}

func unescapeTSVMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	msg = strings.ReplaceAll(msg, "\\t", "\t")
	msg = strings.ReplaceAll(msg, "\\\\", "\\")
	return msg
}
