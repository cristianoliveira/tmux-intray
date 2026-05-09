package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRepo struct {
	notifs []Notification
	err    error
}

func (s *stubRepo) List(_, _, _, _, _, _, _, _ string) ([]Notification, error) {
	return s.notifs, s.err
}

func (s *stubRepo) Add(_, _, _, _, _, _, _ string) (int, error) { return 0, nil }
func (s *stubRepo) GetByID(_ int) (*Notification, error)         { return nil, nil }
func (s *stubRepo) Dismiss(_ int) error                          { return nil }
func (s *stubRepo) DismissAll() error                            { return nil }
func (s *stubRepo) DismissByFilter(_, _, _ string) error         { return nil }
func (s *stubRepo) MarkRead(_ int) error                         { return nil }
func (s *stubRepo) MarkUnread(_ int) error                       { return nil }
func (s *stubRepo) CleanupOld(_ int, _ bool) error               { return nil }
func (s *stubRepo) GetActiveCount() int                          { return 0 }

var _ NotificationRepository = (*stubRepo)(nil)

func makeNotif(id int, message, timestamp string, read bool) Notification {
	var readTS string
	if read {
		readTS = "2025-01-01T13:00:00Z"
	}
	return Notification{
		ID:            id,
		Message:       message,
		Timestamp:     timestamp,
		State:         StateActive,
		Level:         LevelInfo,
		ReadTimestamp: readTS,
	}
}

func TestNotificationService_Query_ReturnsAll(t *testing.T) {
	repo := &stubRepo{notifs: []Notification{
		makeNotif(1, "hello", "2025-01-01T12:00:00Z", false),
		makeNotif(2, "world", "2025-01-01T12:01:00Z", false),
	}}
	svc := NewNotificationService(repo)

	result, err := svc.Query(QueryParams{})
	require.NoError(t, err)
	assert.Len(t, result.Notifications, 2)
}

func TestNotificationService_Query_SearchFilter(t *testing.T) {
	repo := &stubRepo{notifs: []Notification{
		makeNotif(1, "hello world", "2025-01-01T12:00:00Z", false),
		makeNotif(2, "goodbye", "2025-01-01T12:01:00Z", false),
		makeNotif(3, "HELLO again", "2025-01-01T12:02:00Z", false),
	}}
	svc := NewNotificationService(repo)

	result, err := svc.Query(QueryParams{Search: "hello"})
	require.NoError(t, err)
	assert.Len(t, result.Notifications, 2)
}

func TestNotificationService_Query_SortByTimestampDesc(t *testing.T) {
	repo := &stubRepo{notifs: []Notification{
		makeNotif(1, "oldest", "2025-01-01T10:00:00Z", false),
		makeNotif(2, "newest", "2025-01-01T12:00:00Z", false),
		makeNotif(3, "middle", "2025-01-01T11:00:00Z", false),
	}}
	svc := NewNotificationService(repo)

	result, err := svc.Query(QueryParams{SortBy: "timestamp", SortOrder: "desc"})
	require.NoError(t, err)
	assert.Len(t, result.Notifications, 3)
	assert.Equal(t, "newest", result.Notifications[0].Message)
	assert.Equal(t, "oldest", result.Notifications[2].Message)
}

func TestNotificationService_Query_UnreadFirst(t *testing.T) {
	repo := &stubRepo{notifs: []Notification{
		makeNotif(1, "read1", "2025-01-01T10:00:00Z", true),
		makeNotif(2, "unread1", "2025-01-01T11:00:00Z", false),
		makeNotif(3, "read2", "2025-01-01T12:00:00Z", true),
		makeNotif(4, "unread2", "2025-01-01T09:00:00Z", false),
	}}
	svc := NewNotificationService(repo)

	result, err := svc.Query(QueryParams{SortBy: "timestamp", SortOrder: "asc", UnreadFirst: true})
	require.NoError(t, err)
	assert.Len(t, result.Notifications, 4)
	// First two should be unread
	assert.False(t, result.Notifications[0].IsRead())
	assert.False(t, result.Notifications[1].IsRead())
	// Last two should be read
	assert.True(t, result.Notifications[2].IsRead())
	assert.True(t, result.Notifications[3].IsRead())
}

func TestNotificationService_Query_RepoError(t *testing.T) {
	repo := &stubRepo{err: errors.New("db down")}
	svc := NewNotificationService(repo)

	_, err := svc.Query(QueryParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}
