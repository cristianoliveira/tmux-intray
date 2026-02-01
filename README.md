# tmux-intray

<img width="300" height="300" alt="tmux-intray-300" src="https://github.com/user-attachments/assets/4fd9f030-9bb3-43a7-b800-c0d0f479e2a5" align="right" />

A quiet inbox for things that happen while you’re not looking.

<div>
    
tmux-intray provides a persistent in-tmux in-tray where panes, windows, and scripts can drop messages and events without interrupting your flow. Instead of loud notifications or forced context switches, events accumulate calmly until you’re ready to review them. Each item keeps its origin, survives pane and window changes, and can be inspected, jumped to, or cleared at your own pace. It’s designed for deferred attention: notice now if you want, act later when it makes sense.
</div>

## Summary

Quick links to key sections:

### Main Sections
- [Installation](#installation)
- [CLI Installation](#cli-installation)
- [Usage](#usage)
- [Debugging](#debugging)
- [Testing](#testing)
- [Linting](#linting)
- [License](#license)

### Installation Methods
- [Using Tmux Plugin Manager (recommended)](#using-tmux-plugin-manager-recommended)
- [Manual Installation](#manual-installation)
- [One-click installation (curl/bash)](#one-click-installation-curlbash)
- [Homebrew (macOS/Linux)](#homebrew-macoslinux)
- [Docker](#docker)
- [npm](#npm)
- [Go](#go)
- [From Source](#from-source)


## Installation

### Manual Installation

1. Clone the repository:

```bash
git clone https://github.com/tmux-intray/tmux-intray.git ~/.local/share/tmux-plugins/tmux-intray
```

2. Add the plugin to your `.tmux.conf`:

```bash
# Add to the bottom of your .tmux.conf
run '~/.local/share/tmux-plugins/tmux-intray/tmux-intray.tmux'
```

3. Reload TMUX to apply changes:

```bash
tmux source-file ~/.tmux.conf
```

### Alternative Installation Methods

#### Direct Download

```bash
# Download the plugin files
curl -L https://github.com/tmux-intray/tmux-intray/archive/refs/heads/main.zip -o tmux-intray.zip
unzip tmux-intray.zip
mv tmux-intray-main ~/.local/share/tmux-plugins/tmux-intray
```

Then follow steps 2-3 from the manual installation section above.

#### Using a symbolic link

```bash
# Clone the repository to your preferred location
git clone https://github.com/tmux-intray/tmux-intray.git ~/projects/tmux-intray

# Create a symbolic link to the tmux plugins directory
ln -s ~/projects/tmux-intray ~/.local/share/tmux-plugins/tmux-intray
```

Then follow steps 2-3 from the manual installation section above.

#### Using git-archive

```bash
# Clone the repository
git clone https://github.com/tmux-intray/tmux-intray.git ~/.local/share/tmux-plugins/tmux-intray

# Use the plugin
# Add the plugin loader to the bottom of .tmux.conf:
# run '~/.local/share/tmux-plugins/tmux-intray/tmux-intray.tmux'
```

Then reload your tmux configuration.

## Usage

To start using tmux-intray, simply run the command `tmux-intray` in your terminal.

```bash
$ tmux-intray --help
```

### Commands

- `tmux-intray show` - Show all items in the tray (deprecated, use `list`)
- `tmux-intray add <message>` - Add a new item to the tray (options: `--level`, `--session`, `--window`, `--pane`, `--no-associate`)
- `tmux-intray list` - List notifications with filters and formats (e.g., `--active`, `--dismissed`, `--all`, `--level`, `--pane`, `--format=table`)
- `tmux-intray dismiss <id>` - Dismiss a specific notification
- `tmux-intray dismiss --all` - Dismiss all active notifications
- `tmux-intray clear` - Clear all items from the tray (alias for `dismiss --all`)
- `tmux-intray toggle` - Toggle the tray visibility
- `tmux-intray jump <id>` - Jump to the pane of a notification
- `tmux-intray status` - Show notification status summary
- `tmux-intray follow` - Monitor notifications in real-time
- `tmux-intray help` - Show help message
- `tmux-intray version` - Show version information

### Notification Levels

Notifications can have severity levels: `info` (default), `warning`, `error`, `critical`. Levels are used for filtering and color-coded display.

- Add a notification with a level:
  ```bash
  tmux-intray add --level=error "Something went wrong"
  ```
- Filter notifications by level:
  ```bash
  tmux-intray list --level=error
  ```
- The `status` command shows counts per level.

### Hooks System

tmux-intray supports a powerful hooks system that allows you to execute custom scripts before and after notification events. This makes tmux-intray extensible and integratable with other systems.

**Key features:**
- **Hook points**: `pre-add`, `post-add`, `pre-dismiss`, `post-dismiss`, `cleanup`
- **Custom scripts**: Place executable scripts in `~/.config/tmux-intray/hooks/` directories
- **Environment variables**: Receive notification context in your scripts
- **Configurable**: Enable/disable hooks, control error handling, sync/async execution

**Example hook** (log all notifications):
```bash
#!/usr/bin/env bash
# ~/.config/tmux-intray/hooks/pre-add/99-log.sh
LOG_FILE="$HOME/.local/state/tmux-intray/hooks.log"
mkdir -p "$(dirname "$LOG_FILE")"
echo "$(date) [pre-add] ${NOTIFICATION_LEVEL}: ${NOTIFICATION_MESSAGE}" >> "$LOG_FILE"
```

**Learn more**: See the complete [hooks documentation](docs/hooks.md) for detailed examples and configuration.

### Status Bar Integration

tmux-intray can display notification counts in the tmux status bar using the `status-panel` script.

1. Add the following to your `.tmux.conf`:
   ```
   set -g status-right "#(tmux-intray status-panel) %H:%M"
   ```
   This will show a compact indicator with the total notification count. Clicking on the indicator can be bound to open the notification list.

2. Customize the status format by setting environment variables in `config.sh`:
   ```bash
   TMUX_INTRAY_STATUS_FORMAT="detailed"   # compact, detailed, count-only
   TMUX_INTRAY_LEVEL_COLORS="info:green,warning:yellow,error:red,critical:magenta"
   TMUX_INTRAY_SHOW_LEVELS=0              # 0=only total, 1=show level counts
   ```

3. Enable/disable status indicator:
   ```bash
   TMUX_INTRAY_STATUS_ENABLED=1
   ```

The status indicator updates automatically when notifications change.

### Storage

tmux-intray now stores notifications in a file-based TSV storage located at `~/.local/state/tmux-intray/` (following XDG Base Directory Specification). Notifications persist across tmux server restarts.

### Configuration

A sample configuration file is created at `~/.config/tmux-intray/config.sh` on first run. You can customize storage limits, display formats, and more.

### Debugging

You can enable debug logging by setting the `TMUX_INTRAY_DEBUG` environment variable to any non-empty value. When enabled, debug messages will be printed to stderr in cyan color.

Example:
```bash
# Enable debug logging for a single command
TMUX_INTRAY_DEBUG=1 tmux-intray list

# Enable debug logging for the current shell session
export TMUX_INTRAY_DEBUG=1
tmux-intray add "Test notification"
tmux-intray status
```

Debug logs are useful for troubleshooting issues with notification storage, tmux integration, or configuration problems. Note that debug output is sent to stderr, so you can redirect it separately if needed.

## Testing

This project uses [Bats](https://github.com/bats-core/bats-core) for testing.

To run the tests:

With nix (preferable):
```bash
$ nix develop -c make tests
```

Without nix:
```bash
$ bats tests
```

Or:
```bash
make tests
```

With Docker (isolated environment):
```bash
$ ./scripts/docker-test.sh
```

This builds a Docker image with all dependencies and runs the test suite. You can also run specific commands:

```bash
$ ./scripts/docker-test.sh make lint   # Run linter
$ ./scripts/docker-test.sh bash        # Start interactive shell
```

Tests are located in the `tests` directory.

## Linting

This project uses [ShellCheck](https://www.shellcheck.net/) for linting.

To run the linter:

With nix (preferable):
```bash
$ nix develop -c make lint
```

Without nix:

```bash
$ scripts/lint.sh # go over all files in the project and lint them
```

Or:

```bash
make lint
```

## License

[MIT](LICENSE)
