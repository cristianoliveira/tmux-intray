# Tmux Integration Design

## Overview

This document describes the Tmux Abstraction Layer architecture in tmux-intray, which provides a clean, testable interface for interacting with tmux.

## Architecture

### Design Goals

1. **Abstraction**: Provide a clean API that hides the complexity of tmux command execution
2. **Testability**: Make tmux interactions mockable for unit testing
3. **Maintainability**: Centralize tmux-related logic in one place
4. **Consistency**: Ensure all code uses the same patterns for tmux interactions

### Package Structure

```
internal/
├── core/
│   ├── tmux.go        # Low-level tmux abstraction (tmuxRunner, core functions)
│   ├── core.go        # High-level tray management functions
│   └── tmux_test.go   # Tests with mockable tmuxRunner
```

## Core Components

### 1. tmuxRunner - The Abstraction Foundation

The `tmuxRunner` variable is the single point of contact for all tmux command execution:

```go
// internal/core/tmux.go
var tmuxRunner = func(args ...string) (string, string, error) {
    cmd := exec.Command("tmux", args...)
    var stdout, stderr strings.Builder
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    err := cmd.Run()
    return stdout.String(), stderr.String(), err
}
```

**Key Features**:
- **Mockable**: Can be replaced in tests
- **Centralized**: All tmux commands go through this one function
- **Error handling**: Captures both stdout and stderr

**Testing Example**:
```go
func TestSomeFunction(t *testing.T) {
    // Save original
    origRunner := tmuxRunner
    defer func() { tmuxRunner = origRunner }()

    // Replace with mock
    tmuxRunner = func(args ...string) (string, string, error) {
        if args[0] == "has-session" {
            return "", "", nil
        }
        return "", "", fmt.Errorf("unexpected command: %v", args)
    }

    // Test function that uses tmuxRunner
    result := EnsureTmuxRunning()
    require.True(t, result)
}
```

### 2. Low-Level Functions (tmux.go)

These functions directly use `tmuxRunner` and provide basic tmux operations:

| Function | Purpose | Returns |
|----------|---------|---------|
| `EnsureTmuxRunning()` | Check if tmux session exists | `bool` |
| `GetCurrentTmuxContext()` | Get current session/window/pane IDs | `TmuxContext` |
| `ValidatePaneExists(sessionID, windowID, paneID)` | Check if a specific pane exists | `bool` |
| `JumpToPane(sessionID, windowID, paneID)` | Navigate to a specific pane | `bool` |
| `GetTmuxVisibility()` | Get TMUX_INTRAY_VISIBLE variable | `string` ("0" or "1") |
| `SetTmuxVisibility(value)` | Set TMUX_INTRAY_VISIBLE variable | `bool` |

#### TmuxContext Structure

```go
type TmuxContext struct {
    SessionID   string // e.g., "$3"
    WindowID    string // e.g., "@16"
    PaneID      string // e.g., "%21"
    PaneCreated string // e.g., "8443" (pane PID)
}
```

### 3. High-Level Functions (core.go)

These functions provide more convenient APIs for common operations:

| Function | Purpose | Returns |
|----------|---------|---------|
| `GetVisibility()` | Wrapper for GetTmuxVisibility | `string` ("0" or "1") |
| `SetVisibility(visible)` | Wrapper for SetTmuxVisibility | `error` |
| `GetTrayItems(stateFilter)` | Get tray items by state | `string` (newline-separated messages) |
| `AddTrayItem(...)` | Add notification with auto-context | `(string, error)` (ID) |
| `ClearTrayItems()` | Dismiss all active items | `error` |

## Usage Patterns

### Pattern 1: Auto-Context Detection

When adding notifications, automatically detect current tmux context:

```go
// Get current tmux context
ctx := core.GetCurrentTmuxContext()

// Add notification with auto-detected context
id, err := core.AddTrayItem(
    "My notification",
    "", "", "", "", // No manual context
    false,          // Enable auto-context
    "info",         // Level
)
```

**Implementation Details**:
- Uses tmux format string: `#{session_id} #{window_id} #{pane_id} #{pane_pid}`
- Parses 4 fields from tmux output
- Validates all fields are non-empty (Power of 10 Rule 5)
- Returns empty TmuxContext on error

### Pattern 2: Conditional Tmux Operations

Always check if tmux is running before operations:

```go
if !core.EnsureTmuxRunning() {
    colors.Error("tmux not running")
    return fmt.Errorf("tmux not available")
}

// Safe to perform tmux operations now
```

**Implementation Details**:
- Uses `tmux has-session` command
- Returns false if tmux not available or not running
- Logs debug output for troubleshooting

