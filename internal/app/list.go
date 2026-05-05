package app

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/dedupconfig"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/search"
)

var errTypedListUnsupported = errors.New("typed notification listing unsupported")

// ListClient defines dependencies required to list notifications.
type ListClient interface {
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
}

// DomainListClient lists notifications as domain values so internal flow stays typed.
type DomainListClient interface {
	ListDomainNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) ([]*domain.Notification, error)
}

// ListOptions holds all filter parameters for listing notifications.
type ListOptions struct {
	Client         ListClient
	State          string
	Level          string
	Session        string
	Window         string
	Pane           string
	OlderThan      string
	NewerThan      string
	Search         string
	Regex          bool
	GroupBy        string
	GroupCount     bool
	Format         string
	SearchProvider search.Provider
	ReadFilter     string
	DisplayNames   DisplayNames
	RawIDs         bool
	ShowStale      bool
}

// SearchProviderFactory builds a search provider for list behavior.
type SearchProviderFactory func(regex bool) search.Provider

// ListUseCase coordinates list notifications behavior.
type ListUseCase struct {
	client                ListClient
	searchProviderFactory SearchProviderFactory
}

// NewListUseCase creates a new list use-case.
func NewListUseCase(client ListClient, searchProviderFactory SearchProviderFactory) *ListUseCase {
	if client == nil {
		panic("NewListUseCase: client dependency cannot be nil")
	}
	return &ListUseCase{client: client, searchProviderFactory: searchProviderFactory}
}

// Execute prints notifications according to the provided options.
func (u *ListUseCase) Execute(opts ListOptions, w io.Writer) {
	searchProvider := u.getSearchProvider(opts)
	notifications, err := u.fetchNotifications(opts, searchProvider)
	if err != nil {
		_, _ = fmt.Fprintf(w, "list: failed to list notifications: %v\n", err)
		return
	}

	if len(notifications) == 0 {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colors.Blue, "No notifications found", colors.Reset)
		return
	}

	notifications = filterStaleNotifications(notifications, opts)
	if len(notifications) == 0 {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colors.Blue, "No notifications found", colors.Reset)
		return
	}

	notifications = OrderUnreadFirst(notifications)
	printNotifications(notifications, opts, w)
}

func shouldResolveDisplayNames(opts ListOptions) bool {
	if opts.RawIDs || opts.Format == "json" {
		return false
	}
	return opts.Format == "simple" || opts.GroupBy != ""
}

func (u *ListUseCase) fetchNotifications(opts ListOptions, searchProvider search.Provider) ([]*domain.Notification, error) {
	client := u.client
	if opts.Client != nil {
		client = opts.Client
	}

	if typedClient, ok := client.(DomainListClient); ok {
		notifications, err := typedClient.ListDomainNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter)
		if err == nil {
			return filterNotificationsBySearch(notifications, searchProvider, opts.Search), nil
		}
		if !errors.Is(err, errTypedListUnsupported) {
			return nil, err
		}
	}

	lines, err := client.ListNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter)
	if err != nil || lines == "" {
		return nil, err
	}
	return parseAndFilterNotifications(lines, searchProvider, opts.Search), nil
}

func (u *ListUseCase) getSearchProvider(opts ListOptions) search.Provider {
	if opts.SearchProvider != nil {
		return opts.SearchProvider
	}
	if opts.Search == "" || u.searchProviderFactory == nil {
		return nil
	}
	return u.searchProviderFactory(opts.Regex)
}

func parseAndFilterNotifications(lines string, searchProvider search.Provider, searchQuery string) []*domain.Notification {
	var notifications []*domain.Notification
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := domain.ParseNotificationLine(line)
		if err != nil {
			continue
		}
		notifications = append(notifications, &notif)
	}
	return filterNotificationsBySearch(notifications, searchProvider, searchQuery)
}

func filterNotificationsBySearch(notifications []*domain.Notification, searchProvider search.Provider, searchQuery string) []*domain.Notification {
	if searchProvider == nil {
		return notifications
	}

	filtered := make([]*domain.Notification, 0, len(notifications))
	for _, notif := range notifications {
		if searchProvider.Match(*notif, searchQuery) {
			filtered = append(filtered, notif)
		}
	}
	return filtered
}

func filterStaleNotifications(notifs []*domain.Notification, opts ListOptions) []*domain.Notification {
	return KeepOnlyResolvableTmuxRows(notifs, format.FormatterType(opts.Format), opts.DisplayNames, opts.RawIDs, opts.ShowStale)
}

func notificationsToValues(notifs []*domain.Notification) []domain.Notification {
	values := make([]domain.Notification, len(notifs))
	for i, n := range notifs {
		values[i] = *n
	}
	return values
}

func printNotifications(notifications []*domain.Notification, opts ListOptions, w io.Writer) {
	if opts.GroupBy != "" {
		notificationValues := notificationsToValues(notifications)
		var groupResult domain.GroupResult
		if opts.GroupBy == domain.GroupByMessage.String() {
			groupResult = domain.GroupNotificationsWithDedup(notificationValues, domain.GroupByMode(opts.GroupBy), dedupconfig.Load())
		} else {
			groupResult = domain.GroupNotifications(notificationValues, domain.GroupByMode(opts.GroupBy))
		}
		if shouldResolveDisplayNames(opts) {
			groupResult = opts.DisplayNames.EnrichGroupResult(groupResult)
		}

		formatter := format.GetFormatter(opts.Format, opts.GroupCount)
		if err := formatter.FormatGroups(groupResult, w); err != nil {
			_, _ = fmt.Fprintf(w, "list: formatting error: %v\n", err)
		}
		return
	}

	if shouldResolveDisplayNames(opts) {
		notifications = opts.DisplayNames.EnrichNotifications(notifications)
	}

	formatter := format.GetFormatter(opts.Format, false)
	if err := formatter.FormatNotifications(notifications, w); err != nil {
		_, _ = fmt.Fprintf(w, "list: formatting error: %v\n", err)
	}
}

// OrderUnreadFirst places unread notifications before read notifications.
func OrderUnreadFirst(notifs []*domain.Notification) []*domain.Notification {
	if len(notifs) == 0 {
		return notifs
	}

	ordered := make([]*domain.Notification, len(notifs))
	copy(ordered, notifs)

	sort.SliceStable(ordered, func(i, j int) bool {
		iUnread := !ordered[i].IsRead()
		jUnread := !ordered[j].IsRead()
		if iUnread == jUnread {
			return false
		}
		return iUnread && !jUnread
	})

	return ordered
}
