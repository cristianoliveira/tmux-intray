# Status Command Guide

The `tmux-intray status` command displays notification summaries with powerful template-based formatting. This guide covers all variables, presets, and real-world use cases.

## Quick Start

```bash
# Default compact format: [3] Build completed successfully
tmux-intray status

# Detailed format: 3 unread, 5 read | Latest: Build completed successfully
tmux-intray status --format=detailed

# Custom template: "You have 3 notifications"
tmux-intray status --format='You have ${unread-count} notifications'

# JSON output for scripting
tmux-intray status --format=json
```

## Template Syntax

Templates use `${variable-name}` syntax for variable substitution. Variables are resolved to their current values based on your notification database state.

### Valid Characters
- Variables must contain lowercase letters, numbers, and hyphens: `[a-z0-9-]+`
- Variables are case-sensitive: `${unread-count}` works, `${Unread-Count}` does not
- Unknown variables produce an error with a list of available variables

### Example Template
```
You have ${unread-count} active + ${read-count} read = ${total-count} total
```

Output:
```
You have 3 active + 5 read = 3 total
```

## Migration Note (v0.4.0+)

**Behavior Change**: Previously, unknown template variables were silently replaced with empty strings. 

**Current Behavior**: Templates with unknown variables now produce an error that includes a list of all available variables.

**Migration Steps**:
1. Test your existing custom templates - they will now fail if they contain typos
2. Check error messages for the exact list of available variables
3. Update any templates with incorrect variable names

Example of new error output:
```
Error: template substitution error: unknown variable: unead-count

Available variables:
  unread-count
  read-count
  dismissed-count
  ...
```

## Variables Reference

### Count Variables

#### `${unread-count}`
Number of **active** (unread) notifications.

**Use case**: Notification badge on status bar
```bash
# Terminal status bar showing count
tmux-intray status --format='📬 ${unread-count}'

# Output: 📬 3
```

#### `${active-count}`
Alias for `${unread-count}`. Both refer to active notifications.

**Use case**: Explicit naming in complex templates
```bash
tmux-intray status --format='Active: ${active-count} | Dismissed: ${dismissed-count}'
# Output: Active: 3 | Dismissed: 2
```

#### `${total-count}`
Alias for `${unread-count}`. Useful for semantic clarity.

**Use case**: Summary counts
```bash
tmux-intray status --format='Total notifications: ${total-count}'
# Output: Total notifications: 3
```

#### `${read-count}`
Number of **dismissed** (read) notifications. Note: "read" in tmux-intray means "dismissed" internally.

**Use case**: Archive status
```bash
tmux-intray status --format='${read-count} dismissed items'
# Output: 5 dismissed items
```

#### `${dismissed-count}`
Number of dismissed notifications (same as `${read-count}`).

**Use case**: Tracking cleared items
```bash
tmux-intray status --format='Dismissed: ${dismissed-count}'
# Output: Dismissed: 5
```

### Severity Level Count Variables

These break down notifications by severity level (critical, error, warning, info).

#### `${critical-count}`
Number of **critical** severity notifications.

**Use case**: Alert tracking for critical issues
```bash
tmux-intray status --format='🔴 Critical: ${critical-count}'
# Output: 🔴 Critical: 2
```

#### `${error-count}`
Number of **error** severity notifications.

**Use case**: Build system status
```bash
tmux-intray status --format='Build errors: ${error-count}'
# Output: Build errors: 1
```

#### `${warning-count}`
Number of **warning** severity notifications.

**Use case**: Resource monitoring
```bash
tmux-intray status --format='⚠️  Warnings: ${warning-count}'
# Output: ⚠️  Warnings: 3
```

#### `${info-count}`
Number of **info** severity notifications.

**Use case**: Verbose logging
```bash
tmux-intray status --format='ℹ️  Info messages: ${info-count}'
# Output: ℹ️  Info messages: 10
```

### Content Variables

#### `${latest-message}`
Text of the most recently created **active** notification.

**Use case**: Status bar showing the most important message
```bash
tmux-intray status --format='Latest: ${latest-message}'
# Output: Latest: Build completed successfully
```

If no active notifications exist, returns empty string:
```bash
# Output: Latest: 
```

**Important**: The message is truncated to the first line of the notification. Multi-line messages show only the first line.

### Boolean Variables

Boolean variables return the string `"true"` or `"false"` for conditional formatting.

#### `${has-unread}`
Returns `"true"` if any active notifications exist, `"false"` otherwise.

**Use case**: Conditional status display
```bash
tmux-intray status --format='Has unread: ${has-unread}'
# Output: Has unread: true
```

#### `${has-active}`
Alias for `${has-unread}`. Returns `"true"` if any active notifications exist.

