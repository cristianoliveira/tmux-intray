// Package search provides a unified search abstraction for filtering notifications.
// It supports multiple search strategies (substring, regex, token-based) through
// a common Provider interface, eliminating duplicate search logic between CLI and TUI.
package search

import (
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// Provider defines the interface for search providers.
// Implementations can use different strategies (substring, regex, token-based, etc.)
// to match notifications against search queries.
type Provider interface {
	// Match returns true if the notification matches the search query.
	Match(notif notification.Notification, query string) bool

	// Name returns the provider name for identification and debugging.
	Name() string
}

// Options holds configuration options for creating search providers.
type Options struct {
	CaseInsensitive bool     // If true, searches ignore case sensitivity
	Fields          []string // Fields to search in (default: all fields)
}

// DefaultOptions returns the default search options.
func DefaultOptions() Options {
	return Options{
		CaseInsensitive: false,
		Fields:          []string{"message", "session", "window", "pane"},
	}
}

// Option is a function that modifies search options.
type Option func(*Options)

// WithCaseInsensitive sets case-insensitive search.
func WithCaseInsensitive(enabled bool) Option {
	return func(o *Options) {
		o.CaseInsensitive = enabled
	}
}

// WithFields sets the fields to search in.
// Valid fields: "message", "session", "window", "pane", "level", "state".
func WithFields(fields []string) Option {
	return func(o *Options) {
		o.Fields = fields
	}
}

// applyOptions applies the given options to the options struct.
func applyOptions(opts []Option) Options {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
