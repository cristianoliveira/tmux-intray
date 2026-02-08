package search

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/assert"
)

// Test notification used across tests
var testNotification = notification.Notification{
	ID:            1,
	Timestamp:     "2024-01-01T12:00:00Z",
	State:         "active",
	Session:       "$1",
	SessionName:   "my-session",
	Window:        "@0",
	Pane:          "%0",
	Message:       "error: failed to connect to database",
	PaneCreated:   "2024-01-01T11:00:00Z",
	Level:         "error",
	ReadTimestamp: "",
}

var testNotificationRead = notification.Notification{
	ID:            2,
	Timestamp:     "2024-01-01T12:00:00Z",
	State:         "active",
	Session:       "$2",
	Window:        "@1",
	Pane:          "%1",
	Message:       "warning: connection slow",
	PaneCreated:   "2024-01-01T11:00:00Z",
	Level:         "warning",
	ReadTimestamp: "2024-01-01T13:00:00Z",
}

// TestDefaultOptions verifies default option values.
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.False(t, opts.CaseInsensitive, "default should be case-sensitive")
	assert.Equal(t, []string{"message", "session_name", "session", "window", "pane"}, opts.Fields,
		"default fields should include message, session_name, session, window, pane")
}

// TestOptions verifies option application.
func TestOptions(t *testing.T) {
	opts := DefaultOptions()
	WithCaseInsensitive(true)(&opts)
	WithFields([]string{"message", "level"})(&opts)

	assert.True(t, opts.CaseInsensitive)
	assert.Equal(t, []string{"message", "level"}, opts.Fields)
}

