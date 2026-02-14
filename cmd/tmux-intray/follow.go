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

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/spf13/cobra"
)

type followClient interface {
	ListNotifications(state, level, session, window, pane, olderThan, newerThan, readFilter string) string
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
var listFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
	result, _ := fileStorage.ListNotifications(state, level, session, window, pane, olderThan, newerThan, readFilter)
	return result
}

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
func printNotification(n notification.Notification, w io.Writer) {
	timeStr := formatTimestamp(n.Timestamp)
	msg := fmt.Sprintf("[%s] [%s] %s", timeStr, n.Level, n.Message)
	color := colorForLevel(n.Level)
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

	// Determine tick channel
	var tickChan <-chan time.Time
	var ticker *time.Ticker
	if opts.TickChan != nil {
		tickChan = opts.TickChan
	} else {
		ticker = time.NewTicker(opts.Interval)
		tickChan = ticker.C
		defer ticker.Stop()
	}

	// Helper function to fetch notifications
	fetchNotifications := func() string {
		if opts.Client != nil {
			return opts.Client.ListNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, "", "", "")
		}
		return listFunc(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, "", "", "")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-sigChan:
			_, _ = fmt.Fprintf(opts.Output, "\nReceived signal %v, stopping...\n", sig)
			return nil
		case <-tickChan:
			// Fetch notifications with filters
			lines := fetchNotifications()
			if lines == "" {
				continue
			}
			// Parse lines
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
			// Print new notifications
			for _, notif := range notifications {
				if !seen[notif.ID] {
					printNotification(notif, opts.Output)
					seen[notif.ID] = true
				}
			}
		}
	}
}

// defaultFollowClient is the default implementation using listFunc.
type defaultFollowClient struct{}

func (d *defaultFollowClient) ListNotifications(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
	return listFunc(state, level, session, window, pane, olderThan, newerThan, readFilter)
}

// followCmd represents the follow command
var followCmd = NewFollowCmd(&defaultFollowClient{})

func init() {
	cmd.RootCmd.AddCommand(followCmd)
}
