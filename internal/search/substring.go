package search

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// SubstringProvider provides substring-based search.
// Matches if any configured field contains the query as a substring.
type SubstringProvider struct {
	opts Options
}

// NewSubstringProvider creates a new substring search provider.
func NewSubstringProvider(opts ...Option) Provider {
	return &SubstringProvider{
		opts: applyOptions(opts),
	}
}

// Match returns true if any configured field contains the query substring.
func (p *SubstringProvider) Match(notif notification.Notification, query string) bool {
	if query == "" {
		return true
	}

	// Prepare query based on case sensitivity
	searchQuery := query
	if p.opts.CaseInsensitive {
		searchQuery = strings.ToLower(query)
	}

	// Check each configured field
	for _, field := range p.opts.Fields {
		var fieldValue string
		switch field {
		case "message":
			fieldValue = notif.Message
		case "session_name":
			fieldValue = notif.SessionName
		case "session":
			fieldValue = notif.Session
		case "window":
			fieldValue = notif.Window
		case "pane":
			fieldValue = notif.Pane
		case "level":
			fieldValue = notif.Level
		case "state":
			fieldValue = notif.State
		}

		// Skip empty fields
		if fieldValue == "" {
			continue
		}

		// Apply case sensitivity
		if p.opts.CaseInsensitive {
			fieldValue = strings.ToLower(fieldValue)
		}

		// Check for substring match
		if strings.Contains(fieldValue, searchQuery) {
			return true
		}
	}

	return false
}

// Name returns the provider name.
func (p *SubstringProvider) Name() string {
	return "substring"
}
