package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/assert"
)

type stubSearchProvider struct {
	matcher func(notification.Notification, string) bool
}

func (s stubSearchProvider) Match(n notification.Notification, query string) bool {
	return s.matcher(n, query)
}

func (s stubSearchProvider) Name() string {
	return "stub"
}

func TestNotificationServiceFilterNotifications(t *testing.T) {
	svc := newNotificationService()
	notifications := []notification.Notification{
		{ID: 1, Message: "first"},
		{ID: 2, Message: "second"},
		{ID: 3, Message: "third"},
	}

	provider := stubSearchProvider{matcher: func(n notification.Notification, query string) bool {
		return query == "match" && (n.ID == 1 || n.ID == 3)
	}}

	filtered := svc.filterNotifications(notifications, "  match  ", provider)
	assert.Len(t, filtered, 2)
	assert.Equal(t, 1, filtered[0].ID)
	assert.Equal(t, 3, filtered[1].ID)
}

func TestNotificationServiceFilterNotificationsEmptyQueryReturnsAll(t *testing.T) {
	svc := newNotificationService()
	notifications := []notification.Notification{{ID: 1}, {ID: 2}}

	provider := stubSearchProvider{matcher: func(n notification.Notification, query string) bool {
		return false
	}}

	filtered := svc.filterNotifications(notifications, "   ", provider)
	assert.Equal(t, notifications, filtered)
}
