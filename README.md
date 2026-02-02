# tmux-intray

<img width="300" height="300" alt="tmux-intray-300" src="https://github.com/user-attachments/assets/4fd9f030-9bb3-43a7-b800-c0d0f479e2a5" align="right" />

A quiet inbox for things that happen while you're not looking.

<div>

tmux-intray provides a persistent in-tmux in-tray where panes, windows, and scripts can drop messages and events without interrupting your flow. Instead of loud notifications or forced context switches, events accumulate calmly until you're ready to review them. Each item keeps its origin, survives pane and window changes, and can be inspected, jumped to, or cleared at your own pace. It's designed for deferred attention: notice now if you want, act later when it makes sense.
</div>


## Working in Progress

🚧🚧 This plugin is in active development at the moment. It started as a opencode plugin but grew to it's own project.
I use it in my daily basis, I'm a heavy tmux user and so far it works great! At this stage of development I can't promise there won't be
breaking changes.

## Summary

Quick links to key sections:

### Main Sections
- [Installation Options](#installation-options)
- [CLI Installation](#cli-installation)
- [Tmux Plugin Installation](#tmux-plugin-installation)
- [Usage](#usage)
- [Architecture Overview](#architecture-overview)
- [Debugging](#debugging)
- [Testing](#testing)
- [Linting](#linting)
- [License](#license)

### Quick Start
- **Full Setup (Recommended)**: Install CLI + Tmux Plugin via [One-click installation](#one-click-installation)
- **CLI Only**: Install via [Homebrew](#homebrew) or [npm](#npm) for tmux-integrated use
- **Plugin Only**: Install via [Tmux Plugin Manager](#using-tmux-plugin-manager-recommended) if CLI already installed

## Basic usage

```bash
tmux-intray add "my message!"
tmux-intray list 
tmux-intray show | fzf | awk '{ print $1 }' | xargs -I {} tmux-intray jump {}
```

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

#### Homebrew (macOS/Linux)

```bash
# Install from GitHub repository (formula is in the repo)
brew install cristianoliveira/tmux-intray/tmux-intray
```

#### npm

```bash
npm install -g tmux-intray
```

#### Go

```bash
go install github.com/cristianoliveira/tmux-intray/cmd/tmux-intray@latest
```

#### From Source

```bash
git clone https://github.com/cristianoliveira/tmux-intray.git
cd tmux-intray
make install
```

**Note**: The CLI requires tmux to be running for most commands. Installations via package managers provide the CLI only; you'll need to manually set up tmux integration if desired.

### Option 3: Tmux Plugin Only

If you already have the CLI installed (e.g., via Homebrew or npm), install just the tmux integration:

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
- **Bash**: Core implementation is bash-based (4.0+)
- **Standard Unix utilities**: flock, awk, grep, sed, etc.

### CLI Commands

```bash
$ tmux-intray --help
```

#### Core Commands
- `tmux-intray add <message>` - Add a new item to the tray (options: `--level`, `--session`, `--window`, `--pane`, `--no-associate`)
- `tmux-intray list` - List notifications with filters and formats (e.g., `--active`, `--dismissed`, `--all`, `--level`, `--pane`, `--format=table`)
- `tmux-intray dismiss <id>` - Dismiss a specific notification
- `tmux-intray dismiss --all` - Dismiss all active notifications
- `tmux-intray clear` - Clear all items from the tray (alias for `dismiss --all`)

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
- [Man page](man/man1/tmux-intray.1) - Traditional manual page (view with `man -l man/man1/tmux-intray.1`)

Documentation is automatically generated from the command-line help texts.

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

## Tmux Plugin Installation

### What the Plugin Provides

The tmux plugin enhances the CLI with tmux-specific features:

- **Key bindings**: `prefix+i` toggles tray, `prefix+I` shows notifications
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

- `prefix + i` - Toggle tray visibility
- `prefix + I` - Show all notifications

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
   # or use tmux key binding: prefix+I
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

### Debugging

Enable debug logging:
```bash
export TMUX_INTRAY_DEBUG=1
tmux-intray add "Test notification"
```

Debug logs are written to `~/.local/state/tmux-intray/debug.log`.

## Architecture Overview

tmux-intray is built with a modular architecture that separates concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                     tmux-intray System                       │
├─────────────────┬───────────────────────────────────────────┤
│   CLI Core      │           Tmux Integration                │
│  (bash-based)   │        (tmux-intray.tmux)                 │
├─────────────────┼───────────────────────────────────────────┤
│ • Storage       │ • Key bindings (prefix+i, prefix+I)       │
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
2. **Command Layer**: Individual command implementations in `commands/*.sh`
3. **Library Modules**: Core functions in `lib/*.sh` (core, storage, config, hooks)
4. **Tmux Integration**: Plugin loader in `tmux-intray.tmux` and status panel in `scripts/status-panel.sh`

### Data Flow

1. **Notification Creation**: `tmux-intray add` → storage layer → hooks execution
2. **Notification Retrieval**: `tmux-intray list` → storage query → formatted output
3. **Tmux Integration**: Plugin updates status bar via `@tmux_intray_active_count`
4. **Pane Navigation**: `tmux-intray jump` uses captured pane IDs to navigate

## Debugging

### Common Issues

**CLI not found after installation**
- Ensure installation directory is in PATH
- For Homebrew: `brew link tmux-intray`
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
