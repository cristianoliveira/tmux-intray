# Example Hooks for tmux-intray

This directory contains example hook scripts that demonstrate how to extend tmux-intray with custom automation.

## Hook System Overview

tmux-intray provides a hook system that allows you to execute custom scripts at specific points in the notification lifecycle:

| Hook Point | Description | Environment Variables |
|------------|-------------|----------------------|
| `pre-add` | Before a notification is added | `NOTIFICATION_ID`, `LEVEL`, `MESSAGE`, `TIMESTAMP`, `SESSION`, `WINDOW`, `PANE`, `PANE_CREATED` |
| `post-add` | After a notification is added | Same as pre-add |
| `pre-dismiss` | Before a notification is dismissed | Same as pre-add |
| `post-dismiss` | After a notification is dismissed | Same as pre-add |
| `cleanup` | Before cleaning up old notifications | `CLEANUP_DAYS`, `CUTOFF_TIMESTAMP`, `DRY_RUN` |
| `post-cleanup` | After cleaning up old notifications | Same as cleanup plus `DELETED_COUNT` |

## Configuration

Hooks are configured via environment variables in your `~/.config/tmux-intray/config.sh`:

```bash
# Enable hooks globally
TMUX_INTRAY_HOOKS_ENABLED=1

# Hook failure mode: ignore, warn, abort
TMUX_INTRAY_HOOKS_FAILURE_MODE="warn"

# Run hooks asynchronously (0=sync, 1=async)
TMUX_INTRAY_HOOKS_ASYNC=0

# Hook directory (default: $TMUX_INTRAY_CONFIG_DIR/hooks)
TMUX_INTRAY_HOOKS_DIR="$HOME/.config/tmux-intray/hooks"

# Per-hook enable/disable
TMUX_INTRAY_HOOKS_ENABLED_pre_add=1
TMUX_INTRAY_HOOKS_ENABLED_post_add=1
```

## Example Scripts

### 01-log.sh
Logs notification events to a file. Configure log location with `HOOK_LOG_FILE` environment variable.

### 02-notify.sh (optional)
Sends desktop notifications using `notify-send` for certain notification levels.

### 03-webhook.sh (optional)
Sends HTTP POST requests to a webhook URL for integration with external systems.

## Installation

1. Copy the example scripts to your hook directory:
   ```bash
   cp -r examples/hooks/* ~/.config/tmux-intray/hooks/
   ```

2. Make scripts executable:
   ```bash
   chmod +x ~/.config/tmux-intray/hooks/*/*.sh
   ```

3. Customize scripts as needed.

## Writing Your Own Hooks

Hook scripts are simple Bash scripts that have access to environment variables describing the notification context. They should:

- Be executable (`chmod +x`)
- Have a `.sh` extension
- Be placed in the appropriate hook point directory
- Return exit code 0 for success, non-zero for failure

Hook scripts are executed in alphabetical order within each hook point directory.