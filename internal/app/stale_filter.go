package app

import (
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// KeepOnlyResolvableTmuxRows removes stale tmux rows from standard human-readable output.
func KeepOnlyResolvableTmuxRows(notifs []*domain.Notification, ftype format.FormatterType, displayNames DisplayNames, rawIDs, showStale bool) []*domain.Notification {
	if showStale || rawIDs || ftype != format.FormatterTypeSimple {
		return notifs
	}

	filtered := make([]*domain.Notification, 0, len(notifs))
	for _, notif := range notifs {
		if notif == nil {
			continue
		}
		if displayNames.IsResolvedNotification(*notif) {
			filtered = append(filtered, notif)
		}
	}
	return filtered
}

// KeepOnlyResolvableNotifications removes stale tmux notifications unless explicitly requested.
func KeepOnlyResolvableNotifications(notifs []notification.Notification, displayNames DisplayNames, showStale bool) []notification.Notification {
	if showStale {
		return notifs
	}

	filtered := make([]notification.Notification, 0, len(notifs))
	for _, notif := range notifs {
		domainNotif := notification.ToDomainUnsafe(notif)
		if domainNotif != nil && displayNames.IsResolvedNotification(*domainNotif) {
			filtered = append(filtered, notif)
		}
	}
	return filtered
}
