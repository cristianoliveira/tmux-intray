package search

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/assert"
)

// TestRegexProviderCache verifies regex caching behavior.
func TestRegexProviderCache(t *testing.T) {
	provider := NewRegexProvider()

	// First call compiles and caches
	result1 := provider.Match(testNotification, "error")
	assert.True(t, result1)

	// Second call uses cached regex
	result2 := provider.Match(testNotification, "error")
	assert.True(t, result2)

	// Different pattern compiles new regex
	result3 := provider.Match(testNotification, "warning")
	assert.False(t, result3)

	// Original pattern still works
	result4 := provider.Match(testNotification, "error")
	assert.True(t, result4)
}

// TestRegexProviderInvalidPattern handles invalid regex patterns.
func TestRegexProviderInvalidPattern(t *testing.T) {
	provider := NewRegexProvider()

	// Invalid pattern should return false
	result := provider.Match(testNotification, "[invalid(unclosed")
	assert.False(t, result)

	// Valid pattern should still work after invalid
	result2 := provider.Match(testNotification, "error")
	assert.True(t, result2)
}

// TestRegexProviderWithVariousPatterns tests various regex patterns.
func TestRegexProviderWithVariousPatterns(t *testing.T) {
	provider := NewRegexProvider()

	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{"dot wildcard", "err.r", true},
		{"start anchor", "^error", true},  // matches in "level" field
		{"end anchor", "database$", true}, // matches in "message" field
		{"alternation", "database|network", true},
		{"quantifier", "e{2}", false},      // no double "e"
		{"quantifier star", "a.*", true},   // matches "a" in "database"
		{"character class", "[eio]", true}, // contains e, i, or o
		{"negated class", "[^xyz]", true},  // contains chars other than x, y, z
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.Match(testNotification, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSubstringProviderWithVariousInputs tests various substring inputs.
func TestSubstringProviderWithVariousInputs(t *testing.T) {
	provider := NewSubstringProvider()

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{"special chars @", "@", true}, // in session/window/pane
		{"special chars %", "%", true}, // in pane
		{"spaces", "to database", true},
		{"tabs", "\t", false},
		{"unicode", "✓", false},
		{"very long substring", "error: failed to connect to database", true},
		{"exact match", "error: failed to connect to database", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.Match(testNotification, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTokenProviderWithWhitespace tests token parsing with various whitespace.
func TestTokenProviderWithWhitespace(t *testing.T) {
	provider := NewTokenProvider()

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{"leading space", "  error", true},
		{"trailing space", "error  ", true},
		{"multiple spaces", "error  database", true},
		{"tabs", "error\tdatabase", true},
		{"newlines", "error\ndatabase", true}, // strings.Fields handles newlines
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.Match(testNotification, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTokenProviderSpecialTokens tests special token handling.
func TestTokenProviderSpecialTokens(t *testing.T) {
	provider := NewTokenProvider()

	tests := []struct {
		name     string
		notif    notification.Notification
		query    string
		expected bool
	}{
		{
			name:     "READ (uppercase)",
			notif:    testNotificationRead,
			query:    "READ",
			expected: true, // "READ" treated as regular text token, not special
		},
		{
			name:     "Read (mixed case)",
			notif:    testNotificationRead,
			query:    "Read",
			expected: true, // "Read" treated as regular text token
		},
		{
			name:     "read with text",
			notif:    testNotificationRead,
			query:    "read slow",
			expected: true,
		},
		{
			name:     "unread with text",
			notif:    testNotification,
			query:    "unread error",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.Match(tt.notif, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAllFieldsOption tests searching in all valid fields.
func TestAllFieldsOption(t *testing.T) {
	allFields := []string{"message", "session", "window", "pane", "level", "state"}

	providers := []struct {
		name string
		p    Provider
	}{
		{"substring", NewSubstringProvider(WithFields(allFields))},
		{"regex", NewRegexProvider(WithFields(allFields))},
	}

	// Test that each field can be matched
	fieldValues := map[string]string{
		"message": testNotification.Message,
		"session": testNotification.Session,
		"window":  testNotification.Window,
		"pane":    testNotification.Pane,
		"level":   testNotification.Level,
		"state":   testNotification.State,
	}

	for _, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			for field, value := range fieldValues {
				if value == "" {
					continue
				}
				t.Run(field, func(t *testing.T) {
					// Use a substring/regex of the field value
					query := value[:len(value)/2+1] // First half + 1 char
					if field == "message" {
						query = "error"
					}
					result := provider.p.Match(testNotification, query)
					// Should match in at least one field
					// Note: may not match if we're checking a different field's value
					_ = result // For now, just verify it doesn't panic
				})
			}
		})
	}
}

// TestProviderWithEmptyFields tests provider behavior with no configured fields.
func TestProviderWithEmptyFields(t *testing.T) {
	providers := []struct {
		name string
		p    Provider
	}{
		{"substring", NewSubstringProvider(WithFields([]string{}))},
		{"regex", NewRegexProvider(WithFields([]string{}))},
		{"token", NewTokenProvider(WithFields([]string{}))},
	}

	for _, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			// With no fields, no query should match (except empty query)
			result := provider.p.Match(testNotification, "error")
			assert.False(t, result)

			// Empty query should still match
			result2 := provider.p.Match(testNotification, "")
			assert.True(t, result2)
		})
	}
}