**Use case**: Explicit naming
```bash
tmux-intray status --format='Active items: ${has-active}'
# Output: Active items: true
```

#### `${has-dismissed}`
Returns `"true"` if any dismissed notifications exist, `"false"` otherwise.

**Use case**: Archive status
```bash
tmux-intray status --format='Archive exists: ${has-dismissed}'
# Output: Archive exists: true
```

### Severity Variable

#### `${highest-severity}`
Ordinal number representing the highest severity level among all **active** notifications:

| Value | Level    | Meaning      |
|-------|----------|--------------|
| 1     | critical | At least one critical |
| 2     | error    | At least one error (no critical) |
| 3     | warning  | At least one warning (no critical/error) |
| 4     | info     | All are info (or no notifications) |

**Use case**: Priority indicator for alerts
```bash
tmux-intray status --format='Severity: ${highest-severity}'
# Output: Severity: 2  (at least one error)
```

**Color coding example** (with tmux color codes):
```bash
# Using tmux color codes for visual severity
tmux-intray status --format='#[fg=red]Sev:${highest-severity}#[fg=default]'
```

### Session/Window/Pane Variables

These variables list the tmux sessions, windows, and panes containing active notifications. Currently implemented as placeholders for future functionality.

#### `${session-list}`
Comma-separated list of tmux session names with active notifications.

**Current behavior**: Returns empty string (reserved for future implementation)

**Planned use case**:
```bash
tmux-intray status --format='Active sessions: ${session-list}'
# Future output: Active sessions: work, dev, build
```

#### `${window-list}`
Comma-separated list of tmux window IDs with active notifications.

**Current behavior**: Returns empty string (reserved for future implementation)

**Planned use case**:
```bash
tmux-intray status --format='Active windows: ${window-list}'
# Future output: Active windows: 1, 3, 5
```

#### `${pane-list}`
Comma-separated list of tmux pane IDs with active notifications.

**Current behavior**: Returns empty string (reserved for future implementation)

**Current preset usage**:
```bash
tmux-intray status --format=panes
# Output: (0)  [will improve when pane-list is fully implemented]
```

## Presets Reference

Presets are built-in templates for common use cases. Use `--format=preset-name` to apply them.

### 1. `compact` (Default)

**Template**: `[${unread-count}] ${latest-message}`

**Description**: Single-line format showing count and latest message. Perfect for status bars.

**Output example**:
```
[3] Build completed successfully
```

**When to use**:
- Status bar display in tmux
- Minimal output needed
- Space constraints

**Usage**:
```bash
tmux-intray status
# or explicitly:
tmux-intray status --format=compact
```

### 2. `detailed`

**Template**: `${unread-count} unread, ${read-count} read | Latest: ${latest-message}`

**Description**: Multi-line format showing state breakdown and latest message.

**Output example**:
```
3 unread, 5 read | Latest: Build completed successfully
```

**When to use**:
- Terminal status bar with more space
- Want to see archive status
- Need full context at a glance

**Usage**:
```bash
tmux-intray status --format=detailed
```

### 3. `json`

**Template**: `{"unread":${unread-count},"total":${total-count},"message":"${latest-message}"}`

**Description**: JSON format for programmatic consumption by scripts and tools.

**Output example**:
```json
{"unread":3,"total":3,"message":"Build completed successfully"}
```

**When to use**:
- Shell scripts parsing output
- Integration with other tools
- Data pipelines

**Usage**:
```bash
tmux-intray status --format=json | jq '.unread'
# Output: 3
```

**Parsing in bash**:
```bash
#!/bin/bash
status_json=$(tmux-intray status --format=json)
unread_count=$(echo "$status_json" | jq -r '.unread')
echo "Unread: $unread_count"
```

### 4. `count-only`

**Template**: `${unread-count}`

**Description**: Just the notification count, nothing else.

**Output example**:
```
3
```

**When to use**:
- Minimal status indicator
- Piping to other commands
- Scriptable output
- Integration with status bar generators

**Usage**:
```bash
tmux-intray status --format=count-only
```

**In a status bar**:
```bash
# .tmux.conf
set -g status-right "Inbox: #(tmux-intray status --format=count-only) %H:%M"
```

### 5. `levels`

**Template**: `Severity: ${highest-severity} | Unread: ${unread-count}`

**Description**: Shows the highest severity level and unread count for priority-based monitoring.

**Output example**:
```
Severity: 2 | Unread: 3
```

**When to use**:
- SRE/Ops dashboards
- Alert severity monitoring
- Priority-based workflows

**Usage**:
```bash
tmux-intray status --format=levels
```

### 6. `panes`

**Template**: `${pane-list} (${unread-count})`

**Description**: Lists panes with active notifications and the count.

**Output example**:
```
work.0 work.1 (3)
```

