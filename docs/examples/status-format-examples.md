# Status Format Examples

Copy-paste ready examples for `bd status --format`.

## Table of Contents

1. [Tmux Integration](#tmux-integration) (5 examples)
2. [Shell Integration](#shell-integration) (5 examples)
3. [Monitoring & Dashboard](#monitoring--dashboard) (5 examples)
4. [Advanced](#advanced) (5 examples)

---

## Tmux Integration

Examples for integrating status format into your tmux configuration.

### 1. Simple Count in Status Bar

**Goal**: Show just the notification count in your tmux status bar.

**Setup** (add to `.tmux.conf`):
```bash
set -g status-right "Inbox: #(bd status --format=count-only) | %H:%M"
```

**Result**:
```
Inbox: 3 | 21:45
```

**How it works**:
- `count-only` preset outputs just `3`
- Updates each time tmux redraws status bar
- Minimal output, minimal overhead

**Prerequisites**:
- `bd` command in PATH
- Tmux 3.0+

---

### 2. Compact Format with Message

**Goal**: Show count and latest notification message in status bar.

**Setup** (add to `.tmux.conf`):
```bash
set -g status-right "#(bd status --format=compact) | %H:%M"
```

**Result**:
```
[3] Build completed successfully | 21:45
```

**How it works**:
- `compact` preset shows `[count] message`
- Provides context about latest notification
- Still fits in typical status bar

**Tip**: Adjust `status-right-length` if message gets truncated:
```bash
set -g status-right-length 100
```

---

### 3. Color-Coded by Severity

**Goal**: Show different colors based on highest severity level.

**Script** (create `~/.tmux/status-alert.sh`):
```bash
#!/bin/bash

# Get current severity
severity=$(bd status --format='%{highest-severity}')

# Map to colors
case "$severity" in
  1) echo "#[fg=red,bold]🔴 CRITICAL#[default]";;
  2) echo "#[fg=yellow,bold]🟠 ERROR#[default]";;
  3) echo "#[fg=blue]🔵 WARNING#[default]";;
  *) echo "#[fg=green]✅ CLEAR#[default]";;
esac
```

**Make executable**:
```bash
chmod +x ~/.tmux/status-alert.sh
```

**Setup** (add to `.tmux.conf`):
```bash
set -g status-right "#(~/.tmux/status-alert.sh) | %H:%M"
```

**Result**:
```
🔴 CRITICAL | 21:45    (if critical exists)
🟠 ERROR | 21:45       (if error exists)
🔵 WARNING | 21:45     (if warning exists)
✅ CLEAR | 21:45       (otherwise)
```

**How it works**:
- Gets highest severity level (1-4)
- Maps to visual indicators
- Updates in real-time

**Customization**:
- Change colors with tmux color codes
- Add different emoji
- Adjust status-interval for refresh rate:
  ```bash
  set -g status-interval 2  # Refresh every 2 seconds
  ```

---

### 4. Severity Summary in Status Bar

**Goal**: Show breakdown of notifications by severity level.

**Setup** (add to `.tmux.conf`):
```bash
set -g status-right "#(bd status --format='C:%{critical-count} E:%{error-count} W:%{warning-count}') | %H:%M"
```

**Result**:
```
C:1 E:2 W:3 | 21:45
```

**How it works**:
- Shows count for each severity level
- Compact format, easy to scan
- Lets you see full breakdown at glance

**Alternative** (with more detail):
```bash
set -g status-right "#(bd status --format='C:%{critical-count}|E:%{error-count}|W:%{warning-count}|I:%{info-count}') | %H:%M"
```

**Result**:
```
C:1|E:2|W:3|I:4 | 21:45
```

---

### 5. Alert When Critical Exists

**Goal**: Show alert badge only when critical notifications exist.

**Script** (create `~/.tmux/critical-alert.sh`):
```bash
#!/bin/bash

# Get critical count
critical=$(bd status --format='%{critical-count}')

# Show alert only if critical > 0
if [ "$critical" -gt 0 ]; then
  echo "⚠️  CRITICAL: $critical"
else
  echo ""  # Empty string when no critical
fi
```

**Make executable**:
```bash
chmod +x ~/.tmux/critical-alert.sh
```

**Setup** (add to `.tmux.conf`):
```bash
set -g status-right "#(~/.tmux/critical-alert.sh) #(bd status --format=compact) | %H:%M"
```

**Result when critical exists**:
```
⚠️  CRITICAL: 1 [3] Build completed successfully | 21:45
```

**Result when no critical**:
```
[3] Build completed successfully | 21:45
```

**How it works**:
- Runs custom script to check critical count
- Shows alert only when needed
- Reduces clutter when no critical items

**Refinement**:
```bash
#!/bin/bash

critical=$(bd status --format='%{critical-count}')
total=$(bd status --format='%{unread-count}')

if [ "$critical" -gt 0 ]; then
  echo "#[bg=red,fg=white] ⚠️  $critical/$total #[default]"
else
  echo ""
fi
```

---

## Shell Integration

Examples for using status format in shell scripts and aliases.

### 1. Quick Status Alias

**Goal**: Create convenient shell aliases for common status checks.

**Setup** (add to `~/.bashrc` or `~/.zshrc`):
```bash
# Quick aliases
alias bd-status='bd status --format=detailed'
alias bd-count='bd status --format=count-only'
alias bd-json='bd status --format=json'
alias bd-alert='bd status --format="C:%{critical-count} E:%{error-count}"'

# Function with color output
bd-quick() {
  local count=$(bd status --format=count-only)
  if [ "$count" -gt 0 ]; then
    echo "📬 $(tput setaf 1)$count notifications$(tput sgr0)"
  else
    echo "✅ All clear"
  fi
}
```

**Usage**:
```bash
bd-status
# Output: 3 unread, 2 read | Latest: Build completed successfully

bd-count
# Output: 3

bd-json | jq .unread
# Output: 3

bd-quick
# Output: 📬 3 notifications  (in red)
```

**How it works**:
- Aliases provide quick access to formats
- Function can add logic and colors
- Saves time on repeated checks

---

### 2. Conditional Check Function

**Goal**: Run actions based on notification status.

**Script** (add to `~/.bashrc` or `~/.zshrc`):
```bash
# Check if any critical items, return exit code
bd-has-critical() {
  local critical=$(bd status --format='%{critical-count}')
  [ "$critical" -gt 0 ]
}

# Check if any notifications exist
bd-has-unread() {
  [ "$(bd status --format=count-only)" -gt 0 ]
}

# Usage in scripts
if bd-has-critical; then
  echo "Critical issues exist!"
  # Send alert, log, etc.
fi

# In command chains
bd-has-unread && echo "You have notifications" || echo "All clear"
```

**Usage**:
```bash
if bd-has-critical; then
  notify-send "Critical Alert" "There are critical notifications"
fi

# Or in one-liners
cd myproject && bd-has-unread && echo "Check inbox" || true
```

**How it works**:
- Returns proper exit codes (0 = true, 1 = false)
- Can be used in if/else conditions
- Chainable with logical operators

---

### 3. Cron Job Notification

**Goal**: Monitor notifications via cron and send alerts.

**Script** (create `~/.local/bin/bd-cron-monitor`):
```bash
#!/bin/bash

# Check for critical notifications periodically
# Useful in cron: */5 * * * * ~/.local/bin/bd-cron-monitor

ALERT_FILE="$HOME/.local/state/bd-last-alert"
CRITICAL=$(bd status --format='%{critical-count}')

if [ "$CRITICAL" -gt 0 ]; then
  # Only alert once per critical state
  if [ ! -f "$ALERT_FILE" ] || [ "$(cat "$ALERT_FILE")" != "$CRITICAL" ]; then
    # Send email alert
    {
      echo "Subject: BD Critical Alert"
      echo ""
      echo "Critical notifications: $CRITICAL"
      bd status --format=detailed
      echo ""
      echo "Latest: $(bd status --format='%{latest-message}')"
    } | mail -s "BD Alert: $CRITICAL critical" "$USER@localhost"
    
    # Mark alert sent
    echo "$CRITICAL" > "$ALERT_FILE"
  fi
else
  # Clear alert state when critical resolved
  rm -f "$ALERT_FILE"
fi
```

**Make executable**:
```bash
chmod +x ~/.local/bin/bd-cron-monitor
```

**Setup in crontab** (run every 5 minutes):
```bash
*/5 * * * * ~/.local/bin/bd-cron-monitor
```

**Add to crontab**:
```bash
crontab -e
# Add the line above
```

**How it works**:
- Runs every 5 minutes from cron
- Sends alert when critical items exist
- Only sends once per state change
- Uses `ALERT_FILE` to track last sent count

---

### 4. Desktop Notification

**Goal**: Show desktop notifications for new alerts.

**Script** (create `~/.local/bin/bd-notify`):
```bash
#!/bin/bash

# Show desktop notifications for alerts
# Usage: ~/.local/bin/bd-notify
# Install: notify-send required (usually pre-installed)

CRITICAL=$(bd status --format='%{critical-count}')
ERROR=$(bd status --format='%{error-count}')
LATEST=$(bd status --format='%{latest-message}')

if [ "$CRITICAL" -gt 0 ]; then
  notify-send \
    -u critical \
    "🔴 CRITICAL" \
    "$CRITICAL critical notifications\n$LATEST"
elif [ "$ERROR" -gt 0 ]; then
  notify-send \
    -u normal \
    "🟠 ERROR" \
    "$ERROR errors\n$LATEST"
fi
```

**Make executable**:
```bash
chmod +x ~/.local/bin/bd-notify
```

**Setup in crontab** (run every 2 minutes):
```bash
*/2 * * * * ~/.local/bin/bd-notify
```

**Or run manually**:
```bash
~/.local/bin/bd-notify
```

**How it works**:
- Checks severity levels
- Sends desktop notification
- Shows count and latest message
- Uses critical/normal urgency

**System requirements**:
- `notify-send` (usually pre-installed on Linux)
- macOS: Use `osascript` instead:
  ```bash
  osascript -e "display notification \"$LATEST\" with title \"BD: $CRITICAL Critical\""
  ```

---

### 5. Watch Real-Time Status

**Goal**: Monitor status changes in real-time in terminal.

**Command**:
```bash
# Watch status update every 2 seconds
watch -n 2 'bd status --format=detailed'
```

**Output** (updates live):
```
Every 2.0s: bd status --format=detailed          hostname: Thu Feb 27 21:45:00 2026

3 unread, 2 read | Latest: Build completed successfully
```

**With additional info**:
```bash
watch -n 1 'bd status --format="Active:%{unread-count} Critical:%{critical-count} Error:%{error-count}"'
```

**With custom monitoring**:
```bash
watch -n 2 bash -c 'echo "=== Status ==="; bd status --format=detailed; echo "=== Latest ==="; bd list --active | head -3'
```

**How it works**:
- `watch` command runs repeatedly
- `-n 2` means every 2 seconds
- Useful for monitoring while working
- Press `q` to quit

---

## Monitoring & Dashboard

Examples for integration with monitoring systems and dashboards.

### 1. JSON for API Endpoint

**Goal**: Expose status as JSON for monitoring systems.

**Script** (create `~/.local/bin/bd-status-api`):
```bash
#!/bin/bash

# Serve BD status as JSON
# Usage: curl http://localhost:8080/bd-status

STATUS=$(bd status --format=json)
SEVERITY=$(bd status --format='%{highest-severity}')
CRITICAL=$(bd status --format='%{critical-count}')
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

cat <<EOF
{
  "timestamp": "$TIMESTAMP",
  "severity": $SEVERITY,
  "has_critical": $([[ $CRITICAL -gt 0 ]] && echo true || echo false),
  "status": $STATUS
}
EOF
```

**Run it**:
```bash
~/.local/bin/bd-status-api
```

**Output**:
```json
{
  "timestamp": "2026-02-27T21:45:00Z",
  "severity": 1,
  "has_critical": true,
  "status": {"unread":3,"total":3,"message":"Build completed successfully"}
}
```

**With simple HTTP server** (for monitoring):
```bash
#!/bin/bash

# Simple HTTP server on port 8080
PORT=8080

# Check if netcat/nc available
if ! command -v nc &> /dev/null; then
  echo "nc (netcat) not found. Use Python instead:"
  echo "python3 -m http.server 8080"
  exit 1
fi

while true; do
  {
    echo -ne "HTTP/1.1 200 OK\r\n"
    echo -ne "Content-Type: application/json\r\n"
    echo -ne "Connection: close\r\n\r\n"
    
    bd-status-api
  } | nc -l -p $PORT -q 1
done
```

**How it works**:
- Generates JSON response with status
- Can be queried from monitoring systems
- Includes timestamp for tracking

---

### 2. Prometheus Metrics Format

**Goal**: Export metrics in Prometheus format.

**Script** (create `~/.local/bin/bd-prometheus`):
```bash
#!/bin/bash

# Export BD metrics in Prometheus format
# Usage in prometheus.yml:
#   - job_name: 'bd'
#     static_configs:
#       - targets: ['localhost:9100']
#     metrics_path: '/var/lib/node_exporter/bd-metrics.prom'

TIMESTAMP=$(date +%s000)
CRITICAL=$(bd status --format='%{critical-count}')
ERROR=$(bd status --format='%{error-count}')
WARNING=$(bd status --format='%{warning-count}')
UNREAD=$(bd status --format='%{unread-count}')
SEVERITY=$(bd status --format='%{highest-severity}')

cat <<EOF
# HELP bd_critical_notifications Number of critical notifications
# TYPE bd_critical_notifications gauge
bd_critical_notifications $CRITICAL

# HELP bd_error_notifications Number of error notifications
# TYPE bd_error_notifications gauge
bd_error_notifications $ERROR

# HELP bd_warning_notifications Number of warning notifications
# TYPE bd_warning_notifications gauge
bd_warning_notifications $WARNING

# HELP bd_unread_notifications Total unread notifications
# TYPE bd_unread_notifications gauge
bd_unread_notifications $UNREAD

# HELP bd_severity_level Highest severity level (1=critical, 4=info)
# TYPE bd_severity_level gauge
bd_severity_level $SEVERITY
EOF
```

**Make executable**:
```bash
chmod +x ~/.local/bin/bd-prometheus
```

**Run it**:
```bash
~/.local/bin/bd-prometheus
```

**Output**:
```
# HELP bd_critical_notifications Number of critical notifications
# TYPE bd_critical_notifications gauge
bd_critical_notifications 1
# ... more metrics
```

**With cron** (update metrics file every minute):
```bash
* * * * * ~/.local/bin/bd-prometheus > /var/lib/node_exporter/bd-metrics.prom
```

---

### 3. Grafana Dashboard Variable

**Goal**: Use status in Grafana dashboards.

**Script** (create `~/.local/bin/bd-grafana-variable`):
```bash
#!/bin/bash

# Return status suitable for Grafana variables
# Use in Grafana: Query Language -> Custom

FORMAT="$1"  # e.g., "count-only", "json"
DEFAULT="count-only"

# Validate format
case "$FORMAT" in
  count-only|json|detailed|levels)
    bd status --format="$FORMAT"
    ;;
  *)
    bd status --format="$DEFAULT"
    ;;
esac
```

**Make executable**:
```bash
chmod +x ~/.local/bin/bd-grafana-variable
```

**In Grafana**:
1. Create datasource: "Datasource" → "Command/Custom"
2. Set query: `/home/user/.local/bin/bd-grafana-variable count-only`
3. Use variable in panel: `${bd_status_count}`

**Usage in dashboard panels**:
```
Title: Notifications
Value: ${bd_status_count}
```

---

### 4. Status Check Script (Returns Exit Code)

**Goal**: Check status and return exit codes for CI/CD.

**Script** (create `~/.local/bin/bd-check-status`):
```bash
#!/bin/bash

# Check BD status and return appropriate exit code
# Exit 0: OK (no notifications)
# Exit 1: WARNING (has unread)
# Exit 2: CRITICAL (has critical)
# Usage: bd-check-status; echo $?

CRITICAL=$(bd status --format='%{critical-count}')
UNREAD=$(bd status --format='%{unread-count}')

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

**Make executable**:
```bash
chmod +x ~/.local/bin/bd-check-status
```

**Usage**:
```bash
bd-check-status
# Output: OK: All clear
echo $?  # 0

# Or in scripts
if bd-check-status; then
  echo "Deployment OK"
else
  echo "Check notifications before deploying"
  exit 1
fi
```

**In CI/CD pipelines**:
```yaml
# GitHub Actions
- name: Check BD Status
  run: |
    if ! ~/.local/bin/bd-check-status; then
      echo "Status check failed"
      exit 1
    fi
```

---

### 5. Alert Email on Critical

**Goal**: Send email alert when critical notifications appear.

**Script** (create `~/.local/bin/bd-email-alert`):
```bash
#!/bin/bash

# Send email alert when critical notifications exist
# Setup in crontab: */10 * * * * ~/.local/bin/bd-email-alert

EMAIL="your-email@example.com"
ALERT_FILE="$HOME/.local/state/bd-critical-alert"
CRITICAL=$(bd status --format='%{critical-count}')

if [ "$CRITICAL" -gt 0 ]; then
  # Check if we haven't alerted in the last hour
  if [ ! -f "$ALERT_FILE" ] || \
     [ $(date +%s) -gt $(($(stat -f%m "$ALERT_FILE" 2>/dev/null || echo 0) + 3600)) ]; then
    
    # Send email
    {
      echo "Subject: [ALERT] BD Critical Notification"
      echo "To: $EMAIL"
      echo ""
      echo "Critical notifications: $CRITICAL"
      echo ""
      echo "Status:"
      bd status --format=detailed
      echo ""
      echo "Top notifications:"
      bd list --active | head -10
      echo ""
      echo "Time: $(date)"
    } | sendmail "$EMAIL" || mail -s "[ALERT] BD Critical" "$EMAIL"
    
    # Mark alert sent
    touch "$ALERT_FILE"
  fi
else
  # Clear alert state
  rm -f "$ALERT_FILE"
fi
```

**Make executable**:
```bash
chmod +x ~/.local/bin/bd-email-alert
```

**Add to crontab** (every 10 minutes):
```bash
*/10 * * * * ~/.local/bin/bd-email-alert
```

---

## Advanced

Advanced examples for power users.

### 1. Conditional Logic with jq

**Goal**: Parse JSON and apply complex logic.

**Script**:
```bash
#!/bin/bash

# Complex conditional logic using jq
bd status --format=json | jq '
  if .unread > 0 then
    if .unread > 10 then
      "🔴 URGENT: \(.unread) notifications"
    elif .unread > 5 then
      "🟠 WARNING: \(.unread) notifications"
    else
      "🔵 INFO: \(.unread) notifications"
    end
  else
    "✅ All clear"
  end
'
```

**Output**:
```
🔴 URGENT: 15 notifications
```

**Usage**:
```bash
eval "$(bd status --format=json | jq -r '"alert=\(.unread > 0)"')"
if [ "$alert" = "true" ]; then
  echo "Alert needed"
fi
```

---

### 2. Compose with Other CLI Tools

**Goal**: Combine status with other tools.

**Examples**:

```bash
# Show status with timestamp
echo "$(date '+%Y-%m-%d %H:%M:%S'): $(bd status --format=compact)"

# Sort critical items and show count
bd list --active --level=critical | wc -l

# Combine status from multiple sources
echo "Inbox: $(bd status --format=count-only) | Tasks: $(task count)"

# Log status changes
diff <(bd status --format=json) /tmp/bd-status-last.json && \
  echo "No change" || \
  (bd status --format=json | tee /tmp/bd-status-last.json && echo "Status updated")
```

---

### 3. Integration with fzf

**Goal**: Use status with fuzzy finder for browsing.

**Script**:
```bash
#!/bin/bash

# Fuzzy search through notifications based on status
status_summary=$(bd status --format=detailed)
echo "Current: $status_summary"
echo ""

# Show list and allow fuzzy search
bd list --active | \
  fzf \
    --preview 'echo "Selected: {}"' \
    --preview-window=right:30% \
    --bind 'enter:execute(bd jump {+1})'
```

**How it works**:
- Shows current status at top
- Lists notifications with fuzzy search
- Shows preview of selected item
- Jump to notification on select

---

### 4. Watch Real-Time with Formatting

**Goal**: Monitor with custom formatting.

**Script**:
```bash
#!/bin/bash

# Real-time monitoring with custom format
watch -n 1 bash -c '
  echo "=== BD Status $(date +%H:%M:%S) ==="
  echo "Summary: $(bd status --format=detailed)"
  echo ""
  echo "Severity breakdown:"
  echo "  $(bd status --format="C:%{critical-count} E:%{error-count} W:%{warning-count} I:%{info-count}")"
  echo ""
  echo "Recent (last 5):"
  bd list --active --limit=5 | awk "{print \"  \" \$0}"
'
```

**Output** (updates every second):
```
=== BD Status 21:45:30 ===
Summary: 3 unread, 2 read | Latest: Build completed successfully

Severity breakdown:
  C:1 E:2 W:3 I:4

Recent (last 5):
  [notification 1]
  [notification 2]
  ...
```

---

### 5. Log Status to File

**Goal**: Track status changes over time.

**Script** (create `~/.local/bin/bd-log`):
```bash
#!/bin/bash

# Log current status to file
# Usage: bd-log
# Or in crontab: */5 * * * * ~/.local/bin/bd-log

LOG_FILE="$HOME/.local/state/bd-status.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
STATUS=$(bd status --format=json)

# Log with timestamp
echo "[$TIMESTAMP] $STATUS" >> "$LOG_FILE"

# Keep last 10000 lines
tail -10000 "$LOG_FILE" > "$LOG_FILE.tmp"
mv "$LOG_FILE.tmp" "$LOG_FILE"
```

**Make executable**:
```bash
chmod +x ~/.local/bin/bd-log
```

**Add to crontab** (every 5 minutes):
```bash
*/5 * * * * ~/.local/bin/bd-log
```

**View log**:
```bash
tail -f ~/.local/state/bd-status.log

# Output:
[2026-02-27 21:40:00] {"unread":3,"total":3,"message":"Build completed"}
[2026-02-27 21:45:00] {"unread":3,"total":3,"message":"Test passed"}
[2026-02-27 21:50:00] {"unread":4,"total":4,"message":"New error"}
```

**Analyze patterns**:
```bash
# Count how many times status changed
grep -o '"unread":[0-9]*' ~/.local/state/bd-status.log | sort | uniq -c

# Extract timestamps with high unread counts
grep '"unread":[1-9][0-9]' ~/.local/state/bd-status.log
```

---

## Summary

**Quick reference**:

| Category | Purpose | Example |
|----------|---------|---------|
| **Tmux** | Status bar integration | `#(bd status --format=compact)` |
| **Shell** | Quick aliases and checks | `alias bd-status='bd status...'` |
| **Monitoring** | Dashboard and alert integration | `bd status --format=json` |
| **Advanced** | Complex logic and automation | jq parsing, cron jobs, logging |

**Getting started**:
1. Pick an example that matches your use case
2. Copy the code (adjust as needed)
3. Test it locally
4. Deploy to your system

**More help**:
- See [docs/status-format-guide.md](../status-format-guide.md) for concepts
- See [docs/status-format-reference.md](../status-format-reference.md) for technical details
- Run `bd status --help` for built-in help
