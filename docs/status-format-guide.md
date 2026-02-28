# Status Format Guide

A comprehensive user guide for the `bd status --format` feature in tmux-intray.

## Overview

The `bd status --format` feature provides powerful, flexible notification status summaries. Whether you need a simple count for your status bar, JSON output for scripting, or custom templates for specific workflows, this feature has you covered.

**Why this feature matters**:
- **Flexibility**: Choose from 6 presets or create unlimited custom templates
- **Integration**: Seamlessly integrates with tmux, shell scripts, monitoring systems, and dashboards
- **Simplicity**: Easy-to-learn template syntax with 13 available variables
- **Performance**: Sub-50ms response time suitable for real-time status bars

## Quick Start

Get up and running in 3 minutes with these basic examples:

### Example 1: Default Output

```bash
bd status
```

**Output**:
```
[3] Build completed successfully
```

This shows the number of active notifications in brackets `[3]` followed by the latest message.

### Example 2: Show in Tmux Status Bar

```bash
# Add to your .tmux.conf
set -g status-right "#(bd status --format=compact) %H:%M"
```

**Result in status bar**:
```
[3] Build completed successfully 21:45
```

### Example 3: Custom Template

```bash
bd status --format='%{unread-count}/%{total-count} notifications'
```

**Output**:
```
3/3 notifications
```

Notice the syntax: `%{variable-name}` with **percent sign and curly braces** (NOT `${}` or other formats).

## Template Syntax

Templates use the `%{variable-name}` syntax for variable substitution.

### Syntax Rules

- **Variable delimiter**: `%{` and `}`
- **Variable names**: Must contain only lowercase letters, numbers, and hyphens: `[a-z0-9-]+`
- **Case-sensitive**: `%{unread-count}` ✅ works, `%{Unread-Count}` ❌ does not
- **Unknown variables**: Silently replaced with empty strings (no error thrown)
- **Escaping**: Literal `%{` and `}` are part of the syntax—no escape needed for custom text

### Example Templates

```bash
# Simple count
%{unread-count}

# Count with label
You have %{unread-count} notifications

# Multiple variables
Active: %{active-count} | Dismissed: %{dismissed-count}

# With text and emojis
📬 %{unread-count} messages, Latest: %{latest-message}

# Severity breakdown
C:%{critical-count} E:%{error-count} W:%{warning-count}
```

## Available Variables (All 13)

### Count Variables

#### `%{unread-count}`
Number of **active** (unread) notifications.

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `3` |
| Use Case | Notification badges, status indicators |

```bash
bd status --format='%{unread-count}'
# Output: 3
```

#### `%{active-count}`
Alias for `%{unread-count}`. Explicit naming for clarity.

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `3` |
| Use Case | Explicit "active" terminology in templates |

```bash
bd status --format='Active: %{active-count}'
# Output: Active: 3
```

#### `%{total-count}`
Alias for `%{unread-count}`. Useful for semantic clarity in summaries.

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `3` |
| Use Case | Summary messages, simple counters |

```bash
bd status --format='Total: %{total-count}'
# Output: Total: 3
```

#### `%{read-count}`
Number of **dismissed** (read) notifications. Note: "read" means "dismissed" in tmux-intray.

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `5` |
| Use Case | Archive size, dismissed items tracking |

```bash
bd status --format='Dismissed: %{read-count}'
# Output: Dismissed: 5
```

#### `%{dismissed-count}`
Number of dismissed notifications (same as `%{read-count}`).

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `5` |
| Use Case | Explicit "dismissed" terminology |

```bash
bd status --format='Archive: %{dismissed-count} items'
# Output: Archive: 5 items
```

### Severity Level Count Variables

These break down active notifications by severity:

#### `%{critical-count}`
Number of **critical** severity notifications.

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `1` |
| Use Case | Alert tracking for critical issues, SRE dashboards |

```bash
bd status --format='🔴 Critical: %{critical-count}'
# Output: 🔴 Critical: 1
```

#### `%{error-count}`
Number of **error** severity notifications.

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `2` |
| Use Case | Build system status, error tracking |

```bash
bd status --format='Build errors: %{error-count}'
# Output: Build errors: 2
```

#### `%{warning-count}`
Number of **warning** severity notifications.

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `3` |
| Use Case | Resource monitoring, deprecation warnings |

```bash
bd status --format='⚠️  Warnings: %{warning-count}'
# Output: ⚠️  Warnings: 3
```

