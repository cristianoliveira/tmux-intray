/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/spf13/cobra"
)

type listClient interface {
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
}

const listCommandLong = `List notifications with filters and formats.

USAGE:
    tmux-intray list [OPTIONS]

OPTIONS:
    --tab <tab>          Show special tab view: recents, sessions, all
    --active             Show active notifications (default)
    --dismissed          Show dismissed notifications
    --all                Show all notifications
    --pane <id|title>    Filter notifications by pane ID or pane title
    --level <level>      Filter notifications by level: info, warning, error, critical
    --session <id|name>  Filter notifications by session ID or session name
    --window <id|name>   Filter notifications by window ID or window name
    --ids                Show raw tmux session/window/pane IDs instead of resolved names
    --show-stale         Include notifications whose tmux session/window/pane no longer exists
    --older-than <days>  Show notifications older than N days
    --newer-than <days>  Show notifications newer than N days
    --search <pattern>   Search messages (substring match)
    --regex              Use regex search with --search
    --group-by <field>   Group notifications by field (session, window, pane, level, message)
    --group-count        Show only group counts (requires --group-by)
    --filter <status>    Filter notifications by read status: read, unread
    --format=<format>    Output format: simple (default), legacy, table, compact, json

TAB VIEWS:
    --tab=recents        Show recent unread notifications (max 1 per session, last hour)
    --tab=sessions       Show unique sessions with notifications
    --tab=all            Show all notifications (same as --all)

ORDERING:
    Unread notifications are listed first, then read notifications.
    Relative order remains unchanged within each group.
    -h, --help           Show this help`

// NewListCmd creates the list command with explicit dependencies.
//
//nolint:funlen // Command wiring with flags and handlers is intentionally centralized.
func NewListCmd(client listClient, searchProviderFactory appcore.SearchProviderFactory, displayNamesLoader tmuxDisplayNamesLoader) *cobra.Command {
	if client == nil {
		panic("NewListCmd: client dependency cannot be nil")
	}
	if searchProviderFactory == nil {
		panic("NewListCmd: searchProviderFactory dependency cannot be nil")
	}
	if displayNamesLoader == nil {
		panic("NewListCmd: displayNamesLoader dependency cannot be nil")
	}

	// Create the main list command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List notifications with filters and formats",
		Long:  listCommandLong,
	}

	var listPane string
	var listLevel string
	var listSession string
	var listWindow string
	var listOlderThan int
	var listNewerThan int
	var listSearch string
	var listRegex bool
	var listGroupBy string
	var listGroupCount bool
	var listFormat string
	var listFilter string
	var listTab string
	var listJSON bool
	var listRawIDs bool
	var listShowStale bool

	listCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Handle --json flag
		if listJSON {
			listFormat = "json"
		}

		displayNames := appcore.DisplayNames{}
		if shouldLoadListDisplayNames(listJSON, listRawIDs, listSearch, listSession, listWindow, listPane) {
			displayNames = displayNamesLoader()
		}

		listSession = resolveTmuxFilterValue(listSession, displayNames.Sessions)
		listWindow = resolveTmuxFilterValue(listWindow, displayNames.Windows)
		listPane = resolveTmuxFilterValue(listPane, displayNames.Panes)

		// Handle --tab flag
		if listTab != "" {
			validTabs := []string{"recents", "sessions", "all"}
			if !isValidTab(listTab, validTabs) {
				return fmt.Errorf("invalid --tab value: %s (available: %s)", listTab, strings.Join(validTabs, ", "))
			}

			olderCutoff, newerCutoff := computeCutoffTimestamps(listOlderThan, listNewerThan)

			tabOpts := TabOptions{
				Client:       client,
				Tab:          listTab,
				Format:       listFormat,
				Session:      listSession,
				Level:        listLevel,
				Window:       listWindow,
				Pane:         listPane,
				OlderThan:    olderCutoff,
				NewerThan:    newerCutoff,
				ReadFilter:   listFilter,
				DisplayNames: displayNames,
				RawIDs:       listRawIDs,
				ShowStale:    listShowStale,
			}
			PrintTab(tabOpts)
			return nil
		}

		state := determineListState(cmd)
		olderCutoff, newerCutoff := computeCutoffTimestamps(listOlderThan, listNewerThan)
		if err := validateListOptions(listGroupBy, listFilter); err != nil {
			return err
		}

		opts := FilterOptions{
			Client:         client,
			State:          state,
			Level:          listLevel,
			Session:        listSession,
			Window:         listWindow,
			Pane:           listPane,
			OlderThan:      olderCutoff,
			NewerThan:      newerCutoff,
			Search:         listSearch,
			Regex:          listRegex,
			GroupBy:        listGroupBy,
			GroupCount:     listGroupCount,
			Format:         listFormat,
			ReadFilter:     listFilter,
			SearchProvider: buildListSearchProvider(listSearch, listRegex, displayNames),
			DisplayNames:   displayNames,
			RawIDs:         listRawIDs,
			ShowStale:      listShowStale,
		}
		PrintListTo(opts, cmd.OutOrStdout(), searchProviderFactory)
		return nil
	}

	registerListFlags(listCmd, &listPane, &listLevel, &listSession, &listWindow, &listOlderThan, &listNewerThan, &listSearch, &listRegex, &listGroupBy, &listGroupCount, &listFormat, &listFilter)

	// Add --tab flag
	listCmd.Flags().StringVar(&listTab, "tab", "", "Show special tab view: recents, sessions, all")

	// Add --json flag
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	listCmd.Flags().BoolVar(&listRawIDs, "ids", false, "Show raw tmux session/window/pane IDs instead of resolved names")
	listCmd.Flags().BoolVar(&listShowStale, "show-stale", false, "Include notifications whose tmux session/window/pane no longer exists")

	return listCmd
}

