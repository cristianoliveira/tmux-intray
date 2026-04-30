package app

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// DisplayNames holds bulk-resolved tmux labels for presentation.
type DisplayNames struct {
	Sessions map[string]string
	Windows  map[string]string
	Panes    map[string]string
}

// Resolve returns a display value for the given kind, falling back to a readable missing label.
func (d DisplayNames) Resolve(kind, raw string) string {
	if raw == "" {
		return raw
	}

	var names map[string]string
	switch kind {
	case "session":
		names = d.Sessions
	case "window":
		names = d.Windows
	case "pane":
		names = d.Panes
	default:
		return raw
	}

	if resolved := names[raw]; resolved != "" {
		return resolved
	}
	return MissingDisplayName(kind, raw)
}

// MissingDisplayName returns a human-readable fallback for stale tmux IDs.
func MissingDisplayName(kind, raw string) string {
	if raw == "" {
		return raw
	}

	switch kind {
	case "session":
		if strings.HasPrefix(raw, "$") {
			return "stale-session:" + raw
		}
	case "window":
		if strings.HasPrefix(raw, "@") {
			return "stale-window:" + raw
		}
	case "pane":
		if strings.HasPrefix(raw, "%") {
			return "stale-pane:" + raw
		}
	}
	return raw
}

// IsResolvedNotification reports whether all tmux routing IDs have live display names.
func (d DisplayNames) IsResolvedNotification(notif domain.Notification) bool {
	if notif.Session != "" && d.Sessions != nil && d.Sessions[notif.Session] == "" {
		return false
	}
	if notif.Window != "" && d.Windows != nil && d.Windows[notif.Window] == "" {
		return false
	}
	if notif.Pane != "" && d.Panes != nil && d.Panes[notif.Pane] == "" {
		return false
	}
	return true
}

// EnrichNotification returns a copy with human-readable tmux labels.
func (d DisplayNames) EnrichNotification(notif domain.Notification) domain.Notification {
	notif.Session = d.Resolve("session", notif.Session)
	notif.Window = d.Resolve("window", notif.Window)
	notif.Pane = d.Resolve("pane", notif.Pane)
	return notif
}

// EnrichNotifications returns copies with human-readable tmux labels.
func (d DisplayNames) EnrichNotifications(notifs []*domain.Notification) []*domain.Notification {
	if len(notifs) == 0 {
		return notifs
	}

	enriched := make([]*domain.Notification, len(notifs))
	for i, notif := range notifs {
		if notif == nil {
			continue
		}
		copy := d.EnrichNotification(*notif)
		enriched[i] = &copy
	}
	return enriched
}

// EnrichGroupResult updates group headers for presentation while preserving raw group keys.
func (d DisplayNames) EnrichGroupResult(result domain.GroupResult) domain.GroupResult {
	if len(result.Groups) == 0 {
		return result
	}

	enriched := make([]domain.Group, len(result.Groups))
	for i, group := range result.Groups {
		copy := group
		copy.DisplayName = d.ResolveGroupDisplayName(group.Key, result.Mode)
		enriched[i] = copy
	}
	result.Groups = enriched
	return result
}

// ResolveGroupDisplayName maps a raw group key to a human-readable label.
func (d DisplayNames) ResolveGroupDisplayName(key string, mode domain.GroupByMode) string {
	if key == "" {
		return "(empty)"
	}

	switch mode {
	case domain.GroupBySession:
		return d.Resolve("session", key)
	case domain.GroupByWindow:
		parts := splitGroupKey(key)
		if len(parts) >= 2 {
			return d.Resolve("window", parts[1])
		}
	case domain.GroupByPane:
		parts := splitGroupKey(key)
		if len(parts) >= 3 {
			return d.Resolve("pane", parts[2])
		}
	}

	return key
}

func splitGroupKey(key string) []string {
	parts := []string{}
	current := ""
	for _, r := range key {
		if r == '\x00' {
			parts = append(parts, current)
			current = ""
			continue
		}
		current += string(r)
	}
	parts = append(parts, current)
	return parts
}
