package app

import (
	"sort"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// ViewKind describes a high-level notification view orchestration that can be
// shared between CLI and TUI layers.
//
// For now we only standardize the "Recents per session" view used by the TUI
// Recents tab. More kinds can be added incrementally.
type ViewKind string

const (
	// ViewKindRecentsPerSession mirrors the TUI Recents tab behavior:
	// - active notifications only
	// - within the configurable recents_time_window
	// - unread only
	// - one representative notification per session, chosen by severity then recency
	// - at most recentsDatasetLimit sessions (currently 20)
	ViewKindRecentsPerSession ViewKind = "recents-per-session"

	// ViewKindSessionsPerSession mirrors the TUI Sessions tab behavior:
	// - active notifications only
	// - all-time history (no recency time window)
	// - read and unread notifications
	// - one representative notification per session, chosen by severity then recency
	// - no dataset limit
	ViewKindSessionsPerSession ViewKind = "sessions-per-session"
)

// ViewOptions controls how a view is orchestrated.
type ViewOptions struct {
	Kind   ViewKind
	SortBy string // domain.SortByField as string (e.g. "timestamp")
	Order  string // domain.SortOrder as string ("asc" | "desc")
}

// ViewResult contains the orchestrated notifications for a given view.
type ViewResult struct {
	Notifications []domain.Notification
}

// ViewOrchestrator builds high-level notification views on top of the domain
// layer. It is the single source of truth for shared orchestration logic used
// by both CLI and TUI.
type ViewOrchestrator struct{}

// NewViewOrchestrator creates a new ViewOrchestrator.
func NewViewOrchestrator() *ViewOrchestrator {
	return &ViewOrchestrator{}
}

// BuildView orchestrates a notification view based on the provided options and
// input notifications. The input slice is treated as domain-level data (already
// validated and loaded from storage).
func (o *ViewOrchestrator) BuildView(opts ViewOptions, notifs []domain.Notification) ViewResult {
	switch opts.Kind {
	case ViewKindRecentsPerSession:
		return ViewResult{Notifications: o.buildRecentsPerSession(opts, notifs)}
	case ViewKindSessionsPerSession:
		return ViewResult{Notifications: o.buildSessionsPerSession(opts, notifs)}
	default:
		// Unknown kind: return notifications as-is for now.
		return ViewResult{Notifications: notifs}
	}
}

// getRecentsTimeWindow returns the configured time window for the Recents view.
// It reuses the same configuration key as the TUI implementation so that both
// layers remain consistent.
func getRecentsTimeWindow() time.Duration {
	windowStr := config.Get("recents_time_window", "1h")
	duration, err := time.ParseDuration(windowStr)
	if err != nil {
		// Should not happen with validated config; fall back to 1 hour.
		return time.Hour
	}
	return duration
}

const (
	recentsDatasetLimit = 20
)

// buildRecentsPerSession implements the shared Recents-per-session semantics
// currently used by the TUI Recents tab:
//
//   - consider active notifications only
//   - restrict to the configured recents time window
//   - restrict to unread notifications
//   - choose one representative per session by severity then recency
//   - return the most recent recentsDatasetLimit representatives
func (o *ViewOrchestrator) buildRecentsPerSession(opts ViewOptions, notifs []domain.Notification) []domain.Notification {
	if len(notifs) == 0 {
		return nil
	}

	activeOnly := filterActiveNotifications(notifs)
	if len(activeOnly) == 0 {
		return nil
	}

	windowFiltered := domain.FilterByTimeDuration(activeOnly, getRecentsTimeWindow())
	if len(windowFiltered) == 0 {
		return nil
	}

	unreadOnly := filterUnreadNotifications(windowFiltered)
	if len(unreadOnly) == 0 {
		return nil
	}

	result := selectSessionRepresentatives(unreadOnly)
	sortRepresentativesByRecency(result)

	if len(result) > recentsDatasetLimit {
		return result[:recentsDatasetLimit]
	}
	return result
}

func (o *ViewOrchestrator) buildSessionsPerSession(opts ViewOptions, notifs []domain.Notification) []domain.Notification {
	if len(notifs) == 0 {
		return nil
	}

	activeOnly := filterActiveNotifications(notifs)
	if len(activeOnly) == 0 {
		return nil
	}

	result := selectSessionRepresentatives(activeOnly)
	sortRepresentativesByRecency(result)
	return result
}

func filterActiveNotifications(notifs []domain.Notification) []domain.Notification {
	activeOnly := make([]domain.Notification, 0, len(notifs))
	for _, n := range notifs {
		if n.State == "" || n.State == domain.StateActive {
			activeOnly = append(activeOnly, n)
		}
	}
	return activeOnly
}

func filterUnreadNotifications(notifs []domain.Notification) []domain.Notification {
	unreadOnly := make([]domain.Notification, 0, len(notifs))
	for _, n := range notifs {
		if n.ReadTimestamp == "" {
			unreadOnly = append(unreadOnly, n)
		}
	}
	return unreadOnly
}

func selectSessionRepresentatives(notifs []domain.Notification) []domain.Notification {
	sessionBest := make(map[string]domain.Notification)
	for _, n := range notifs {
		current, exists := sessionBest[n.Session]
		if !exists || isBetterDomainRepresentative(n, current) {
			sessionBest[n.Session] = n
		}
	}

	result := make([]domain.Notification, 0, len(sessionBest))
	for _, n := range sessionBest {
		result = append(result, n)
	}
	return result
}

func sortRepresentativesByRecency(notifs []domain.Notification) {
	sort.SliceStable(notifs, func(i, j int) bool {
		it, errI := time.Parse(time.RFC3339, notifs[i].Timestamp)
		jt, errJ := time.Parse(time.RFC3339, notifs[j].Timestamp)
		if errI != nil || errJ != nil {
			return false
		}
		return it.After(jt)
	})
}

// severityRankDomain mirrors the TUI severityRank helper but works on the
// domain.NotificationLevel type.
func severityRankDomain(level domain.NotificationLevel) int {
	switch string(level) {
	case "error":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// isBetterDomainRepresentative compares two domain notifications using the
// same rules as the TUI implementation: higher severity first, then recency.
func isBetterDomainRepresentative(candidate, current domain.Notification) bool {
	cr := severityRankDomain(candidate.Level)
	pr := severityRankDomain(current.Level)
	if cr != pr {
		return cr > pr
	}

	ct, errC := time.Parse(time.RFC3339, candidate.Timestamp)
	pt, errP := time.Parse(time.RFC3339, current.Timestamp)
	if errC != nil || errP != nil {
		return false
	}

	return ct.After(pt)
}
