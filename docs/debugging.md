# Debugging Guide for tmux-intray

This guide will help you troubleshoot issues with tmux-intray using the built-in logging system and debugging techniques.

## Quick Start

To enable debug logging, set the `TMUX_INTRAY_LOG_LEVEL` environment variable:

```bash
# Enable debug logging
export TMUX_INTRAY_LOG_LEVEL=debug

# Run a command
tmux-intray list

# Logs will appear on stderr with timestamps and log levels
```

## Log Levels

tmux-intray supports five logging levels, from most verbose to least:

### `debug`
- **When to use**: Detailed troubleshooting, understanding application flow
- **What it shows**: Every function call, configuration parsing, internal state changes
- **Output**: `[DEBUG]` prefix with full timestamp
- **Example**:
  ```
  2026-03-16 14:32:15 [DEBUG] Initializing configuration from /home/user/.config/tmux-intray
  2026-03-16 14:32:15 [DEBUG] Loading notification storage backend: sqlite
  2026-03-16 14:32:15 [DEBUG] Database opened: /home/user/.local/state/tmux-intray/notifications.db
  ```

### `info` (default)
- **When to use**: Normal operation, monitoring application startup and key events
- **What it shows**: Important events like commands completed, configuration loaded
- **Output**: `[INFO]` prefix with timestamp
- **Example**:
  ```
  2026-03-16 14:32:15 [INFO] Notification added: 'Build failed' (ID: 42)
  2026-03-16 14:32:16 [INFO] Notification dismissed: ID 42
  ```

### `warn`
- **When to use**: Monitoring for potential issues that don't prevent operation
- **What it shows**: Unusual conditions, missing optional features, deprecated usage
- **Output**: `[WARN]` prefix with timestamp
- **Example**:
  ```
  2026-03-16 14:32:15 [WARN] Hook script not found: /home/user/.config/tmux-intray/hooks/pre-add.sh
  2026-03-16 14:32:15 [WARN] Configuration file not found, using defaults
  ```

### `error`
- **When to use**: Diagnosing failures and errors
- **What it shows**: Errors that prevent operations from completing
- **Output**: `[ERROR]` prefix with timestamp
- **Example**:
  ```
  2026-03-16 14:32:15 [ERROR] Failed to open database: permission denied
  2026-03-16 14:32:15 [ERROR] Invalid notification level: 'critical'
  ```

### `off`
- **When to use**: Disabling all logging
- **What it shows**: Nothing
- **Useful for**: Silent operation, automated scripts where output must be clean
- **Example**:
  ```bash
  export TMUX_INTRAY_LOG_LEVEL=off
  tmux-intray list  # No log output
  ```

## Understanding Log Output

### Log Format

All logs follow this format:
```
YYYY-MM-DD HH:MM:SS [LEVEL] Message
```

### Where Logs Go

Logs are written to **stderr**, not stdout. This is intentional:
- Stdout remains clean for data output (JSON, tables, notifications)
- Stderr carries diagnostic information
- You can redirect them separately:

```bash
# Separate stdout and stderr
tmux-intray list > notifications.txt 2> debug.log

# Capture both
tmux-intray list &> combined.log

# View logs while command runs
tmux-intray list 2>&1 | grep ERROR
```

## Common Debugging Scenarios

### Scenario 1: Notifications Not Appearing

**Problem**: You add a notification but it doesn't show up in the tray.

**Debug steps**:
```bash
# 1. Enable debug logging
export TMUX_INTRAY_LOG_LEVEL=debug

# 2. Try adding a notification
tmux-intray add "test message" 2>&1

# 3. Check if it was stored
tmux-intray list

# 4. Look for these patterns in output:
#    - "Notification added" → Stored successfully
#    - "Failed to open database" → Storage issue
#    - "Invalid notification level" → Format issue
```

**What to look for**:
- `[INFO] Notification added` → Success
- `[ERROR] Failed to add notification` → Storage failure
- `[WARN] Hook failed` → Pre-add hook is blocking

### Scenario 2: Command-Line Arguments Not Working

**Problem**: Command line flags or options are being ignored.

**Debug steps**:
```bash
# Enable debug logging to see argument parsing
export TMUX_INTRAY_LOG_LEVEL=debug

# Run the command with flags
tmux-intray add --level=error "Something failed" 2>&1 | head -20

# Look for:
# - Argument parsing debug messages
# - "level: error" in configuration output
# - "Notification added" with correct level
```

### Scenario 3: Tmux Integration Problems

**Problem**: Tmux plugin isn't loading or status bar isn't updating.

**Debug steps**:
```bash
# 1. Check if tmux-intray binary is accessible
which tmux-intray
echo $PATH

# 2. Test the status command directly
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray status --format=compact 2>&1

# 3. Check for tmux connectivity issues
# Look for messages like:
#    - "Failed to connect to tmux"
#    - "Invalid tmux session"
#    - "Pane not found"
```

