package main

import (
	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
)

func keepOnlyResolvableTmuxRows(notifs []*domain.Notification, ftype format.FormatterType, displayNames appcore.DisplayNames, rawIDs, showStale bool) []*domain.Notification {
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
