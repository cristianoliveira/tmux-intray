package search

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// TokenProvider provides token-based search.
// The query is split into whitespace-separated tokens.
// Each token must match at least one field (AND logic).
// Special tokens: "read" (match only read), "unread" (match only unread).
type TokenProvider struct {
	opts Options
}

// NewTokenProvider creates a new token search provider.
func NewTokenProvider(opts ...Option) Provider {
	return &TokenProvider{
		opts: applyOptions(opts),
	}
}

// Match returns true if all text tokens match at least one field
// and the notification matches the read/unread filter if specified.
func (p *TokenProvider) Match(notif notification.Notification, query string) bool {
	if query == "" {
		return true
	}

	// Parse tokens
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}

	// Parse special tokens (read/unread)
	tokens := strings.Fields(query)
	readFilter := false
	unreadFilter := false
	textTokens := []string{}

	for _, token := range tokens {
		tokenLower := strings.ToLower(token)
		switch tokenLower {
		case "read":
			readFilter = true
		case "unread":
			unreadFilter = true
		default:
			// Apply case sensitivity to text tokens
			if p.opts.CaseInsensitive {
				textTokens = append(textTokens, strings.ToLower(token))
			} else {
				textTokens = append(textTokens, token)
			}
		}
	}

	// If both read and unread specified, ignore both (contradiction)
	if readFilter && unreadFilter {
		readFilter = false
		unreadFilter = false
	}

	// Apply read/unread filter
	if readFilter && !notif.IsRead() {
		return false
	}
	if unreadFilter && notif.IsRead() {
		return false
	}

	// If no text tokens, match passed the read/unread filter
	if len(textTokens) == 0 {
		return true
	}

	// Each token must match at least one field (AND logic)
	for _, token := range textTokens {
		matched := false
		for _, field := range p.opts.Fields {
			var fieldValues []string
			switch field {
			case "message":
				fieldValues = []string{notif.Message}
			case "session":
				fieldValues = p.getFieldValuesWithNames(notif.Session, p.opts.SessionNames)
			case "window":
				fieldValues = p.getFieldValuesWithNames(notif.Window, p.opts.WindowNames)
			case "pane":
				fieldValues = p.getFieldValuesWithNames(notif.Pane, p.opts.PaneNames)
			case "level":
				fieldValues = []string{notif.Level}
			case "state":
				fieldValues = []string{notif.State}
			}

			// Check all field values (ID and name)
			for _, fieldValue := range fieldValues {
				// Skip empty fields
				if fieldValue == "" {
					continue
				}

				// Apply case sensitivity
				if p.opts.CaseInsensitive {
					fieldValue = strings.ToLower(fieldValue)
				}

				// Check if token is found in field
				if strings.Contains(fieldValue, token) {
					matched = true
					break
				}
			}

			if matched {
				break
			}
		}

		// Token didn't match any field
		if !matched {
			return false
		}
	}

	// All tokens matched
	return true
}

// Name returns the provider name.
func (p *TokenProvider) Name() string {
	return "token"
}

// getFieldValuesWithNames returns a slice containing both the ID and resolved name.
// If nameMap is nil or ID not found, returns only the ID.
func (p *TokenProvider) getFieldValuesWithNames(id string, nameMap map[string]string) []string {
	if id == "" {
		return []string{}
	}

	values := []string{id}

	// If name map is provided and ID exists in map, add the name
	if nameMap != nil {
		if name, ok := nameMap[id]; ok {
			values = append(values, name)
		}
	}

	return values
}
