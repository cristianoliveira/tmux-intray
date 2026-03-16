package storage

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStorage is a mock implementation of Storage for testing.
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	args := m.Called(message, timestamp, session, window, pane, paneCreated, level)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	args := m.Called(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetNotificationByID(id string) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) DismissNotification(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorage) DismissAll() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) DismissByFilter(session, window, pane string) error {
	args := m.Called(session, window, pane)
	return args.Error(0)
}

func (m *MockStorage) MarkNotificationRead(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorage) MarkNotificationUnread(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorage) MarkNotificationReadWithTimestamp(id, timestamp string) error {
	args := m.Called(id, timestamp)
	return args.Error(0)
}

func (m *MockStorage) MarkNotificationUnreadWithTimestamp(id, timestamp string) error {
	args := m.Called(id, timestamp)
	return args.Error(0)
}

func (m *MockStorage) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	args := m.Called(daysThreshold, dryRun)
	return args.Error(0)
}

func (m *MockStorage) GetActiveCount() int {
	args := m.Called()
	return args.Int(0)
}

var _ Storage = (*MockStorage)(nil)

func TestDomainRepositoryAdapter_Add(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("AddNotification", "test message", "", "session1", "window0", "pane0", "", "info").
			Return("42", nil)

		id, err := adapter.Add("test message", "", "session1", "window0", "pane0", "", "info")
		require.NoError(t, err)
		assert.Equal(t, 42, id)
		mockStorage.AssertExpectations(t)
	})

	t.Run("storage error", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		expectedErr := errors.New("storage failure")
		mockStorage.On("AddNotification", "msg", "", "", "", "", "", "warning").
			Return("", expectedErr)

		id, err := adapter.Add("msg", "", "", "", "", "", "warning")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStorageFailed)
		assert.Contains(t, err.Error(), "storage failure")
		assert.Equal(t, 0, id)
		mockStorage.AssertExpectations(t)
	})

	t.Run("invalid ID format", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("AddNotification", "msg", "", "", "", "", "", "info").
			Return("not-a-number", nil)

		id, err := adapter.Add("msg", "", "", "", "", "", "info")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID format")
		assert.Equal(t, 0, id)
		mockStorage.AssertExpectations(t)
	})
}

func TestDomainRepositoryAdapter_List(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("ListNotifications", "", "", "", "", "", "", "", "").
			Return("", nil)

		notifs, err := adapter.List("", "", "", "", "", "", "", "")
		require.NoError(t, err)
		assert.Empty(t, notifs)
		mockStorage.AssertExpectations(t)
	})

	t.Run("multiple notifications", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		// Create valid TSV lines (ID, Timestamp, State, Session, Window, Pane, Message, PaneCreated, Level, ReadTimestamp)
		tsvData := "1\t2025-01-01T12:00:00Z\tactive\tsess1\twin0\tpane0\ttest message\t123456789\tinfo\t\n" +
			"2\t2025-01-01T12:30:00Z\tdismissed\tsess2\twin1\tpane1\tanother message\t987654321\twarning\t2025-01-01T13:00:00Z"
		mockStorage.On("ListNotifications", "all", "", "", "", "", "", "", "").
			Return(tsvData, nil)

		notifs, err := adapter.List("all", "", "", "", "", "", "", "")
		require.NoError(t, err)
		require.Len(t, notifs, 2)
		// Check first notification
		assert.Equal(t, 1, notifs[0].ID)
		assert.Equal(t, "2025-01-01T12:00:00Z", notifs[0].Timestamp)
		assert.Equal(t, domain.StateActive, notifs[0].State)
		assert.Equal(t, "sess1", notifs[0].Session)
		assert.Equal(t, "win0", notifs[0].Window)
		assert.Equal(t, "pane0", notifs[0].Pane)
		assert.Equal(t, "test message", notifs[0].Message)
		assert.Equal(t, "123456789", notifs[0].PaneCreated)
		assert.Equal(t, domain.LevelInfo, notifs[0].Level)
		assert.Empty(t, notifs[0].ReadTimestamp)
		// Check second notification
		assert.Equal(t, 2, notifs[1].ID)
		assert.Equal(t, domain.StateDismissed, notifs[1].State)
		assert.Equal(t, domain.LevelWarning, notifs[1].Level)
		assert.Equal(t, "2025-01-01T13:00:00Z", notifs[1].ReadTimestamp)
		mockStorage.AssertExpectations(t)
	})

	t.Run("handle storage error", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		expectedErr := errors.New("storage failure")
		mockStorage.On("ListNotifications", "active", "info", "", "", "", "", "", "").
			Return("", expectedErr)

		notifs, err := adapter.List("active", "info", "", "", "", "", "", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStorageFailed)
		assert.Contains(t, err.Error(), "storage failure")
		assert.Nil(t, notifs)
		mockStorage.AssertExpectations(t)
	})
}