#### `%{info-count}`
Number of **info** severity notifications.

| Property | Value |
|----------|-------|
| Type | Integer (0+) |
| Example | `10` |
| Use Case | Verbose logging, status messages |

```bash
bd status --format='ℹ️  Info: %{info-count}'
# Output: ℹ️  Info: 10
```

### Content Variables

#### `%{latest-message}`
Text of the most recently created **active** notification.

| Property | Value |
|----------|-------|
| Type | String |
| Example | `Build completed successfully` |
| Use Case | Status bar showing latest event, context in summaries |

```bash
bd status --format='Latest: %{latest-message}'
# Output: Latest: Build completed successfully
```

**Note**: Shows only the first line of multi-line messages.

### Boolean Variables

Boolean variables return the string `"true"` or `"false"` for conditional logic:

#### `%{has-unread}`
Returns `"true"` if any active notifications exist, `"false"` otherwise.

| Property | Value |
|----------|-------|
| Type | String: "true" or "false" |
| Example | `true` |
| Use Case | Conditional status display, alert activation |

```bash
bd status --format='Has unread: %{has-unread}'
# Output: Has unread: true
```

#### `%{has-active}`
Alias for `%{has-unread}`. Returns `"true"` if any active notifications exist.

| Property | Value |
|----------|-------|
| Type | String: "true" or "false" |
| Example | `false` |
| Use Case | Explicit "active" terminology |

```bash
bd status --format='Any active: %{has-active}'
# Output: Any active: true
```

#### `%{has-dismissed}`
Returns `"true"` if any dismissed notifications exist, `"false"` otherwise.

| Property | Value |
|----------|-------|
| Type | String: "true" or "false" |
| Example | `true` |
| Use Case | Archive status, cleanup indicators |

```bash
bd status --format='Has archive: %{has-dismissed}'
# Output: Has archive: true
```

### Severity Variable

#### `%{highest-severity}`
Ordinal number representing the **highest severity** among all active notifications.

| Property | Value |
|----------|-------|
| Type | Integer: 1-4 |
| Range | 1=critical, 2=error, 3=warning, 4=info |
| Example | `2` |
| Use Case | Priority indicators, SLA tracking, alert routing |

```bash
bd status --format='Severity: %{highest-severity}'
# Output: Severity: 2  (means at least one error)
```

**Severity ordinals** (lower = more severe):
- `1` - At least one **critical**
- `2` - At least one **error** (no critical)
- `3` - At least one **warning** (no critical/error)
- `4` - All are **info** (or no notifications)

### Session/Window/Pane Variables

These variables identify which tmux sessions, windows, and panes have notifications:

#### `%{session-list}`
Comma-separated list of tmux session names with active notifications.

| Property | Value |
|----------|-------|
| Type | String (comma-separated) |
| Example | `work,dev,build` |
| Status | Reserved for future implementation |
| Current | Returns empty string |

#### `%{window-list}`
Comma-separated list of tmux window IDs with active notifications.

| Property | Value |
|----------|-------|
| Type | String (comma-separated) |
| Example | `1,3,5` |
| Status | Reserved for future implementation |
| Current | Returns empty string |

#### `%{pane-list}`
Comma-separated list of tmux pane IDs with active notifications.

| Property | Value |
|----------|-------|
| Type | String (comma-separated) |
| Example | `0.0,0.1,1.2` |
| Status | Reserved for future implementation |
| Current | Returns empty string |

## Preset Formats (6 Built-in Templates)

Presets are ready-made templates for common use cases. Use `--format=preset-name`.

### 1. `compact` (Default)

**Template**: `[%{unread-count}] %{latest-message}`

**Description**: Single-line format showing count in brackets and latest message.

**Output example**:
```
[3] Build completed successfully
```

**When to use**:
- Status bar display (space-constrained)
- Minimal context needed
- Most common use case

**Usage**:
```bash
bd status
# or explicitly:
bd status --format=compact
```

**In .tmux.conf**:
```bash
set -g status-right "#(bd status) %H:%M"
```

### 2. `detailed`

**Template**: `%{unread-count} unread, %{read-count} read | Latest: %{latest-message}`

**Description**: Shows state breakdown (active and dismissed) with latest message.

**Output example**:
```
3 unread, 5 read | Latest: Build completed successfully
```

**When to use**:
- Want to see both active and dismissed counts
- More space available in status bar
- Full context at a glance

