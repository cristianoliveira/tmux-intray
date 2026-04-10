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

	// 1) Active-only subset
	activeOnly := make([]domain.Notification, 0, len(notifs))
	for _, n := range notifs {
		if n.State == "" || n.State == domain.StateActive {
			activeOnly = append(activeOnly, n)
		}
	}
	if len(activeOnly) == 0 {
		return nil
	}

	// 2) Apply time window
	window := getRecentsTimeWindow()
	activeOnly = domain.FilterByTimeDuration(activeOnly, window)
	if len(activeOnly) == 0 {
		return nil
	}

	// 3) Unread only
	unreadOnly := make([]domain.Notification, 0, len(activeOnly))
	for _, n := range activeOnly {
		if n.ReadTimestamp == "" {
			unreadOnly = append(unreadOnly, n)
		}
	}
	if len(unreadOnly) == 0 {
		return nil
	}

	// 4) One representative per session: choose best by severity, then recency
	sessionBest := make(map[string]domain.Notification)
	for _, n := range unreadOnly {
		key := n.Session
		if current, ok := sessionBest[key]; !ok {
			// first notification for this session
			sessionBest[key] = n
		} else if isBetterDomainRepresentative(n, current) {
			// better candidate for this session
			sessionBest[key] = n
		}
	}

	// 5) Re-sort representatives by recency (desc) and severity for ties
	result := make([]domain.Notification, 0, len(sessionBest))
	for _, n := range sessionBest {
		result = append(result, n)
	}

	// Order sessions by most recent activity first, matching the TUI tests:
	// newer representative timestamps should come first regardless of severity.
	sort.SliceStable(result, func(i, j int) bool {
		it, errI := time.Parse(time.RFC3339, result[i].Timestamp)
		jt, errJ := time.Parse(time.RFC3339, result[j].Timestamp)
		if errI != nil || errJ != nil {
			return false
		}
		return it.After(jt)
	})

	// 6) Limit to recentsDatasetLimit
	if len(result) > recentsDatasetLimit {
		result = result[:recentsDatasetLimit]
	}

	return result
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
