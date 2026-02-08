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

			// Check for substring match
			if strings.Contains(fieldValue, searchQuery) {
				return true
			}
		}
	}

	return false
}

// Name returns the provider name.
func (p *SubstringProvider) Name() string {
	return "substring"
}

// getFieldValuesWithNames returns a slice containing both the ID and resolved name.
// If nameMap is nil or ID not found, returns only the ID.
func (p *SubstringProvider) getFieldValuesWithNames(id string, nameMap map[string]string) []string {
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
