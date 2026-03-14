# Privacy Policy

This document outlines the privacy guarantees and data handling practices for tmux-intray.

## Telemetry Privacy

**Telemetry is local-only and never transmitted.**

tmux-intray includes an optional telemetry feature that tracks feature usage patterns to inform development decisions. This data is stored exclusively on your device and is never sent to any external service.

### What Data is Collected

When telemetry is enabled, tmux-intray collects:

- **Feature name**: Which command or TUI feature was used (e.g., "add", "list", "jump", "tui")
- **Feature category**: Whether it's a CLI or TUI feature
- **Timestamp**: When the feature was used (ISO 8601 format)
- **Context data**: Optional JSON-formatted additional information about feature usage

Example telemetry event:
```json
{
  "id": 1,
  "timestamp": "2026-03-14T13:00:00Z",
  "feature_name": "add",
  "feature_category": "cli",
  "context_data": "{}"
}
```

### What Data is NOT Collected

tmux-intray telemetry **does NOT** collect:

- Personal information (name, email, IP address, etc.)
- Notification content or message text
- Tmux session names, window names, or pane contents
- File paths or directory contents
- Any information that could identify you or your specific environment
- Any data from other applications or processes

### Data Storage

- **Location**: `~/.local/state/tmux-intray/notifications.db` (respects `TMUX_INTRAY_STATE_DIR` and XDG Base Directory Specification)
- **Format**: SQLite database with encrypted-at-rest optional (not currently implemented)
- **Access**: Only accessible by your user account (standard Unix file permissions)
- **Transmission**: **No network calls are ever made**

### Data Control

You have complete control over your telemetry data:

#### View Telemetry Data

```bash
# Show feature usage summary
tmux-intray telemetry show

# Show usage from the last 7 days
tmux-intray telemetry show --days 7

# Show telemetry status
tmux-intray telemetry status
```

#### Export Telemetry Data

```bash
# Export all telemetry data to JSONL format
tmux-intray telemetry export --output telemetry.jsonl
```

This allows you to:
- Backup your telemetry data
- Analyze it with your own tools
- Share it with developers for debugging (optional)
- Migrate data to another system

#### Clear Telemetry Data

```bash
# Clear telemetry data older than 90 days (default)
tmux-intray telemetry clear

# Clear telemetry data older than 30 days
tmux-intray telemetry clear --days 30

# Clear all telemetry data
tmux-intray telemetry clear --days 0
```

**Important**: The `clear` command requires confirmation before deleting data.

#### Disable Telemetry

```bash
# Disable telemetry via environment variable
export TMUX_INTRAY_TELEMETRY_ENABLED=false

# Or in config.toml
echo "telemetry_enabled = false" >> ~/.config/tmux-intray/config.toml
```

When disabled, no telemetry data is collected, but existing data remains until you explicitly clear it.

### Telemetry is Opt-In

- **Default**: Telemetry is **disabled by default**
- **Opt-in**: You must explicitly enable it to start collection
- **Opt-out**: You can disable it at any time without losing existing data
- **Data deletion**: You can delete all telemetry data with a single command

### Why Telemetry Exists

The telemetry feature exists to:

1. **Understand feature usage**: Identify which features are most used and which are rarely used
2. **Inform development decisions**: Make data-driven decisions about feature development and deprecation
3. **Improve user experience**: Focus development effort on features that matter most to users
4. **Local analytics**: Provide you with insights into your own usage patterns

### Data Ownership

- **Your data, your choice**: All telemetry data belongs to you
- **No sharing**: No data is ever shared with third parties
- **No profiling**: No user profiling or behavior analysis beyond feature usage counts
- **No advertising**: No use of data for advertising or marketing purposes

### Technical Implementation

The telemetry system is implemented with privacy as a core principle:

- **Local-first architecture**: All data storage and processing happens locally
- **No external dependencies**: No cloud services or APIs are used
- **Open source**: The telemetry implementation is fully auditable in the source code
- **Transparent**: Data schema and storage format are documented and open

### Configuration

See the [Configuration Guide](./configuration.md#telemetry) for detailed telemetry configuration options.

### Architecture

For technical details about the telemetry system architecture, see [Telemetry Architecture](./design/telemetry-architecture.md).

### Questions or Concerns

If you have questions or concerns about privacy:

1. Review this privacy documentation
2. Check the [Configuration Guide](./configuration.md#telemetry)
3. Examine the telemetry source code in `internal/storage/sqlite/schema.sql` and `cmd/tmux-intray/telemetry.go`
4. File an issue on [GitHub](https://github.com/cristianoliveira/tmux-intray/issues)

### Summary

| Aspect | Details |
|--------|---------|
| **Data Collection** | Opt-in, disabled by default |
| **Data Storage** | Local SQLite database only |
| **Data Transmission** | Never transmitted over network |
| **Personal Information** | Not collected |
| **Data Control** | Full control: view, export, clear, disable |
| **Data Ownership** | Belongs entirely to the user |
| **Third-party Access** | Never shared with third parties |
| **Purpose** | Feature usage analytics for development decisions |

## Additional Privacy Considerations

### Notification Data

Beyond telemetry, tmux-intray stores notification data in the same SQLite database:

- **Location**: Same as telemetry (`~/.local/state/tmux-intray/notifications.db`)
- **Content**: Message text, severity levels, timestamps, tmux context (session/window/pane IDs)
- **Access**: Only accessible by your user account
- **Transmission**: No network calls are made for notification data either

### Hooks System

The hooks system allows you to execute custom scripts before/after events. If you configure hooks to send data to external services, this is your responsibility and independent of tmux-intray's privacy practices.

### Configuration Files

Configuration files are stored in `~/.config/tmux-intray/` (or `TMUX_INTRAY_CONFIG_DIR`):

- `config.toml` - Main configuration
- `tui.toml` - TUI settings
- `hooks/` - Hook scripts

These files contain your preferences and settings, not telemetry or notification data.

---

**Last Updated**: 2026-03-14

This privacy policy is part of tmux-intray's commitment to user privacy and transparent data handling.
