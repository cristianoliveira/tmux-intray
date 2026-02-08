package search

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/assert"
)

// TestRegexProviderWithEmptyQueryField tests regex with various field configurations.
func TestRegexProviderWithDifferentFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		query    string
		expected bool
	}{
		{
			name:     "only level field",
			fields:   []string{"level"},
			query:    "error",
			expected: true,
		},
		{
			name:     "only message field",
			fields:   []string{"message"},
			query:    "database",
			expected: true,
		},
		{
			name:     "only state field",
			fields:   []string{"state"},
			query:    "active",
			expected: true,
		},
		{
			name:     "level and state",
			fields:   []string{"level", "state"},
			query:    "error",
			expected: true,
		},
		{
			name:     "non-matching field only",
			fields:   []string{"session"},
			query:    "error",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewRegexProvider(WithFields(tt.fields))
			result := provider.Match(testNotification, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTokenProviderFieldFiltering tests token provider with limited fields.
func TestTokenProviderFieldFiltering(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		query    string
		expected bool
	}{
		{
			name:     "only message field with match",
			fields:   []string{"message"},
			query:    "database",
			expected: true,
		},
		{
			name:     "only message field no match",
			fields:   []string{"message"},
			query:    "$1",
			expected: false,
		},
		{
			name:     "only session field with match",
			fields:   []string{"session"},
			query:    "$1",
			expected: true,
		},
		{
			name:     "only session field no match",
			fields:   []string{"session"},
			query:    "database",
			expected: false,
		},
		{
			name:     "multiple fields some match",
			fields:   []string{"message", "session"},
			query:    "$1 database",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewTokenProvider(WithFields(tt.fields))
			result := provider.Match(testNotification, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSubstringProviderFieldConfigurations tests substring provider with field configurations.
func TestSubstringProviderFieldConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		query    string
		expected bool
	}{
		{
			name:     "single field match",
			fields:   []string{"message"},
			query:    "database",
			expected: true,
		},
		{
			name:     "single field no match",
			fields:   []string{"message"},
			query:    "$1",
			expected: false,
		},
		{
			name:     "multiple fields all match",
			fields:   []string{"message", "level"},
			query:    "error",
			expected: true,
		},
		{
			name:     "multiple fields partial match",
			fields:   []string{"message", "session"},
			query:    "$1",
			expected: true,
		},
		{
			name:     "empty fields list",
			fields:   []string{},
			query:    "database",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewSubstringProvider(WithFields(tt.fields))
			result := provider.Match(testNotification, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTokenProviderReadUnreadOnly tests various read/unread combinations.
func TestTokenProviderReadUnreadOnly(t *testing.T) {
	provider := NewTokenProvider()

	// Create read and unread notifications
	readNotif := notification.Notification{
		ID:            1,
		Message:       "test message",
		ReadTimestamp: "2024-01-01T12:00:00Z",
	}
	unreadNotif := notification.Notification{
		ID:            2,
		Message:       "test message",
		ReadTimestamp: "",
	}

	tests := []struct {
		name     string
		notif    notification.Notification
		query    string
		expected bool
	}{
		{
			name:     "unread notif with unread query",
			notif:    unreadNotif,
			query:    "unread",
			expected: true,
		},
		{
			name:     "unread notif with read query",
			notif:    unreadNotif,
			query:    "read",
			expected: false,
		},
		{
			name:     "read notif with read query",
			notif:    readNotif,
			query:    "read",
			expected: true,
		},
		{
			name:     "read notif with unread query",
			notif:    readNotif,
			query:    "unread",
			expected: false,
		},
		{
			name:     "unread notif with unread and text",
			notif:    unreadNotif,
			query:    "unread test",
			expected: true,
		},
		{
			name:     "read notif with read and text",
			notif:    readNotif,
			query:    "read test",
			expected: true,
		},
		{
			name:     "read notif with unread and text",
			notif:    readNotif,
			query:    "unread test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.Match(tt.notif, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
