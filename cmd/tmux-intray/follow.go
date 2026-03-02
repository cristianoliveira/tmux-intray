/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/spf13/cobra"
)

type followClient interface {
	ListNotifications(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error)
}

var (
	followAll       bool
	followDismissed bool
	followLevel     string
	followPane      string
	followInterval  float64
)

// NewFollowCmd creates the follow command with explicit dependencies.
func NewFollowCmd(client followClient) *cobra.Command {
	if client == nil {
		panic("NewFollowCmd: client dependency cannot be nil")
	}

	cmd := &cobra.Command{
		Use:   "follow",
		Short: "Monitor notifications in real-time",
		Long: `Monitor notifications in real-time.

USAGE:
    tmux-intray follow [OPTIONS]

OPTIONS:
    --all              Show all notifications (not just active)
    --dismissed        Show only dismissed notifications
    --level <level>   Filter by level (error, warning, info)
    --pane <id>       Filter by pane ID
    --interval <secs>  Poll interval (default: 1)
    -h, --help         Show this help`,
		RunE: func(c *cobra.Command, args []string) error {
			// Determine state filter
			state := "active"
			if followAll {
				state = "all"
			} else if followDismissed {
				state = "dismissed"
			}

			opts := FollowOptions{
				Client:   client,
				State:    state,
				Level:    followLevel,
				Pane:     followPane,
				Interval: time.Duration(followInterval * float64(time.Second)),
			}
			ctx := context.Background()
			return Follow(ctx, opts)
		},
	}

	cmd.Flags().BoolVar(&followAll, "all", false, "Show all notifications (not just active)")
	cmd.Flags().BoolVar(&followDismissed, "dismissed", false, "Show only dismissed notifications")
	cmd.Flags().StringVar(&followLevel, "level", "", "Filter by level (error, warning, info)")
	cmd.Flags().StringVar(&followPane, "pane", "", "Filter by pane ID")
	cmd.Flags().Float64Var(&followInterval, "interval", 1.0, "Poll interval in seconds (default: 1)")

	return cmd
}

// FollowOptions holds all parameters for following notifications.
type FollowOptions struct {
	Client   followClient     // client for fetching notifications
	State    string           // "active", "dismissed", "all"
	Level    string           // "error", "warning", "info", ""
	Pane     string           // pane ID filter
	Session  string           // session ID filter
	Window   string           // window ID filter
	Interval time.Duration    // polling interval (default 1 second)
	Output   io.Writer        // where to write notifications (default os.Stdout)
	TickChan <-chan time.Time // optional tick channel for testing (if nil, a ticker is created)
}

// listFunc is the function used to retrieve notifications. Can be changed for testing.
var listFunc func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error)

// formatTimestamp converts ISO timestamp to display format.
func formatTimestamp(ts string) string {
	// Expected format: "2006-01-02T15:04:05Z"
	// Convert to "2006-01-02 15:04:05"
	if len(ts) >= 20 && ts[10] == 'T' && ts[len(ts)-1] == 'Z' {
		return ts[:10] + " " + ts[11:19]
	}
	return ts
}

// colorForLevel returns the appropriate color code for a notification level.
func colorForLevel(level string) string {
	switch level {
	case "error":
		return colors.Red
	case "warning":
		return colors.Yellow
	default:
		return "" // default color
	}
}

// printNotification prints a single notification to the writer with formatting.
func printNotification(n domain.Notification, w io.Writer) {
	timeStr := formatTimestamp(n.Timestamp)
	msg := fmt.Sprintf("[%s] [%s] %s", timeStr, n.Level.String(), n.Message)
	color := colorForLevel(n.Level.String())
	reset := colors.Reset
	if color != "" {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", color, msg, reset)
	} else {
		_, _ = fmt.Fprintln(w, msg)
	}
	if n.Pane != "" {
		_, _ = fmt.Fprintf(w, "  └─ From pane: %s\n", n.Pane)
	}
}

// Follow starts monitoring notifications according to the provided options.
// It runs until interrupted (Ctrl+C) or the context is cancelled.
func Follow(ctx context.Context, opts FollowOptions) error {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.Interval <= 0 {
		opts.Interval = time.Second
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Clear screen (optional) but we can just print header
	_, _ = fmt.Fprint(opts.Output, "\033[2J\033[H") // ANSI clear screen and home cursor
	colors.Info("Monitoring notifications (Ctrl+C to stop)...")
	_, _ = fmt.Fprintln(opts.Output)

	// Map from notification ID to whether we've seen it
	seen := make(map[int]bool)

	tickChan, stopTicker := resolveTickChannel(opts)
	defer stopTicker()

	for {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-sigChan:
			_, _ = fmt.Fprintf(opts.Output, "\nReceived signal %v, stopping...\n", sig)
			return nil
		case <-tickChan:
			err := handleTick(opts, seen)
			if err != nil {
				_, _ = fmt.Fprintf(opts.Output, "follow: failed to list notifications: %v\n", err)
			}
		}
	}
}

func resolveTickChannel(opts FollowOptions) (<-chan time.Time, func()) {
	if opts.TickChan != nil {
		return opts.TickChan, func() {}
	}
	ticker := time.NewTicker(opts.Interval)
	return ticker.C, ticker.Stop
}

func handleTick(opts FollowOptions, seen map[int]bool) error {
	lines, err := fetchFollowNotifications(opts)
	if err != nil {
		return err
	}
	if lines == "" {
		return nil
	}

	notifications := parseNotifications(lines)
	printNewNotifications(notifications, seen, opts.Output)
	return nil
}

func fetchFollowNotifications(opts FollowOptions) (string, error) {
	if opts.Client != nil {
		return opts.Client.ListNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, "", "", "")
	}
	if listFunc == nil {
		return "", fmt.Errorf("follow: missing list client")
	}
	return listFunc(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, "", "", "")
}

func parseNotifications(lines string) []notification.Notification {
	var notifications []notification.Notification
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := notification.ParseNotification(line)
		if err != nil {
			continue
		}
		notifications = append(notifications, notif)
	}
	return notifications
}

func printNewNotifications(notifications []notification.Notification, seen map[int]bool, output io.Writer) {
	for _, notif := range notifications {
		if seen[notif.ID] {
			continue
		}
		domainNotif := notification.ToDomainUnsafe(notif)
		printNotification(*domainNotif, output)
		seen[notif.ID] = true
	}
}
