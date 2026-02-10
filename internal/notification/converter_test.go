package notification

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validOldNotification() Notification {
	return Notification{
		ID:            42,
		Timestamp:     "2025-01-01T12:00:00Z",
		State:         "active",
		Session:       "session1",
		Window:        "window2",
		Pane:          "pane3",
		Message:       "test message",
		PaneCreated:   "123",
		Level:         "info",
		ReadTimestamp: "2025-01-02T01:02:03Z",
	}
}

func validDomainNotification() *domain.Notification {
	// This matches validOldNotification after conversion
	state, _ := domain.ParseNotificationState("active")
	level, _ := domain.ParseNotificationLevel("info")
	return &domain.Notification{
		ID:            42,
		Timestamp:     "2025-01-01T12:00:00Z",
		State:         state,
		Session:       "session1",
		Window:        "window2",
		Pane:          "pane3",
		Message:       "test message",
		PaneCreated:   "123",
		Level:         level,
		ReadTimestamp: "2025-01-02T01:02:03Z",
	}
}

func TestToDomain_ValidNotification(t *testing.T) {
	old := validOldNotification()

	domainNotif, err := ToDomain(old)
	require.NoError(t, err)
	require.NotNil(t, domainNotif)

	assert.Equal(t, old.ID, domainNotif.ID)
	assert.Equal(t, old.Timestamp, domainNotif.Timestamp)
	assert.Equal(t, old.State, domainNotif.State.String())
	assert.Equal(t, old.Session, domainNotif.Session)
	assert.Equal(t, old.Window, domainNotif.Window)
	assert.Equal(t, old.Pane, domainNotif.Pane)
	assert.Equal(t, old.Message, domainNotif.Message)
	assert.Equal(t, old.PaneCreated, domainNotif.PaneCreated)
	assert.Equal(t, old.Level, domainNotif.Level.String())
	assert.Equal(t, old.ReadTimestamp, domainNotif.ReadTimestamp)
}

func TestToDomain_EmptyStateLevel(t *testing.T) {
	old := validOldNotification()
	old.State = ""
	old.Level = ""

	_, err := ToDomain(old)
	require.Error(t, err)
	// Should be validation error because empty state/level are invalid
	assert.Contains(t, err.Error(), "validation failed")
}

func TestToDomain_InvalidState(t *testing.T) {
	old := validOldNotification()
	old.State = "invalid"

	_, err := ToDomain(old)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state")
}

func TestToDomain_InvalidLevel(t *testing.T) {
	old := validOldNotification()
	old.Level = "invalid"

	_, err := ToDomain(old)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid level")
}

func TestToDomain_RoundTrip(t *testing.T) {
	old := validOldNotification()

	domainNotif, err := ToDomain(old)
	require.NoError(t, err)

	old2 := FromDomain(domainNotif)
	assert.Equal(t, old, old2)
}

func TestFromDomain(t *testing.T) {
	domainNotif := validDomainNotification()

	old := FromDomain(domainNotif)

	assert.Equal(t, domainNotif.ID, old.ID)
	assert.Equal(t, domainNotif.Timestamp, old.Timestamp)
	assert.Equal(t, domainNotif.State.String(), old.State)
	assert.Equal(t, domainNotif.Session, old.Session)
	assert.Equal(t, domainNotif.Window, old.Window)
	assert.Equal(t, domainNotif.Pane, old.Pane)
	assert.Equal(t, domainNotif.Message, old.Message)
	assert.Equal(t, domainNotif.PaneCreated, old.PaneCreated)
	assert.Equal(t, domainNotif.Level.String(), old.Level)
	assert.Equal(t, domainNotif.ReadTimestamp, old.ReadTimestamp)
}

func TestToDomainSlice(t *testing.T) {
	olds := []Notification{
		validOldNotification(),
		validOldNotification(),
	}

	domainNotifs, err := ToDomainSlice(olds)
	require.NoError(t, err)
	require.Len(t, domainNotifs, 2)

	for i, d := range domainNotifs {
		assert.Equal(t, olds[i].ID, d.ID)
	}
}

func TestToDomainSlice_WithInvalid(t *testing.T) {
	olds := []Notification{
		validOldNotification(),
		{State: "invalid"},
		validOldNotification(),
	}

	_, err := ToDomainSlice(olds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification at index 1")
}

func TestFromDomainSlice(t *testing.T) {
	domainNotifs := []*domain.Notification{
		validDomainNotification(),
		validDomainNotification(),
	}

	olds := FromDomainSlice(domainNotifs)
	assert.Len(t, olds, 2)

	for i, old := range olds {
		assert.Equal(t, domainNotifs[i].ID, old.ID)
	}
}

func TestFromDomainSlice_WithNil(t *testing.T) {
	domainNotifs := []*domain.Notification{
		validDomainNotification(),
		nil,
		validDomainNotification(),
	}

	olds := FromDomainSlice(domainNotifs)
	assert.Len(t, olds, 2) // nil is skipped
	for _, old := range olds {
		assert.NotZero(t, old.ID)
	}
}

// Additional edge cases

func TestToDomain_EmptyMessage(t *testing.T) {
	old := validOldNotification()
	old.Message = ""

	_, err := ToDomain(old)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestToDomain_InvalidTimestamp(t *testing.T) {
	old := validOldNotification()
	old.Timestamp = "not a timestamp"

	_, err := ToDomain(old)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestToDomain_InvalidReadTimestamp(t *testing.T) {
	old := validOldNotification()
	old.ReadTimestamp = "not a timestamp"

	_, err := ToDomain(old)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}
