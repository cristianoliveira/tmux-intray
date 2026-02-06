# Troubleshooting tmux-intray

This guide covers common issues and how to resolve them. If you encounter a problem not listed here, please [open an issue](https://github.com/cristianoliveira/tmux-intray/issues).

## Quick Diagnostic Checklist

1. **Is tmux running?**  
   `tmux-intray` requires an active tmux session for most commands. Run `tmux has-session` to verify.

2. **Is the CLI in your PATH?**  
   Run `which tmux-intray`. If not found, ensure your installation directory (e.g., `~/.local/bin`) is in `$PATH`.

3. **Are environment variables set correctly?**  
   Check `echo $TMUX_INTRAY_STATE_DIR`, `echo $TMUX_INTRAY_CONFIG_DIR`. Ensure these directories exist and are writable.

4. **Enable debug mode**  
   `export TMUX_INTRAY_DEBUG=1` and re‑run the failing command. Look for debug messages on stderr.

5. **Check the log file**  
   If `TMUX_INTRAY_LOG_FILE` is set, examine that file. Otherwise, debug output goes to stderr.

6. **Verify hooks permissions**  
   Ensure hook scripts are executable (`chmod +x ~/.config/tmux-intray/hooks/*/*.sh`).

---

## Common Issues

### “tmux-intray: command not found” or sourcing errors

**Symptoms:**  
- `tmux-intray` not found after installation.
- Error messages about missing scripts or sourcing failures.

**Causes:**  
- Installation directory not in `$PATH`.
- The installation script may have placed the binary in a location not included in your shell’s search path.
- If you installed via npm, the global npm bin directory may not be in your PATH.

**Solutions:**  
1. **Check installation location**  
   - npm: `npm bin -g`  
   - Manual: look in `~/.local/bin` or `/usr/local/bin`

2. **Add directory to PATH**  
   Add the following to your shell profile (`.bashrc`, `.zshrc`, etc.):
   ```bash
   export PATH="$HOME/.local/bin:$PATH"
   ```
   Then restart your shell or run `source ~/.bashrc`.

3. **Re‑run installation**  
   The installation script now uses absolute paths to avoid sourcing issues. Re‑install using the one‑line installer:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/cristianoliveira/tmux-intray/main/install.sh | bash
   ```

### Hooks not running

**Symptoms:**  
- Custom hook scripts are not executed.
- No output from hooks.

**Causes:**  
- Hooks are disabled globally (`TMUX_INTRAY_HOOKS_ENABLED=0`).
- The specific hook point is disabled (e.g., `TMUX_INTRAY_HOOKS_ENABLED_pre_add=0`).
- Hook scripts are not executable.
- Hook directory structure is incorrect.
- Hook failure mode is set to `abort` and a previous hook failed.

**Solutions:**  
1. **Verify hooks are enabled**  
   ```bash
   export TMUX_INTRAY_DEBUG=1
   tmux-intray add "test hook"
   ```
   Look for debug messages about hooks.

2. **Check hook directory structure**  
   Hooks should be placed in `$TMUX_INTRAY_HOOKS_DIR/<hook-point>/`. Example:
   ```
   ~/.config/tmux-intray/hooks/
   ├── pre-add/
   │   └── 01-log.sh
   ├── post-add/
   │   └── 99-notify.sh
   └── cleanup/
       └── 50-archive.sh
   ```
   Each script must be executable (`chmod +x`).

3. **Enable per‑hook logging**  
   Set `TMUX_INTRAY_HOOKS_FAILURE_MODE=warn` to see warnings when a hook fails.

4. **Test a simple hook**  
   Create a test hook that writes to a file:
   ```bash
   echo '#!/bin/bash' > ~/.config/tmux-intray/hooks/post-add/test.sh
   echo 'date >> /tmp/tmux-intray-hook.log' >> ~/.config/tmux-intray/hooks/post-add/test.sh
   chmod +x ~/.config/tmux-intray/hooks/post-add/test.sh
   tmux-intray add "hook test"
   cat /tmp/tmux-intray-hook.log
   ```

### Cleanup not removing entries

**Symptoms:**  
- Dismissed notifications remain in storage.
- `tmux-intray cleanup` reports “nothing to clean up” but old entries persist.

**Causes:**  
- `TMUX_INTRAY_AUTO_CLEANUP_DAYS` is set too high.
- Cleanup hooks are failing and the failure mode is `abort`.
- Storage file permissions prevent deletion.

**Solutions:**  
1. **Run manual cleanup with explicit days**  
   ```bash
   tmux-intray cleanup --days 0
   ```
   This removes **all** dismissed notifications (use with caution).

2. **Check the auto‑cleanup threshold**  
   ```bash
   echo $TMUX_INTRAY_AUTO_CLEANUP_DAYS
   ```
   The default is 30 days. If you want more aggressive cleanup, set it lower:
   ```bash
   export TMUX_INTRAY_AUTO_CLEANUP_DAYS=7
   tmux-intray cleanup
   ```

3. **Dry‑run to see what would be removed**  
   ```bash
   tmux-intray cleanup --dry-run
   ```

4. **Inspect storage file**  
   The storage file is at `$TMUX_INTRAY_STATE_DIR/notifications.tsv`. You can view its contents (but do not edit while tmux-intray is running):
   ```bash
   head -20 "$TMUX_INTRAY_STATE_DIR/notifications.tsv"
   ```

### Storage permission problems

**Symptoms:**  
- Errors like “Cannot create directory”, “Permission denied”, “Failed to lock storage”.
- Notifications are not saved.

**Causes:**  
- The state directory (`$TMUX_INTRAY_STATE_DIR`) is not writable.
- The storage file is locked by another process (e.g., a previous tmux-intray command that crashed).
- Running tmux-intray under a different user (e.g., via sudo).

**Solutions:**  
1. **Check directory permissions**  
   ```bash
   ls -ld "$TMUX_INTRAY_STATE_DIR"
   ```
   Ensure your user has write access.

2. **Recreate the directory**  
   ```bash
   rm -rf "$TMUX_INTRAY_STATE_DIR"
   tmux-intray add "test"
   ```
   This will recreate the directory with proper permissions.

3. **Remove stale lock files**  
   The lock file is `$TMUX_INTRAY_STATE_DIR/notifications.tsv.lock`. If you’re sure no other tmux-intray process is running:
   ```bash
   rm -f "$TMUX_INTRAY_STATE_DIR/notifications.tsv.lock"
   ```

4. **Avoid running with sudo**  
   tmux-intray should run as your regular user, not root.

### Debugging tips

**Enable verbose output**  
```bash
export TMUX_INTRAY_DEBUG=1
tmux-intray list
```

Debug messages are printed to stderr and include:
- Configuration loading
- Storage operations
- Hook execution
- Command‑line parsing

**Check the storage file directly**  
If you suspect data corruption, you can examine the TSV file:
```bash
column -t -s $'\t' "$TMUX_INTRAY_STATE_DIR/notifications.tsv" | head -20
```

**Run the test suite**  
The project includes a comprehensive test suite that can verify your installation:
```bash
make test
```

**Increase debug level**  
Some components may have additional debug levels (e.g., `TMUX_INTRAY_DEBUG=2`). Check the source code for details.

### “No tmux session running”

**Symptoms:**  
Commands fail with “No tmux session running”.

**Causes:**  
- You are not inside a tmux session.
- tmux is running but the environment variables (`TMUX`, `TMUX_PANE`) are not set (e.g., when running via `sudo`).

**Solutions:**  
- Start a tmux session (`tmux new -s mysession`) or attach to an existing one (`tmux attach`).
- If you need to run tmux-intray outside tmux (e.g., from a cron job), use the `--no-associate` flag with `add` and set `TMUX_INTRAY_STATUS_ENABLED=0`.

---

## Still stuck?

If none of the above solutions work, please collect the following information and open an issue:

1. **tmux-intray version**  
   `tmux-intray version`

2. **Debug output**  
   `TMUX_INTRAY_DEBUG=1 tmux-intray <command> 2>&1`

3. **Environment**  
   `env | grep TMUX_INTRAY`

4. **Storage directory listing**  
   `ls -la "$TMUX_INTRAY_STATE_DIR"`

5. **tmux version**  
   `tmux -V`

6. **Operating system**  
   `uname -a`