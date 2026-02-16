package format

import (
	"fmt"
	"io"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// TableConfig holds configuration for table formatting.
type TableConfig struct {
	// ShowHeaders determines whether to show column headers.
	ShowHeaders bool

	// HeaderColor is the color to use for headers.
	HeaderColor string

	// ColumnWidths defines the width for each column.
	ColumnWidths map[string]int

	// ColumnAlignments defines the alignment for each column (left, right, center).
	ColumnAlignments map[string]string
}

// DefaultTableConfig returns a default table configuration.
func DefaultTableConfig() *TableConfig {
	return &TableConfig{
		ShowHeaders: true,
		HeaderColor: colors.Blue,
		ColumnWidths: map[string]int{
			"ID":      4,
			"Date":    23,
			"Message": 32,
			"Session": 10,
			"Window":  10,
			"Pane":    10,
			"Level":   8,
		},
		ColumnAlignments: map[string]string{
			"ID":    "right",
			"Date":  "left",
			"Level": "left",
		},
	}
}

// TableColumn represents a column in a table.
type TableColumn struct {
	// Name is the column name displayed in the header.
	Name string

	// Width is the column width in characters.
	Width int

	// Alignment is the text alignment (left, right, center).
	Alignment string

	// Extractor extracts the value from a notification.
	Extractor func(*domain.Notification) string
}

// ExtendedTableFormatter extends the basic table formatter with more columns.
type ExtendedTableFormatter struct {
	config  *TableConfig
	columns []TableColumn
}

// NewExtendedTableFormatter creates a new ExtendedTableFormatter with default columns.
func NewExtendedTableFormatter() *ExtendedTableFormatter {
	config := DefaultTableConfig()
	columns := []TableColumn{
		{
			Name:      "ID",
			Width:     config.ColumnWidths["ID"],
			Alignment: config.ColumnAlignments["ID"],
			Extractor: func(n *domain.Notification) string {
				return formatIntToString(n.ID, config.ColumnWidths["ID"], config.ColumnAlignments["ID"])
			},
		},
		{
			Name:      "Date",
			Width:     config.ColumnWidths["Date"],
			Alignment: config.ColumnAlignments["Date"],
			Extractor: func(n *domain.Notification) string {
				return formatString(n.Timestamp, config.ColumnWidths["Date"], config.ColumnAlignments["Date"])
			},
		},
		{
			Name:      "Level",
			Width:     config.ColumnWidths["Level"],
			Alignment: config.ColumnAlignments["Level"],
			Extractor: func(n *domain.Notification) string {
				return formatString(n.Level.String(), config.ColumnWidths["Level"], config.ColumnAlignments["Level"])
			},
		},
		{
			Name:  "Message",
			Width: config.ColumnWidths["Message"],
			Extractor: func(n *domain.Notification) string {
				return truncateString(n.Message, config.ColumnWidths["Message"])
			},
		},
	}
	return &ExtendedTableFormatter{
		config:  config,
		columns: columns,
	}
}

// WithColumns adds custom columns to the formatter.
func (f *ExtendedTableFormatter) WithColumns(columns ...TableColumn) *ExtendedTableFormatter {
	f.columns = append(f.columns, columns...)
	return f
}

// FormatNotifications formats notifications in an extended table format.
func (f *ExtendedTableFormatter) FormatNotifications(notifications []*domain.Notification, writer io.Writer) error {
	if len(notifications) == 0 {
		return nil
	}

	// Write header if enabled
	if f.config.ShowHeaders {
		err := f.writeHeader(writer)
		if err != nil {
			return err
		}
	}

	// Write separator
	err := f.writeSeparator(writer)
	if err != nil {
		return err
	}

	// Write rows
	for _, n := range notifications {
		err := f.writeRow(n, writer)
		if err != nil {
			return err
		}
	}

	return nil
}

// FormatGroups formats grouped notifications in an extended table format.
func (f *ExtendedTableFormatter) FormatGroups(groups domain.GroupResult, writer io.Writer) error {
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

// writeHeader writes the table header.
func (f *ExtendedTableFormatter) writeHeader(writer io.Writer) error {
	reset := colors.Reset
	for i, col := range f.columns {
		header := formatString(col.Name, col.Width, "left")
		if i == 0 {
			_, err := fmt.Fprintf(writer, "%s%s%s", f.config.HeaderColor, header, reset)
			if err != nil {
				return err
			}
		} else {
			_, err := fmt.Fprintf(writer, "  %s", header)
			if err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(writer)
	return err
}

// writeSeparator writes the table separator.
func (f *ExtendedTableFormatter) writeSeparator(writer io.Writer) error {
	reset := colors.Reset
	for i, col := range f.columns {
		separator := makeSeparator(col.Width)
		if i == 0 {
			_, err := fmt.Fprintf(writer, "%s%s%s", f.config.HeaderColor, separator, reset)
			if err != nil {
				return err
			}
		} else {
			_, err := fmt.Fprintf(writer, "  %s", separator)
			if err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(writer)
	return err
}

// writeRow writes a single table row.
func (f *ExtendedTableFormatter) writeRow(notification *domain.Notification, writer io.Writer) error {
	for i, col := range f.columns {
		value := col.Extractor(notification)
		if i > 0 {
			_, err := fmt.Fprintf(writer, "  %s", value)
			if err != nil {
				return err
			}
		} else {
			_, err := fmt.Fprintf(writer, "%s", value)
			if err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(writer)
	return err
}

// Helper functions

// formatIntToString formats an integer to a string with the specified width and alignment.
func formatIntToString(i int, width int, alignment string) string {
	s := fmt.Sprintf("%d", i)
	return formatString(s, width, alignment)
}

// formatString formats a string with the specified width and alignment.
func formatString(s string, width int, alignment string) string {
	if len(s) >= width {
		return s[:width]
	}

	switch alignment {
	case "right":
		return strings.Repeat(" ", width-len(s)) + s
	case "center":
		left := (width - len(s)) / 2
		right := width - len(s) - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // left
		return s + strings.Repeat(" ", width-len(s))
	}
}

// truncateString truncates a string to the specified width, adding "..." if truncated.
func truncateString(s string, width int) string {
	if len(s) <= width {
		return s + strings.Repeat(" ", width-len(s))
	}
	if width < 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

// makeSeparator creates a separator line of the specified width.
func makeSeparator(width int) string {
	return strings.Repeat("-", width)
}
