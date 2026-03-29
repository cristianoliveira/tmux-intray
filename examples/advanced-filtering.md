# Advanced Filtering Examples

This guide demonstrates the powerful filtering capabilities of `tmux-intray list`. It shows how to combine multiple filters to find exactly the notifications you need.

Before running these examples, ensure you have an active tmux session and some notifications in your tray (you can add test notifications with `tmux-intray add`).

---

## Example 1: Filter by session and level

```bash
# Notifications from a specific session with error level
tmux-intray list --session=$SESSION --level=error
# SESSION can be session ID (e.g., '$1') or session name
```

---

## Example 2: Time-based filtering

```bash
# Notifications older than 7 days but newer than 1 day
tmux-intray list --older-than=7 --newer-than=1
# Useful for finding stale notifications that aren't too old
```

---

## Example 3: Search within messages (substring match)

```bash
# Search for notifications containing 'error' or 'failed'
tmux-intray list --search=error
tmux-intray list --search=failed

# Combine with other filters
tmux-intray list --search=error --level=error
```

---

## Example 4: Regex search

```bash
# Regex search for patterns (e.g., 'ERR[0-9]+')
tmux-intray list --search='ERR[0-9]+' --regex
# Note: regex syntax is Bash's extended regular expressions
```

---

## Example 5: Grouping notifications

```bash
# Group notifications by session
tmux-intray list --group-by=session
# Groups are shown with headers and counts
```

---

## Example 6: Group counts only

```bash
# Show only group counts (no individual items)
tmux-intray list --group-by=session --group-count
# Output: 'Group: session_name (count)'
```

---

## Example 7: Combined filters: session + time + level

```bash
# Complex filter: notifications from session 'work', level warning or error, older than 1 day
tmux-intray list --session=work --level=warning --older-than=1
tmux-intray list --session=work --level=error --older-than=1
# (Run each level separately, or filter for multiple levels in a single command)
```

---

## Example 8: Pane-specific filtering with search

```bash
# Find notifications from a specific pane that mention 'TODO'
tmux-intray list --pane=%0 --search=TODO
# Pane ID can be obtained from tmux (e.g., '%0', '%1')
```

---

## Example 9: Window-based grouping with counts

```bash
# Group by window and show only counts
tmux-intray list --group-by=window --group-count
# Useful for seeing which windows have most notifications
```

---

## Example 10: Format output as table with filters

```bash
# Table format with session filter
tmux-intray list --session=$SESSION --format=table
# Table format includes ID, timestamp, pane, level, and message
```

---

## Example 11: Show dismissed notifications older than 30 days (cleanup candidates)

```bash
# Dismissed notifications older than 30 days
tmux-intray list --dismissed --older-than=30
# These would be removed by 'tmux-intray cleanup'
```

---

## Example 12: Live monitoring with filters

```bash
# Monitor new error notifications in real-time
tmux-intray follow --level=error
# Press Ctrl+C to stop
```

---

## Tips

- Use `tmux-intray list --help` to see all available options
- Filters can be combined: `--session`, `--window`, `--pane`, `--level`, `--older-than`, `--newer-than`, `--search`, `--regex`
- Grouping works with any filter combination
- The `--format` option controls output style: `legacy`, `table`, `compact`, `json`

---

## Real-world use cases

### Find all error notifications from the last 24 hours, grouped by session

```bash
tmux-intray list --level=error --newer-than=1 --group-by=session
```

### Find notifications from a specific project (session) that mention 'deprecated'

```bash
tmux-intray list --session=myproject --search=deprecated
```

### Monitor warnings in real-time while debugging

```bash
tmux-intray follow --level=warning
```

---

For complete command reference, see [CLI Reference](../docs/cli/CLI_REFERENCE.md).
