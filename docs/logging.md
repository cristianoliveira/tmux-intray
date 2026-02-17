# Structured Logging

**Status**: *Implemented* - Available since tmux-intray v1.0

## Overview

tmux-intray includes a structured logging system that writes detailed operational logs to disk. These logs are useful for debugging, auditing, and understanding the internal behavior of the application. The logging system:

- Writes logs in JSON format for easy parsing and analysis
- Automatically rotates log files to prevent disk space exhaustion
- Redacts sensitive information (passwords, tokens, etc.) from log output
- Integrates with existing debug/quiet options
- Mirrors console output (colors package) to structured logs

## Configuration

Structured logging is configured via environment variables (or the configuration file) with the following options:

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_LOGGING_ENABLED` | `false` | Enable structured file logging when set to `true` |
| `TMUX_INTRAY_LOGGING_LEVEL` | `info` | Minimum log level to record (`debug`, `info`, `warn`, `error`) |
| `TMUX_INTRAY_LOGGING_MAX_FILES` | `10` | Maximum number of log files to retain (older files are rotated out) |

### Configuration File Example

Add the following to your `~/.config/tmux-intray/config.toml`:

```toml
# Enable structured logging
logging_enabled = true
logging_level = "debug"
logging_max_files = 5
```

### Environment Variable Example

```bash
export TMUX_INTRAY_LOGGING_ENABLED=true
export TMUX_INTRAY_LOGGING_LEVEL=debug
export TMUX_INTRAY_LOGGING_MAX_FILES=5
```

## Log File Location

Log files are stored in the following location priority:

1. **Primary location**: `{state_dir}/logs`  
   Where `state_dir` defaults to `$XDG_STATE_HOME/tmux-intray` (typically `~/.local/state/tmux-intray`).  
   This directory is created with `0700` permissions.

2. **Fallback location**: `{os.TempDir()}/tmux-intray/logs`  
   If the primary location is not writable, logs are written to a temporary directory.

You can check the current log file location by running:

```bash
tmux-intray list 2>&1 | grep "Logging to file:"
```

Or programmatically via the `CurrentLogFile()` function in the `logging` package.

## File Naming

Each log file is named with the following pattern:

```
tmux-intray_YYYYMMDD_HHMMSS_PID{pid}_{command}.log
```

Example: `tmux-intray_20250217_142305_PID12345_tmux-intray.log`

- **YYYYMMDD_HHMMSS**: Timestamp when the log file was created
- **PID{pid}**: Process ID of the tmux-intray instance
- **{command}**: Command name (spaces replaced with underscores)

## File Rotation

The logging system automatically rotates log files to prevent unlimited disk usage:

- When logging is initialized, the system checks the number of existing log files matching the pattern `tmux-intray_*.log`.
- If the count exceeds `logging_max_files`, the oldest files (by modification time) are deleted until only `logging_max_files` remain.
- Rotation occurs **before** creating a new log file, ensuring the total number never exceeds the limit.
- Default limit is 10 files; set `logging_max_files` to `0` to disable rotation (not recommended).

## Sensitive Data Redaction

To protect sensitive information, the logging system automatically redacts values whose keys contain certain sensitive words.

### Redacted Keywords

The following keywords trigger redaction (case-insensitive):

- `secret`
- `password`
- `token`
- `key`
- `auth`
- `credential`

### How Redaction Works

When a log entry includes key‑value pairs, the system checks each key against the sensitive word list. If a key contains a sensitive word as a separate segment (split by non‑alphanumeric characters), the corresponding value is replaced with `"[REDACTED]"`.

**Examples:**

| Key | Value Before Redaction | Value After Redaction |
|-----|------------------------|-----------------------|
| `password` | `supersecret` | `"[REDACTED]"` |
| `api_token` | `xyz123` | `"[REDACTED]"` |
| `auth_header` | `Bearer abc` | `"[REDACTED]"` |
| `normal_field` | `some value` | `some value` |
| `secret_key` | `mysecret` | `"[REDACTED]"` |

Redaction is applied to all structured log output, including fields added via `With()`.

## Integration with Debug/Quiet Options

Structured logging integrates with the existing `TMUX_INTRAY_DEBUG` and `TMUX_INTRAY_QUIET` options:

| Configuration | Logging Level | Notes |
|---------------|---------------|-------|
| `TMUX_INTRAY_DEBUG=true` | `debug` | Debug messages are logged at the `debug` level |
| `TMUX_INTRAY_QUIET=true` | `error` | Only errors are logged; other messages are suppressed |
| Both set | `debug` | Debug takes precedence over quiet |

These mappings are applied automatically; you do not need to manually set `logging_level` when using debug/quiet flags.

## Console Output Mirroring

When structured logging is enabled, the `colors` package mirrors all console output to the structured logger:

| Console Function | Log Level | Additional Fields |
|-----------------|-----------|-------------------|
| `colors.Error()` | `error` | None |
| `colors.Success()` | `info` | `type: "success"` |
| `colors.Warning()` | `warn` | None |
| `colors.Info()` | `info` | None |
| `colors.LogInfo()` | `info` | None |
| `colors.Debug()` | `debug` | None |

This ensures that everything printed to the console (via colors) is also captured in the structured logs, providing a complete audit trail.

## Usage Examples

### Basic Logging in Code

```go
import "github.com/cristianoliveira/tmux-intray/internal/logging"

