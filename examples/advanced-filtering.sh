#!/usr/bin/env bash
# Advanced filtering examples for tmux-intray
#
# This script demonstrates the powerful filtering capabilities of `tmux-intray list`.
# It shows how to combine multiple filters to find exactly the notifications you need.
#
# Before running these examples, ensure you have an active tmux session and some
# notifications in your tray (you can add test notifications with `tmux-intray add`).

set -euo pipefail

echo "=== tmux-intray Advanced Filtering Examples ==="
echo

# Example 1: Filter by session and level
echo "1. Notifications from a specific session with error level:"
echo "   tmux-intray list --session=\$SESSION --level=error"
echo "   # SESSION can be session ID (e.g., '\$1') or session name"
echo

# Example 2: Time-based filtering
echo "2. Notifications older than 7 days but newer than 1 day:"
echo "   tmux-intray list --older-than=7 --newer-than=1"
echo "   # Useful for finding stale notifications that aren't too old"
echo

# Example 3: Search within messages (substring match)
echo "3. Search for notifications containing 'error' or 'failed':"
echo "   tmux-intray list --search=error"
echo "   tmux-intray list --search=failed"
echo "   # Combine with other filters:"
echo "   tmux-intray list --search=error --level=error"
echo

# Example 4: Regex search
echo "4. Regex search for patterns (e.g., 'ERR[0-9]+'):"
echo "   tmux-intray list --search='ERR[0-9]+' --regex"
echo "   # Note: regex syntax is Bash's extended regular expressions"
echo

# Example 5: Grouping notifications
echo "5. Group notifications by session:"
echo "   tmux-intray list --group-by=session"
echo "   # Groups are shown with headers and counts"
echo

# Example 6: Group counts only
echo "6. Show only group counts (no individual items):"
echo "   tmux-intray list --group-by=session --group-count"
echo "   # Output: 'Group: session_name (count)'"
echo

# Example 7: Combined filters: session + time + level
echo "7. Complex filter: notifications from session 'work', level warning or error, older than 1 day:"
echo "   tmux-intray list --session=work --level=warning --older-than=1"
echo "   tmux-intray list --session=work --level=error --older-than=1"
echo "   # (Run each level separately, or filter for multiple levels in a single command)"
echo

# Example 8: Pane-specific filtering with search
echo "8. Find notifications from a specific pane that mention 'TODO':"
echo "   tmux-intray list --pane=%0 --search=TODO"
echo "   # Pane ID can be obtained from tmux (e.g., '%0', '%1')"
echo

# Example 9: Window-based grouping with counts
echo "9. Group by window and show only counts:"
echo "   tmux-intray list --group-by=window --group-count"
echo "   # Useful for seeing which windows have the most notifications"
echo

# Example 10: Format output as table with filters
echo "10. Table format with session filter:"
echo "    tmux-intray list --session=\$SESSION --format=table"
echo "    # Table format includes ID, timestamp, pane, level, and message"
echo

# Example 11: Show dismissed notifications older than 30 days (cleanup candidates)
echo "11. Dismissed notifications older than 30 days:"
echo "    tmux-intray list --dismissed --older-than=30"
echo "    # These would be removed by 'tmux-intray cleanup'"
echo

# Example 12: Live monitoring with filters
echo "12. Monitor new error notifications in real-time:"
echo "    tmux-intray follow --level=error"
echo "    # Press Ctrl+C to stop"
echo

echo "=== Tips ==="
echo
echo "- Use 'tmux-intray list --help' to see all available options."
echo "- Filters can be combined: --session, --window, --pane, --level, --older-than, --newer-than, --search, --regex."
echo "- Grouping works with any filter combination."
echo "- The '--format' option controls output style: legacy, table, compact, json (coming soon)."
echo "- For scripting, use '--format=compact' to get just messages (one per line)."
echo
echo "=== Real-world use case ==="
echo
echo "# Find all error notifications from the last 24 hours, group by session"
echo "tmux-intray list --level=error --newer-than=1 --group-by=session"
echo
echo "# Find notifications from a specific project (session) that mention 'deprecated'"
echo "tmux-intray list --session=myproject --search=deprecated"
echo
echo "# Monitor warnings in real-time while debugging"
echo "tmux-intray follow --level=warning"
