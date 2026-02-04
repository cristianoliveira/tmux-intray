# Hooks System

**Status**: *Implemented* - Available since tmux-intray v1.0

## Overview

The hooks system in tmux-intray allows you to execute custom scripts before and after key notification events. This makes tmux-intray a pluggable and modular notification system that can integrate with other tools, automate workflows, and extend functionality without modifying the core codebase.

## Hook Points

tmux-intray supports the following hook points:

| Hook Point | Trigger | Use Cases |
|------------|---------|-----------|
| `pre-add` | Before a notification is added to storage | Validate notifications, enrich with metadata, filter spam |
| `post-add` | After a notification is successfully added | Trigger external alerts (Slack, email), log to external systems, update dashboards |
| `pre-dismiss` | Before a notification is dismissed | Confirm dismissal, check conditions, backup before removal |
| `post-dismiss` | After a notification is dismissed | Clean up related resources, update external systems, trigger follow-up actions |
| `cleanup` | After garbage collection runs | Archive old notifications, update metrics, perform maintenance |
| `post-list` | After listing notifications (planned) | Audit logging, analytics, usage tracking |

## Hook Script Location

Hook scripts are placed in the following directory structure:

```
~/.config/tmux-intray/hooks/
├── pre-add/
│   ├── 01-validate.sh
│   ├── 02-enrich.sh
│   └── 99-log.sh
├── post-add/
│   ├── 01-slack-notify.sh
│   └── 99-log.sh
├── pre-dismiss/
│   └── 01-confirm.sh
├── post-dismiss/
│   └── 99-log.sh
└── cleanup/
    └── 01-archive.sh
```

- **Directory naming**: Hook scripts are organized in directories named after hook points
- **Execution order**: Scripts are executed in alphabetical/numerical order within each hook point
- **File extensions**: Use `.sh` extension for shell scripts, but any executable file will work

## Environment Variables

Each hook script receives relevant context via environment variables:

### Common Variables (available to all hooks)
- `HOOK_POINT` - Name of the hook point being executed (e.g., "pre-add")
- `HOOK_TIMESTAMP` - ISO 8601 timestamp when hook was triggered
- `TMUX_INTRAY_STATE_DIR` - Path to tmux-intray state directory
- `TMUX_INTRAY_CONFIG_DIR` - Path to tmux-intray config directory

### Notification-Specific Variables (available where applicable)

Variables are available in two forms for backward compatibility (e.g., `LEVEL` and `NOTIFICATION_LEVEL` both contain the same value):

- `NOTIFICATION_ID` - Unique ID of the notification
- `LEVEL` / `NOTIFICATION_LEVEL` - Severity level (info, warning, error, critical)
- `MESSAGE` / `NOTIFICATION_MESSAGE` - The notification message content
- `TIMESTAMP` / `NOTIFICATION_TIMESTAMP` - When the notification was created (ISO 8601)
- `SESSION` / `NOTIFICATION_SESSION` - tmux session ID where notification originated
- `WINDOW` / `NOTIFICATION_WINDOW` - tmux window ID where notification originated
- `PANE` / `NOTIFICATION_PANE` - tmux pane ID where notification originated
- `PANE_CREATED` / `NOTIFICATION_PANE_CREATED` - Timestamp when pane was created
- `ESCAPED_MESSAGE` / `NOTIFICATION_ESCAPED_MESSAGE` - Escaped message for safe shell usage
- `NOTIFICATION_STATE` - Current state (active, dismissed) - defaults to "active"

### Example Hook Script

```bash
#!/usr/bin/env bash
# ~/.config/tmux-intray/hooks/pre-add/99-log.sh
# Log all notifications to a file

LOG_FILE="${HOOK_LOG_FILE:-$HOME/.local/state/tmux-intray/hooks.log}"

# Ensure directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Append log entry
{
    echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ") [${HOOK_POINT}] ID=${NOTIFICATION_ID:-new} level=${NOTIFICATION_LEVEL:-unknown}"
    echo "  message: ${NOTIFICATION_MESSAGE:-}"
    echo "  context: ${NOTIFICATION_SESSION:-none}/${NOTIFICATION_WINDOW:-none}/${NOTIFICATION_PANE:-none}"
} >>"$LOG_FILE" 2>/dev/null || true
```

## Hook Execution

### Synchronous vs Asynchronous Hooks
- **Synchronous (default)**: Hook script runs and tmux-intray waits for completion
- **Asynchronous**: Hook script runs in background, tmux-intray continues immediately
- **Configuration**: Set `TMUX_INTRAY_HOOKS_ASYNC=1` for all hooks, or per-hook with `TMUX_INTRAY_HOOKS_ASYNC_pre_add=1`

### Error Handling
- **Ignore (default)**: Hook failures are logged but don't block the operation
- **Warn**: Hook failures generate warnings but don't block the operation
- **Abort**: Hook failures abort the tmux-intray operation
- **Configuration**: `TMUX_INTRAY_HOOKS_FAILURE_MODE=ignore|warn|abort`

### Performance Considerations
- Hooks add overhead to notification operations
- Asynchronous hooks use double-fork to avoid zombie processes; the parent process does not wait for child termination
- Asynchronous hooks minimize impact but lose error feedback
- Consider batching external calls in hooks that run frequently

## Configuration

Hooks are configured via environment variables or the configuration file:

