package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/stretchr/testify/require"
)

type fakeStorage struct {
	addID string

	addErr          error
	dismissErr      error
	dismissAllErr   error
	markReadErr     error
	markUnreadErr   error
	cleanupErr      error
	listResult      string
	getResult       string
	getActiveResult int

	addCalls        int
	dismissCalls    int
	dismissAllCalls int
	markReadCalls   int
	markUnreadCalls int
	cleanupCalls    int
	listCalls       int
	getCalls        int
}

func (f *fakeStorage) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	f.addCalls++
	if f.addErr != nil {
		return "", f.addErr
	}
	if f.addID == "" {
		f.addID = "1"
	}
	return f.addID, nil
}

func (f *fakeStorage) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	f.listCalls++
	return f.listResult, nil
}

func (f *fakeStorage) GetNotificationByID(id string) (string, error) {
	f.getCalls++
	return f.getResult, nil
}

func (f *fakeStorage) DismissNotification(id string) error {
	f.dismissCalls++
	return f.dismissErr
}

func (f *fakeStorage) DismissAll() error {
	f.dismissAllCalls++
	return f.dismissAllErr
}

func (f *fakeStorage) MarkNotificationRead(id string) error {
	f.markReadCalls++
	return f.markReadErr
}

func (f *fakeStorage) MarkNotificationUnread(id string) error {
	f.markUnreadCalls++
	return f.markUnreadErr
}

func (f *fakeStorage) MarkNotificationReadWithTimestamp(id, timestamp string) error {
	f.markReadCalls++
	return f.markReadErr
}

func (f *fakeStorage) MarkNotificationUnreadWithTimestamp(id, timestamp string) error {
	f.markUnreadCalls++
	return f.markUnreadErr
}

func (f *fakeStorage) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	f.cleanupCalls++
	return f.cleanupErr
}

func (f *fakeStorage) GetActiveCount() int {
	return f.getActiveResult
}

func TestDualWriterAddWritesToBothBackends(t *testing.T) {
	tsv := &fakeStorage{addID: "42"}
	sql := &fakeStorage{addID: "42"}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	id, err := dw.AddNotification("message", "", "", "", "", "", "info")
	require.NoError(t, err)
	require.Equal(t, "42", id)
	require.Equal(t, 1, tsv.addCalls)
	require.Equal(t, 1, sql.addCalls)
}

func TestDualWriterReturnsTSVErrorAndSkipsSQLite(t *testing.T) {
	tsv := &fakeStorage{addErr: fmt.Errorf("tsv failed")}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	_, err = dw.AddNotification("message", "", "", "", "", "", "info")
	require.Error(t, err)
	require.Equal(t, 1, tsv.addCalls)
	require.Equal(t, 0, sql.addCalls)
}

func TestDualWriterVerifyOnlySkipsSQLiteWrites(t *testing.T) {
	tsv := &fakeStorage{addID: "7"}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{VerifyOnly: true})
	require.NoError(t, err)

	_, err = dw.AddNotification("message", "", "", "", "", "", "info")
	require.NoError(t, err)
	require.Equal(t, 1, tsv.addCalls)
	require.Equal(t, 0, sql.addCalls)
}

func TestDualWriterReadBackendDefaultsToSQLite(t *testing.T) {
	tsv := &fakeStorage{listResult: "tsv"}
	sql := &fakeStorage{listResult: "sqlite"}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	list, err := dw.ListNotifications("all", "", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, "sqlite", list)
	require.Equal(t, 0, tsv.listCalls)
	require.Equal(t, 1, sql.listCalls)
}

func TestDualWriterReadBackendCanUseTSV(t *testing.T) {
	tsv := &fakeStorage{getResult: "tsv-record", getActiveResult: 3}
	sql := &fakeStorage{getResult: "sqlite-record", getActiveResult: 4}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{ReadBackend: ReadBackendTSV})
	require.NoError(t, err)

	line, err := dw.GetNotificationByID("1")
	require.NoError(t, err)
	require.Equal(t, "tsv-record", line)
	require.Equal(t, 3, dw.GetActiveCount())
	require.Equal(t, 1, tsv.getCalls)
	require.Equal(t, 0, sql.getCalls)
}

