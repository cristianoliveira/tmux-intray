# tmux-intray

<img width="300" height="300" alt="tmux-intray-300" src="https://github.com/user-attachments/assets/4fd9f030-9bb3-43a7-b800-c0d0f479e2a5" align="right" />

A quiet inbox for things that happen while you're not looking.

<div>

tmux-intray provides a persistent in-tmux in-tray where panes, windows, and scripts can drop messages and events without interrupting your flow. Instead of loud notifications or forced context switches, events accumulate calmly until you're ready to review them. Each item keeps its origin, survives pane and window changes, and can be inspected, jumped to, or cleared at your own pace. It's designed for deferred attention: notice now if you want, act later when it makes sense.
</div>
[![codecov](https://codecov.io/gh/cristianoliveira/tmux-intray/branch/main/graph/badge.svg)](https://codecov.io/gh/cristianoliveira/tmux-intray)

## Work in Progress

> [!WARNING]
> 🚧 This plugin is in active development at the moment. It started as an opencode plugin but grew into its own project.
I use it on a daily basis, I'm a heavy tmux user and so far it works great! At this stage of development I can't promise there won't be
breaking changes.

## Summary

Quick links to key sections:

### Main Sections
- [Installation Options](#installation-options)
- [CLI Installation](#cli-installation)
- [Tmux Plugin Installation](#tmux-plugin-installation)
- [Usage](#usage)
- [Fzf Integration](#fzf-integration)
- [Architecture Overview](#architecture-overview)
- [Debugging](#debugging)
- [Testing](#testing)
- [Linting](#linting)
- [License](#license)

### Quick Start
- **Full Setup (Recommended)**: Install CLI + Tmux Plugin via [One-click installation](#one-click-installation)
- **CLI Only**: Install via [npm](#npm) or [Go](#go) for tmux-integrated use
- **Plugin Only**: Install via [Tmux Plugin Manager](#using-tmux-plugin-manager-recommended) if CLI already installed

## Basic usage

```bash
tmux-intray add "my message!"
tmux-intray list
tmux-intray jump <id>
# or using fzf
tmux-intray list | fzf | awk '{ print $1 }' | xargs -I {} tmux-intray jump {}
```

## Using SQLite Storage (Beta Opt-in)

> [!WARNING]
> SQLite storage is in a gradual opt-in rollout. The default backend remains TSV.

SQLite support is available for users who want transactional storage and better scalability on larger inboxes. The SQLite backend uses sqlc-generated queries from `internal/storage/sqlite/queries.sql` (generated into `internal/storage/sqlite/sqlcgen/`).

Quick opt-in:

```bash
# one session
export TMUX_INTRAY_STORAGE_BACKEND=sqlite

# or persist in ~/.config/tmux-intray/config.sh
TMUX_INTRAY_STORAGE_BACKEND="sqlite"
```

Recommended rollout path:

1. Start with `TMUX_INTRAY_STORAGE_BACKEND=dual` to keep TSV as source-of-truth while validating SQLite writes.
2. Move to `TMUX_INTRAY_STORAGE_BACKEND=sqlite` after a stable period.
3. Roll back quickly by setting `TMUX_INTRAY_STORAGE_BACKEND=tsv`.

See the complete migration and rollback guide in [docs/storage-migration.md](docs/storage-migration.md).

## Installation Options

tmux-intray has two main components that can be installed separately or together:

1. **CLI (Command Line Interface)**: The core notification system that can be installed via package managers
2. **Tmux Plugin**: Integration layer that provides key bindings, status bar updates, and pane tracking

### Option 1: CLI + Tmux Plugin (Recommended)

For full functionality with tmux integration, install both components:

#### One-click installation (curl/bash)

```bash
# Installs both CLI and tmux plugin
curl -fsSL https://raw.githubusercontent.com/cristianoliveira/tmux-intray/main/install.sh | bash
```

This installs the CLI to `~/.local/bin` (or custom prefix) and the tmux plugin to `~/.local/share/tmux-plugins/tmux-intray`.

#### Manual Installation

```bash
# Clone the repository
git clone https://github.com/cristianoliveira/tmux-intray.git ~/.local/share/tmux-plugins/tmux-intray

# Add to your .tmux.conf
echo "run '~/.local/share/tmux-plugins/tmux-intray/tmux-intray.tmux'" >> ~/.tmux.conf

# Reload tmux configuration
tmux source-file ~/.tmux.conf
```

### Option 2: CLI Only

Install just the command-line interface for use within tmux sessions:

#### npm

```bash
npm install -g tmux-intray
```

#### Go (Recommended)

```bash
go install github.com/cristianoliveira/tmux-intray@latest
```

#### From Source

```bash
git clone https://github.com/cristianoliveira/tmux-intray.git
cd tmux-intray
make install
```

**Note**: The CLI requires tmux to be running for most commands. Installations via package managers provide the CLI only; you'll need to manually set up tmux integration if desired.

### Option 3: Tmux Plugin Only

If you already have the CLI installed (e.g., via npm), install just the tmux integration:

#### Using Tmux Plugin Manager (recommended)

Add to your `.tmux.conf`:

```bash
set -g @plugin 'cristianoliveira/tmux-intray'
```

Then press `prefix + I` to install.

#### Manual Plugin Installation

```bash
# Clone just the plugin files
git clone https://github.com/cristianoliveira/tmux-intray.git ~/.local/share/tmux-plugins/tmux-intray

# Add to .tmux.conf
echo "run '~/.local/share/tmux-plugins/tmux-intray/tmux-intray.tmux'" >> ~/.tmux.conf

# Reload tmux
tmux source-file ~/.tmux.conf
```

### Option 4: OpenCode Plugin

If you use [OpenCode](https://github.com/opencode/opencode), you can install the OpenCode plugin to receive notifications when OpenCode sessions complete, error, or require permissions.

#### Using the installation script

```bash
# Navigate to the plugin directory after cloning the repository
cd tmux-intray/opencode/plugins/opencode-tmux-intray

# Install globally (recommended)
./install.sh --global

# Or install locally
./install.sh --local
```

The installation script will:
- Copy plugin files to OpenCode plugin directories
- Install npm dependencies
- Add npm scripts for easy management

For detailed installation instructions and configuration, see the [plugin README](opencode/plugins/opencode-tmux-intray/README.md).

## CLI Installation

### What the CLI Provides

The tmux-intray CLI is a command-line interface for managing notifications within tmux sessions. It provides:

- Notification storage and retrieval
- Severity levels (info, warning, error, critical)
- Filtering and formatting options
- Pane association for notification origin tracking
- Hooks system for extensibility

### CLI Requirements

- **tmux**: Most commands require an active tmux session
- **Go**: Required for installation and execution

### CLI Commands

```bash
$ tmux-intray --help
```

#### Core Commands
- `tmux-intray add <message>` - Add a new item to the tray (options: `--level`, `--session`, `--window`, `--pane`, `--no-associate`)
- `tmux-intray list` - List notifications with filters and formats (e.g., `--active`, `--dismissed`, `--all`, `--level`, `--pane`, `--session`, `--window`, `--older-than`, `--newer-than`, `--search`, `--regex`, `--group-by`, `--group-count`, `--format=table`)
- `tmux-intray dismiss <id>` - Dismiss a specific notification
- `tmux-intray dismiss --all` - Dismiss all active notifications
- `tmux-intray clear` - Clear all items from the tray (alias for `dismiss --all`)
- `tmux-intray cleanup` - Clean up old dismissed notifications (configurable retention)

#### Navigation Commands
- `tmux-intray toggle` - Toggle the tray visibility
- `tmux-intray jump <id>` - Jump to the pane of a notification
- `tmux-intray status` - Show notification status summary
- `tmux-intray follow` - Monitor notifications in real-time

#### Utility Commands
- `tmux-intray help` - Show help message
- `tmux-intray version` - Show version information
- `tmux-intray status-panel` - Generate status bar output for tmux integration

## Documentation

Comprehensive documentation is available:

- [CLI Reference](docs/cli/CLI_REFERENCE.md) - Complete command reference
- [Configuration Guide](docs/configuration.md) - All environment variables and settings (including TUI settings persistence)
- [Storage Migration Guide](docs/storage-migration.md) - Gradual SQLite opt-in plan, safeguards, and rollback
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues and solutions
- [Release Notes](RELEASE_NOTES.md) - Current rollout status and release communication
- [Advanced Filtering Example](examples/advanced-filtering.sh) - Complex filter combinations
- [Man page](man/man1/tmux-intray.1) - Traditional manual page (view with `man -l man/man1/tmux-intray.1`)

Documentation is automatically generated from the command-line help texts.

### TUI Settings Persistence

The TUI automatically saves your preferences on exit:
- **Settings file**: `~/.config/tmux-intray/settings.json`
- **Manual save**: Press `:w` in TUI command mode
- **Auto-save**: Settings are saved when you quit (q, :q, Ctrl+C)
- **Reset settings**: Run `tmux-intray settings reset`
- **View settings**: Run `tmux-intray settings show`

See [Configuration Guide](docs/configuration.md) for details on available settings.

### Notification Levels

Notifications can have severity levels: `info` (default), `warning`, `error`, `critical`. Levels are used for filtering and color-coded display.

```bash
# Add a notification with a level
tmux-intray add --level=error "Something went wrong"

# Filter notifications by level
tmux-intray list --level=error

# The `status` command shows counts per level
tmux-intray status
```

### Advanced Filtering

tmux-intray's `list` command supports powerful filtering options to help you find notifications based on various criteria.

**Common Filters:**
- `--session <id>` / `--window <id>` / `--pane <id>` – filter by tmux context
- `--older-than <days>` / `--newer-than <days>` – time‑based filtering
- `--search <pattern>` – substring search in messages (use `--regex` for regular expressions)
- `--group-by <field>` – group notifications by session, window, pane, or level
- `--group-count` – show only group counts (requires `--group-by`)

**Examples:**

```bash
# Notifications from a specific session with error level
tmux-intray list --session=work --level=error

# Notifications older than 7 days but newer than 1 day
tmux-intray list --older-than=7 --newer-than=1

# Search for notifications containing "error" (substring match)
tmux-intray list --search=error

# Regex search for patterns
tmux-intray list --search='ERR[0-9]+' --regex

# Group notifications by session
tmux-intray list --group-by=session

# Show only group counts
tmux-intray list --group-by=session --group-count
```

For a comprehensive list of filters and detailed examples, see the [CLI Reference](docs/cli/CLI_REFERENCE.md) and the [advanced filtering example](examples/advanced-filtering.sh).

## Tmux Plugin Installation

### What the Plugin Provides

The tmux plugin enhances the CLI with tmux-specific features:

- **Key bindings**: `prefix+I` shows notifications in real-time (follow mode), `prefix+J` opens interactive TUI in popup window
- **Status bar integration**: Real-time notification count in status-right
- **Pane context capture**: Automatic tracking of notification origins
- **Environment setup**: Proper PATH and configuration for CLI access

### Plugin Configuration

Add to your `.tmux.conf`:

```bash
# Basic setup
run '~/.local/share/tmux-plugins/tmux-intray/tmux-intray.tmux'

# Optional: Custom status bar configuration
set -g status-right "#(tmux-intray status-panel) %H:%M"
```

### Key Bindings

- `prefix + I` - Show notifications in real-time (follow mode)
- `prefix + J` - Open interactive TUI in popup window

### Interactive TUI

The `tmux-intray tui` command provides an interactive terminal user interface for managing notifications. It can be accessed via the `prefix + J` key binding, which opens the TUI in a tmux popup window (80% width, 80% height).

**TUI Key Bindings:**

| Key          | Action                                     |
|--------------|--------------------------------------------|
| j/k          | Navigate up/down in the list               |
| /            | Enter search mode                          |
| :            | Enter command mode                         |
| ESC          | Exit search/command mode, or quit TUI      |
| d            | Dismiss selected notification              |
| Enter        | Jump to pane (or execute command in command mode) |
| q            | Quit TUI                                   |
| :w           | Save settings manually                     |
| i            | Edit search query (when in search mode)    |

**Features:**
- Table view with TYPE, STATUS, SUMMARY, SOURCE, AGE columns
- Real-time search filtering
- Vim-like navigation
- Dismiss notifications directly
- Jump to source panes
- Notifications sorted by most recent first
- **Settings persistence**: TUI preferences (column order, sort order, filters, view mode) are automatically saved on exit and restored on startup
- Settings file location: `~/.config/tmux-intray/settings.json`

### Status Bar Integration

The plugin updates `@tmux_intray_active_count` and provides `status-panel` command for status bar display. Configure the format in `~/.config/tmux-intray/config.sh`:

```bash
# Status panel formats: compact, detailed, count-only
export TMUX_INTRAY_STATUS_FORMAT="compact"
```

## Usage

### Basic Workflow

1. **Add notifications** from scripts or manually:
   ```bash
   tmux-intray add "Build completed"
   tmux-intray add --level=warning "High memory usage detected"
   ```

2. **Review notifications** when ready:
   ```bash
   tmux-intray list
    # or use tmux key bindings: prefix+I (follow mode) or prefix+J (interactive TUI)
   ```

3. **Manage notifications**:
   ```bash
   tmux-intray dismiss 1          # Dismiss specific notification
   tmux-intray clear              # Clear all notifications
   tmux-intray jump 2             # Jump to notification source pane
   ```

### Hooks System

tmux-intray supports a powerful hooks system that allows you to execute custom scripts before and after notification events. This makes tmux-intray extensible and integratable with other systems.

**Key features:**
- **Hook points**: `pre-add`, `post-add`, `pre-dismiss`, `post-dismiss`, `cleanup`
- **Configurable failure modes**: ignore, warn, or abort on hook failure
- **Environment variables**: Provide notification context to hook scripts

**Example hook script** (`~/.config/tmux-intray/hooks/post-add.sh`):
```bash
#!/bin/bash
# Send notification to external system
curl -X POST https://api.example.com/notifications \
  -d "message=$TMUX_INTRAY_MESSAGE&level=$TMUX_INTRAY_LEVEL"
```

### Cleanup

tmux-intray automatically cleans up old dismissed notifications to prevent storage bloat. The `tmux-intray cleanup` command removes notifications that have been dismissed for more than a configured number of days (default: 30 days).

**Retention Configuration:**
- Set `TMUX_INTRAY_AUTO_CLEANUP_DAYS` environment variable (e.g., `export TMUX_INTRAY_AUTO_CLEANUP_DAYS=7`)
- The default is 30 days; set to `0` to disable auto‑cleanup.

**Manual Cleanup:**
```bash
# Dry-run to see what would be removed
tmux-intray cleanup --dry-run

# Remove notifications dismissed more than 7 days ago
tmux-intray cleanup --days=7

# Remove all dismissed notifications (use with caution)
tmux-intray cleanup --days=0
```

**Automation:**
- Cleanup can be run periodically via cron or systemd timer.
- Hooks are available for `cleanup` and `post-cleanup` to integrate with external systems.

For detailed configuration options, see the [configuration guide](docs/configuration.md).

### Debugging

Enable debug logging:
```bash
export TMUX_INTRAY_DEBUG=1
tmux-intray add "Test notification"
```

Debug logs are written to `~/.local/state/tmux-intray/debug.log`.

## Fzf Integration

tmux-intray works well with [fzf](https://github.com/junegunn/fzf) for interactive notification management. The `--format=table` output is structured and easy to parse.

### Basic fzf Examples

#### Interactive Notification Dismissal
Select a notification with fzf and dismiss it:
```bash
tmux-intray list --format=table | tail -n +4 | fzf --header-lines=0 --with-nth=2.. | awk '{print $1}' | xargs -I {} tmux-intray dismiss {}
```

#### Multi-Select Batch Dismissal
Select multiple notifications and dismiss them all:
```bash
tmux-intray list --format=table | tail -n +3 | fzf --multi --header-lines=0 --with-nth=2.. | awk '{print $1}' | xargs tmux-intray dismiss
```

#### Jump to Notification Pane
Select a notification and jump to its source pane:
```bash
tmux-intray list --format=table | tail -n +3 | fzf --header-lines=0 --with-nth=2.. | awk '{print $1}' | xargs -I {} tmux-intray jump {}
```

#### Fzf Preview with tmux Pane Context
Preview the pane metadata and recent pane output (inside tmux):
```bash
tmux-intray list --format=table | tail -n +3 | fzf --header-lines=0 \
  --with-nth=2.. \
  --preview='tmux display-message -p -t {3} "#{session_name}:#{window_index}.#{pane_index} #{pane_current_command} #{pane_current_path}"; echo; tmux capture-pane -pt {3} -S -20' \
  --preview-window=right:60%:wrap \
  | awk '{print $1}' | xargs -I {} tmux-intray jump {}
```

### Reusable Shell Functions

Add these to your `.bashrc` or `.zshrc`:

```bash
# Fuzzy dismiss notifications
tray-dismiss() {
  local selected=$(tmux-intray list --format=table | tail -n +3 | fzf --header-lines=0 --with-nth=2.. | awk '{print $1}')
  if [[ -n "$selected" ]]; then
    echo "Dismissing: $selected"
    tmux-intray dismiss $selected
  fi
}

# Fuzzy jump to notification pane
tray-jump() {
  local selected=$(tmux-intray list --format=table | tail -n +3 | fzf --header-lines=0 --with-nth=2.. | awk '{print $1}')
  if [[ -n "$selected" ]]; then
    echo "Jumping to: $selected"
    tmux-intray jump $selected
  fi
}

# Multi-select dismissal
tray-dismiss-multi() {
  local selected=$(tmux-intray list --format=table | tail -n +3 | fzf --multi --header-lines=0 --with-nth=2.. | awk '{print $1}' | tr '\n' ' ')
  if [[ -n "$selected" ]]; then
    echo "Dismissing: $selected"
    tmux-intray dismiss $selected
  fi
}
```

### Tmux Key Bindings with fzf

Add to your `.tmux.conf` for quick access:

```bash
# Bind prefix + F to fuzzy dismiss
bind-key -T prefix F run-shell "tmux-intray list --format=table | tail -n +3 | fzf --header-lines=0 --with-nth=2.. | awk '{print \\\$1}' | xargs tmux-intray dismiss"

# Bind prefix + J to fuzzy jump
bind-key -T prefix J run-shell "tmux-intray list --format=table | tail -n +3 | fzf --header-lines=0 --with-nth=2.. | awk '{print \\\$1}' | xargs tmux-intray jump"
```

### How It Works

1. `tail -n +3` skips the table header (first 3 lines)
2. `fzf --header-lines=0` treats the input as headerless
3. `--with-nth=2..` hides the ID column from display
4. `awk '{print $1}'` extracts the notification ID
5. `xargs tmux-intray dismiss` or `tmux-intray jump` runs the action

## Architecture Overview

tmux-intray is built with a modular architecture that separates concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                     tmux-intray System                       │
├─────────────────┬───────────────────────────────────────────┤
│   CLI Core      │           Tmux Integration                │
│   (Go-based)    │        (tmux-intray.tmux)                 │
├─────────────────┼───────────────────────────────────────────┤
│ • Storage       │ • Key bindings (prefix+I, prefix+J)       │
│ • Commands      │ • Status bar updates                      │
│ • Hooks system  │ • Pane context capture                    │
│ • Configuration │ • Environment setup                       │
└─────────────────┴───────────────────────────────────────────┘
                            │
                     ┌──────┴──────┐
                     │   tmux      │
                     │  session    │
                     └─────────────┘
```

### Core Components

1. **Storage Layer**: File-based TSV storage with `flock` locking in `~/.local/state/tmux-intray/`
2. **Command Layer**: Individual command implementations in `cmd/*.go`
3. **Tmux Integration**: Plugin loader in `tmux-intray.tmux` and status panel command (`tmux-intray status-panel`)

### Data Flow

1. **Notification Creation**: `tmux-intray add` → storage layer → hooks execution
2. **Notification Retrieval**: `tmux-intray list` → storage query → formatted output
3. **Tmux Integration**: Plugin updates status bar via `@tmux_intray_active_count`
4. **Pane Navigation**: `tmux-intray jump` uses captured pane IDs to navigate

## Debugging

### Common Issues

**CLI not found after installation**
- Ensure installation directory is in PATH
- For npm: May require `npm bin -g` to be in PATH

**Tmux plugin not loading**
- Check `.tmux.conf` syntax
- Verify plugin path exists
- Reload tmux with `tmux source-file ~/.tmux.conf`

**Notifications not appearing in status bar**
- Ensure `status-panel` command works: `tmux-intray status-panel`
- Check status-right configuration in `.tmux.conf`

### Debug Logging

```bash
# Enable verbose debugging
export TMUX_INTRAY_DEBUG=2

# Check debug log
tail -f ~/.local/state/tmux-intray/debug.log
```

## Testing

Run the test suite:
```bash
make test
# or directly
bats tests/
```

Plugin tests (OpenCode integration) are located in `opencode/plugins/opencode-tmux-intray/` and can be run with:
```bash
cd opencode/plugins/opencode-tmux-intray && npm test
```

## Linting

Check code style:
```bash
make lint
```

## License

tmux-intray is licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

## Additional Resources

- [GitHub Repository](https://github.com/cristianoliveira/tmux-intray)
- [Issue Tracker](https://github.com/cristianoliveira/tmux-intray/issues)
- [Contributing Guidelines](CONTRIBUTING.md)

## Acknowledgments

tmux-intray builds upon the tmux plugin ecosystem and follows XDG Base Directory Specification for configuration and data storage.
