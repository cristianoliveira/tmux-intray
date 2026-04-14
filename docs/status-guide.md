# Status Format Guide

Complete guide to the `tmux-intray status --format` feature for displaying notification summaries with custom templates.

## Quick Start

```bash
# Default compact format
tmux-intray status
# Output: [3] Build completed successfully

# Detailed format
tmux-intray status --format=detailed
# Output: 3 unread, 5 read | Latest: Build completed successfully

# Custom template
tmux-intray status --format='You have {{unread-count}} notifications'
# Output: You have 3 notifications

# JSON output for scripting
tmux-intray status --format=json | jq '.unread'
# Output: 3
```

## Template Syntax

Templates use `{{variable-name}}` syntax for variable substitution.

**Rules:**
- **Opening delimiter**: `{{`
- **Closing delimiter**: `}}`
- **Variable names**: Lowercase letters, numbers, and hyphens only: `[a-z0-9-]+`
- **Case-sensitive**: `{{unread-count}}` ✅ works, `{{Unread-Count}}` ❌ does not
- **Unknown variables**: Returns error with list of available variables

**Examples:**
```bash
# Simple count
tmux-intray status --format='{{unread-count}}'

# Multiple variables
tmux-intray status --format='Active: {{active-count}} | Dismissed: {{dismissed-count}}'

# With text
tmux-intray status --format='📬 {{unread-count}} messages, Latest: {{latest-message}}'

# Severity breakdown
tmux-intray status --format='C:{{critical-count}} E:{{error-count}} W:{{warning-count}}'
```

## Variables Reference

The status formatter exposes 13 core variables plus 4 severity-count variables.

### Count Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `{{unread-count}}` | Integer | Active (unread) notifications | `3` |
| `{{active-count}}` | Integer | Alias for unread-count | `3` |
| `{{total-count}}` | Integer | Alias for unread-count | `3` |
| `{{read-count}}` | Integer | Dismissed notifications | `5` |
| `{{dismissed-count}}` | Integer | Dismissed notifications | `5` |

### Severity Count Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `{{critical-count}}` | Integer | Critical severity notifications | `1` |
| `{{error-count}}` | Integer | Error severity notifications | `2` |
| `{{warning-count}}` | Integer | Warning severity notifications | `3` |
| `{{info-count}}` | Integer | Info severity notifications | `10` |

### Content Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `{{latest-message}}` | String | Latest active notification message | `Build completed` |

### Boolean Variables

| Variable | Type | Values | Description | Example |
|----------|------|--------|-------------|---------|
| `{{has-unread}}` | String | "true", "false" | Has active notifications | `true` |
| `{{has-active}}` | String | "true", "false" | Has active notifications (alias) | `true` |
| `{{has-dismissed}}` | String | "true", "false" | Has dismissed notifications | `false` |

### Severity Variable

| Variable | Type | Values | Description | Example |
|----------|------|--------|-------------|---------|
| `{{highest-severity}}` | Integer | 1-4 | Highest severity ordinal | `2` |

**Severity mapping** (lower = more severe):
- `1` - At least one **critical**
- `2` - At least one **error** (no critical)
- `3` - At least one **warning** (no critical/error)
- `4` - All are **info** (or no notifications)

### Session/Window/Pane Variables

These variables are recognized by the formatter. Today they return empty strings in normal template rendering:

| Variable | Current behavior |
|----------|------------------|
| `{{session-list}}` | Empty string |
| `{{window-list}}` | Empty string |
| `{{pane-list}}` | Empty string |

## Presets

Presets are built-in templates for common use cases. Use `--format=preset-name`.

### 1. `compact` (Default)

**Template**: `[{{unread-count}}] {{latest-message}}`

**Output**: `[3] Build completed successfully`

**Usage**: `tmux-intray status` or `tmux-intray status --format=compact`

### 2. `detailed`

**Template**: `{{unread-count}} unread, {{read-count}} read | Latest: {{latest-message}}`

**Output**: `3 unread, 5 read | Latest: Build completed successfully`

**Usage**: `tmux-intray status --format=detailed`

### 3. `json`

**Special format** - Returns structured JSON with all counts and pane breakdown.

> Note: although `json` also exists in the template preset registry, the `status` command handles `json` as a dedicated special format before template expansion.

**Output example**:
```json
{
  "active": 107,
  "info": 77,
  "warning": 3,
  "error": 27,
  "critical": 0,
  "panes": {
    "$10:@48:%85": 1,
    "$2:@6:%36": 4
  }
}
```

**Usage**: `tmux-intray status --format=json | jq '.active'`

**Note**: This is a special format (not a template) that returns complete status data.

### 4. `count-only`