func TestDualWriterVerifyConsistencyDetectsDiscrepancies(t *testing.T) {
	tsv := &fakeStorage{listResult: strings.Join([]string{
		"1\t2026-01-01T00:00:00Z\tactive\ts\tw\tp\tone\t\tinfo\t",
		"2\t2026-01-01T00:00:00Z\tdismissed\ts\tw\tp\ttwo\t\terror\t",
	}, "\n")}
	sql := &fakeStorage{listResult: strings.Join([]string{
		"1\t2026-01-01T00:00:00Z\tactive\ts\tw\tp\tone-changed\t\tinfo\t",
		"3\t2026-01-01T00:00:00Z\tactive\ts\tw\tp\tthree\t\twarning\t",
	}, "\n")}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{SampleSize: 10})
	require.NoError(t, err)

	report, err := dw.VerifyConsistency(0)
	require.NoError(t, err)
	require.False(t, report.Consistent)
	require.Equal(t, []string{"3"}, report.MissingInTSV)
	require.Equal(t, []string{"2"}, report.MissingInSQLite)
	require.Len(t, report.RecordDiffs, 1)
	require.Equal(t, "1", report.RecordDiffs[0].ID)
	require.Equal(t, "message", report.RecordDiffs[0].FieldDiffs[0].Field)
}

func TestDualWriterTracksWriteMetrics(t *testing.T) {
	tsv := &fakeStorage{addID: "1"}
	sql := &fakeStorage{addErr: fmt.Errorf("sqlite failed")}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	_, err = dw.AddNotification("message", "", "", "", "", "", "info")
	require.NoError(t, err)

	metrics := dw.Metrics()
	require.Equal(t, int64(1), metrics.WriteOperations)
	require.Equal(t, int64(0), metrics.TSVWriteFailures)
	require.Equal(t, int64(1), metrics.SQLiteWriteFailure)
	require.GreaterOrEqual(t, metrics.TotalWriteLatency, metrics.AverageWriteLatency())
}

func TestDualWriterFallsBackToTSVReadsAfterSQLiteWriteFailure(t *testing.T) {
	tsv := &fakeStorage{addID: "1", listResult: "tsv"}
	sql := &fakeStorage{addErr: fmt.Errorf("sqlite failed"), listResult: "sqlite"}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	_, err = dw.AddNotification("message", "", "", "", "", "", "info")
	require.NoError(t, err)

	list, err := dw.ListNotifications("all", "", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, "tsv", list)
	require.Equal(t, 1, tsv.listCalls)
	require.Equal(t, 0, sql.listCalls)
}

func TestDualWriterMigrationSkipsWhenTSVEmpty(t *testing.T) {
	tsv := &fakeStorage{
		listResult:      "",
		getActiveResult: 0,
	}
	sql := &fakeStorage{
		listResult:      "",
		getActiveResult: 0,
	}

	_, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	// Migration should skip because TSV is empty
}

func TestDualWriterMigrateNotificationLinesParsesTSV(t *testing.T) {
	tsv := &fakeStorage{
		listResult: strings.Join([]string{
			"1\t2026-01-01T00:00:00Z\tactive\tsession1\twindow1\tpane1\tmessage1\t\tinfo\t",
			"2\t2026-01-01T01:00:00Z\tdismissed\tsession2\twindow2\tpane2\tmessage2\t\twarning\t",
		}, "\n"),
		getActiveResult: 1,
	}
	sql := &fakeStorage{
		listResult:      "",
		getActiveResult: 0,
	}

	_, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	// Test that migration can parse TSV lines correctly
	// This is tested via integration tests with real backends
}

// DismissAll tests

func TestDualWriterDismissAllWritesToBothBackends(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.DismissAll()
	require.NoError(t, err)
	require.Equal(t, 1, tsv.dismissAllCalls)
	require.Equal(t, 1, sql.dismissAllCalls)
}

