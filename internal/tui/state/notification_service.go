package state

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

type notificationService struct{}

func newNotificationService() *notificationService {
	return &notificationService{}
}

func (s *notificationService) loadActiveNotifications() ([]notification.Notification, error) {
	lines, err := storage.ListNotifications("active", "", "", "", "", "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to load notifications: %w", err)
	}
	if lines == "" {
		return []notification.Notification{}, nil
	}

	items := make([]notification.Notification, 0)
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, parseErr := notification.ParseNotification(line)
		if parseErr != nil {
			continue
		}
		items = append(items, notif)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp > items[j].Timestamp
	})

	return items, nil
}

func (s *notificationService) filterNotifications(notifications []notification.Notification, query string, provider search.Provider) []notification.Notification {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return notifications
	}

	filtered := make([]notification.Notification, 0, len(notifications))
	for _, notif := range notifications {
		if provider.Match(notif, trimmed) {
			filtered = append(filtered, notif)
		}
	}

	return filtered
}

func (s *notificationService) dismissNotification(id string) error {
	return storage.DismissNotification(id)
}

func (s *notificationService) markNotificationRead(id string) error {
	return storage.MarkNotificationRead(id)
}

func (s *notificationService) markNotificationUnread(id string) error {
	return storage.MarkNotificationUnread(id)
}