// Initialize logging with default config (from global config)
err := logging.InitGlobal()
if err != nil {
    // handle error
}
defer logging.ShutdownGlobal()

// Log messages at different levels
logging.Info("Processing notification", "id", 123, "level", "warning")
logging.Warn("Disk space low", "available_mb", 42)
logging.Error("Failed to connect", "error", err)

// Add context with With()
logger := logging.With("request_id", "abc123")
logger.Info("Request started")
```

### Checking Log File Location

```go
import "github.com/cristianoliveira/tmux-intray/internal/logging"

path := logging.CurrentLogFile()
if path != "" {
    fmt.Println("Logging to:", path)
}
```

### Disabling Logging for a Specific Operation

```go
// Get the global logger
logger := logging.GetGlobal()
// If logging is disabled, logger is a no-op logger
logger.Debug("This won't appear anywhere if logging is disabled")
```

## Troubleshooting

### Logs Are Not Being Written

1. **Check if logging is enabled**: Verify `TMUX_INTRAY_LOGGING_ENABLED` is set to `true`.
2. **Verify directory permissions**: Ensure the log directory is writable. Check fallback location (`/tmp/tmux-intray/logs`).
3. **Check log level**: If level is set to `error`, only error messages will be logged.
4. **Look for initialization errors**: Run with `TMUX_INTRAY_DEBUG=true` to see any logging initialization failures.

### Log Files Are Not Rotating

1. **Check `logging_max_files` setting**: Ensure it's a positive integer (default 10).
2. **Verify file pattern**: Rotation only affects files matching `tmux-intray_*.log`.
3. **Permission issues**: The process must have permission to delete old log files.

### Sensitive Data Appears in Logs

1. **Verify redaction keywords**: The system redacts keys containing the sensitive words listed above.
2. **Check key naming**: Keys must contain sensitive words as separate segments (e.g., `api_token` redacts, `apitoken` does not).
3. **Custom redaction**: If you need additional redaction, modify the `redactor.go` file in the logging package.

## Differences from `TMUX_INTRAY_LOG_FILE`

tmux-intray has two distinct logging mechanisms:

| Feature | Structured Logging | `TMUX_INTRAY_LOG_FILE` |
|---------|-------------------|------------------------|
| **Format** | Structured JSON | Plain text (debug output) |
| **Content** | Operational logs, mirrored console output | Debug messages only (when `TMUX_INTRAY_DEBUG=true`) |
| **Configuration** | `logging_enabled`, `logging_level`, `logging_max_files` | Single file path |
| **Rotation** | Automatic file rotation | No rotation (single file) |
| **Use Case** | Long‑term auditing, debugging, analysis | Temporary debugging sessions |

Use structured logging for ongoing monitoring and `TMUX_INTRAY_LOG_FILE` for ad‑hoc debugging.

## Internal Architecture

The logging system is implemented in the `internal/logging` package:

- **`config.go`**: Configuration structures and loading
- **`logger.go`**: Logger implementation using charmbracelet/log
- **`redactor.go`**: Sensitive data redaction
- **`rotation.go`**: File rotation logic

The logger integrates with the `colors` package via `SetLogger()`, ensuring all console output is mirrored.

## See Also

- [Configuration Guide](./configuration.md) – General configuration options
- [Troubleshooting Guide](./troubleshooting.md) – Debugging tips
- [Project Philosophy](./philosophy.md) – Design principles