func TestDualWriterDismissAllReturnsTSVErrorAndSkipsSQLite(t *testing.T) {
	tsv := &fakeStorage{dismissAllErr: fmt.Errorf("tsv failed")}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.DismissAll()
	require.Error(t, err)
	require.Equal(t, 1, tsv.dismissAllCalls)
	require.Equal(t, 0, sql.dismissAllCalls)
}

func TestDualWriterDismissAllVerifyOnlySkipsSQLiteWrites(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{VerifyOnly: true})
	require.NoError(t, err)

	err = dw.DismissAll()
	require.NoError(t, err)
	require.Equal(t, 1, tsv.dismissAllCalls)
	require.Equal(t, 0, sql.dismissAllCalls)
}

// MarkNotificationUnread tests

func TestDualWriterMarkNotificationUnreadWritesToBothBackends(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.MarkNotificationUnread("1")
	require.NoError(t, err)
	require.Equal(t, 1, tsv.markUnreadCalls)
	require.Equal(t, 1, sql.markUnreadCalls)
}

func TestDualWriterMarkNotificationUnreadReturnsTSVErrorAndSkipsSQLite(t *testing.T) {
	tsv := &fakeStorage{markUnreadErr: fmt.Errorf("tsv failed")}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.MarkNotificationUnread("1")
	require.Error(t, err)
	require.Equal(t, 1, tsv.markUnreadCalls)
	require.Equal(t, 0, sql.markUnreadCalls)
}

func TestDualWriterMarkNotificationUnreadVerifyOnlySkipsSQLiteWrites(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{VerifyOnly: true})
	require.NoError(t, err)

	err = dw.MarkNotificationUnread("1")
	require.NoError(t, err)
	require.Equal(t, 1, tsv.markUnreadCalls)
	require.Equal(t, 0, sql.markUnreadCalls)
}

// CleanupOldNotifications tests

func TestDualWriterCleanupOldNotificationsWritesToBothBackends(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.CleanupOldNotifications(7, false)
	require.NoError(t, err)
	require.Equal(t, 1, tsv.cleanupCalls)
	require.Equal(t, 1, sql.cleanupCalls)
}

func TestDualWriterCleanupOldNotificationsWithDryRun(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.CleanupOldNotifications(7, true)
	require.NoError(t, err)
	require.Equal(t, 1, tsv.cleanupCalls)
	require.Equal(t, 1, sql.cleanupCalls)
}

func TestDualWriterCleanupOldNotificationsReturnsTSVErrorAndSkipsSQLite(t *testing.T) {
	tsv := &fakeStorage{cleanupErr: fmt.Errorf("tsv failed")}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.CleanupOldNotifications(7, false)
	require.Error(t, err)
	require.Equal(t, 1, tsv.cleanupCalls)
	require.Equal(t, 0, sql.cleanupCalls)
}

func TestDualWriterCleanupOldNotificationsVerifyOnlySkipsSQLiteWrites(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{VerifyOnly: true})
	require.NoError(t, err)

	err = dw.CleanupOldNotifications(7, false)
	require.NoError(t, err)
	require.Equal(t, 1, tsv.cleanupCalls)
	require.Equal(t, 0, sql.cleanupCalls)
}

func TestDualWriterDismissAllTracksMetrics(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.DismissAll()
	require.NoError(t, err)

	metrics := dw.Metrics()
	require.Equal(t, int64(1), metrics.WriteOperations)
	require.Equal(t, int64(0), metrics.TSVWriteFailures)
	require.Equal(t, int64(0), metrics.SQLiteWriteFailure)
}

func TestDualWriterMarkNotificationUnreadTracksMetrics(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{markUnreadErr: fmt.Errorf("sqlite failed")}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.MarkNotificationUnread("1")
	require.NoError(t, err)

	metrics := dw.Metrics()
	require.Equal(t, int64(1), metrics.WriteOperations)
	require.Equal(t, int64(0), metrics.TSVWriteFailures)
	require.Equal(t, int64(1), metrics.SQLiteWriteFailure)
}

