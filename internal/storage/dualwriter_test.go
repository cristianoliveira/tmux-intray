package storage

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeStorage struct {
	mu sync.Mutex

	addFn            func(message, timestamp, session, window, pane, paneCreated, level string) (string, error)
	listFn           func(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error)
	getFn            func(id string) (string, error)
	dismissFn        func(id string) error
	dismissAllFn     func() error
	markReadFn       func(id string) error
	markUnreadFn     func(id string) error
	cleanupFn        func(daysThreshold int, dryRun bool) error
	getActiveCountFn func() int

	addCalls int
}

func (f *fakeStorage) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	f.mu.Lock()
	f.addCalls++
	f.mu.Unlock()
	if f.addFn != nil {
		return f.addFn(message, timestamp, session, window, pane, paneCreated, level)
	}
	return "1", nil
}

func (f *fakeStorage) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
	if f.listFn != nil {
		return f.listFn(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
	}
	return "", nil
}

func (f *fakeStorage) GetNotificationByID(id string) (string, error) {
	if f.getFn != nil {
		return f.getFn(id)
	}
	return "", nil
}

func (f *fakeStorage) DismissNotification(id string) error {
	if f.dismissFn != nil {
		return f.dismissFn(id)
	}
	return nil
}

func (f *fakeStorage) DismissAll() error {
	if f.dismissAllFn != nil {
		return f.dismissAllFn()
	}
	return nil
}

func (f *fakeStorage) MarkNotificationRead(id string) error {
	if f.markReadFn != nil {
		return f.markReadFn(id)
	}
	return nil
}

func (f *fakeStorage) MarkNotificationUnread(id string) error {
	if f.markUnreadFn != nil {
		return f.markUnreadFn(id)
	}
	return nil
}

func (f *fakeStorage) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	if f.cleanupFn != nil {
		return f.cleanupFn(daysThreshold, dryRun)
	}
	return nil
}

func (f *fakeStorage) GetActiveCount() int {
	if f.getActiveCountFn != nil {
		return f.getActiveCountFn()
	}
	return 0
}

func TestDualWriterAddWritesBothBackends(t *testing.T) {
	tsvStore := &fakeStorage{addFn: func(_, _, _, _, _, _, _ string) (string, error) { return "7", nil }}
	sqliteStore := &fakeStorage{addFn: func(_, _, _, _, _, _, _ string) (string, error) { return "7", nil }}

	dw, err := NewDualWriter(tsvStore, sqliteStore, DualWriterOptions{ReadFromSQLite: true, VerifyEveryNWrites: 0})
	require.NoError(t, err)

	id, err := dw.AddNotification("hello", "", "", "", "", "", "info")
	require.NoError(t, err)
	require.Equal(t, "7", id)
	require.Equal(t, 1, tsvStore.addCalls)
	require.Equal(t, 1, sqliteStore.addCalls)

	metrics := dw.Metrics()
	require.Equal(t, int64(1), metrics.WriteOperations)
}

func TestDualWriterSQLiteFailureFallsBackToTSVOnly(t *testing.T) {
	tsvStore := &fakeStorage{addFn: func(_, _, _, _, _, _, _ string) (string, error) { return "3", nil }}
	sqliteStore := &fakeStorage{addFn: func(_, _, _, _, _, _, _ string) (string, error) {
		return "", fmt.Errorf("sqlite write failed")
	}}

	dw, err := NewDualWriter(tsvStore, sqliteStore, DualWriterOptions{ReadFromSQLite: false, VerifyEveryNWrites: 0})
	require.NoError(t, err)

	id, err := dw.AddNotification("first", "", "", "", "", "", "info")
	require.NoError(t, err)
	require.Equal(t, "3", id)

	_, err = dw.AddNotification("second", "", "", "", "", "", "info")
	require.NoError(t, err)

	require.Equal(t, 2, tsvStore.addCalls)
	require.Equal(t, 1, sqliteStore.addCalls)
}

func TestDualWriterReadOnlyVerificationModeSkipsSQLiteWrites(t *testing.T) {
	tsvStore := &fakeStorage{addFn: func(_, _, _, _, _, _, _ string) (string, error) { return "9", nil }}
	sqliteStore := &fakeStorage{}

	dw, err := NewDualWriter(tsvStore, sqliteStore, DualWriterOptions{ReadOnlyVerificationMode: true, VerifyEveryNWrites: 0})
	require.NoError(t, err)

	id, err := dw.AddNotification("readonly", "", "", "", "", "", "info")
	require.NoError(t, err)
	require.Equal(t, "9", id)
	require.Equal(t, 1, tsvStore.addCalls)
	require.Equal(t, 0, sqliteStore.addCalls)
}

func TestDualWriterVerifyConsistencyDetectsMismatch(t *testing.T) {
	tsvStore := &fakeStorage{
		listFn: func(_, _, _, _, _, _, _ string) (string, error) {
			return "1\t2026-01-01T00:00:00Z\tactive\t\t\t\ttsv\t\tinfo\t", nil
		},
		getActiveCountFn: func() int { return 1 },
	}
	sqliteStore := &fakeStorage{
		listFn: func(_, _, _, _, _, _, _ string) (string, error) {
			return "1\t2026-01-01T00:00:00Z\tactive\t\t\t\tsqlite\t\tinfo\t", nil
		},
		getActiveCountFn: func() int { return 1 },
	}

	dw, err := NewDualWriter(tsvStore, sqliteStore, DualWriterOptions{ConsistencySampleSize: 1, VerifyEveryNWrites: 0})
	require.NoError(t, err)

	report, err := dw.VerifyConsistency()
	require.Error(t, err)
	require.Contains(t, report.MismatchedIDs, "1")
}