**Manual status bar test**:
```bash
# Check what the status bar would show
tmux-intray status --format=compact

# If it shows output, the CLI is working
# If it shows nothing or errors, check the logs above
```

### Scenario 4: Configuration Issues

**Problem**: Configuration settings aren't being applied.

**Debug steps**:
```bash
# 1. Enable debug and view what configuration is loaded
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray list 2>&1 | grep -i config

# 2. Verify configuration file exists
cat ~/.config/tmux-intray/config.toml

# 3. Check for parse errors
# Look for "Configuration file not found" or parse error messages

# 4. Try using environment variables instead
export TMUX_INTRAY_LOG_LEVEL=debug
export TMUX_INTRAY_MAX_NOTIFICATIONS=500
tmux-intray list 2>&1
```

**Configuration validation**:
```bash
# Check if config file is valid TOML
# Use online TOML validator or:
python3 -m pip install toml
python3 -c "import toml; toml.load(open(open(os.path.expanduser('~/.config/tmux-intray/config.toml'))))"
```

### Scenario 5: Hook Scripts Failing

**Problem**: Custom hook scripts are causing errors.

**Debug steps**:
```bash
# 1. Enable debug and add a notification to trigger pre-add hook
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray add "test" 2>&1

# 2. Look for hook-related messages:
#    - "[WARN] Hook failed: pre-add"
#    - "[ERROR] Hook script not executable"

# 3. Test the hook directly
bash ~/.config/tmux-intray/hooks/pre-add.sh

# 4. Check hook script permissions
ls -la ~/.config/tmux-intray/hooks/
chmod +x ~/.config/tmux-intray/hooks/*.sh
```

### Scenario 6: Database Issues

**Problem**: Corruption, permission errors, or storage failures.

**Debug steps**:
```bash
# 1. Enable debug logging
export TMUX_INTRAY_LOG_LEVEL=debug

# 2. Try a basic operation
tmux-intray list 2>&1 | grep -i "database\|storage\|sql"

# 3. Check database file exists
ls -lah ~/.local/state/tmux-intray/notifications.db

# 4. Check permissions
# Should be readable/writable by current user
stat ~/.local/state/tmux-intray/notifications.db

# 5. If corrupted, backup and reset
mv ~/.local/state/tmux-intray/notifications.db ~/.local/state/tmux-intray/notifications.db.bak
tmux-intray list  # Creates fresh database
```

## Combining Logs with Other Tools

### Filtering by Level

```bash
# Show only errors and warnings
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray list 2>&1 | grep -E "\[WARN\]|\[ERROR\]"

# Show only debug messages
tmux-intray list 2>&1 | grep "\[DEBUG\]"

# Show everything except debug
tmux-intray list 2>&1 | grep -v "\[DEBUG\]"
```

### Following Logs in Real Time

```bash
# Run a command and see all logs as they happen
tmux-intray add "something" 2>&1 | tee full-logs.txt

# Watch for errors
tmux-intray list 2>&1 | grep --line-buffered ERROR
```

### Capturing Logs to File

```bash
# Save all output to file
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray list 2>&1 | tee debug-session.log

# Keep only the log file, discard stdout
tmux-intray list > /dev/null 2>&1 | tee debug.log

# Rotate logs manually
mv debug.log debug.log.1
```

### Searching for Specific Issues

```bash
# Find all database-related messages
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray list 2>&1 > full.log
grep -i "database\|sqlite\|sql" full.log

# Find timeline of an operation
grep "Notification added.*test" full.log
# See what happened before and after
```

## Troubleshooting Common Issues

### Issue: "command not found: tmux-intray"

**Solution**:
```bash
# 1. Check if binary is installed
which tmux-intray

# 2. Add installation directory to PATH
export PATH="$HOME/.local/bin:$PATH"

# 3. Verify installation
tmux-intray version  # or appropriate version command
```

**Debug**:
```bash
# Check npm global installation
npm bin -g
which tmux-intray

# Check Go installation
go list -m all | grep tmux-intray
```

### Issue: "Failed to connect to tmux"

**Solution**:
```bash
# 1. Check if tmux is running
tmux list-sessions

# 2. Try creating a session
tmux new-session -d -s test

# 3. Run tmux-intray inside tmux
tmux send-keys -t test 'tmux-intray list' Enter

# 4. Enable debug to see connection details
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray list 2>&1 | grep -i "tmux\|session"
```

### Issue: "Permission denied" on database

**Solution**:
```bash
# 1. Check directory permissions
ls -la ~/.local/state/tmux-intray/

# 2. Fix permissions
chmod 700 ~/.local/state/tmux-intray/
chmod 600 ~/.local/state/tmux-intray/notifications.db

# 3. Verify
tmux-intray list
```

### Issue: "Invalid configuration" errors

**Solution**:
```bash
# 1. Check configuration file syntax
cat ~/.config/tmux-intray/config.toml

# 2. Reset to defaults (creates sample config)
rm ~/.config/tmux-intray/config.toml
tmux-intray status

# 3. Enable debug to see what's happening
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray list 2>&1 | grep -i config
```