func TestDomainRepositoryAdapter_GetByID(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		tsvData := "5\t2025-01-01T12:00:00Z\tactive\tsess\twin\tpane\ttest message\t123456789\tinfo\t"
		mockStorage.On("GetNotificationByID", "5").Return(tsvData, nil)

		notif, err := adapter.GetByID(5)
		require.NoError(t, err)
		require.NotNil(t, notif)
		assert.Equal(t, 5, notif.ID)
		assert.Equal(t, "2025-01-01T12:00:00Z", notif.Timestamp)
		assert.Equal(t, domain.StateActive, notif.State)
		assert.Equal(t, "sess", notif.Session)
		assert.Equal(t, "win", notif.Window)
		assert.Equal(t, "pane", notif.Pane)
		assert.Equal(t, "test message", notif.Message)
		assert.Equal(t, "123456789", notif.PaneCreated)
		assert.Equal(t, domain.LevelInfo, notif.Level)
		assert.Empty(t, notif.ReadTimestamp)
		mockStorage.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("GetNotificationByID", "99").Return("", nil)

		notif, err := adapter.GetByID(99)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
		assert.Nil(t, notif)
		mockStorage.AssertExpectations(t)
	})

	t.Run("handle storage error", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		expectedErr := errors.New("storage failure")
		mockStorage.On("GetNotificationByID", "7").Return("", expectedErr)

		notif, err := adapter.GetByID(7)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStorageFailed)
		assert.Contains(t, err.Error(), "storage failure")
		assert.Nil(t, notif)
		mockStorage.AssertExpectations(t)
	})
}

func TestDomainRepositoryAdapter_Dismiss(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("DismissNotification", "42").Return(nil)

		err := adapter.Dismiss(42)
		require.NoError(t, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("handle storage error", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		expectedErr := errors.New("storage failure")
		mockStorage.On("DismissNotification", "99").Return(expectedErr)

		err := adapter.Dismiss(99)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStorageFailed)
		assert.Contains(t, err.Error(), "storage failure")
		mockStorage.AssertExpectations(t)
	})
}

func TestDomainRepositoryAdapter_DismissAll(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("DismissAll").Return(nil)

		err := adapter.DismissAll()
		require.NoError(t, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("handle storage error", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		expectedErr := errors.New("storage failure")
		mockStorage.On("DismissAll").Return(expectedErr)

		err := adapter.DismissAll()
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStorageFailed)
		assert.Contains(t, err.Error(), "storage failure")
		mockStorage.AssertExpectations(t)
	})
}

func TestDomainRepositoryAdapter_MarkRead(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("MarkNotificationRead", "42").Return(nil)

		err := adapter.MarkRead(42)
		require.NoError(t, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("handle storage error", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		expectedErr := errors.New("storage failure")
		mockStorage.On("MarkNotificationRead", "99").Return(expectedErr)

		err := adapter.MarkRead(99)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStorageFailed)
		assert.Contains(t, err.Error(), "storage failure")
		mockStorage.AssertExpectations(t)
	})
}

func TestDomainRepositoryAdapter_MarkUnread(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("MarkNotificationUnread", "42").Return(nil)

		err := adapter.MarkUnread(42)
		require.NoError(t, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("handle storage error", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		expectedErr := errors.New("storage failure")
		mockStorage.On("MarkNotificationUnread", "99").Return(expectedErr)

		err := adapter.MarkUnread(99)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStorageFailed)
		assert.Contains(t, err.Error(), "storage failure")
		mockStorage.AssertExpectations(t)
	})
}

func TestDomainRepositoryAdapter_CleanupOld(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("CleanupOldNotifications", 30, true).Return(nil)

		err := adapter.CleanupOld(30, true)
		require.NoError(t, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("handle storage error", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		expectedErr := errors.New("storage failure")
		mockStorage.On("CleanupOldNotifications", 7, false).Return(expectedErr)

		err := adapter.CleanupOld(7, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStorageFailed)
		assert.Contains(t, err.Error(), "storage failure")
		mockStorage.AssertExpectations(t)
	})
}

func TestDomainRepositoryAdapter_GetActiveCount(t *testing.T) {
	t.Run("returns correct value", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("GetActiveCount").Return(5)

		count := adapter.GetActiveCount()
		assert.Equal(t, 5, count)
		mockStorage.AssertExpectations(t)
	})

	t.Run("returns zero when storage returns zero", func(t *testing.T) {
		mockStorage := new(MockStorage)
		adapter := NewDomainRepositoryAdapter(mockStorage)

		mockStorage.On("GetActiveCount").Return(0)

		count := adapter.GetActiveCount()
		assert.Equal(t, 0, count)
		mockStorage.AssertExpectations(t)
	})
}