### Pattern 3: Safe Pane Navigation

Validate pane exists before attempting to jump:

```go
// First validate pane exists
if !core.ValidatePaneExists(sessionID, windowID, paneID) {
    colors.Warning("Pane no longer exists")
}

// Then attempt to jump (function handles fallback)
if !core.JumpToPane(sessionID, windowID, paneID) {
    colors.Error("Failed to jump to pane")
}
```

**Implementation Details**:
- `ValidatePaneExists()` uses `tmux list-panes -t session:window -F #{pane_id}`
- `JumpToPane()` validates pane, then calls `tmux select-window` and `tmux select-pane`
- Falls back to window if pane doesn't exist
- Returns false on complete failure

### Pattern 4: Visibility Management

Use global tmux environment variables for cross-command state:

```go
// Get current visibility
visible := core.GetVisibility() // "0" or "1"

// Toggle visibility
newVisible := (visible != "1")
if err := core.SetVisibility(newVisible); err != nil {
    return err
}
```

**Implementation Details**:
- Uses `tmux show-environment -g TMUX_INTRAY_VISIBLE`
- Uses `tmux set-environment -g TMUX_INTRAY_VISIBLE <value>`
- Returns "0" as default if variable not set
- Global scope means all commands can access the value

## Message Escaping

Messages are escaped to handle special characters in TSV format:

```go
// Escape function
func escapeMessage(msg string) string {
    msg = strings.ReplaceAll(msg, "\\", "\\\\")  // Escape backslashes first
    msg = strings.ReplaceAll(msg, "\t", "\\t")    // Escape tabs
    msg = strings.ReplaceAll(msg, "\n", "\\n")    // Escape newlines
    return msg
}

// Unescape function
func unescapeMessage(msg string) string {
    msg = strings.ReplaceAll(msg, "\\n", "\n")    // Unescape newlines first
    msg = strings.ReplaceAll(msg, "\\t", "\t")    // Unescape tabs
    msg = strings.ReplaceAll(msg, "\\\\", "\\")   // Unescape backslashes
    return msg
}
```

**Order matters**: Backslashes must be escaped first, then special characters.

## Error Handling

### Standard Error Handling Pattern

```go
stdout, stderr, err := tmuxRunner("command", "args")
if err != nil {
    colors.Error("Command failed: " + err.Error())
    if stderr != "" {
        colors.Debug("stderr: " + stderr)
    }
    return fmt.Errorf("operation failed: %w", err)
}
```

### Error Types

| Error Type | When It Happens | How to Handle |
|------------|-----------------|---------------|
| Tmux not running | `tmux has-session` fails | Return error with clear message |
| Invalid format | tmux output unexpected | Return empty context/zero value |
| Pane not found | `ValidatePaneExists` returns false | Log warning, fall back to window |
| Permission denied | Can't write tmux environment | Return error with details |

## Testing Strategy

### Unit Testing with Mocked tmuxRunner

```go
func TestJumpToPane(t *testing.T) {
    // Save original
    origRunner := tmuxRunner
    defer func() { tmuxRunner = origRunner }()

    tests := []struct {
        name         string
        sessionID    string
        windowID     string
        paneID       string
        mockOutput   string
        wantSuccess  bool
    }{
        {
            name:        "successful jump",
            sessionID:   "$3",
            windowID:    "@16",
            paneID:      "%21",
            mockOutput:  "%21",
            wantSuccess: true,
        },
        {
            name:        "pane not found",
            sessionID:   "$3",
            windowID:    "@16",
            paneID:      "%99",
            mockOutput:  "%21\n%22",
            wantSuccess: true, // Falls back to window
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Set up mock
            tmuxRunner = func(args ...string) (string, string, error) {
                switch args[0] {
                case "list-panes":
                    return tt.mockOutput, "", nil
                case "select-window":
                    return "", "", nil
                case "select-pane":
                    return "", "", nil
                }
                return "", "", fmt.Errorf("unexpected command: %v", args)
            }

            // Test function
            result := JumpToPane(tt.sessionID, tt.windowID, tt.paneID)
            require.Equal(t, tt.wantSuccess, result)
        })
    }
}
```

### Integration Testing

Integration tests run in actual tmux sessions (see `tests/` directory):

```bash
# Start tmux session
tmux new-session -d -s test_session

# Run commands in tmux context
tmux send-keys -t test_session "tmux-intray add 'test'" Enter

# Verify storage
tmux-intray list --format=table

# Cleanup
tmux kill-session -t test_session
```

## Migration Guide

### When NOT to Use exec.Command