**Usage**:
```bash
bd status --format=detailed
```

### 3. `json`

**Template**: `{"unread":%{unread-count},"total":%{total-count},"message":"%{latest-message}"}`

**Description**: JSON format for programmatic consumption and scripting.

**Output example**:
```json
{"unread":3,"total":3,"message":"Build completed successfully"}
```

**When to use**:
- Shell scripts parsing output
- Integration with tools like `jq`
- Data pipelines and APIs
- Programmatic data handling

**Usage with jq**:
```bash
bd status --format=json | jq '.unread'
# Output: 3

bd status --format=json | jq '.message'
# Output: "Build completed successfully"
```

**Bash parsing**:
```bash
status_json=$(bd status --format=json)
unread=$(echo "$status_json" | jq -r '.unread')
echo "Unread: $unread"
```

### 4. `count-only`

**Template**: `%{unread-count}`

**Description**: Just the notification count, nothing else.

**Output example**:
```
3
```

**When to use**:
- Minimal status indicator
- Piping to other commands
- Simple count-only displays

**Usage**:
```bash
bd status --format=count-only
```

**In .tmux.conf**:
```bash
set -g status-right "Inbox: #(bd status --format=count-only) %H:%M"
```

### 5. `levels`

**Template**: `Severity: %{highest-severity} | Unread: %{unread-count}`

**Description**: Shows highest severity level and unread count.

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
bd status --format=levels
```

### 6. `panes`

**Template**: `%{pane-list} (%{unread-count})`

**Description**: Lists panes with active notifications and count.

**Output example**:
```
0.0 0.1 (3)
```

**When to use**:
- Debugging which panes have notifications
- Multi-pane tracking
- Future enhanced pane tracking

**Usage**:
```bash
bd status --format=panes
```

## Real-World Use Cases

### 1. Simple Status Bar Badge

**Goal**: Show just the count in your tmux status bar.

**Setup**:
```bash
# In .tmux.conf
set -g status-right "Inbox: #(bd status --format=count-only) | %H:%M"
```

**Result**:
```
Inbox: 3 | 21:45
```

### 2. Status Bar with Message

**Goal**: Show count and latest event.

**Setup**:
```bash
# In .tmux.conf
set -g status-right "#(bd status --format=compact) | %H:%M"
```

**Result**:
```
[3] Build completed successfully | 21:45
```

### 3. Severity-Based Alert Display

**Goal**: Show different indicators based on notification severity.

**Script** (`~/.local/bin/bd-status-alert`):
```bash
#!/bin/bash

severity=$(bd status --format='%{highest-severity}')
count=$(bd status --format='%{unread-count}')

case "$severity" in
  1) echo "🔴 CRITICAL: $count";;
  2) echo "🟠 ERROR: $count";;
  3) echo "🔵 WARNING: $count";;
  *) echo "✅ OK";;
esac
```

**Setup in .tmux.conf**:
```bash
set -g status-right "#(~/.local/bin/bd-status-alert) | %H:%M"
```

### 4. Severity Breakdown

**Goal**: Show count breakdown by severity level.

**Command**:
```bash
bd status --format='C:%{critical-count} E:%{error-count} W:%{warning-count} I:%{info-count}'
```

**Output**:
```
C:1 E:2 W:3 I:4
```

### 5. Conditional Display (Archive Status)

**Goal**: Show archive info only when dismissed items exist.

**Script** (`~/.local/bin/bd-status-archive`):
```bash
#!/bin/bash

active=$(bd status --format='%{unread-count}')
dismissed=$(bd status --format='%{dismissed-count}')

if [ "$dismissed" -gt 0 ]; then
  echo "Active: $active | Archive: $dismissed"
else
  echo "Active: $active"
fi
```

### 6. Shell Alias for Quick Check

**Goal**: Quick access to notification status from command line.

**Setup** (in `~/.bashrc` or `~/.zshrc`):
```bash
# Quick status check
alias bd-status='bd status --format=detailed'

# Just the count
alias bd-count='bd status --format=count-only'

# JSON for scripting
alias bd-json='bd status --format=json'
```

**Usage**:
```bash
bd-status
# Output: 3 unread, 2 read | Latest: Build completed successfully

bd-count
# Output: 3
```

### 7. Cron Job Notification

**Goal**: Alert when critical notifications exist.

**Script** (`~/.local/bin/bd-cron-alert`):
```bash
#!/bin/bash

