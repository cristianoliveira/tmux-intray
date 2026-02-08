package search

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/assert"
)

// TestAllFieldValues tests matching specific field values.
func TestAllFieldValues(t *testing.T) {
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
			// Test matching each individual field
			assert.True(t, provider.p.Match(testNotification, "error"), "level field")
			assert.True(t, provider.p.Match(testNotification, "database"), "message field")
			// Regex needs escaping for special chars
			if provider.name == "regex" {
				assert.True(t, provider.p.Match(testNotification, `\$1`), "session field (regex escaped)")
			} else {
				assert.True(t, provider.p.Match(testNotification, "$1"), "session field")
			}
			assert.True(t, provider.p.Match(testNotification, "@0"), "window field")
			assert.True(t, provider.p.Match(testNotification, "%0"), "pane field")
		})
	}
}

// TestTokenProviderEdgeCases tests edge cases for token matching.
func TestTokenProviderEdgeCases(t *testing.T) {
	provider := NewTokenProvider()

	// Create notifications with specific characteristics
	notifWithAllFields := notification.Notification{
		ID:        1,
		Timestamp: "2024-01-01T12:00:00Z",
		State:     "active",
		Session:   "$1",
		Window:    "@1",
		Pane:      "%1",
		Message:   "test message",
		Level:     "info",
	}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{"empty after whitespace", "   ", true},        // trimmed to empty string
		{"single whitespace", " ", true},               // trimmed to empty string
		{"only special tokens read", "read", false},    // notif is unread
		{"only special tokens unread", "unread", true}, // notif is unread
		{"same token twice", "test test", true},
		{"overlapping tokens", "test messa", true},
		{"token that spans multiple fields", "test $1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.Match(notifWithAllFields, tt.query)
			assert.Equal(t, tt.expected, result, "query: %q", tt.query)
		})
	}
}

// TestRegexProviderCaseInsensitiveMode tests case-insensitive regex mode.
func TestRegexProviderCaseInsensitiveMode(t *testing.T) {
	providerCI := NewRegexProvider(WithCaseInsensitive(true))
	providerCS := NewRegexProvider(WithCaseInsensitive(false))

	// Test that case-insensitive mode works
	assert.True(t, providerCI.Match(testNotification, "ERROR"), "case-insensitive should match uppercase")
	assert.True(t, providerCI.Match(testNotification, "Error"), "case-insensitive should match mixed case")
	assert.True(t, providerCI.Match(testNotification, "error"), "case-insensitive should match lowercase")

	// Test that case-sensitive mode works
	assert.False(t, providerCS.Match(testNotification, "ERROR"), "case-sensitive should not match uppercase")
	assert.False(t, providerCS.Match(testNotification, "Error"), "case-sensitive should not match mixed case")
	assert.True(t, providerCS.Match(testNotification, "error"), "case-sensitive should match lowercase")
}

// TestSubstringProviderCaseInsensitive tests case-insensitive substring search.
func TestSubstringProviderCaseInsensitive(t *testing.T) {
	providerCI := NewSubstringProvider(WithCaseInsensitive(true))
	providerCS := NewSubstringProvider(WithCaseInsensitive(false))

	// Test that case-insensitive mode works
	assert.True(t, providerCI.Match(testNotification, "ERROR"), "case-insensitive should match uppercase")
	assert.True(t, providerCI.Match(testNotification, "Error"), "case-insensitive should match mixed case")
	assert.True(t, providerCI.Match(testNotification, "error"), "case-insensitive should match lowercase")

	// Test that case-sensitive mode works
	assert.False(t, providerCS.Match(testNotification, "ERROR"), "case-sensitive should not match uppercase")
	assert.False(t, providerCS.Match(testNotification, "Error"), "case-sensitive should not match mixed case")
	assert.True(t, providerCS.Match(testNotification, "error"), "case-sensitive should match lowercase")
}

// TestTokenProviderCaseInsensitive tests case-insensitive token search.
func TestTokenProviderCaseInsensitive(t *testing.T) {
	providerCI := NewTokenProvider(WithCaseInsensitive(true))
	providerCS := NewTokenProvider(WithCaseInsensitive(false))

	// Test that case-insensitive mode works
	assert.True(t, providerCI.Match(testNotification, "ERROR DATABASE"), "case-insensitive should match uppercase tokens")
	assert.True(t, providerCI.Match(testNotification, "Error Database"), "case-insensitive should match mixed case tokens")

	// Test that case-sensitive mode works
	assert.False(t, providerCS.Match(testNotification, "ERROR"), "case-sensitive should not match uppercase token")
	assert.False(t, providerCS.Match(testNotification, "Error"), "case-sensitive should not match mixed case token")
	assert.True(t, providerCS.Match(testNotification, "error"), "case-sensitive should match lowercase token")
}