```bash
# In ~/.config/tmux-intray/config.sh

# Enable/disable hooks globally
TMUX_INTRAY_HOOKS_ENABLED=1

# Hook failure mode (ignore, warn, abort)
TMUX_INTRAY_HOOKS_FAILURE_MODE="ignore"

# Run hooks asynchronously (0=sync, 1=async)
TMUX_INTRAY_HOOKS_ASYNC=0

# Per-hook configuration
TMUX_INTRAY_HOOKS_ENABLED_pre_add=1
TMUX_INTRAY_HOOKS_ASYNC_pre_add=0
TMUX_INTRAY_HOOKS_FAILURE_MODE_pre_add="warn"
```

## Example Use Cases

### 1. Notification Validation
```bash
#!/usr/bin/env bash
# ~/.config/tmux-intray/hooks/pre-add/01-validate.sh
# Reject notifications containing sensitive data

if [[ "$NOTIFICATION_MESSAGE" =~ (password|secret|token)= ]]; then
    echo "ERROR: Notification contains sensitive data" >&2
    exit 1  # This will prevent the notification from being added
fi
```

### 2. External Alerting
```bash
#!/usr/bin/env bash
# ~/.config/tmux-intray/hooks/post-add/01-slack.sh
# Send critical notifications to Slack

if [[ "$NOTIFICATION_LEVEL" == "critical" ]]; then
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"🚨 tmux-intray: $NOTIFICATION_MESSAGE\"}" \
        "$SLACK_WEBHOOK_URL" >/dev/null 2>&1 &
fi
```

### 3. Notification Enrichment
```bash
#!/usr/bin/env bash
# ~/.config/tmux-intray/hooks/pre-add/02-enrich.sh
# Add hostname and user to notification message

HOSTNAME=$(hostname -s)
USERNAME=$(whoami)
export NOTIFICATION_MESSAGE="[$USERNAME@$HOSTNAME] $NOTIFICATION_MESSAGE"
```

### 4. Audit Logging
```bash
#!/usr/bin/env bash
# ~/.config/tmux-intray/hooks/post-dismiss/99-audit.sh
# Log all dismissals for compliance

AUDIT_LOG="$HOME/.local/state/tmux-intray/audit.log"
echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ") DISMISSED $NOTIFICATION_ID by user $(whoami)" >> "$AUDIT_LOG"
```

## Security Considerations

### Hook Script Security
- Hook scripts run with the same permissions as the tmux-intray process
- Only install hook scripts from trusted sources
- Review hook scripts for security issues
- Consider setting restrictive file permissions on hook directories

### Environment Variable Sanitization
- Sensitive data should not be passed via environment variables
- Consider using a secure configuration store for secrets
- Hook scripts should sanitize inputs if passing to external systems

## Best Practices

### 1. Keep Hooks Simple
- Hooks should do one thing well
- Complex logic belongs in separate scripts called by hooks
- Use exit codes appropriately (0=success, non-zero=failure)

### 2. Handle Errors Gracefully
- Don't let hook failures break tmux-intray core functionality
- Log errors to appropriate destinations
- Consider fallback mechanisms for critical hooks

### 3. Performance Optimization
- Use asynchronous hooks for long-running operations
- Batch external API calls when possible
- Implement rate limiting for hooks that call external services

### 4. Testing Hooks
- Test hooks in isolation before deploying
- Use dry-run mode to verify hook execution
- Monitor hook performance and error rates

## Troubleshooting

### Common Issues

1. **Hooks not executing**
   - Check `TMUX_INTRAY_HOOKS_ENABLED` setting
   - Verify hook script permissions (`chmod +x script.sh`)
   - Check hook script location and naming

2. **Hook failures blocking operations**
   - Check `TMUX_INTRAY_HOOKS_FAILURE_MODE` setting
   - Review hook script error messages
   - Test hook script manually with same environment variables

3. **Performance issues**
   - Consider enabling asynchronous execution
   - Review hook script performance
   - Check for infinite loops or blocking calls

### Debugging Hooks

Enable debug logging for hooks:

```bash
export TMUX_INTRAY_LOG_LEVEL=DEBUG
export TMUX_INTRAY_LOG_FILE="$HOME/.local/state/tmux-intray/debug.log"
```

Check hook execution logs:

```bash
tail -f ~/.local/state/tmux-intray/debug.log | grep -i hook
```

## Migration from Previous Versions

If you're using custom integrations with tmux-intray, the hooks system provides a structured way to migrate:

1. **Identify integration points** - What external systems does tmux-intray interact with?
2. **Map to hook points** - Which hook points match your integration needs?
3. **Create hook scripts** - Convert existing integration code to hook scripts
4. **Test and deploy** - Test hooks in a staging environment before production

## Future Enhancements

The hooks system is designed to be extensible. Future enhancements may include:

1. **Dynamic hook registration** - Register hooks at runtime without file system changes
2. **Hook dependencies** - Define execution order and dependencies between hooks
3. **Hook templates** - Reusable hook script templates for common use cases
4. **Remote hooks** - Execute hooks on remote systems via SSH or HTTP
5. **Hook marketplace** - Share and discover community-contributed hook scripts

## Getting Help

- Check the [tmux-intray documentation](../README.md)
- Review example hooks in the `examples/hooks/` directory
- File issues on the [GitHub repository](https://github.com/tmux-plugins/tmux-intray)
- Join the discussion in the tmux-intray community

---

*Note: The hooks system is a planned feature. Implementation details may change before release. Check the [tmux-intray changelog](../CHANGELOG.md) for updates.*