critical=$(bd status --format='%{critical-count}')
if [ "$critical" -gt 0 ]; then
  echo "ALERT: $critical critical notifications" | mail -s "Critical Alert" you@example.com
fi
```

**Setup in crontab**:
```bash
*/5 * * * * ~/.local/bin/bd-cron-alert
```

### 8. Desktop Notification

**Goal**: Show desktop notification when critical items appear.

**Script** (`~/.local/bin/bd-desktop-notify`):
```bash
#!/bin/bash

critical=$(bd status --format='%{critical-count}')
error=$(bd status --format='%{error-count}')

if [ "$critical" -gt 0 ]; then
  notify-send "Critical Alert" "$critical critical notifications"
elif [ "$error" -gt 0 ]; then
  notify-send "Error Alert" "$error error notifications"
fi
```

### 9. Monitoring Dashboard

**Goal**: Expose status as JSON for monitoring tools.

**Script** (`~/.local/bin/bd-status-api`):
```bash
#!/bin/bash

# Serve status JSON
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
status=$(bd status --format=json)

# Wrap in API response
cat <<EOF
{
  "timestamp": "$timestamp",
  "status": $status,
  "severity": "$(bd status --format='%{highest-severity}')",
  "has_critical": "$(bd status --format='%{critical-count}' | grep -q '^[1-9]' && echo true || echo false)"
}
EOF
```

### 10. Watch Real-Time Updates

**Goal**: Monitor status changes in real-time.

**Command**:
```bash
watch -n 2 "bd status --format=detailed"
```

This updates every 2 seconds, showing live notification counts.

## Integration Examples

### Tmux Configuration

Add to your `.tmux.conf`:

```bash
# Simple count display
set -g status-right "Inbox: #(bd status --format=count-only) %H:%M"

# With latest message
set -g status-right "#(bd status --format=compact) %H:%M"

# Refresh interval (seconds)
set -g status-interval 2

# More space for longer output
set -g status-right-length 100
```

### Shell Alias/Function

Add to `~/.bashrc` or `~/.zshrc`:

```bash
# Quick aliases
alias bd-status='bd status --format=detailed'
alias bd-count='bd status --format=count-only'
alias bd-alert='bd status --format="%{critical-count}:%{error-count}"'

# Function with conditional logic
bd-check() {
  local critical=$(bd status --format='%{critical-count}')
  if [ "$critical" -gt 0 ]; then
    echo "⚠️  $critical critical issues!"
    return 1
  else
    echo "✅ All clear"
    return 0
  fi
}
```

### Bash Monitoring Script

Create `~/.local/bin/bd-monitor`:

```bash
#!/bin/bash
set -e

# Get status
status=$(bd status --format=json)
unread=$(echo "$status" | jq -r '.unread // 0')
message=$(echo "$status" | jq -r '.message // ""')

# Check for alerts
if [ "$unread" -gt 0 ]; then
  echo "🔔 $(date '+%Y-%m-%d %H:%M:%S'): $unread notifications"
  echo "   Latest: $message"

  # Optional: log to file
  echo "$(date '+%Y-%m-%d %H:%M:%S'): $unread - $message" >> ~/.local/state/bd-monitor.log
else
  echo "✅ $(date '+%Y-%m-%d %H:%M:%S'): All clear"
fi
```

Make executable:
```bash
chmod +x ~/.local/bin/bd-monitor
```

### Python Integration

```python
#!/usr/bin/env python3
import subprocess
import json

def get_status():
    """Get notification status as dictionary."""
    result = subprocess.run(
        ['bd', 'status', '--format=json'],
        capture_output=True,
        text=True
    )
    return json.loads(result.stdout)

# Usage
status = get_status()
print(f"Unread: {status['unread']}")
print(f"Message: {status['message']}")

# Check for alerts
if status['unread'] > 0:
    print("Alert: New notifications!")
```

### Desktop Notification Example

Create `~/.local/bin/bd-notify`:

```bash
#!/bin/bash

# Get current status
critical=$(bd status --format='%{critical-count}')
error=$(bd status --format='%{error-count}')
latest=$(bd status --format='%{latest-message}')

# Determine urgency
if [ "$critical" -gt 0 ]; then
  urgency="critical"
  title="🔴 CRITICAL"
  body="$critical critical notifications"
elif [ "$error" -gt 0 ]; then
  urgency="normal"
  title="🟠 ERROR"
  body="$error errors"
else
  exit 0  # No alert needed