// TestProviderWithEmptyNotification tests provider behavior with mostly empty notification.
func TestProviderWithEmptyNotification(t *testing.T) {
	emptyNotif := notification.Notification{
		ID:        1,
		Timestamp: "2024-01-01T12:00:00Z",
		Message:   "test",
	}

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
			assert.True(t, provider.p.Match(emptyNotif, "test"), "should match message")
			assert.False(t, provider.p.Match(emptyNotif, "error"), "should not match")
			assert.True(t, provider.p.Match(emptyNotif, ""), "empty query should match")
		})
	}
}

// TestTokenProviderOnlyReadTokens tests with only read/unread tokens.
func TestTokenProviderOnlyReadTokens(t *testing.T) {
	provider := NewTokenProvider()

	// Unread notification
	assert.True(t, provider.Match(testNotification, "unread"), "unread should match unread notif")
	assert.False(t, provider.Match(testNotification, "read"), "read should not match unread notif")

	// Read notification
	assert.False(t, provider.Match(testNotificationRead, "unread"), "unread should not match read notif")
	assert.True(t, provider.Match(testNotificationRead, "read"), "read should match read notif")

	// With case-insensitive
	providerCI := NewTokenProvider(WithCaseInsensitive(true))
	assert.True(t, providerCI.Match(testNotificationRead, "READ"), "READ should match as read token when CI")
	assert.True(t, providerCI.Match(testNotificationRead, "Read"), "Read should match as read token when CI")
}

// TestSubstringProviderExactFieldMatch tests exact matching on specific fields.
func TestSubstringProviderExactFieldMatch(t *testing.T) {
	// Provider that only searches in message field
	provider := NewSubstringProvider(WithFields([]string{"message"}))

	// testNotification has Message: "error: failed to connect to database"
	assert.True(t, provider.Match(testNotification, "database"), "should match in message")
	assert.True(t, provider.Match(testNotification, "error"), "should match in message")
	assert.False(t, provider.Match(testNotification, "$1"), "should not match session field (not message)")
	assert.False(t, provider.Match(testNotification, "@0"), "should not match window field (not message)")
	assert.False(t, provider.Match(testNotification, "%0"), "should not match pane field (not message)")

	// Provider that only searches in level field
	levelProvider := NewSubstringProvider(WithFields([]string{"level"}))

	assert.False(t, levelProvider.Match(testNotification, "database"), "should not match message")
	assert.False(t, levelProvider.Match(testNotification, "$1"), "should not match session")
	assert.True(t, levelProvider.Match(testNotification, "error"), "should match level")
}

// TestMultipleEmptyQueries tests various empty query variations.
func TestMultipleEmptyQueries(t *testing.T) {
	providers := []struct {
		name string
		p    Provider
	}{
		{"substring", NewSubstringProvider()},
		{"regex", NewRegexProvider()},
		{"token", NewTokenProvider()},
	}

	queries := []struct {
		query    string
		expected bool
	}{
		{"", true},     // empty string
		{"   ", false}, // whitespace is trimmed, but token provider treats as empty
		{"\t", false},
		{"\n", false},
		{"  \t  \n  ", false},
	}

	for _, provider := range providers {
		for _, q := range queries {
			t.Run(provider.name+"/"+q.query, func(t *testing.T) {
				result := provider.p.Match(testNotification, q.query)
				// Whitespace is trimmed to empty string, so should match
				expected := true
				if q.query != "" {
					// For token provider, trimmed empty matches, for others it's actual whitespace
					if provider.name != "token" {
						expected = false // whitespace doesn't match anything
					}
				}
				assert.Equal(t, expected, result, "query: %q", q.query)
			})
		}
	}
}

// TestTokenProviderComplexQueries tests complex token combinations.
func TestTokenProviderComplexQueries(t *testing.T) {
	provider := NewTokenProvider()

	// Create notification with multiple words in message
	multiWordNotif := notification.Notification{
		ID:            1,
		Timestamp:     "2024-01-01T12:00:00Z",
		State:         "active",
		Message:       "error connecting to database server timeout",
		Level:         "error",
		ReadTimestamp: "",
	}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{"multiple matching tokens", "error database", true},
		{"tokens from different words", "connecting server", true},
		{"partial match of token", "time", true}, // "timeout" contains "time"
		{"one matching one not", "error network", false},
		{"many tokens all match", "error database connecting server timeout", true},
		{"read filter with tokens", "unread error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.Match(multiWordNotif, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
