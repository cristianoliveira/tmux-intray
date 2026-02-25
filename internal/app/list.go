package app

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/dedupconfig"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

// ListClient defines dependencies required to list notifications.
type ListClient interface {
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
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
}

// ListUseCase coordinates list notifications behavior.
type ListUseCase struct {
	client ListClient
}

// NewListUseCase creates a new list use-case.
func NewListUseCase(client ListClient) *ListUseCase {
	if client == nil {
		panic("NewListUseCase: client dependency cannot be nil")
	}
	return &ListUseCase{client: client}
}

// Execute prints notifications according to the provided options.
func (u *ListUseCase) Execute(opts ListOptions, w io.Writer) {
	lines, err := u.fetchNotifications(opts)
	if err != nil {
		_, _ = fmt.Fprintf(w, "list: failed to list notifications: %v\n", err)
		return
	}

	if lines == "" {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colors.Blue, "No notifications found", colors.Reset)
		return
	}

	searchProvider := u.getSearchProvider(opts)
	notifications := parseAndFilterNotifications(lines, searchProvider, opts.Search)
	if len(notifications) == 0 {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colors.Blue, "No notifications found", colors.Reset)
		return
	}

	notifications = OrderUnreadFirst(notifications)
	printNotifications(notifications, opts, w)
}

func (u *ListUseCase) fetchNotifications(opts ListOptions) (string, error) {
	if opts.Client != nil {
		return opts.Client.ListNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter)
	}
	return u.client.ListNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter)
}

func (u *ListUseCase) getSearchProvider(opts ListOptions) search.Provider {
	if opts.SearchProvider != nil {
		return opts.SearchProvider
	}
	if opts.Search == "" {
		return nil
	}

	client := tmux.NewDefaultClient()
	sessionNames, _ := client.ListSessions()
	if sessionNames == nil {
		sessionNames = make(map[string]string)
	}
	windowNames, _ := client.ListWindows()
	if windowNames == nil {
		windowNames = make(map[string]string)
	}
	paneNames, _ := client.ListPanes()
	if paneNames == nil {
		paneNames = make(map[string]string)
	}

	if opts.Regex {
		return search.NewRegexProvider(
			search.WithCaseInsensitive(false),
			search.WithSessionNames(sessionNames),
			search.WithWindowNames(windowNames),
			search.WithPaneNames(paneNames),
		)
	}

	return search.NewSubstringProvider(
		search.WithCaseInsensitive(false),
		search.WithSessionNames(sessionNames),
		search.WithWindowNames(windowNames),
		search.WithPaneNames(paneNames),
	)
}

func parseAndFilterNotifications(lines string, searchProvider search.Provider, searchQuery string) []*domain.Notification {
	var notifications []*domain.Notification
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := notification.ParseNotification(line)
		if err != nil {
			continue
		}
		if searchProvider != nil && !searchProvider.Match(notif, searchQuery) {
			continue
		}
		notifications = append(notifications, notification.ToDomainUnsafe(notif))
	}
	return notifications
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

		formatter := format.GetFormatter(opts.Format, opts.GroupCount)
		if err := formatter.FormatGroups(groupResult, w); err != nil {
			_, _ = fmt.Fprintf(w, "list: formatting error: %v\n", err)
		}
		return
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
