package search

import (
	"regexp"
	"sync"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// RegexProvider provides regex-based search.
// Matches if any configured field matches the regex pattern.
type RegexProvider struct {
	opts    Options
	cache   map[string]*regexp.Regexp
	cacheMu sync.RWMutex
}

// NewRegexProvider creates a new regex search provider.
func NewRegexProvider(opts ...Option) Provider {
	return &RegexProvider{
		opts:  applyOptions(opts),
		cache: make(map[string]*regexp.Regexp),
	}
}

// Match returns true if any configured field matches the regex pattern.
// If the query is not a valid regex, it returns false for all notifications.
func (p *RegexProvider) Match(notif notification.Notification, query string) bool {
	if query == "" {
		return true
	}

	// Get or compile regex
	re, err := p.getRegex(query)
	if err != nil {
		// Invalid regex, return false
		return false
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

		// Check for regex match
		if re.MatchString(fieldValue) {
			return true
		}
	}

	return false
}

// getRegex returns a compiled regex for the given pattern, using cache.
func (p *RegexProvider) getRegex(pattern string) (*regexp.Regexp, error) {
	p.cacheMu.RLock()
	re, ok := p.cache[pattern]
	p.cacheMu.RUnlock()

	if ok {
		return re, nil
	}

	// Compile with case-insensitive flag if configured
	var re2 *regexp.Regexp
	var err error
	if p.opts.CaseInsensitive {
		re2, err = regexp.Compile("(?i)" + pattern)
	} else {
		re2, err = regexp.Compile(pattern)
	}

	if err != nil {
		return nil, err
	}

	p.cacheMu.Lock()
	p.cache[pattern] = re2
	p.cacheMu.Unlock()

	return re2, nil
}

// Name returns the provider name.
func (p *RegexProvider) Name() string {
	return "regex"
}
