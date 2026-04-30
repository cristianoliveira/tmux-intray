package app

import (
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
)

// StaleFilterOptions controls whether stale tmux targets should be visible.
type StaleFilterOptions struct {
	DisplayNames DisplayNames
	ShowStale    bool
}

// ShouldShowNotification is the canonical stale-target visibility decision.
func ShouldShowNotification(notif domain.Notification, opts StaleFilterOptions) bool {
	if opts.ShowStale {
		return true
	}
	return opts.DisplayNames.IsResolvedNotification(notif)
}

// KeepOnlyResolvableTmuxRows removes stale tmux rows from standard human-readable output.
func KeepOnlyResolvableTmuxRows(notifs []*domain.Notification, ftype format.FormatterType, displayNames DisplayNames, rawIDs, showStale bool) []*domain.Notification {
	if rawIDs || ftype != format.FormatterTypeSimple {
		return notifs
	}

	filtered := make([]*domain.Notification, 0, len(notifs))
	for _, notif := range notifs {
		if notif == nil {
			continue
		}
		if ShouldShowNotification(*notif, StaleFilterOptions{DisplayNames: displayNames, ShowStale: showStale}) {
			filtered = append(filtered, notif)
		}
	}
	return filtered
}

// KeepOnlyResolvableNotifications removes stale tmux domain notifications unless explicitly requested.
func KeepOnlyResolvableNotifications(notifs []domain.Notification, displayNames DisplayNames, showStale bool) []domain.Notification {
	filtered := make([]domain.Notification, 0, len(notifs))
	for _, notif := range notifs {
		if ShouldShowNotification(notif, StaleFilterOptions{DisplayNames: displayNames, ShowStale: showStale}) {
			filtered = append(filtered, notif)
		}
	}
	return filtered
}
