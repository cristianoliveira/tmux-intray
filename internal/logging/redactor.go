package logging

import (
	"regexp"
	"strings"
)

// redactor redacts sensitive values in log key‑value pairs.
type redactor struct {
	sensitiveWords map[string]bool
}

// newRedactor creates a new redactor with the default sensitive key pattern.
func newRedactor() *redactor {
	words := []string{"secret", "password", "token", "key", "auth", "credential"}
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return &redactor{
		sensitiveWords: m,
	}
}

// redact walks through a slice of key‑value pairs (flattened as [key1, value1, key2, value2, …]).
// If a key contains a sensitive word as a separate segment, its value is replaced with "[REDACTED]".
// Returns a new slice with redacted values; the original slice is not modified.
func (r *redactor) redact(pairs []any) []any {
	if len(pairs) == 0 {
		return pairs
	}
	result := make([]any, len(pairs))
	copy(result, pairs)
	for i := 0; i+1 < len(result); i += 2 {
		key, ok := result[i].(string)
		if !ok {
			continue
		}
		if r.isSensitive(key) {
			result[i+1] = "[REDACTED]"
		}
	}
	return result
}

// isSensitive returns true if the key contains any sensitive word as a separate segment.
// Segments are split by non-alphanumeric characters (including underscore).
func (r *redactor) isSensitive(key string) bool {
	// Convert to lower case for case‑insensitive comparison
	key = strings.ToLower(key)
	// Split by non-alphanumeric
	parts := splitByNonAlphanumeric(key)
	for _, part := range parts {
		if r.sensitiveWords[part] {
			return true
		}
	}
	return false
}

// splitByNonAlphanumeric splits a string by sequences of non‑alphanumeric characters.
func splitByNonAlphanumeric(s string) []string {
	// Use regexp for simplicity; we already have the import
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return re.Split(s, -1)
}

// redactString redacts sensitive substrings within a string value.
// This is a more aggressive redaction that scans the whole value for patterns.
// Not currently used but kept for future extension.
func (r *redactor) redactString(value string) string {
	// Simple implementation: if value contains any sensitive keyword, redact whole value
	if r.isSensitive(value) {
		return "[REDACTED]"
	}
	return value
}