fi

# Send notification
notify-send -u "$urgency" "$title" "$body: $latest"
```

Setup cron to run every 5 minutes:
```bash
*/5 * * * * ~/.local/bin/bd-notify
```

## Error Handling & Troubleshooting

### Common Mistakes

#### ❌ Wrong Syntax: Using `${}` instead of `%{}`

**Incorrect**:
```bash
bd status --format='You have ${unread-count} notifications'
# Output: You have  notifications  (empty variable)
```

**Correct**:
```bash
bd status --format='You have %{unread-count} notifications'
# Output: You have 3 notifications
```

#### ❌ Variable Name Typo

**Incorrect**:
```bash
bd status --format='%{unread_count}'  # underscore instead of hyphen
# Output: (empty string)
```

**Correct**:
```bash
bd status --format='%{unread-count}'  # hyphen
# Output: 3
```

#### ❌ Case Sensitivity

**Incorrect**:
```bash
bd status --format='%{Unread-Count}'  # uppercase
# Output: (empty string)
```

**Correct**:
```bash
bd status --format='%{unread-count}'  # lowercase
# Output: 3
```

### Debug Tips

**1. Test individual variables**:
```bash
bd status --format='%{unread-count}'
bd status --format='%{latest-message}'
bd status --format='%{critical-count}'
```

**2. Check if notifications exist**:
```bash
bd list --active
```

**3. View full help text**:
```bash
bd status --help
```

**4. Test JSON output**:
```bash
bd status --format=json | jq .
```

**5. Verify variable names**:
```bash
# These 13 are all valid:
bd status --format='%{unread-count} %{active-count} %{total-count} %{read-count} %{dismissed-count}'
bd status --format='%{critical-count} %{error-count} %{warning-count} %{info-count}'
bd status --format='%{latest-message} %{has-unread} %{has-active} %{has-dismissed} %{highest-severity}'
```

### Error Messages

**"tmux not running"**:
```bash
# Solution: Start tmux first
tmux new-session -d -s work

# Then try again
bd status
```

**Empty output**:
```bash
# Check if there are notifications
bd list --active

# If empty, add a test notification
bd add "Test notification"
```

**Template substitution error**:
```bash
# Check for mismatched braces
bd status --format='%{unread-count'  # Missing closing }
# Error: mismatched variable delimiters

# Fix by adding closing brace
bd status --format='%{unread-count}'
```

## Environment Variables

### `TMUX_INTRAY_STATUS_FORMAT`

Set a default format without needing `--format` flag every time.

**Syntax**:
```bash
export TMUX_INTRAY_STATUS_FORMAT="compact"
```

**Values**: Any preset name or custom template.

**Examples**:
```bash
# Use preset
export TMUX_INTRAY_STATUS_FORMAT="detailed"

# Use custom template
export TMUX_INTRAY_STATUS_FORMAT="C:%{critical-count} E:%{error-count}"

# Verify it works
bd status  # Uses env var instead of 'compact' default
```

**Precedence** (CLI flag overrides env):
1. `--format` flag (if provided) - highest priority
2. `TMUX_INTRAY_STATUS_FORMAT` env var
3. `"compact"` default

**Example**:
```bash
export TMUX_INTRAY_STATUS_FORMAT="detailed"

bd status
# Uses: detailed (from env var)

bd status --format=json
# Uses: json (from CLI flag, overrides env)
```

**Persistence**:

Add to `~/.bashrc` or `~/.zshrc` to persist across sessions:
```bash
# ~/.bashrc
export TMUX_INTRAY_STATUS_FORMAT="compact"
```

## Summary

The `bd status --format` feature provides:

✅ **13 template variables** for comprehensive status reporting
✅ **6 presets** for common use cases
✅ **Custom templates** for unlimited flexibility
✅ **Real-time updates** suitable for status bars
✅ **JSON output** for programmatic consumption
✅ **Environment variable** support for defaults

**Next steps**:
1. Try the [Quick Start](#quick-start) examples
2. Explore the [Real-World Use Cases](#real-world-use-cases)
3. Check [Integration Examples](#integration-examples) for your workflow
4. Reference the [Available Variables](#available-variables-all-13) for detailed information

**More help**:
- See [docs/status-format-reference.md](status-format-reference.md) for technical details
- See [docs/examples/status-format-examples.md](examples/status-format-examples.md) for more runnable examples
- Run `bd status --help` for built-in command help