**Template**: `{{unread-count}}`

**Output**: `3`

**Usage**: `tmux-intray status --format=count-only`

### 5. `levels`

**Special format** - Shows counts for each severity level.

> Note: although `levels` also exists in the template preset registry, the `status` command handles it as a dedicated special format before template expansion.

**Output example**:
```
info:77
warning:3
error:29
critical:0
```

**Usage**: `tmux-intray status --format=levels`

**Note**: This is a special format (not a template). For severity with highest level, use custom template:
```bash
tmux-intray status --format='Severity: {{highest-severity}} | Unread: {{unread-count}}'
```

### 6. `panes`

**Special format** - Shows pane counts sorted by pane identifier.

> Note: although `panes` also exists in the template preset registry, the `status` command handles it as a dedicated special format before template expansion.

**Output example**:
```
$2:@6:%36:4
$2:@9:%38:6
$10:@48:%85:1
```

**Usage**: `tmux-intray status --format=panes`

**Note**: This is a special format (not a template) that lists all panes with active notifications.

## Real-World Use Cases

### Use Case 1: Simple Status Bar Badge

Show just the unread count in your tmux status bar.

**Setup in `.tmux.conf`:**
```bash
set -g status-right "Inbox: #(tmux-intray status --format=count-only) %H:%M"
```

**Result**: `Inbox: 3 21:45`

### Use Case 2: Status Bar with Latest Message

Show count and latest notification.

**Setup in `.tmux.conf`:**
```bash
set -g status-right "#(tmux-intray status --format=compact) | %H:%M"
```

**Result**: `[3] Build completed successfully | 21:45`

### Use Case 3: Color-Coded Severity Badge

Display different colors based on highest severity level.

**Script** (`~/.tmux/status-alert.sh`):**
```bash
#!/bin/bash
severity=$(tmux-intray status --format='{{highest-severity}}')
case "$severity" in
  1) echo "#[fg=red,bold]🔴 CRITICAL#[default]";;
  2) echo "#[fg=yellow,bold]🟠 ERROR#[default]";;
  3) echo "#[fg=blue]🔵 WARNING#[default]";;
  *) echo "#[fg=green]✅ CLEAR#[default]";;
esac
```

**Setup in `.tmux.conf`:**
```bash
set -g status-right "#(~/.tmux/status-alert.sh) | %H:%M"
```

### Use Case 4: Severity Breakdown

Show count breakdown by severity level.

**Command:**
```bash
tmux-intray status --format='C:{{critical-count}} E:{{error-count}} W:{{warning-count}} I:{{info-count}}'
```

**Output**: `C:1 E:2 W:3 I:4`

### Use Case 5: Conditional Display (Archive Status)

Show archive information only when dismissed items exist.

**Script** (`~/.local/bin/status-archive.sh`):**
```bash
#!/bin/bash
active=$(tmux-intray status --format='{{unread-count}}')
dismissed=$(tmux-intray status --format='{{dismissed-count}}')

if [ "$dismissed" -gt 0 ]; then
  echo "Active: $active | Archive: $dismissed"
else
  echo "Active: $active"
fi
```

### Use Case 6: Shell Aliases

Quick access to notification status from command line.

**Add to `~/.bashrc` or `~/.zshrc`:**
```bash
# Quick aliases
alias inbox='tmux-intray status --format=detailed'
alias inbox-count='tmux-intray status --format=count-only'
alias inbox-json='tmux-intray status --format=json'

# Function with color output
inbox-check() {
  local count=$(tmux-intray status --format=count-only)
  if [ "$count" -gt 0 ]; then
    echo "📬 $(tput setaf 1)$count notifications$(tput sgr0)"
  else
    echo "✅ All clear"
  fi
}
```

### Use Case 7: Monitoring Script with Exit Codes

Check status and return exit codes for automation.

**Script** (`~/.local/bin/inbox-check-status`):**
```bash
#!/bin/bash
# Exit 0: OK (no notifications)
# Exit 1: WARNING (has unread)
# Exit 2: CRITICAL (has critical)

CRITICAL=$(tmux-intray status --format='{{critical-count}}')
UNREAD=$(tmux-intray status --format='{{unread-count}}')

if [ "$CRITICAL" -gt 0 ]; then
  echo "CRITICAL: $CRITICAL critical notifications" >&2
  exit 2
elif [ "$UNREAD" -gt 0 ]; then
  echo "WARNING: $UNREAD unread notifications" >&2
  exit 1
else
  echo "OK: All clear" >&2
  exit 0
fi
```

### Use Case 8: Cron Job Notification

Monitor notifications via cron and send alerts.

