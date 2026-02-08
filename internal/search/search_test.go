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
	assert.Equal(t, []string{"message", "session", "window", "pane"}, opts.Fields,
		"default fields should include message, session, window, pane")
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

// TestNameBasedSearch tests name-based search for sessions, windows, and panes.
func TestNameBasedSearch(t *testing.T) {
	// Create test notifications
	notif1 := notification.Notification{
		ID:        1,
		Session:   "$1",
		Window:    "@0",
		Pane:      "%0",
		Message:   "error: failed to connect",
		Level:     "error",
		State:     "active",
		Timestamp: "2024-01-01T12:00:00Z",
	}

	notif2 := notification.Notification{
		ID:        2,
		Session:   "$2",
		Window:    "@1",
		Pane:      "%1",
		Message:   "warning: slow connection",
		Level:     "warning",
		State:     "active",
		Timestamp: "2024-01-01T12:00:00Z",
	}

	// Create name maps
	sessionNames := map[string]string{
		"$1": "my-work",
		"$2": "personal",
	}
	windowNames := map[string]string{
		"@0": "main",
		"@1": "editor",
	}
	paneNames := map[string]string{
		"%0": "terminal",
		"%1": "vim",
	}

	tests := []struct {
		name        string
		provider    Provider
		notif       notification.Notification
		query       string
		expected    bool
		description string
	}{
		// Substring provider tests
		{
			name:        "substring: session name matches",
			provider:    NewSubstringProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "my-work",
			expected:    true,
			description: "should match by session name",
		},
		{
			name:        "substring: window name matches",
			provider:    NewSubstringProvider(WithWindowNames(windowNames)),
			notif:       notif1,
			query:       "main",
			expected:    true,
			description: "should match by window name",
		},
		{
			name:        "substring: pane name matches",
			provider:    NewSubstringProvider(WithPaneNames(paneNames)),
			notif:       notif1,
			query:       "terminal",
			expected:    true,
			description: "should match by pane name",
		},
		{
			name:        "substring: session ID still works",
			provider:    NewSubstringProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "$1",
			expected:    true,
			description: "should still match by session ID",
		},
		{
			name:        "substring: no name match",
			provider:    NewSubstringProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "nonexistent",
			expected:    false,
			description: "should not match nonexistent name",
		},
		{
			name:        "substring: all name maps",
			provider:    NewSubstringProvider(WithSessionNames(sessionNames), WithWindowNames(windowNames), WithPaneNames(paneNames)),
			notif:       notif1,
			query:       "my-work",
			expected:    true,
			description: "should match with all name maps provided",
		},
		{
			name:        "substring: different notification",
			provider:    NewSubstringProvider(WithSessionNames(sessionNames), WithWindowNames(windowNames), WithPaneNames(paneNames)),
			notif:       notif2,
			query:       "personal",
			expected:    true,
			description: "should match second notification by session name",
		},

		// Regex provider tests
		{
			name:        "regex: session name matches",
			provider:    NewRegexProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "my-work",
			expected:    true,
			description: "should match by session name",
		},
		{
			name:        "regex: window name pattern",
			provider:    NewRegexProvider(WithWindowNames(windowNames)),
			notif:       notif1,
			query:       "ma..",
			expected:    true,
			description: "should match window name with pattern",
		},
		{
			name:        "regex: session ID still works",
			provider:    NewRegexProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       `\$1`,
			expected:    true,
			description: "should still match by session ID",
		},
		{
			name:        "regex: no name match",
			provider:    NewRegexProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "xyz",
			expected:    false,
			description: "should not match nonexistent name",
		},

		// Token provider tests
		{
			name:        "token: session name matches",
			provider:    NewTokenProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "my-work",
			expected:    true,
			description: "should match by session name",
		},
		{
			name:        "token: window name matches",
			provider:    NewTokenProvider(WithWindowNames(windowNames)),
			notif:       notif1,
			query:       "main",
			expected:    true,
			description: "should match by window name",
		},
		{
			name:        "token: pane name matches",
			provider:    NewTokenProvider(WithPaneNames(paneNames)),
			notif:       notif1,
			query:       "terminal",
			expected:    true,
			description: "should match by pane name",
		},
		{
			name:        "token: session ID still works",
			provider:    NewTokenProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "$1",
			expected:    true,
			description: "should still match by session ID",
		},
		{
			name:        "token: multiple tokens with name",
			provider:    NewTokenProvider(WithSessionNames(sessionNames), WithCaseInsensitive(true)),
			notif:       notif1,
			query:       "my-work error",
			expected:    true,
			description: "should match multiple tokens including name",
		},
		{
			name:        "token: case-insensitive name",
			provider:    NewTokenProvider(WithSessionNames(sessionNames), WithCaseInsensitive(true)),
			notif:       notif1,
			query:       "MY-WORK",
			expected:    true,
			description: "should match name case-insensitively",
		},
		{
			name:        "token: no name match",
			provider:    NewTokenProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "nonexistent",
			expected:    false,
			description: "should not match nonexistent name",
		},

		// Case sensitivity tests
		{
			name:        "substring: case-sensitive name",
			provider:    NewSubstringProvider(WithSessionNames(sessionNames)),
			notif:       notif1,
			query:       "MY-WORK",
			expected:    false,
			description: "should not match case-sensitive",
		},
		{
			name:        "substring: case-insensitive name",
			provider:    NewSubstringProvider(WithSessionNames(sessionNames), WithCaseInsensitive(true)),
			notif:       notif1,
			query:       "MY-WORK",
			expected:    true,
			description: "should match name case-insensitively",
		},

		// Nil name maps (backward compatibility)
		{
			name:        "substring: nil name maps",
			provider:    NewSubstringProvider(WithSessionNames(nil)),
			notif:       notif1,
			query:       "$1",
			expected:    true,
			description: "should work with nil name maps (backward compatible)",
		},
		{
			name:        "token: nil name maps",
			provider:    NewTokenProvider(WithSessionNames(nil)),
			notif:       notif1,
			query:       "$1",
			expected:    true,
			description: "should work with nil name maps (backward compatible)",
		},
		{
			name:        "regex: nil name maps",
			provider:    NewRegexProvider(WithSessionNames(nil)),
			notif:       notif1,
			query:       `\$1`,
			expected:    true,
			description: "should work with nil name maps (backward compatible)",
		},

		// Empty ID tests
		{
			name:        "substring: empty session ID",
			provider:    NewSubstringProvider(WithSessionNames(sessionNames)),
			notif:       notification.Notification{ID: 1, Session: "", Message: "test"},
			query:       "my-work",
			expected:    false,
			description: "should not match empty session ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.Match(tt.notif, tt.query)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

// TestNameBasedSearchCrossField tests searching across multiple fields with names.
func TestNameBasedSearchCrossField(t *testing.T) {
	notif := notification.Notification{
		ID:        1,
		Session:   "$1",
		Window:    "@0",
		Pane:      "%0",
		Message:   "error in my-work session",
		Level:     "error",
		State:     "active",
		Timestamp: "2024-01-01T12:00:00Z",
	}

	sessionNames := map[string]string{
		"$1": "my-work",
	}
	windowNames := map[string]string{
		"@0": "main",
	}
	paneNames := map[string]string{
		"%0": "terminal",
	}

	tests := []struct {
		name     string
		provider Provider
		query    string
		expected bool
	}{
		{
			name:     "substring: match in name across fields",
			provider: NewSubstringProvider(WithSessionNames(sessionNames), WithWindowNames(windowNames), WithPaneNames(paneNames)),
			query:    "my-work",
			expected: true,
		},
		{
			name:     "token: match name across fields",
			provider: NewTokenProvider(WithSessionNames(sessionNames), WithWindowNames(windowNames), WithPaneNames(paneNames)),
			query:    "my-work",
			expected: true,
		},
		{
			name:     "regex: match name across fields",
			provider: NewRegexProvider(WithSessionNames(sessionNames), WithWindowNames(windowNames), WithPaneNames(paneNames)),
			query:    "my-work",
			expected: true,
		},
		{
			name:     "token: match name and message",
			provider: NewTokenProvider(WithSessionNames(sessionNames), WithWindowNames(windowNames), WithPaneNames(paneNames)),
			query:    "my-work error",
			expected: true,
		},
		{
			name:     "substring: different notification",
			provider: NewSubstringProvider(WithSessionNames(sessionNames), WithWindowNames(windowNames), WithPaneNames(paneNames)),
			query:    "main",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.Match(notif, tt.query)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
