# Hook Migration Guide for Go Binary

## Issue
Hooks that reference `bin/tmux-intray` or `tmux-intray.sh` are failing because these files no longer exist in the Go implementation.

## Solution

### Option 1: Use tmux-intray from PATH (Recommended)
If tmux-intray is installed in your PATH, simply call `tmux-intray` directly:
```bash
#!/bin/bash
# Old way
# bin/tmux-intray list

# New way
tmux-intray list
```

### Option 2: Use TMUX_INTRAY_BINARY environment variable
The Go implementation now provides the binary path as `TMUX_INTRAY_BINARY`:
```bash
#!/bin/bash
# Use the environment variable
if [ -n "$TMUX_INTRAY_BINARY" ]; then
    "$TMUX_INTRAY_BINARY" list
else
    # Fallback to PATH
    tmux-intray list
fi
```

### Option 3: Find the binary dynamically
```bash
#!/bin/bash
# Find the binary
TMUX_INTRAY_CMD=""
if [ -n "$TMUX_INTRAY_BINARY" ]; then
    TMUX_INTRAY_CMD="$TMUX_INTRAY_BINARY"
elif command -v tmux-intray >/dev/null 2>&1; then
    TMUX_INTRAY_CMD="tmux-intray"
fi

if [ -n "$TMUX_INTRAY_CMD" ]; then
    "$TMUX_INTRAY_CMD" list
else
    echo "Error: tmux-intray not found"
    exit 1
fi
```

## Common Hook Patterns

### Listing notifications
```bash
#!/bin/bash
# List all active notifications
if [ -n "$TMUX_INTRAY_BINARY" ]; then
    "$TMUX_INTRAY_BINARY" list --state active
else
    tmux-intray list --state active
fi
```

### Adding a notification from a hook
```bash
#!/bin/bash
# Add a notification from a hook
if [ -n "$TMUX_INTRAY_BINARY" ]; then
    "$TMUX_INTRAY_BINARY" add "Hook triggered: $HOOK_POINT"
else
    tmux-intray add "Hook triggered: $HOOK_POINT"
fi
```

### Dismissing notifications
```bash
#!/bin/bash
# Dismiss all notifications older than 1 day
if [ -n "$TMUX_INTRAY_BINARY" ]; then
    "$TMUX_INTRAY_BINARY" cleanup --days 1
else
    tmux-intray cleanup --days 1
fi
```

## Testing Your Hooks

To test if your hooks work with the new implementation:

1. Create a test hook:
```bash
#!/bin/bash
echo "Hook executed at $(date)"
echo "TMUX_INTRAY_BINARY: $TMUX_INTRAY_BINARY"

# Test if the binary works
if [ -n "$TMUX_INTRAY_BINARY" ]; then
    echo "Testing TMUX_INTRAY_BINARY:"
    "$TMUX_INTRAY_BINARY" version
elif command -v tmux-intray >/dev/null 2>&1; then
    echo "Testing tmux-intray from PATH:"
    tmux-intray version
else
    echo "Error: Cannot find tmux-intray binary"
fi
```

2. Make it executable:
```bash
chmod +x ~/.config/tmux-intray/hooks/pre-add/test.sh
```

3. Test it:
```bash
tmux-intray add "Test hook"
```

## Environment Variables Available to Hooks

The Go implementation provides these environment variables to hooks:
- `HOOK_POINT`: The hook point being executed (e.g., "pre-add", "post-add")
- `HOOK_TIMESTAMP`: ISO 8601 timestamp of when the hook was triggered
- `TMUX_INTRAY_HOOKS_FAILURE_MODE`: How to handle hook failures (abort, warn, ignore)
- `TMUX_INTRAY_BINARY`: Path to the tmux-intray binary that triggered the hook
- `NOTIFICATION_ID`: ID of the notification (for add/dismiss hooks)
- `LEVEL`: Notification level (info, warning, error, critical)
- `MESSAGE`: The notification message
- `TIMESTAMP`: Notification timestamp
- `SESSION`, `WINDOW`, `PANE`: Tmux context (if applicable)
- `PANE_CREATED`: Pane creation timestamp (if applicable)

## Migration Checklist

- [ ] Replace `bin/tmux-intray` with `tmux-intray` or `"$TMUX_INTRAY_BINARY"`
- [ ] Make sure all hooks are executable (`chmod +x`)
- [ ] Test each hook individually
- [ ] Check that hooks can find the tmux-intray binary
- [ ] Verify hooks work with the new environment variables