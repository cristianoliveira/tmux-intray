package format

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// SimpleFormatter formats notifications in a simple format with ID, timestamp, and message.
type SimpleFormatter struct{}

// NewSimpleFormatter creates a new SimpleFormatter.
func NewSimpleFormatter() *SimpleFormatter {
	return &SimpleFormatter{}
}

// FormatNotifications formats notifications in simple format.
func (f *SimpleFormatter) FormatNotifications(notifications []*domain.Notification, writer io.Writer) error {
	for _, n := range notifications {
		// Truncate message for display (50 chars max)
		displayMsg := n.Message
		if len(displayMsg) > 50 {
			displayMsg = displayMsg[:47] + "..."
		}
		_, err := fmt.Fprintf(writer, "%-4d  %-25s  - %s\n", n.ID, n.Timestamp, displayMsg)
		if err != nil {
			return err
		}
	}
	return nil
}

// FormatGroups formats grouped notifications in simple format.
func (f *SimpleFormatter) FormatGroups(groups domain.GroupResult, writer io.Writer) error {
	// Groups are already sorted by display name
	for _, group := range groups.Groups {
		_, err := fmt.Fprintf(writer, "=== %s (%d) ===\n", group.DisplayName, group.Count)
		if err != nil {
			return err
		}
		notifs := make([]*domain.Notification, len(group.Notifications))
		for i := range group.Notifications {
			notifs[i] = &group.Notifications[i]
		}
		err = f.FormatNotifications(notifs, writer)
		if err != nil {
			return err
		}
	}
	return nil
}

// LegacyFormatter formats notifications with only messages (one per line).
type LegacyFormatter struct{}

// NewLegacyFormatter creates a new LegacyFormatter.
func NewLegacyFormatter() *LegacyFormatter {
	return &LegacyFormatter{}
}

// FormatNotifications formats notifications in legacy format.
func (f *LegacyFormatter) FormatNotifications(notifications []*domain.Notification, writer io.Writer) error {
	for _, n := range notifications {
		_, err := fmt.Fprintln(writer, n.Message)
		if err != nil {
			return err
		}
	}
	return nil
}

// FormatGroups formats grouped notifications in legacy format.
func (f *LegacyFormatter) FormatGroups(groups domain.GroupResult, writer io.Writer) error {
	// Groups are already sorted by display name
	for _, group := range groups.Groups {
		_, err := fmt.Fprintf(writer, "=== %s (%d) ===\n", group.DisplayName, group.Count)
		if err != nil {
			return err
		}
		notifs := make([]*domain.Notification, len(group.Notifications))
		for i := range group.Notifications {
			notifs[i] = &group.Notifications[i]
		}
		err = f.FormatNotifications(notifs, writer)
		if err != nil {
			return err
		}
	}
	return nil
}

// TableFormatter formats notifications in a table format with headers.
type TableFormatter struct{}

// NewTableFormatter creates a new TableFormatter.
func NewTableFormatter() *TableFormatter {
	return &TableFormatter{}
}

// FormatNotifications formats notifications in table format.
func (f *TableFormatter) FormatNotifications(notifications []*domain.Notification, writer io.Writer) error {
	if len(notifications) == 0 {
		return nil
	}
	headerColor := colors.Blue
	reset := colors.Reset
	_, err := fmt.Fprintf(writer, "%sID    DATE                   - Message%s\n", headerColor, reset)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "%s----  ---------------------  - --------------------------------%s\n", headerColor, reset)
	if err != nil {
		return err
	}
	for _, n := range notifications {
		// Truncate message for display (32 chars max)
		displayMsg := n.Message
		if len(displayMsg) > 32 {
			displayMsg = displayMsg[:29] + "..."
		}
		_, err := fmt.Fprintf(writer, "%-4d  %-23s  - %s\n", n.ID, n.Timestamp, displayMsg)
		if err != nil {
			return err
		}
	}
	return nil
}

// FormatGroups formats grouped notifications in table format.
func (f *TableFormatter) FormatGroups(groups domain.GroupResult, writer io.Writer) error {
	// Groups are already sorted by display name
	for _, group := range groups.Groups {
		_, err := fmt.Fprintf(writer, "=== %s (%d) ===\n", group.DisplayName, group.Count)
		if err != nil {
			return err
		}
		notifs := make([]*domain.Notification, len(group.Notifications))
		for i := range group.Notifications {
			notifs[i] = &group.Notifications[i]
		}
		err = f.FormatNotifications(notifs, writer)
		if err != nil {
			return err
		}
	}
	return nil
}

