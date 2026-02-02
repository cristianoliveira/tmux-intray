# tmux-intray Configuration

tmux-intray can be configured via environment variables, which can be set in your shell profile or in a dedicated configuration file. This guide covers all available configuration options.

## Configuration File

The primary way to configure tmux-intray is through the configuration file at `$TMUX_INTRAY_CONFIG_DIR/config.sh` (default: `~/.config/tmux-intray/config.sh`). This file is sourced as a bash script when tmux-intray starts, allowing you to set environment variables and define custom behavior.

If the file doesn't exist, tmux-intray creates a sample configuration file with default values and helpful comments.

## Environment Variables

All configuration options are controlled by environment variables with the `TMUX_INTRAY_` prefix. You can set these variables in your configuration file or export them in your shell.

### Storage & Paths

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_STATE_DIR` | `$XDG_STATE_HOME/tmux-intray` (`~/.local/state/tmux-intray`) | Directory where notification data is stored. Follows XDG Base Directory Specification. |
| `TMUX_INTRAY_CONFIG_DIR` | `$XDG_CONFIG_HOME/tmux-intray` (`~/.config/tmux-intray`) | Directory for configuration files and hooks. |
| `TMUX_INTRAY_MAX_NOTIFICATIONS` | `1000` | Maximum number of notifications to keep (oldest are automatically cleaned up). |
| `TMUX_INTRAY_AUTO_CLEANUP_DAYS` | `30` | Automatically clean up notifications that have been dismissed for more than this many days. |

### Display & Formatting

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_DATE_FORMAT` | `%Y-%m-%d %H:%M:%S` | Date format used in output (see `man date` for format codes). |
| `TMUX_INTRAY_TABLE_FORMAT` | `default` | Table style for `--format=table` output (`default`, `minimal`, `fancy`). |
| `TMUX_INTRAY_LEVEL_COLORS` | `info:green,warning:yellow,error:red,critical:magenta` | Color mapping for notification levels in status bar. Available colors: black, red, green, yellow, blue, magenta, cyan, white. |

### Status Bar Integration

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_STATUS_ENABLED` | `1` | Enable (1) or disable (0) the status bar indicator. |
| `TMUX_INTRAY_STATUS_FORMAT` | `compact` | Status panel format (`compact`, `detailed`, `count-only`). |
| `TMUX_INTRAY_SHOW_LEVELS` | `0` | Show level counts (1) or only total count (0) in status bar. |

### Hook System

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_HOOKS_ENABLED` | `1` | Enable (1) or disable (0) hooks globally. |
| `TMUX_INTRAY_HOOKS_FAILURE_MODE` | `warn` | Behavior when a hook fails: `ignore` (silently continue), `warn` (log warning), `abort` (stop operation). |
| `TMUX_INTRAY_HOOKS_ASYNC` | `0` | Run hooks asynchronously (1) or synchronously (0). *Not yet implemented*. |
| `TMUX_INTRAY_HOOKS_DIR` | `$TMUX_INTRAY_CONFIG_DIR/hooks` | Directory containing hook scripts. |
| `TMUX_INTRAY_HOOKS_ENABLED_pre_add` | `1` | Enable/disable pre‑add hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_post_add` | `1` | Enable/disable post‑add hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_pre_dismiss` | `1` | Enable/disable pre‑dismiss hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_post_dismiss` | `1` | Enable/disable post‑dismiss hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_cleanup` | `1` | Enable/disable cleanup hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_post_cleanup` | `1` | Enable/disable post‑cleanup hooks (0/1). |

### Debugging & Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_DEBUG` | *unset* | Enable debug output when set to `1`, `true`, `yes`, or `on`. Debug messages are printed to stderr. |
| `TMUX_INTRAY_QUIET` | *unset* | Suppress non‑error output when set to `1`, `true`, `yes`, or `on`. |

## Sample Configuration File

```bash
# tmux-intray configuration
# This file is sourced by tmux-intray on startup.

# Storage directories (follow XDG Base Directory Specification)
TMUX_INTRAY_STATE_DIR="$HOME/.local/state/tmux-intray"
TMUX_INTRAY_CONFIG_DIR="$HOME/.config/tmux-intray"

# Storage limits
TMUX_INTRAY_MAX_NOTIFICATIONS=1000
TMUX_INTRAY_AUTO_CLEANUP_DAYS=30

# Display settings
TMUX_INTRAY_DATE_FORMAT="%Y-%m-%d %H:%M:%S"
TMUX_INTRAY_TABLE_FORMAT="default"

# Status bar integration
TMUX_INTRAY_STATUS_ENABLED=1
TMUX_INTRAY_STATUS_FORMAT="compact"
TMUX_INTRAY_SHOW_LEVELS=0
TMUX_INTRAY_LEVEL_COLORS="info:green,warning:yellow,error:red,critical:magenta"

# Hook system
TMUX_INTRAY_HOOKS_ENABLED=1
TMUX_INTRAY_HOOKS_FAILURE_MODE="warn"
TMUX_INTRAY_HOOKS_ASYNC=0
TMUX_INTRAY_HOOKS_DIR="$HOME/.config/tmux-intray/hooks"

# Per-hook enable/disable
TMUX_INTRAY_HOOKS_ENABLED_pre_add=1
TMUX_INTRAY_HOOKS_ENABLED_post_add=1
TMUX_INTRAY_HOOKS_ENABLED_pre_dismiss=1
TMUX_INTRAY_HOOKS_ENABLED_post_dismiss=1
TMUX_INTRAY_HOOKS_ENABLED_cleanup=1
TMUX_INTRAY_HOOKS_ENABLED_post_cleanup=1

# Debugging
TMUX_INTRAY_DEBUG=0
TMUX_INTRAY_QUIET=0
```

## Overriding Configuration

You can also set environment variables directly in your shell, which take precedence over the configuration file. For example:

```bash
export TMUX_INTRAY_DEBUG=1
export TMUX_INTRAY_AUTO_CLEANUP_DAYS=7
tmux-intray list
```

This is useful for temporary debugging or for per‑session customization.

## Notes

- All paths support `~` expansion and environment variables.
- If `XDG_STATE_HOME` or `XDG_CONFIG_HOME` are not set, the default `~/.local/state` and `~/.config` are used.
- The hook system is extensible; see [hooks documentation](hooks.md) for details.
- For troubleshooting, enable `TMUX_INTRAY_DEBUG` and check the output on stderr.