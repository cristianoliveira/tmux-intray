package views

import (
	"sort"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// Kind describes a high-level notification view orchestration shared across surfaces.
type Kind string

const (
	// KindActiveNotificationTimeline returns all active notifications (read + unread), preserving input ordering.
	KindActiveNotificationTimeline Kind = "active-notification-timeline"

	// KindRecentUnreadTimeline returns recent unread notifications across sessions (no per-session collapse).
	KindRecentUnreadTimeline Kind = "recent-unread-timeline"

	// KindRecentUnreadSessionHighlights returns recent unread highlights (one representative per session).
	KindRecentUnreadSessionHighlights Kind = "recent-unread-session-highlights"

	// KindSessionHistory returns session history (one representative per active session, all-time).
	KindSessionHistory Kind = "session-history"
)

// Options controls how a view is orchestrated.
type Options struct {
	Kind   Kind
	SortBy string
	Order  string
	Limit  int
}

// Result contains orchestrated notifications for a given view.
type Result struct {
	Notifications []domain.Notification
}

// Orchestrator builds high-level notification views on top of the domain layer.
type Orchestrator struct{}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{}
}

// Build orchestrates a notification view based on options and input notifications.
func (o *Orchestrator) Build(opts Options, notifs []domain.Notification) Result {
	switch opts.Kind {
	case KindActiveNotificationTimeline:
		return Result{Notifications: o.buildActiveNotificationTimeline(notifs)}
	case KindRecentUnreadTimeline:
		return Result{Notifications: o.buildRecentUnreadTimeline(opts, notifs)}
	case KindRecentUnreadSessionHighlights:
		return Result{Notifications: o.buildRecentUnreadSessionHighlights(notifs)}
	case KindSessionHistory:
		return Result{Notifications: o.buildSessionHistory(notifs)}
	default:
		return Result{Notifications: notifs}
	}
}

func getRecentsTimeWindow() time.Duration {
	windowStr := config.Get("recents_time_window", "1h")
	duration, err := time.ParseDuration(windowStr)
	if err != nil {
		return time.Hour
	}
	return duration
}

const recentsDatasetLimit = 20

func (o *Orchestrator) buildActiveNotificationTimeline(notifs []domain.Notification) []domain.Notification {
	if len(notifs) == 0 {
		return nil
	}

	return filterActiveNotifications(notifs)
}

func (o *Orchestrator) buildRecentUnreadTimeline(opts Options, notifs []domain.Notification) []domain.Notification {
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

	sorted := sortByOptions(unreadOnly, opts.SortBy, opts.Order)
	if opts.Limit > 0 && len(sorted) > opts.Limit {
		return sorted[:opts.Limit]
	}
	return sorted
}

func (o *Orchestrator) buildRecentUnreadSessionHighlights(notifs []domain.Notification) []domain.Notification {
	result := o.buildRecentUnreadTimeline(Options{SortBy: "timestamp", Order: "desc", Limit: recentsDatasetLimit}, notifs)
	if len(result) == 0 {
		return nil
	}

	result = selectSessionRepresentatives(result)
	sortRepresentativesByRecency(result)

	if len(result) > recentsDatasetLimit {
		return result[:recentsDatasetLimit]
	}
	return result
}

func (o *Orchestrator) buildSessionHistory(notifs []domain.Notification) []domain.Notification {
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

func sortByOptions(notifs []domain.Notification, sortBy, order string) []domain.Notification {
	field, err := domain.ParseSortByField(sortBy)
	if err != nil {
		field = domain.SortByTimestampField
	}

	sortOrder, err := domain.ParseSortOrder(order)
	if err != nil {
		sortOrder = domain.SortOrderDesc
	}

	return domain.SortNotifications(notifs, domain.SortOptions{Field: field, Order: sortOrder})
}

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