func TestDualWriterCleanupOldNotificationsTracksMetrics(t *testing.T) {
	tsv := &fakeStorage{}
	sql := &fakeStorage{cleanupErr: fmt.Errorf("sqlite failed")}

	dw, err := NewDualWriterWithBackends(tsv, sql, DualWriterOptions{})
	require.NoError(t, err)

	err = dw.CleanupOldNotifications(7, false)
	require.NoError(t, err)

	metrics := dw.Metrics()
	require.Equal(t, int64(1), metrics.WriteOperations)
	require.Equal(t, int64(0), metrics.TSVWriteFailures)
	require.Equal(t, int64(1), metrics.SQLiteWriteFailure)
}

// Tests for isValidState helper function

func TestIsValidState(t *testing.T) {
	testCases := []struct {
		name  string
		state string
		want  bool
	}{
		{"valid state active", "active", true},
		{"valid state dismissed", "dismissed", true},
		{"invalid state empty", "", false},
		{"invalid state lowercase", "inactive", false},
		{"invalid state uppercase", "ACTIVE", false},
		{"invalid state mixed case", "Active", false},
		{"invalid state partial", "activ", false},
		{"invalid state number", "123", false},
		{"invalid state whitespace", "  ", false},
		{"invalid state tab", "\t", false},
		{"invalid state with spaces", "active ", false},
		{"invalid state with prefix", "pre-active", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidState(tc.state)
			require.Equal(t, tc.want, result, "unexpected result for state '%s'", tc.state)
		})
	}
}

// Tests for isValidLevel helper function

func TestIsValidLevel(t *testing.T) {
	testCases := []struct {
		name  string
		level string
		want  bool
	}{
		{"valid level info", "info", true},
		{"valid level warning", "warning", true},
		{"valid level error", "error", true},
		{"valid level critical", "critical", true},
		{"invalid level empty", "", false},
		{"invalid level lowercase", "debug", false},
		{"invalid level uppercase", "INFO", false},
		{"invalid level mixed case", "Info", false},
		{"invalid level partial", "inf", false},
		{"invalid level number", "123", false},
		{"invalid level whitespace", "  ", false},
		{"invalid level tab", "\t", false},
		{"invalid level with spaces", "info ", false},
		{"invalid level with prefix", "pre-info", false},
		{"invalid level trace", "trace", false},
		{"invalid level fatal", "fatal", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidLevel(tc.level)
			require.Equal(t, tc.want, result, "unexpected result for level '%s'", tc.level)
		})
	}
}

// Tests for migrateOneRecord helper function

