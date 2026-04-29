# Pi Tmux Intray Extension

Send Pi agent lifecycle notifications to `tmux-intray`.

## What it does

The extension subscribes to Pi events and writes quiet tmux notifications:

- `agent_end` → `tmux-intray add --level=info -- "Task completed"`
- failed `tool_result` → `tmux-intray add --level=error -- "Tool error: <tool>"`
- `session_shutdown` → `tmux-intray add --level=warning -- "Session shutdown"`

It captures the current tmux session, window, and pane with:

```bash
tmux display-message -p '#{session_id}'
tmux display-message -p '#{window_id}'
tmux display-message -p '#{pane_id}'
```

Then forwards them as `--session`, `--window`, and `--pane` flags so `tmux-intray jump` can return to the right place.

Failures are logged to `/tmp/pi-tmux-intray.log` and never crash Pi.

## Requirements

- Pi `>= 0.70`
- `tmux-intray` CLI in `PATH`
- Running inside tmux for pane context

## Easy install

Install this repository as a Pi package:

```bash
pi install git:github.com/cristianoliveira/tmux-intray
```

Or from a local checkout:

```bash
pi install /path/to/tmux-intray
```

Restart Pi or run `/reload`.

## Try without installing

```bash
pi -e git:github.com/cristianoliveira/tmux-intray
```

## Configuration

By default the extension runs `tmux-intray` from `PATH`.

Override the binary path with either variable:

```bash
export TMUX_INTRAY_PATH=/absolute/path/to/tmux-intray
# or
export TMUX_INTRAY_BIN=/absolute/path/to/tmux-intray
```

## Manual install

Copy the extension directory to Pi's global extension directory:

```bash
mkdir -p ~/.pi/agent/extensions
cp -R pi/agent/extensions/tmux-intray ~/.pi/agent/extensions/tmux-intray
```

Then restart Pi or run `/reload`.
