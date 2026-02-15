# Format Package

The `format` package provides output formatting functionality for CLI commands in the tmux-intray project. It includes formatters for different output styles and notification display.

## Features

- Multiple output formats: simple, legacy, table, compact, and JSON
- Support for grouped notifications
- Extensible table formatting with custom columns
- Proper truncation of long messages
- Consistent interface for all formatters

## Usage

### Basic Usage

```go
import "github.com/cristianoliveira/tmux-intray/internal/format"

// Create a formatter
formatter := format.NewSimpleFormatter()

// Format notifications
var buf bytes.Buffer
err := formatter.FormatNotifications(notifications, &buf)
if err != nil {
    // Handle error
}
```

### Using Different Formats

```go
// Simple format: ID DATE - Message
simpleFormatter := format.NewSimpleFormatter()

// Legacy format: only messages
legacyFormatter := format.NewLegacyFormatter()

// Table format: headers and columns
tableFormatter := format.NewTableFormatter()

// Compact format: only messages
compactFormatter := format.NewCompactFormatter()

// JSON format: structured output
jsonFormatter := format.NewJSONFormatter()
```

### Grouped Notifications

```go
// Group notifications by level
groups := domain.GroupNotifications(notifications, domain.GroupByLevel)

// Format groups
err := formatter.FormatGroups(groups, &buf)
```

### Extended Table Formatting

```go
// Create an extended table formatter
formatter := format.NewExtendedTableFormatter()

// Add custom columns
formatter.WithColumns(format.TableColumn{
    Name:      "Session",
    Width:     10,
    Alignment: "left",
    Extractor: func(n *domain.Notification) string {
        return format.formatString(n.Session, 10, "left")
    },
})
```

### Getting Formatters by Type

```go
// Get formatter by type name
formatter := format.GetFormatter("simple", false)

// Get group count formatter
groupCountFormatter := format.GetFormatter("simple", true)
```

## Format Details

### Simple Format
- Displays ID, timestamp, and message
- Message truncated to 50 characters
- Format: `ID DATE - Message`

### Legacy Format
- Displays only messages, one per line
- No truncation
- Original format for backward compatibility

### Table Format
- Displays headers and columns
- Message truncated to 32 characters
- Color-coded headers
- Format: `ID DATE - Message`

### Compact Format
- Displays only messages
- Message truncated to 60 characters
- Minimal output for quick scanning

### JSON Format
- Structured JSON output
- Full notification data
- No truncation
- Ideal for programmatic processing

## Extensibility

The format package is designed to be extensible:

1. **Custom Formatters**: Implement the `Formatter` interface
2. **Extended Tables**: Add custom columns with `WithColumns()`
3. **Group Counters**: Use `NewGroupCountFormatter` for count-only output

## Examples

See the `formatter_test.go` and `table_test.go` files for more detailed examples.