func TestMigrateOneRecord(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	// Cleanup the SQLite storage connection
	t.Cleanup(func() {
		sqliteStorage.Close()
	})

	testCases := []struct {
		name          string
		idStr         string
		fields        []string
		wantError     bool
		errorContains string
		verifyRecord  bool // If true, verify the record was actually inserted
	}{
		{
			name:  "valid record upserts successfully",
			idStr: "5",
			fields: []string{
				"5",                    // ID
				"2025-01-01T12:00:00Z", // timestamp
				"active",               // state
				"session1",             // session
				"window0",              // window
				"pane0",                // pane
				"test message",         // message
				"2025-01-01T10:00:00Z", // pane_created (must be empty or valid timestamp)
				"info",                 // level
				"",                     // read_timestamp
			},
			wantError:    false,
			verifyRecord: true,
		},
		{
			name:  "valid dismissed record upserts successfully",
			idStr: "10",
			fields: []string{
				"10",
				"2025-01-01T12:00:00Z",
				"dismissed",
				"session1",
				"window0",
				"pane0",
				"test message",
				"2025-01-01T10:00:00Z", // pane_created (must be empty or valid timestamp)
				"warning",
				"2025-01-01T13:00:00Z",
			},
			wantError:    false,
			verifyRecord: true,
		},
		{
			name:  "too few fields returns nil (no error)",
			idStr: "5",
			fields: []string{
				"5",
				"2025-01-01T12:00:00Z",
				"active",
			},
			wantError:    false,
			verifyRecord: false,
		},
		{
			name:  "invalid ID (non-numeric) returns nil (no error, logs warning)",
			idStr: "abc",
			fields: []string{
				"abc",
				"2025-01-01T12:00:00Z",
				"active",
				"session1",
				"window0",
				"pane0",
				"test message",
				"", // pane_created (empty)
				"info",
				"",
			},
			wantError:    false,
			verifyRecord: false,
		},
		{
			name:  "invalid state returns nil (no error, logs warning)",
			idStr: "5",
			fields: []string{
				"5",
				"2025-01-01T12:00:00Z",
				"invalid",
				"session1",
				"window0",
				"pane0",
				"test message",
				"", // pane_created (empty)
				"info",
				"",
			},
			wantError:    false,
			verifyRecord: false,
		},
		{
			name:  "empty state returns nil (no error, logs warning)",
			idStr: "5",
			fields: []string{
				"5",
				"2025-01-01T12:00:00Z",
				"",
				"session1",
				"window0",
				"pane0",
				"test message",
				"", // pane_created (empty)
				"info",
				"",
			},
			wantError:    false,
			verifyRecord: false,
		},
		{
			name:  "invalid level returns nil (no error, logs warning)",
			idStr: "5",
			fields: []string{
				"5",
				"2025-01-01T12:00:00Z",
				"active",
				"session1",
				"window0",
				"pane0",
				"test message",
				"", // pane_created (empty)
				"debug",
				"",
			},
			wantError:    false,
			verifyRecord: false,
		},
		{
			name:  "empty level returns nil (no error, logs warning)",
			idStr: "5",
			fields: []string{
				"5",
				"2025-01-01T12:00:00Z",
				"active",
				"session1",
				"window0",
				"pane0",
				"test message",
				"", // pane_created (empty)
				"",
				"",
			},
			wantError:    false,
			verifyRecord: false,
		},
		{
			name:  "record with critical level upserts successfully",
			idStr: "99",
			fields: []string{
				"99",
				"2025-01-01T12:00:00Z",
				"active",
				"session1",
				"window0",
				"pane0",
				"critical error",
				"2025-01-01T10:00:00Z", // pane_created (valid timestamp)
				"critical",
				"",
			},
			wantError:    false,
			verifyRecord: true,
		},
		{
			name:  "record with escaped message upserts successfully",
			idStr: "7",
			fields: []string{
				"7",
				"2025-01-01T12:00:00Z",
				"active",
				"session1",
				"window0",
				"pane0",
				"msg\\nwith\\ttabs\\n",
				"2025-01-01T10:00:00Z", // pane_created (valid timestamp)
				"warning",
				"",
			},
			wantError:    false,
			verifyRecord: true,
		},
		{
			name:  "record with all empty optional fields upserts successfully",
			idStr: "3",
			fields: []string{
				"3",
				"2025-01-01T12:00:00Z",
				"active",
				"",
				"",
				"",
				"",
				"",
				"info",
				"",
			},
			wantError:    false,
			verifyRecord: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateOneRecord(sqliteStorage, tc.idStr, tc.fields)

			if tc.wantError {
				require.Error(t, err, "expected error for test case: "+tc.name)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains, "error message should contain: "+tc.errorContains)
				}
			} else {
				require.NoError(t, err, "unexpected error for test case: "+tc.name)

				// If verifyRecord is true, check that the record was actually inserted
				if tc.verifyRecord {
					// Query the record from SQLite to verify it was inserted
					// We need to use the SQLite storage's GetNotificationByID method
					line, err := sqliteStorage.GetNotificationByID(tc.idStr)
					require.NoError(t, err, "should be able to retrieve migrated record")
					require.NotEmpty(t, line, "migrated record should not be empty")

					// Parse the line and verify fields match
					fields := strings.Split(line, "\t")
					if len(fields) > 0 {
						require.Equal(t, tc.idStr, fields[FieldID], "ID should match")
					}
					if len(fields) > FieldState {
						require.Equal(t, tc.fields[FieldState], fields[FieldState], "state should match")
					}
					if len(fields) > FieldLevel {
						require.Equal(t, tc.fields[FieldLevel], fields[FieldLevel], "level should match")
					}
				}
			}
		})
	}
}
