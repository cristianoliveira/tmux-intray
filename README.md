tmux-intray

A quiet inbox for things that happen while you’re not looking.

tmux-intray provides a persistent in-tmux in-tray where panes, windows, and scripts can drop messages and events without interrupting your flow. Instead of loud notifications or forced context switches, events accumulate calmly until you’re ready to review them. Each item keeps its origin, survives pane and window changes, and can be inspected, jumped to, or cleared at your own pace. It’s designed for deferred attention: notice now if you want, act later when it makes sense.

## Installation

### Using [Tmux Plugin Manager](https://github.com/tmux-plugins/tpm) (recommended)

Add plugin to the list of TPM plugins in `.tmux.conf`:

```
set -g @plugin 'tmux-plugins/tmux-intray'
```

### Manual Installation

Clone the repo:

```bash
$ git clone https://github.com/tmux-plugins/tmux-intray ~/clone/path
```

Add this line to the bottom of `.tmux.conf`:

```
run-shell ~/clone/path/tmux-intray.tmux
```

Reload TMUX environment:

```bash
# type this in terminal
$ tmux source-file ~/.tmux.conf
```

### One-click installation (curl/bash)

For the easiest installation without package managers, you can use our one-click installer:

```bash
curl -fsSL https://raw.githubusercontent.com/cristianoliveira/tmux-intray/main/install.sh | bash
```

This will:
- Download the latest release from GitHub
- Install tmux-intray to `~/.local/bin/` (or `/usr/local/bin/` if needed)
- Add the installation directory to your PATH if necessary
- Provide clear success/error messages

**Troubleshooting tips:**
- If you get a "permission denied" error, ensure `~/.local/bin/` is in your PATH and executable
- For system-wide installation, you can run with sudo: `curl ... | sudo bash`
- If curl fails, ensure you have curl installed (install via package manager)

**Alternative:** If you prefer to inspect the installer before running, you can download and run it separately:

```bash
curl -fsSL -o install.sh https://raw.githubusercontent.com/cristianoliveira/tmux-intray/main/install.sh
chmod +x install.sh
./install.sh
```

## CLI Installation

The tmux-intray CLI tool can be installed via several package managers:

### Homebrew (macOS/Linux)

```bash
brew install https://raw.githubusercontent.com/cristianoliveira/tmux-intray/main/Formula/tmux-intray.rb
```

### Docker

```bash
# Run directly via Docker (build locally)
git clone https://github.com/cristianoliveira/tmux-intray
cd tmux-intray
docker build -t tmux-intray .
docker run --rm tmux-intray version
```

### npm

```bash
npm install -g tmux-intray
```

### Go

```bash
go install github.com/cristianoliveira/tmux-intray/cmd/tmux-intray@latest
```

### From Source

```bash
git clone https://github.com/cristianoliveira/tmux-intray
cd tmux-intray
make install
```

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