### Issue: Slow performance or high memory usage

**Solution**:
```bash
# 1. Check notification count
tmux-intray list | wc -l

# 2. Enable debug to see what's slow
export TMUX_INTRAY_LOG_LEVEL=debug
time tmux-intray list 2>&1

# 3. Clean up old notifications
tmux-intray cleanup --days=7

# 4. Reduce max notifications
export TMUX_INTRAY_MAX_NOTIFICATIONS=100
```

## Development Guide: Adding Logging to Code

If you're extending tmux-intray or contributing, here's how to add logging to your code:

### Basic Usage

```go
package mypackage

import (
    "github.com/cristianoliveira/tmux-intray/internal/log"
)

func DoSomething() {
    // Log at different levels
    log.Debug("Entering DoSomething with value: %v", someValue)
    log.Info("Operation started")
    
    if err := risky(); err != nil {
        log.Error("Operation failed: %v", err)
        return
    }
    
    log.Info("Operation completed successfully")
}
```

### What to Log

**DO log**:
- Important state transitions (started, completed, failed)
- Configuration values (helps diagnose config issues)
- Input validation (why something was rejected)
- External interactions (file I/O, database queries, tmux commands)
- Performance milestones (helps diagnose slowness)

**DON'T log**:
- Sensitive data (passwords, tokens, private information)
- Every line of code execution (use `debug` level only)
- Noise that doesn't help debugging
- User-visible output (stdout is for that)

### Log Level Selection

- **Debug**: Tracing function entry/exit, variable values, internal flow
- **Info**: User-facing events, key milestones, normal operation
- **Warn**: Unusual but non-fatal conditions, missing optional files
- **Error**: Failures and exceptions

### Examples

```go
// Good debug logging for troubleshooting
log.Debug("Loading config from %s", configPath)
log.Debug("Config parsed: %+v", config)
log.Debug("Opening database at %s", dbPath)

// Good info logging for operation tracking
log.Info("Notification added with ID: %s", id)
log.Info("Running hook: %s", hookName)

// Good warn logging for unusual conditions
log.Warn("Hook script not found: %s", hookPath)
log.Warn("Configuration file missing, using defaults")

// Good error logging for failures
log.Error("Failed to parse configuration: %v", err)
log.Error("Database query failed: %v", err)
```

## Tips & Tricks

### Quick Debug Session

```bash
# Create a debug environment
export TMUX_INTRAY_LOG_LEVEL=debug
export PS1='[DEBUG] $ '  # Mark your prompt

# Now run commands, all output includes logs
tmux-intray add "test"
tmux-intray list
```

### Save & Share Logs for Support

```bash
# Capture everything for bug reports
export TMUX_INTRAY_LOG_LEVEL=debug

# Run the failing command and save logs
tmux-intray <failing-command> 2>&1 > issue-logs.txt

# Share the logs (redact sensitive info first)
# Check for passwords, tokens, home paths, etc.
cat issue-logs.txt | less
```

### Piping Logs to External Tools

```bash
# Send logs to system log (if available)
tmux-intray list 2>&1 | logger -t tmux-intray

# Create a searchable log with timestamps
tmux-intray list 2>&1 | while IFS= read -r line; do
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $line"
done | tee session.log
```

### Monitor in Tmux Window

```bash
# Create a dedicated tmux window for logs
tmux new-window -n debug

# Run debug command and keep it running
tmux send-keys -t debug 'export TMUX_INTRAY_LOG_LEVEL=debug; tmux-intray list' Enter

# Watch from another window
tmux send-keys -t <other> 'tail -f session.log' Enter
```

## When Logging Isn't Enough

If you've exhausted logging options, try these:

1. **Examine the database directly**:
   ```bash
   sqlite3 ~/.local/state/tmux-intray/notifications.db "SELECT * FROM notifications LIMIT 5;"
   ```

2. **Check environment variables**:
   ```bash
   env | grep TMUX_INTRAY
   ```

3. **Verify tmux setup**:
   ```bash
   tmux show-environment -g
   tmux list-panes -a
   ```

4. **Test with minimal configuration**:
   ```bash
   # Create a fresh config directory
   mkdir -p /tmp/tmux-intray-test/.config
   mkdir -p /tmp/tmux-intray-test/.local/state
   
   export TMUX_INTRAY_CONFIG_DIR=/tmp/tmux-intray-test/.config
   export TMUX_INTRAY_STATE_DIR=/tmp/tmux-intray-test/.local/state
   export TMUX_INTRAY_LOG_LEVEL=debug
   
   tmux-intray list 2>&1
   ```

## Related Documentation

- [Configuration Guide](./configuration.md) - Full list of environment variables and config options
- [CLI Reference](./cli/CLI_REFERENCE.md) - All available commands
- [Troubleshooting Guide](./troubleshooting.md) - Other common issues and solutions
- [Architecture](./design/) - Design documents if you need to understand internals
