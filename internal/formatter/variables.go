// Package formatter provides template parsing, variable resolution, and preset management
// for formatting notification output with customizable templates and variables.
package formatter

import (
	"fmt"
	"strconv"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// VariableContext contains all data needed for template variable resolution.
type VariableContext struct {
	// Count-related variables
	UnreadCount    int
	TotalCount     int
	ReadCount      int
	ActiveCount    int
	DismissedCount int

	// Level-specific count variables (for backward compatibility)
	InfoCount     int
	WarningCount  int
	ErrorCount    int
	CriticalCount int

	// Content variables
	LatestMessage string

	// State variables
	HasUnread    bool
	HasActive    bool
	HasDismissed bool

	// Level variables
	HighestSeverity domain.NotificationLevel

	// Session/Window/Pane variables
	SessionList string
	WindowList  string
	PaneList    string
}

// VariableResolver resolves template variables to their values.
type VariableResolver interface {
	// Resolve returns the string value for a given variable name and context.
	Resolve(varName string, ctx VariableContext) (string, error)
}

// variableResolver implements VariableResolver interface.
type variableResolver struct{}

// NewVariableResolver creates a new variable resolver instance.
func NewVariableResolver() VariableResolver {
	return &variableResolver{}
}

// Resolve returns the string value for a variable from the context.
// Handles all 13 template variables and their aliases.
func (vr *variableResolver) Resolve(varName string, ctx VariableContext) (string, error) {
	switch varName {
	// Count variables
	case "unread-count":
		return strconv.Itoa(ctx.UnreadCount), nil

	case "total-count":
		// Alias for unread-count
		return strconv.Itoa(ctx.UnreadCount), nil

	case "read-count":
		return strconv.Itoa(ctx.ReadCount), nil

	case "active-count":
		return strconv.Itoa(ctx.ActiveCount), nil

	case "dismissed-count":
		return strconv.Itoa(ctx.DismissedCount), nil

	// Level-specific count variables
	case "info-count":
		return strconv.Itoa(ctx.InfoCount), nil

	case "warning-count":
		return strconv.Itoa(ctx.WarningCount), nil

	case "error-count":
		return strconv.Itoa(ctx.ErrorCount), nil

	case "critical-count":
		return strconv.Itoa(ctx.CriticalCount), nil

	// Content variables
	case "latest-message":
		return ctx.LatestMessage, nil

	// Boolean variables (as strings)
	case "has-unread":
		return boolToString(ctx.HasUnread), nil

	case "has-active":
		return boolToString(ctx.HasActive), nil

	case "has-dismissed":
		return boolToString(ctx.HasDismissed), nil

	// Severity variable with ordinal mapping
	case "highest-severity":
		return severityToOrdinal(ctx.HighestSeverity), nil

	// Session/Window/Pane variables
	case "session-list":
		return ctx.SessionList, nil

	case "window-list":
		return ctx.WindowList, nil

	case "pane-list":
		return ctx.PaneList, nil

	default:
		return "", fmt.Errorf("unknown variable: %s", varName)
	}
}

// boolToString converts a boolean to the string "true" or "false".
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// severityToOrdinal maps NotificationLevel to ordinal severity numbers.
// Lower numbers = more severe.
// CRITICAL=1, HIGH=2, MEDIUM=3, LOW=4
func severityToOrdinal(level domain.NotificationLevel) string {
	switch level {
	case domain.LevelCritical:
		return "1"
	case domain.LevelError:
		// Treating error as HIGH priority
		return "2"
	case domain.LevelWarning:
		// Treating warning as MEDIUM priority
		return "3"
	case domain.LevelInfo:
		// Treating info as LOW priority
		return "4"
	default:
		// Default to lowest priority
		return "4"
	}
}