// isValidTab checks if a tab value is valid.
func isValidTab(tab string, validTabs []string) bool {
	for _, t := range validTabs {
		if t == tab {
			return true
		}
	}
	return false
}

func shouldLoadListDisplayNames(listJSON, listRawIDs bool, listSearch, listSession, listWindow, listPane string) bool {
	if listSearch != "" || listSession != "" || listWindow != "" || listPane != "" {
		return true
	}
	return !listJSON && !listRawIDs
}

func resolveTmuxFilterValue(raw string, names map[string]string) string {
	if raw == "" || names == nil {
		return raw
	}
	for id, name := range names {
		if name == raw {
			return id
		}
	}
	return raw
}

func buildListSearchProvider(query string, regex bool, names appcore.DisplayNames) search.Provider {
	if query == "" {
		return nil
	}

	opts := []search.Option{
		search.WithCaseInsensitive(false),
		search.WithSessionNames(names.Sessions),
		search.WithWindowNames(names.Windows),
		search.WithPaneNames(names.Panes),
	}
	if regex {
		return search.NewRegexProvider(opts...)
	}
	return search.NewSubstringProvider(opts...)
}

// registerListFlags registers all flags for the list command.
func registerListFlags(cmd *cobra.Command, listPane, listLevel, listSession, listWindow *string, listOlderThan, listNewerThan *int, listSearch *string, listRegex *bool, listGroupBy *string, listGroupCount *bool, listFormat, listFilter *string) {
	cmd.Flags().Bool("active", false, "Show active notifications (default)")
	cmd.Flags().Bool("dismissed", false, "Show dismissed notifications")
	cmd.Flags().Bool("all", false, "Show all notifications")
	cmd.Flags().StringVar(listPane, "pane", "", "Filter notifications by pane ID or pane title")
	cmd.Flags().StringVar(listLevel, "level", "", "Filter notifications by level: info, warning, error, critical")
	cmd.Flags().StringVar(listSession, "session", "", "Filter notifications by session ID or session name")
	cmd.Flags().StringVar(listWindow, "window", "", "Filter notifications by window ID or window name")
	cmd.Flags().IntVar(listOlderThan, "older-than", 0, "Show notifications older than N days")
	cmd.Flags().IntVar(listNewerThan, "newer-than", 0, "Show notifications newer than N days")
	cmd.Flags().StringVar(listSearch, "search", "", "Search messages (substring match)")
	cmd.Flags().BoolVar(listRegex, "regex", false, "Use regex search with --search")
	cmd.Flags().StringVar(listGroupBy, "group-by", "", "Group notifications by field (session, window, pane, level, message)")
	cmd.Flags().BoolVar(listGroupCount, "group-count", false, "Show only group counts (requires --group-by)")
	cmd.Flags().StringVar(listFormat, "format", "simple", "Output format: simple (default), legacy, table, compact, json")
	cmd.Flags().StringVar(listFilter, "filter", "", "Filter notifications by read status: read, unread")
}

// determineListState determines the state filter based on flags.
func determineListState(cmd *cobra.Command) string {
	state := "active"
	if cmd.Flag("dismissed").Changed {
		state = "dismissed"
	}
	if cmd.Flag("all").Changed {
		state = "all"
	}
	return state
}

// listNow is the time source used for list timestamp filters.
var listNow = time.Now

// computeCutoffTimestamps computes timestamp cutoffs for older/newer-than filters.
func computeCutoffTimestamps(olderThan, newerThan int) (olderCutoff, newerCutoff string) {
	base := listNow().UTC().Truncate(time.Second)
	if olderThan > 0 {
		olderCutoff = base.AddDate(0, 0, -olderThan).Format("2006-01-02T15:04:05Z")
	}
	if newerThan > 0 {
		newerCutoff = base.AddDate(0, 0, -newerThan).Format("2006-01-02T15:04:05Z")
	}
	return
}

// validateListOptions validates list command options.
func validateListOptions(groupBy, filter string) error {
	// Validate group-by field
	if groupBy != "" && groupBy != "session" && groupBy != "window" && groupBy != "pane" && groupBy != "level" && groupBy != "message" {
		return fmt.Errorf("invalid group-by field: %s (must be session, window, pane, level, message)", groupBy)
	}

	// Validate read filter
	if filter != "" && filter != "read" && filter != "unread" {
		return fmt.Errorf("invalid filter value: %s (must be read or unread)", filter)
	}

	return nil
}

// FilterOptions holds all filter parameters for listing notifications.
type FilterOptions = appcore.ListOptions

// PrintList prints notifications according to the provided filter options.
func PrintList(opts FilterOptions) {
	PrintListTo(opts, os.Stdout, defaultListSearchProvider)
}

// PrintListTo prints notifications to the provided writer using the injected search provider factory.
func PrintListTo(opts FilterOptions, w io.Writer, searchProviderFactory appcore.SearchProviderFactory) {
	if opts.Client == nil {
		_, _ = fmt.Fprintln(w, "list: missing client")
		return
	}

	useCase := appcore.NewListUseCase(opts.Client, searchProviderFactory)
	useCase.Execute(appcore.ListOptions(opts), w)
}

// orderUnreadFirst places unread notifications before read notifications.
// It keeps the existing relative order within each bucket (stable).
func orderUnreadFirst(notifs []*domain.Notification) []*domain.Notification {
	return appcore.OrderUnreadFirst(notifs)
}