// TestSubstringProvider tests substring-based search.
func TestSubstringProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		query    string
		expected bool
	}{
		{
			name:     "empty query matches all",
			provider: NewSubstringProvider(),
			query:    "",
			expected: true,
		},
		{
			name:     "substring in message",
			provider: NewSubstringProvider(),
			query:    "database",
			expected: true,
		},
		{
			name:     "substring not found",
			provider: NewSubstringProvider(),
			query:    "network",
			expected: false,
		},
		{
			name:     "case-sensitive match",
			provider: NewSubstringProvider(),
			query:    "Error",
			expected: false,
		},
		{
			name:     "case-insensitive match",
			provider: NewSubstringProvider(WithCaseInsensitive(true)),
			query:    "Error",
			expected: true,
		},
		{
			name:     "match in session",
			provider: NewSubstringProvider(),
			query:    "$1",
			expected: true,
		},
		{
			name:     "match in pane",
			provider: NewSubstringProvider(),
			query:    "%0",
			expected: true,
		},
		{
			name:     "match in window",
			provider: NewSubstringProvider(),
			query:    "@0",
			expected: true,
		},
		{
			name:     "custom fields only message",
			provider: NewSubstringProvider(WithFields([]string{"message"})),
			query:    "$1",
			expected: false,
		},
		{
			name:     "custom fields include level",
			provider: NewSubstringProvider(WithFields([]string{"message", "level"})),
			query:    "error",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.Match(testNotification, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSubstringProviderName verifies provider name.
func TestSubstringProviderName(t *testing.T) {
	provider := NewSubstringProvider()
	assert.Equal(t, "substring", provider.Name())
}

// TestRegexProvider tests regex-based search.
func TestRegexProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		query    string
		expected bool
	}{
		{
			name:     "empty query matches all",
			provider: NewRegexProvider(),
			query:    "",
			expected: true,
		},
		{
			name:     "simple regex match",
			provider: NewRegexProvider(),
			query:    "database",
			expected: true,
		},
		{
			name:     "regex pattern match",
			provider: NewRegexProvider(),
			query:    "error.*database",
			expected: true,
		},
		{
			name:     "regex not found",
			provider: NewRegexProvider(),
			query:    "network",
			expected: false,
		},
		{
			name:     "case-sensitive regex with uppercase",
			provider: NewRegexProvider(),
			query:    "Error",
			expected: false,
		},
		{
			name:     "case-insensitive regex with (?i) flag",
			provider: NewRegexProvider(),
			query:    "(?i)[Ee]rror",
			expected: true,
		},
		{
			name:     "case-insensitive option",
			provider: NewRegexProvider(WithCaseInsensitive(true)),
			query:    "Error",
			expected: true,
		},
		{
			name:     "invalid regex returns false",
			provider: NewRegexProvider(),
			query:    "[invalid",
			expected: false,
		},
		{
			name:     "regex in session",
			provider: NewRegexProvider(),
			query:    `\$1`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.Match(testNotification, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRegexProviderName verifies provider name.
func TestRegexProviderName(t *testing.T) {
	provider := NewRegexProvider()
	assert.Equal(t, "regex", provider.Name())
}

// TestTokenProvider tests token-based search.
func TestTokenProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		notif    notification.Notification
		query    string
		expected bool
	}{
		{
			name:     "empty query matches all",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "",
			expected: true,
		},
		{
			name:     "single token in message",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "database",
			expected: true,
		},
		{
			name:     "multiple tokens all match",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "error database",
			expected: true,
		},
		{
			name:     "one token doesn't match",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "error network",
			expected: false,
		},
		{
			name:     "case-sensitive token",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "Error",
			expected: false,
		},
		{
			name:     "case-insensitive token",
			provider: NewTokenProvider(WithCaseInsensitive(true)),
			notif:    testNotification,
			query:    "Error",
			expected: true,
		},
		{
			name:     "read filter matches read notification",
			provider: NewTokenProvider(),
			notif:    testNotificationRead,
			query:    "read",
			expected: true,
		},
		{
			name:     "read filter doesn't match unread",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "read",
			expected: false,
		},
		{
			name:     "unread filter matches unread notification",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "unread",
			expected: true,
		},
		{
			name:     "unread filter doesn't match read",
			provider: NewTokenProvider(),
			notif:    testNotificationRead,
			query:    "unread",
			expected: false,
		},
		{
			name:     "both read and unread cancel out",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "read unread",
			expected: true,
		},
		{
			name:     "read filter with text token",
			provider: NewTokenProvider(),
			notif:    testNotificationRead,
			query:    "read slow",
			expected: true,
		},
		{
			name:     "read filter with unmatched text token",
			provider: NewTokenProvider(),
			notif:    testNotificationRead,
			query:    "read database",
			expected: false,
		},
		{
			name:     "tokens match across different fields",
			provider: NewTokenProvider(),
			notif:    testNotification,
			query:    "%0 error",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.Match(tt.notif, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTokenProviderName verifies provider name.
func TestTokenProviderName(t *testing.T) {
	provider := NewTokenProvider()
	assert.Equal(t, "token", provider.Name())
}

// TestProviderEdgeCases tests edge cases for all providers.
func TestProviderEdgeCases(t *testing.T) {
	providers := []struct {
		name string
		p    Provider
	}{
		{"substring", NewSubstringProvider()},
		{"regex", NewRegexProvider()},
		{"token", NewTokenProvider()},
	}

	for _, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			// Test with notification with empty fields
			emptyNotif := notification.Notification{
				ID:        1,
				Timestamp: "2024-01-01T12:00:00Z",
				State:     "",
				Session:   "",
				Window:    "",
				Pane:      "",
				Message:   "",
				Level:     "",
			}

			// Empty query should always match
			assert.True(t, provider.p.Match(emptyNotif, ""), "empty query should match")

			// Non-empty query on empty notification should not match
			assert.False(t, provider.p.Match(emptyNotif, "test"), "query should not match empty notification")
		})
	}
}

// TestApplyOptions verifies applyOptions function.
func TestApplyOptions(t *testing.T) {
	opts := applyOptions([]Option{
		WithCaseInsensitive(true),
		WithFields([]string{"message"}),
	})

	assert.True(t, opts.CaseInsensitive)
	assert.Equal(t, []string{"message"}, opts.Fields)
}

// BenchmarkSubstringProvider benchmarks substring provider.
func BenchmarkSubstringProvider(b *testing.B) {
	provider := NewSubstringProvider()
	query := "database"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.Match(testNotification, query)
	}
}

// BenchmarkRegexProvider benchmarks regex provider.
func BenchmarkRegexProvider(b *testing.B) {
	provider := NewRegexProvider()
	query := "database"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.Match(testNotification, query)
	}
}

// BenchmarkTokenProvider benchmarks token provider.
func BenchmarkTokenProvider(b *testing.B) {
	provider := NewTokenProvider()
	query := "error database"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.Match(testNotification, query)
	}
}
