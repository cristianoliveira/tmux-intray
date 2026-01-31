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

## Usage

To start using tmux-intray, simply run the command `tmux-intray` in your terminal.

```bash
$ tmux-intray --help
```

### Commands

- `tmux-intray show` - Show all items in the tray (deprecated, use `list`)
- `tmux-intray add <message>` - Add a new item to the tray
- `tmux-intray list` - List notifications with filters and formats (e.g., `--active`, `--dismissed`, `--all`, `--format=table`)
- `tmux-intray dismiss <id>` - Dismiss a specific notification
- `tmux-intray dismiss --all` - Dismiss all active notifications
- `tmux-intray clear` - Clear all items from the tray (alias for `dismiss --all`)
- `tmux-intray toggle` - Toggle the tray visibility
- `tmux-intray help` - Show help message
- `tmux-intray version` - Show version information

### Storage

tmux-intray now stores notifications in a file-based TSV storage located at `~/.local/state/tmux-intray/` (following XDG Base Directory Specification). Notifications persist across tmux server restarts.

### Configuration

A sample configuration file is created at `~/.config/tmux-intray/config.sh` on first run. You can customize storage limits, display formats, and more.

## Testing

This project uses [Bats](https://github.com/bats-core/bats-core) for testing.

To run the tests:

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

```bash
$ scripts/lint.sh # go over all files in the project and lint them
```

Or:

```bash
make lint
```

## License

[MIT](LICENSE)