// CompactFormatter formats notifications with message only in a compact format.
type CompactFormatter struct{}

// NewCompactFormatter creates a new CompactFormatter.
func NewCompactFormatter() *CompactFormatter {
	return &CompactFormatter{}
}

// FormatNotifications formats notifications in compact format.
func (f *CompactFormatter) FormatNotifications(notifications []*domain.Notification, writer io.Writer) error {
	for _, n := range notifications {
		// Truncate message for display (60 chars max)
		displayMsg := n.Message
		if len(displayMsg) > 60 {
			displayMsg = displayMsg[:57] + "..."
		}
		_, err := fmt.Fprintln(writer, displayMsg)
		if err != nil {
			return err
		}
	}
	return nil
}

// FormatGroups formats grouped notifications in compact format.
func (f *CompactFormatter) FormatGroups(groups domain.GroupResult, writer io.Writer) error {
	// Groups are already sorted by display name
	for _, group := range groups.Groups {
		_, err := fmt.Fprintf(writer, "=== %s (%d) ===\n", group.DisplayName, group.Count)
		if err != nil {
			return err
		}
		notifs := make([]*domain.Notification, len(group.Notifications))
		for i := range group.Notifications {
			notifs[i] = &group.Notifications[i]
		}
		err = f.FormatNotifications(notifs, writer)
		if err != nil {
			return err
		}
	}
	return nil
}

// JSONFormatter formats notifications as JSON.
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSONFormatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// FormatNotifications formats notifications as JSON.
func (f *JSONFormatter) FormatNotifications(notifications []*domain.Notification, writer io.Writer) error {
	data, err := json.MarshalIndent(notifications, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notifications to JSON: %w", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer)
	return err
}

// FormatGroups formats grouped notifications as JSON.
func (f *JSONFormatter) FormatGroups(groups domain.GroupResult, writer io.Writer) error {
	data, err := json.MarshalIndent(groups, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal group result to JSON: %w", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer)
	return err
}

// GroupCountFormatter formats only group counts.
type GroupCountFormatter struct {
	formatter Formatter
}

// NewGroupCountFormatter creates a new GroupCountFormatter.
func NewGroupCountFormatter(formatter Formatter) *GroupCountFormatter {
	return &GroupCountFormatter{formatter: formatter}
}

// FormatNotifications is not applicable for GroupCountFormatter.
func (f *GroupCountFormatter) FormatNotifications(notifications []*domain.Notification, writer io.Writer) error {
	return fmt.Errorf("formatNotifications not supported for GroupCountFormatter")
}

// FormatGroups formats only group counts.
func (f *GroupCountFormatter) FormatGroups(groups domain.GroupResult, writer io.Writer) error {
	// Groups are already sorted by display name
	for _, group := range groups.Groups {
		_, err := fmt.Fprintf(writer, "Group: %s (%d)\n", group.DisplayName, group.Count)
		if err != nil {
			return err
		}
	}
	return nil
}

// Helper function to convert notification values to pointers.
func notificationsToPointers(notifs []domain.Notification) []*domain.Notification {
	ptrs := make([]*domain.Notification, len(notifs))
	for i := range notifs {
		ptrs[i] = &notifs[i]
	}
	return ptrs
}

// Helper function to get the appropriate formatter.
func GetFormatter(format string, groupCount bool) Formatter {
	formatterType := FormatterType(format)

	// Check if this is a valid formatter type
	valid := false
	for _, ft := range []FormatterType{
		FormatterTypeSimple,
		FormatterTypeLegacy,
		FormatterTypeTable,
		FormatterTypeCompact,
		FormatterTypeJSON,
	} {
		if ft == formatterType {
			valid = true
			break
		}
	}

	// Default to simple formatter for unknown types
	if !valid {
		formatterType = FormatterTypeSimple
	}

	if groupCount {
		return NewGroupCountFormatter(NewFormatter(formatterType))
	}
	return NewFormatter(formatterType)
}