**Note**: Currently shows limited pane information. Will improve when pane tracking is fully implemented.

**When to use**:
- Debugging which panes have notifications
- Multi-pane status tracking
- Future enhanced pane tracking

**Usage**:
```bash
tmux-intray status --format=panes
```

## Real-World Use Cases

### Use Case 1: Simple Status Bar Badge

Show just the unread count in your tmux status bar.

**Setup in `.tmux.conf`**:
```bash
set -g status-right "Inbox: #(tmux-intray status --format=count-only) %H:%M"
```

**Output**:
```
Inbox: 3 21:45
```

**Why this works**:
- `count-only` preset gives just the number
- Minimal output for status bar space
- Updates in real-time with status bar refresh

### Use Case 2: Status Bar with Latest Message

Show both count and the most recent notification.

**Setup in `.tmux.conf`**:
```bash
set -g status-right "#(tmux-intray status --format=compact) | %H:%M"
```

**Output**:
```
[3] Build completed successfully | 21:45
```

**Why this works**:
- `compact` preset provides context
- Users see what's most recent at a glance
- Stays readable in terminal width

### Use Case 3: Color-Coded Severity Badge

Display different colors based on the highest severity level.

**Template**:
```bash
tmux-intray status --format='#{if-shell \
  "test $(tmux-intray status --format=json | jq .unread) -gt 0", \
  "#[bg=red,fg=white]Alert: ${critical-count}#[default]", \
  "#[fg=green]Clear#[default]"}'
```

**Alternative (simpler)**:
```bash
# Use custom script to parse severity and apply colors
cat > ~/.tmux/scripts/inbox-color.sh << 'EOF'
#!/bin/bash
severity=$(tmux-intray status --format=json | jq -r '.highest_severity // 4')
case "$severity" in
  1) echo "#[bg=red,fg=white]CRITICAL${unread-count}#[default]";;
  2) echo "#[bg=yellow,fg=black]ERROR${unread-count}#[default]";;
  3) echo "#[bg=blue,fg=white]WARN${unread-count}#[default]";;
  *) echo "#[fg=green]OK#[default]";;
esac
EOF
chmod +x ~/.tmux/scripts/inbox-color.sh
```

### Use Case 4: Multi-Level Summary

Show breakdown of notifications by severity in a verbose format.

**Template**:
```bash
tmux-intray status --format='C:${critical-count} E:${error-count} W:${warning-count} I:${info-count}'
```

**Output**:
```
C:1 E:2 W:3 I:4
```

**Why this works**:
- Users see full severity breakdown
- Easy to scan for critical issues
- Fits in status bar or scripts

### Use Case 5: Conditional Display (Archive Status)

Show archive information only when dismissed items exist.

**Template**:
```bash
tmux-intray status --format='Active: ${unread-count}${if-shell \
  "test $(tmux-intray status --format=json | jq .read_count) -gt 0", \
  " | Archived: ${read-count}", ""}'
```

**Outputs**:
```
# When no dismissed items:
Active: 3

# When dismissed items exist:
Active: 3 | Archived: 5
```

### Use Case 6: Integration with tmux-right-status

Use status command in a more complex status bar setup.

**`.tmux.conf` setup**:
```bash
# Refresh status bar every 2 seconds
set -g status-interval 2

# Use custom formatting for status-right
set -g status-right-length 100
set -g status-right '#(~/.config/tmux-intray/status-bar.sh) | %H:%M:%S'
```

**`~/.config/tmux-intray/status-bar.sh`**:
```bash
#!/bin/bash

# Get status JSON
status=$(tmux-intray status --format=json)

# Extract values
unread=$(echo "$status" | jq -r '.unread')
message=$(echo "$status" | jq -r '.message' | cut -c1-20)

# Format output
if [ "$unread" -gt 0 ]; then
  echo "📬 $unread: $message"
else
  echo "📭 No notifications"
fi
```

## Troubleshooting

### "Command not found" for tmux-intray

**Symptoms**: `tmux-intray: command not found` when running `tmux-intray status`

**Solutions**:
1. Verify installation: `which tmux-intray`
2. Check PATH includes installation directory: `echo $PATH | grep -q ~/.local/bin && echo "OK" || echo "Missing"`
3. Reinstall if needed: `go install github.com/cristianoliveira/tmux-intray@latest`

### Status command output is empty

**Symptoms**: `tmux-intray status` returns no output or error

**Solutions**:
1. Ensure tmux is running: `tmux list-sessions`
2. Check if notifications exist: `tmux-intray list --active`
3. Verify database file: `ls -l ~/.local/state/tmux-intray/notifications.db`

### Template error with unknown variable

**Symptoms**: Command fails with "unknown variable" error

