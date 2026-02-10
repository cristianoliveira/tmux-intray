package notification

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// ToDomain converts an old notification.Notification to a domain.Notification.
func ToDomain(n Notification) (*domain.Notification, error) {
	state, err := domain.ParseNotificationState(n.State)
	if err != nil && n.State != "" {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	level, err := domain.ParseNotificationLevel(n.Level)
	if err != nil && n.Level != "" {
		return nil, fmt.Errorf("invalid level: %w", err)
	}

	domainNotif, err := domain.NewNotification(
		n.ID,
		n.Timestamp,
		state,
		n.Session,
		n.Window,
		n.Pane,
		n.Message,
		n.PaneCreated,
		level,
		n.ReadTimestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return domainNotif, nil
}

// FromDomain converts a domain.Notification to an old notification.Notification.
func FromDomain(n *domain.Notification) Notification {
	return Notification{
		ID:            n.ID,
		Timestamp:     n.Timestamp,
		State:         n.State.String(),
		Session:       n.Session,
		Window:        n.Window,
		Pane:          n.Pane,
		Message:       n.Message,
		PaneCreated:   n.PaneCreated,
		Level:         n.Level.String(),
		ReadTimestamp: n.ReadTimestamp,
	}
}

// ToDomainSlice converts a slice of old notifications to domain notifications.
func ToDomainSlice(notifs []Notification) ([]*domain.Notification, error) {
	domainNotifs := make([]*domain.Notification, 0, len(notifs))
	for i, n := range notifs {
		domainNotif, err := ToDomain(n)
		if err != nil {
			return nil, fmt.Errorf("notification at index %d: %w", i, err)
		}
		domainNotifs = append(domainNotifs, domainNotif)
	}
	return domainNotifs, nil
}

// FromDomainSlice converts a slice of domain notifications to old notifications.
func FromDomainSlice(notifs []*domain.Notification) []Notification {
	oldNotifs := make([]Notification, 0, len(notifs))
	for _, n := range notifs {
		if n == nil {
			continue
		}
		oldNotifs = append(oldNotifs, FromDomain(n))
	}
	return oldNotifs
}
