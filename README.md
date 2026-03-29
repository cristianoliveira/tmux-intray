# tmux-intray
[![CI](https://github.com/cristianoliveira/tmux-intray/actions/workflows/ci.yml/badge.svg)](https://github.com/cristianoliveira/tmux-intray/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/cristianoliveira/tmux-intray/branch/main/graph/badge.svg)](https://codecov.io/gh/cristianoliveira/tmux-intray)

<div style="display:flex">
    <div>
        A quiet inbox for things that happen while you're not looking.
        tmux-intray provides a persistent in-tmux in-tray where panes, windows, and scripts can drop messages and events without interrupting your flow. Instead of loud notifications or forced context switching.
    </div>
    <div>
      <img width="300" height="300" alt="tmux-intray-300" src="https://github.com/user-attachments/assets/4fd9f030-9bb3-43a7-b800-c0d0f479e2a5" align="right" />
    </div>
</div>
</br>
</br>

**Work in Progress**

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
- [Hooks system](#hooks-system)
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

## Basic Usage

```bash
tmux-intray add "my message!"
tmux-intray list
tmux-intray jump <id>

```

### Managing Notifications
Once messages arrive you can manage them with the following commands:
```bash
tmux-intray tui
# or using fzf
tmux-intray list | fzf | awk '{ print $1 }' | xargs -I {} tmux-intray jump {}
```

### Tmux Integration
We recommend attaching <prefix> + J to open TUI in popup window

```bash
bind-key -T prefix J run-shell "tmux popup -E -h 50% -w 70% 'tmux-intray tui'"
```

Using `tmux-intray status` create a status bar in `.tmux.conf`:

```bash
# Shows the tmux-intray status panel
set -g status-right "#(tmux-intray status --format='📨 {{unread-count}}/{{total-count}}') %H:%M %a %d-%b-%y"
```

See [tmux.conf](tmux-intray.tmux) for a full example.

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

#### Go (Recommended for Go users)

```bash
go install github.com/cristianoliveira/tmux-intray@latest
```

#### Nix (Flakes)

```bash
# Run directly from the flake
nix run github:cristianoliveira/tmux-intray

# Or build and install locally
nix build .#
nix run .# -- --version

# Install tmux-intray globally
nix profile install github:cristianoliveira/tmux-intray
```

#### From Source

```bash
git clone https://github.com/cristianoliveira/tmux-intray.git
cd tmux-intray
make install
```

#### Manual Plugin Installation

```bash
# Clone just the plugin files
git clone https://github.com/cristianoliveira/tmux-intray.git ~/.local/share/tmux-plugins/tmux-intray

# Add to .tmux.conf
echo "run '~/.local/share/tmux-plugins/tmux-intray/tmux-intray.tmux'" >> ~/.tmux.conf

# Reload tmux
tmux source-file ~/.tmux.conf
```

### Integrations With Code Agents

- [OpenCode](opencode/plugins/opencode-tmux-intray) - See the readme
- [Pi](pi/agent/extensions/tmux-intray) - See the Readme

### CLI Commands

```bash
$ tmux-intray --help
```

See the [CLI Reference](docs/cli/CLI_REFERENCE.md) for a complete list of commands and options.

## Documentation

Comprehensive documentation is available:

- [CLI Reference](docs/cli/CLI_REFERENCE.md) - Complete command reference
- [Status Command Guide](docs/status-command-guide.md) - Template variables, presets, real-world examples, and troubleshooting
- [Configuration Guide](docs/configuration.md) - All environment variables and settings (including TUI settings persistence)
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues and solutions
- [Advanced Filtering Example](examples/advanced-filtering.sh) - Complex filter combinations
- [Man page](man/man1/tmux-intray.1) - Traditional manual page (view with `man -l man/man1/tmux-intray.1`)

Documentation is automatically generated from the command-line help texts.

### TUI Settings Persistence

The TUI automatically saves your preferences on exit:
- **Settings file**: `~/.config/tmux-intray/tui.toml`
- **Auto-save**: Settings are saved when you quit (q, :q, Ctrl+C)
- **Reset settings**: Run `tmux-intray settings reset`
- **View settings**: Run `tmux-intray settings show`

See [Configuration Guide](docs/configuration.md) for details on available settings.

For a comprehensive list of filters and detailed examples, see the [CLI Reference](docs/cli/CLI_REFERENCE.md) and the [advanced filtering example](examples/advanced-filtering.sh).

### Hooks system

tmux-intray supports a hooks system that allows you to execute custom scripts before and after notification events. This makes tmux-intray extensible and integratable with other systems.

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

See more in the [hooks guide](docs/hooks.md)

### Debugging

Enable debug logging:
```bash
export TMUX_INTRAY_LOG_LEVEL=debug
tmux-intray add "Test notification"
```

Logs are written to stderr. For detailed debugging guidance, see the [Debugging Guide](./docs/debugging.md).

### Getting More Help

For detailed troubleshooting and debugging scenarios:
- **[Debugging Guide](./docs/debugging.md)** - Complete guide with examples for all common issues
- **[Configuration Guide](./docs/configuration.md)** - All environment variables and config options
- **[Troubleshooting Guide](./docs/troubleshooting.md)** - Additional solutions and tips

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

1. **Storage Layer**: SQLite database with transactional storage in `~/.local/state/tmux-intray/notifications.db`
2. **Command Layer**: Individual command implementations in `cmd/*.go`
3. **Tmux Integration**: Plugin loader in `tmux-intray.tmux` and status command (`tmux-intray status`)

### Data Flow

1. **Notification Creation**: `tmux-intray add` → storage layer → hooks execution
2. **Notification Retrieval**: `tmux-intray list` → storage query → formatted output
3. **Tmux Integration**: Plugin updates status bar via `@tmux_intray_active_count`
4. **Pane Navigation**: `tmux-intray jump` uses captured pane IDs to navigate

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
