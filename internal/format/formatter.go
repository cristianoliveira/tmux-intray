// Package format provides output formatting functionality for CLI commands.
// It includes formatters for different output styles and notification display.
package format

import (
	"io"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// Formatter defines the interface for output formatters.
type Formatter interface {
	// FormatNotifications formats a slice of notifications and writes to the writer.
	FormatNotifications(notifications []*domain.Notification, writer io.Writer) error

	// FormatGroups formats grouped notifications and writes to the writer.
	FormatGroups(groups domain.GroupResult, writer io.Writer) error
}

// FormatterType represents the type of formatter to use.
type FormatterType string

const (
	// FormatterTypeSimple displays notifications in a simple format with ID, timestamp, and message.
	FormatterTypeSimple FormatterType = "simple"

	// FormatterTypeLegacy displays only messages, one per line (original format).
	FormatterTypeLegacy FormatterType = "legacy"

	// FormatterTypeTable displays notifications in a table format with headers.
	FormatterTypeTable FormatterType = "table"

	// FormatterTypeCompact displays only messages in a compact format.
	FormatterTypeCompact FormatterType = "compact"

	// FormatterTypeJSON displays notifications in JSON format.
	FormatterTypeJSON FormatterType = "json"
)

// NewFormatter creates a new formatter of the specified type.
func NewFormatter(formatterType FormatterType) Formatter {
	switch formatterType {
	case FormatterTypeSimple:
		return NewSimpleFormatter()
	case FormatterTypeLegacy:
		return NewLegacyFormatter()
	case FormatterTypeTable:
		return NewTableFormatter()
	case FormatterTypeCompact:
		return NewCompactFormatter()
	case FormatterTypeJSON:
		return NewJSONFormatter()
	default:
		// Default to simple formatter for unknown types
		return NewSimpleFormatter()
	}
}
