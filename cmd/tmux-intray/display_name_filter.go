package main

import (
	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
)

func keepOnlyResolvableTmuxRows(notifs []*domain.Notification, ftype format.FormatterType, displayNames appcore.DisplayNames, rawIDs bool) []*domain.Notification {
	if rawIDs || ftype != format.FormatterTypeSimple {
		return notifs
	}

	filtered := make([]*domain.Notification, 0, len(notifs))
	for _, notif := range notifs {
		if notif == nil {
			continue
		}
		if hasResolvedTmuxNames(*notif, displayNames) {
			filtered = append(filtered, notif)
		}
	}
	return filtered
}

func hasResolvedTmuxNames(notif domain.Notification, displayNames appcore.DisplayNames) bool {
	if notif.Session == "" || notif.Window == "" || notif.Pane == "" {
		return false
	}

	if displayNames.Sessions[notif.Session] == "" {
		return false
	}
	if displayNames.Windows[notif.Window] == "" {
		return false
	}
	if displayNames.Panes[notif.Pane] == "" {
		return false
	}

	return true
}
