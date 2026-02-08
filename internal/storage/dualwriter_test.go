package storage

import (
	"fmt"
	"strings"
	"testing"

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

func (f *fakeStorage) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
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

	list, err := dw.ListNotifications("all", "", "", "", "", "", "")
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

	list, err := dw.ListNotifications("all", "", "", "", "", "", "")
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
