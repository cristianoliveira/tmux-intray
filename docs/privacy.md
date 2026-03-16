# Privacy Policy

This document outlines the privacy guarantees and data handling practices for tmux-intray.

## Simple Logging

**tmux-intray uses simple, local-only logging for debugging.**

tmux-intray includes optional debug logging to help troubleshoot issues. This logging is purely for diagnostic purposes and is stored exclusively on your device. No data is ever transmitted to external services.

### What Gets Logged

When logging is enabled via `TMUX_INTRAY_LOG_LEVEL`, tmux-intray may log:

- **Command execution**: Which command was run and basic outcome
- **Configuration loading**: Which configuration files were loaded
- **Settings changes**: When settings are modified
- **TUI actions**: Basic interactions in the terminal UI
- **Error messages**: Diagnostic information when errors occur

### What Data is NOT Logged

tmux-intray does NOT log:

- Personal information (name, email, IP address, etc.)
- Notification content or message text
- Tmux session names, window names, or pane contents
- File paths or directory contents (except config file locations)
- Any information that could identify you or your specific environment
- Any data from other applications or processes

### Controlling Logging

Logging is **disabled by default**. Enable it only when debugging:

```bash
# Enable debug logging via environment variable
export TMUX_INTRAY_LOG_LEVEL=debug

# Supported levels: debug, info, warn, error, off
# Run with logging
tmux-intray list
```

To disable logging:

```bash
unset TMUX_INTRAY_LOG_LEVEL
```

### Log Output

- **Default**: Logs are written to stderr
- **Location**: Terminal output only (no persistent log file by default)
- **Access**: Only visible to you in your terminal session
- **Transmission**: No network calls are made

### Technical Implementation

The logging system is implemented with simplicity and privacy as core principles:

- **Local-only**: All logging stays on your machine
- **Minimal output**: Only essential diagnostic information is logged
- **No external dependencies**: No telemetry services or tracking
- **Open source**: The logging implementation is fully auditable in the source code
- **Transparent**: Log levels are documented and easy to control

## Notification Data

tmux-intray stores notification data in a SQLite database:

- **Location**: `~/.local/state/tmux-intray/notifications.db` (respects `TMUX_INTRAY_STATE_DIR` and XDG Base Directory Specification)
- **Content**: Message text, severity levels, timestamps, tmux context (session/window/pane IDs)
- **Access**: Only accessible by your user account (standard Unix file permissions)
- **Transmission**: No network calls are made for notification data
- **Privacy**: This data is local-only and never shared

## Hooks System

The hooks system allows you to execute custom scripts before/after events. If you configure hooks to send data to external services, this is your responsibility and independent of tmux-intray's privacy practices.

## Configuration Files

Configuration files are stored in `~/.config/tmux-intray/` (or `TMUX_INTRAY_CONFIG_DIR`):

- `config.toml` - Main configuration
- `tui.toml` - TUI settings
- `hooks/` - Hook scripts

These files contain your preferences and settings, not diagnostic or notification data.

## Privacy Summary

| Aspect | Details |
|--------|---------|
| **Data Collection** | Minimal, diagnostic only |
| **Data Storage** | Local SQLite database and stderr output |
| **Data Transmission** | Never transmitted over network |
| **Personal Information** | Not collected |
| **Data Control** | You control all logging via environment variables |
| **Third-party Access** | Never shared with third parties |
| **Purpose** | Local debugging and problem diagnosis |

## Questions or Concerns

If you have questions or concerns about privacy:

1. Review this privacy documentation
2. Check the [Configuration Guide](./configuration.md)
3. Examine the logging source code in `internal/log/log.go`
4. File an issue on [GitHub](https://github.com/cristianoliveira/tmux-intray/issues)

---

**Last Updated**: 2026-03-16

This privacy policy reflects tmux-intray's commitment to user privacy and transparent data handling.