**Possible causes**:
- Variable name has typo (case-sensitive: `{{unread-count}}` not `{{Unread-Count}}`)
- Variable doesn't exist (all available variables are listed in the error)

**Solution**: The error message includes a complete list of available variables:
```bash
# Example error output:
Error: template substitution error: unknown variable: unead-count

Available variables:
  unread-count
  read-count
  dismissed-count
  latest-message
  ...
```

### Variable shows as empty string

**Symptoms**: Template like `{{latest-message}}` shows empty output

**Possible causes**:
- No active notifications (check `tmux-intray list --active`)

**Solution**: Verify notifications exist and are active:
```bash
tmux-intray list --active
```

### JSON output not valid JSON

**Symptoms**: Trying to parse output with `jq` fails with parse error

**Causes**:
- Using wrong preset (not `--format=json`)
- Output contains literal newlines in message

**Solutions**:
```bash
# Make sure to use json preset
tmux-intray status --format=json | jq .

# If still failing, check actual output
tmux-intray status --format=json | od -c  # see special characters
```

### Tmux color codes not working

**Symptoms**: Status bar shows literal text like `#[fg=red]` instead of colors

**Causes**:
- Using `#[...]` syntax outside tmux context
- Tmux color codes in wrong format

**Solutions**:
```bash
# In .tmux.conf, use raw color codes
set -g status-right "#(tmux-intray status --format=count-only)"

# For colored output, use tmux helper script
tmux set-option -g status-right "Inbox: #(tmux-intray status --format=count-only)"

# Or handle colors in shell script, not in template
cat > ~/.config/tmux/inbox.sh << 'EOF'
#!/bin/bash
count=$(tmux-intray status --format=count-only)
if [ "$count" -gt 0 ]; then
  echo "#[bg=red,fg=white]$count#[default]"
else
  echo "#[fg=green]OK#[default]"
fi
EOF
```

### Status bar updates too slowly

**Symptoms**: Status bar shows stale data, doesn't update in real-time

**Causes**:
- Default tmux status refresh rate is 15 seconds
- Running expensive commands in status bar

**Solutions**:
```bash
# In .tmux.conf, increase refresh rate
set -g status-interval 2  # Update every 2 seconds

# Cache results to avoid hammering status command
set -g @tmux_intray_format "count-only"  # Simplest format
```

## Advanced Examples

### Example 1: Scriptable Status JSON with Error Handling

```bash
#!/bin/bash
set -e

# Get status with timeout
status_json=$(timeout 2 tmux-intray status --format=json 2>/dev/null || echo '{"unread":0,"total":0,"message":""}')

# Parse safely
unread=$(echo "$status_json" | jq -r '.unread // 0')
latest=$(echo "$status_json" | jq -r '.message // ""' | head -c 30)

# Output
if [ "$unread" -gt 0 ]; then
  echo "🔔 $unread: $latest"
else
  echo "✓ All clear"
fi
```

### Example 2: Custom Severity Alert

```bash
#!/bin/bash

# Only alert if critical notifications exist
critical=$(tmux-intray status --format='${critical-count}')
if [ "$critical" -gt 0 ]; then
  # Send alert (email, Slack, etc.)
  notify-send "Critical notifications: $critical"
  
  # Log for monitoring
  echo "$(date): Critical alert triggered" >> ~/.local/state/tmux-intray/alerts.log
fi
```

### Example 3: Dynamic Status Color

```bash
# Color-coded status based on severity
get_status_color() {
  local severity=$(tmux-intray status --format='${highest-severity}')
  case "$severity" in
    1) echo "#[bg=red,fg=white]🔴 CRITICAL#[default]";;
    2) echo "#[bg=yellow,fg=black]🟠 ERROR#[default]";;
    3) echo "#[bg=blue,fg=white]🔵 WARNING#[default]";;
    *) echo "#[fg=green]✅ NORMAL#[default]";;
  esac
}

# Use in .tmux.conf
# set -g status-right "#(get_status_color) %H:%M"
```

## Performance Considerations

- Status command is optimized to respond in **< 50ms** for typical databases
- Each invocation queries the SQLite database once
- Templates with many variables have minimal performance impact
- For high-frequency updates (< 1 second), consider caching JSON output

## Format String Limits

- Help text is truncated at ~500 characters when displayed
- For full documentation, see this guide
- Maximum template length: No practical limit (tested up to 1MB)
- Variable names must be lowercase alphanumeric + hyphens

## Summary

The `status` command provides flexible notification summary formatting through:
- **13 template variables** for counts, content, and state
- **6 presets** for common use cases  
- **Custom templates** for specific needs
- **Real-time updates** suitable for status bars and scripts
- **JSON output** for programmatic consumption

Start with presets, customize with templates as needed.