❌ **Direct exec.Command calls (avoid)**:
```go
cmd := exec.Command("tmux", "has-session")
if err := cmd.Run(); err != nil {
    return err
}
```

✅ **Use core functions (correct)**:
```go
if !core.EnsureTmuxRunning() {
    return fmt.Errorf("tmux not running")
}
```

### When to Add New Core Functions

Consider adding a new core function when:
1. The operation is used in multiple places
2. The operation is complex (requires parsing tmux output)
3. The operation should be mockable for testing
4. The operation involves multiple tmux commands

**Example**: Adding session name lookup

```go
// In internal/core/tmux.go

// GetSessionName returns the name of a session.
func GetSessionName(sessionID string) string {
    stdout, stderr, err := tmuxRunner("display-message", "-t", sessionID, "-p", "#S")
    if err != nil {
        colors.Debug("tmux display-message failed: " + err.Error())
        if stderr != "" {
            colors.Debug("stderr: " + stderr)
        }
        return sessionID // fallback to session ID
    }
    return strings.TrimSpace(stdout)
}
```

### When to Keep Direct exec.Command

It's acceptable to use `exec.Command("tmux", ...)` directly when:
1. **It's inside internal/core/tmux.go** - This is the core implementation
2. **It's in TUI-specific helpers** - These are already mockable (variable assignment)
3. **It's in test files** - Tests need direct access for setup/teardown

## Performance Considerations

### Minimize Tmux Calls

Each `tmuxRunner` call spawns a subprocess, which has overhead:

❌ **Multiple calls (slow)**:
```go
ctx := core.GetCurrentTmuxContext()  // Call 1
name := core.GetSessionName(ctx.SessionID)  // Call 2
windows := core.ListWindows(ctx.SessionID)  // Call 3
```

✅ **Single call with format string (fast)**:
```go
// Get everything in one call
format := "#{session_id}\t#{session_name}\t#{window_count}"
stdout, _, _ := tmuxRunner("list-sessions", "-F", format)
// Parse output to get all needed data
```

### Caching

Consider caching session/window names in TUI to avoid repeated lookups:

```go
// In TUI code
type tuiModel struct {
    sessionNames map[string]string // Cache of session names
}

func (m *tuiModel) getSessionName(sessionID string) string {
    if name, ok := m.sessionNames[sessionID]; ok {
        return name // Return cached value
    }
    // Fetch and cache
    name := fetchSessionName(sessionID)
    m.sessionNames[sessionID] = name
    return name
}
```

## Security Considerations

### Command Injection Prevention

The `tmuxRunner` uses variadic args which are properly escaped by Go's `exec.Command`:

```go
// Safe - args are properly escaped
tmuxRunner("display-message", "-t", userProvidedSessionID, "-p", "#S")

// Unsafe - don't do this (vulnerable to injection)
cmdStr := fmt.Sprintf("tmux display-message -t %s -p #S", userInput)
exec.Command("sh", "-c", cmdStr)
```

### Input Validation

Validate session/window/pane IDs before use:

```go
// Validate format
if !strings.HasPrefix(sessionID, "$") {
    return fmt.Errorf("invalid session ID format: %s", sessionID)
}

// Ensure non-empty
if sessionID == "" || windowID == "" || paneID == "" {
    return fmt.Errorf("missing session, window, or pane ID")
}
```

## Future Enhancements

### Potential Improvements

1. **Formal TmuxClient Interface**
   - Create `type TmuxClient interface` for better dependency injection
   - Enable easier testing with multiple implementations

2. **Additional Core Functions**
   - `GetSessionName(sessionID string) string`
   - `ListSessionNames() map[string]string`
   - `SetTmuxStatusOption(name, value string) error`

3. **Connection Pooling**
   - Reuse tmux server connections for better performance
   - Reduce subprocess overhead

4. **Async Operations**
   - Run multiple tmux commands concurrently using goroutines
   - Aggregate results

## Conclusion

The Tmux Abstraction Layer provides a clean, testable, and maintainable way to interact with tmux. By following the patterns and guidelines in this document, developers can:

- ✅ Write code that is easy to test
- ✅ Avoid common pitfalls with tmux integration
- ✅ Maintain consistency across the codebase
- ✅ Keep tmux-related logic centralized
- ✅ Mock tmux interactions for unit tests

For questions or contributions, please refer to:
- [DEVELOPMENT.md](../../DEVELOPMENT.md) - Development workflow
- [CLI Reference](./cli/CLI_REFERENCE.md) - Command usage examples
- [Testing Strategy](../testing/testing-strategy.md) - Testing guidelines
