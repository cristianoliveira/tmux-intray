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

type tokenQuery struct {
	readFilter   bool
	unreadFilter bool
	textTokens   []string
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

	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}

	parsed := p.parseTokenQuery(query)
	if !parsed.matchesReadFilter(notif) {
		return false
	}

	if len(parsed.textTokens) == 0 {
		return true
	}

	return p.matchTextTokens(notif, parsed.textTokens)
}

// Name returns the provider name.
func (p *TokenProvider) Name() string {
	return "token"
}

func (p *TokenProvider) parseTokenQuery(query string) tokenQuery {
	tokens := strings.Fields(query)
	parsed := tokenQuery{}

	for _, token := range tokens {
		tokenLower := strings.ToLower(token)
		switch tokenLower {
		case "read":
			parsed.readFilter = true
		case "unread":
			parsed.unreadFilter = true
		default:
			if p.opts.CaseInsensitive {
				parsed.textTokens = append(parsed.textTokens, strings.ToLower(token))
			} else {
				parsed.textTokens = append(parsed.textTokens, token)
			}
		}
	}

	if parsed.readFilter && parsed.unreadFilter {
		parsed.readFilter = false
		parsed.unreadFilter = false
	}

	return parsed
}

func (q tokenQuery) matchesReadFilter(notif notification.Notification) bool {
	if q.readFilter && !notif.IsRead() {
		return false
	}
	if q.unreadFilter && notif.IsRead() {
		return false
	}
	return true
}

func (p *TokenProvider) matchTextTokens(notif notification.Notification, tokens []string) bool {
	for _, token := range tokens {
		if !p.matchToken(notif, token) {
			return false
		}
	}
	return true
}

func (p *TokenProvider) matchToken(notif notification.Notification, token string) bool {
	for _, field := range p.opts.Fields {
		for _, fieldValue := range p.getFieldValues(notif, field) {
			if fieldValue == "" {
				continue
			}
			if p.opts.CaseInsensitive {
				fieldValue = strings.ToLower(fieldValue)
			}
			if strings.Contains(fieldValue, token) {
				return true
			}
		}
	}
	return false
}

func (p *TokenProvider) getFieldValues(notif notification.Notification, field string) []string {
	switch field {
	case "message":
		return []string{notif.Message}
	case "session":
		return p.getFieldValuesWithNames(notif.Session, p.opts.SessionNames)
	case "window":
		return p.getFieldValuesWithNames(notif.Window, p.opts.WindowNames)
	case "pane":
		return p.getFieldValuesWithNames(notif.Pane, p.opts.PaneNames)
	case "level":
		return []string{notif.Level}
	case "state":
		return []string{notif.State}
	default:
		return []string{}
	}
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
