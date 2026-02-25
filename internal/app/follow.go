package app

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
)

// FollowClient defines dependencies for fetching notifications.
type FollowClient interface {
	ListNotifications(state, level, session, window, pane, olderThan, newerThan, readFilter string) string
}

// FollowOptions holds all parameters for follow behavior.
type FollowOptions struct {
	Client   FollowClient
	State    string
	Level    string
	Pane     string
	Session  string
	Window   string
	Interval time.Duration
	Output   io.Writer
	TickChan <-chan time.Time
	ListFunc func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string
}

// FollowUseCase coordinates follow behavior.
type FollowUseCase struct{}

// NewFollowUseCase creates a follow use-case.
func NewFollowUseCase() *FollowUseCase {
	return &FollowUseCase{}
}

// Execute starts monitoring notifications until interruption/cancellation.
func (u *FollowUseCase) Execute(ctx context.Context, opts FollowOptions) error {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.Interval <= 0 {
		opts.Interval = time.Second
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	_, _ = fmt.Fprint(opts.Output, "\033[2J\033[H")
	colors.Info("Monitoring notifications (Ctrl+C to stop)...")
	_, _ = fmt.Fprintln(opts.Output)

	seen := make(map[int]bool)
	tickChan, cleanupTicker := setupFollowTickChan(opts)
	defer cleanupTicker()

	for {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-sigChan:
			_, _ = fmt.Fprintf(opts.Output, "\nReceived signal %v, stopping...\n", sig)
			return nil
		case <-tickChan:
			u.handleFollowTick(opts, seen)
		}
	}
}

func setupFollowTickChan(opts FollowOptions) (<-chan time.Time, func()) {
	if opts.TickChan != nil {
		return opts.TickChan, func() {}
	}

	ticker := time.NewTicker(opts.Interval)
	return ticker.C, ticker.Stop
}

func (u *FollowUseCase) handleFollowTick(opts FollowOptions, seen map[int]bool) {
	lines := u.fetchFollowNotifications(opts)
	if lines == "" {
		return
	}

	notifications := parseFollowNotifications(lines)
	printNewFollowNotifications(notifications, seen, opts.Output)
}

func (u *FollowUseCase) fetchFollowNotifications(opts FollowOptions) string {
	if opts.Client != nil {
		return opts.Client.ListNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, "", "", "")
	}
	if opts.ListFunc != nil {
		return opts.ListFunc(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, "", "", "")
	}

	return ""
}

func parseFollowNotifications(lines string) []notification.Notification {
	notifications := make([]notification.Notification, 0)
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

func printNewFollowNotifications(notifications []notification.Notification, seen map[int]bool, output io.Writer) {
	for _, notif := range notifications {
		if seen[notif.ID] {
			continue
		}

		domainNotif := notification.ToDomainUnsafe(notif)
		printFollowNotification(*domainNotif, output)
		seen[notif.ID] = true
	}
}

func printFollowNotification(n domain.Notification, w io.Writer) {
	timeStr := formatFollowTimestamp(n.Timestamp)
	msg := fmt.Sprintf("[%s] [%s] %s", timeStr, n.Level.String(), n.Message)
	color := followColorForLevel(n.Level.String())
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

func formatFollowTimestamp(ts string) string {
	if len(ts) >= 20 && ts[10] == 'T' && ts[len(ts)-1] == 'Z' {
		return ts[:10] + " " + ts[11:19]
	}
	return ts
}

func followColorForLevel(level string) string {
	switch level {
	case "error":
		return colors.Red
	case "warning":
		return colors.Yellow
	default:
		return ""
	}
}
