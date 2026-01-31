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