**Script** (`~/.local/bin/inbox-cron-monitor`):**
```bash
#!/bin/bash
ALERT_FILE="$HOME/.local/state/inbox-last-alert"
CRITICAL=$(tmux-intray status --format='{{critical-count}}')

if [ "$CRITICAL" -gt 0 ]; then
  if [ ! -f "$ALERT_FILE" ] || [ "$(cat "$ALERT_FILE")" != "$CRITICAL" ]; then
    echo "Critical notifications: $CRITICAL" | mail -s "Inbox Alert" "$USER"
    echo "$CRITICAL" > "$ALERT_FILE"
  fi
else
  rm -f "$ALERT_FILE"
fi
```

**Add to crontab** (every 5 minutes):
```bash
*/5 * * * * ~/.local/bin/inbox-cron-monitor
```

### Use Case 9: Desktop Notification

Show desktop notifications for alerts.

**Script** (`~/.local/bin/inbox-notify`):**
```bash
#!/bin/bash
CRITICAL=$(tmux-intray status --format='{{critical-count}}')
ERROR=$(tmux-intray status --format='{{error-count}}')
LATEST=$(tmux-intray status --format='{{latest-message}}')

if [ "$CRITICAL" -gt 0 ]; then
  notify-send -u critical "🔴 CRITICAL" "$CRITICAL critical notifications\n$LATEST"
elif [ "$ERROR" -gt 0 ]; then
  notify-send -u normal "🟠 ERROR" "$ERROR errors\n$LATEST"
fi
```

### Use Case 10: Real-Time Monitoring

Watch status changes in terminal.

```bash
watch -n 2 'tmux-intray status --format=detailed'
```

## Tmux Configuration Examples

Add to your `.tmux.conf`:

```bash
# Simple count display
set -g status-right "Inbox: #(tmux-intray status --format=count-only) %H:%M"

# With latest message
set -g status-right "#(tmux-intray status --format=compact) %H:%M"

# Refresh interval (seconds)
set -g status-interval 2

# More space for longer output
set -g status-right-length 100
```

## Error Handling & Troubleshooting

### "Unknown variable" Error

**Symptom**: Command fails with unknown variable error

**Cause**: Variable name has typo or doesn't exist

**Solution**: The error message includes a complete list of available variables:
```bash
tmux-intray status --format='{{unred-count}}'
# Error: unknown variable: unred-count
# Available variables: unread-count, read-count, ...
```

### Variable Shows as Empty

**Symptom**: Template shows empty output

**Cause**: No active notifications exist

**Solution**: Check if notifications exist:
```bash
tmux-intray list --active
```

### JSON Parsing Fails

**Symptom**: `jq` fails to parse output

**Cause**: Using wrong format or output contains newlines

**Solution**: Ensure using json preset:
```bash
tmux-intray status --format=json | jq .
```

### Status Bar Updates Too Slowly

**Symptom**: Status bar shows stale data

**Cause**: Default tmux refresh rate is 15 seconds

**Solution**: Increase refresh rate in `.tmux.conf`:
```bash
set -g status-interval 2  # Update every 2 seconds
```

## Environment Variable

### TMUX_INTRAY_STATUS_FORMAT

Set a default format without needing `--format` flag every time.

**Usage:**
```bash
export TMUX_INTRAY_STATUS_FORMAT="detailed"
```

**Values**: Any preset name or custom template.

**Precedence**:
1. `--format` flag (highest priority)
2. `TMUX_INTRAY_STATUS_FORMAT` env var
3. `"compact"` default

**Example:**
```bash
export TMUX_INTRAY_STATUS_FORMAT="detailed"

tmux-intray status
# Uses: detailed (from env var)

tmux-intray status --format=json
# Uses: json (from CLI flag, overrides env)
```

## Performance

- **Response time**: < 50ms for typical databases
- **Template overhead**: < 1ms for templates with 5 variables
- **Status bar updates**: Recommended interval of 2+ seconds

**Best practices:**
- Use simple formats like `count-only` for high-frequency updates
- Cache JSON output when using multiple values
- Status interval of 2+ seconds to reduce query frequency

## Command Reference

```
tmux-intray status [OPTIONS]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | String | `compact` | Output format: preset name or custom template |
| `-h, --help` | Flag | - | Show help message |

**Exit codes:**
- `0` - Success
- `1` - Error (tmux not running, invalid template, database error)

## Summary

The `status` command provides flexible notification summary formatting through:

- **13 template variables** for counts, content, and state
- **6 presets** for common use cases
- **Custom templates** with `{{variable-name}}` syntax
- **Real-time updates** suitable for status bars
- **JSON output** for programmatic consumption
- **Environment variable** support for defaults

**Start with presets, customize with templates as needed.**
