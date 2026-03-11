# Tmux Intray shortcuts

This document lists keyboard shortcuts currently implemented in code.

## Tmux plugin bindings

These are tmux prefix bindings installed by `tmux-intray.tmux` and recommended:

| Shortcut | Context | Action |
|---|---|---|
| `prefix + J` | tmux | Open `tmux-intray tui` in a tmux popup |

## TUI shortcuts (normal mode)

Applies when the TUI is open and not in search input or confirmation mode.

| Shortcut | Action | Notes |
|---|---|---|
| `j` / `k` | Move selection down/up | Works in all list views |
| `gg` | Move to top | Two-key sequence |
| `G` | Move to bottom | |
| `Enter` | Jump to target | In grouped view, first expands/collapses a group row when applicable |
| `d` | Dismiss selected notification | |
| `D` | Dismiss selected group | Grouped view only; opens confirmation dialog |
| `R` | Mark selected notification as read | Uppercase `R` |
| `u` | Mark selected notification as unread | |
| `r` | Switch tab to Recents | |
| `a` | Switch tab to All | |
| `/` | Enter search input mode | |
| `v` | Cycle view mode | `compact -> detailed -> grouped -> search -> compact` |
| `?` | Toggle help text | |
| `q` | Quit TUI | Saves settings before quitting |
| `Esc` | Quit TUI | If not in search input |
| `Ctrl+c` | Quit TUI | Saves settings before quitting |

## Grouped view only

These shortcuts only have effect when current view mode is grouped.

| Shortcut | Action | Notes |
|---|---|---|
| `h` | Collapse current group node | No effect on leaf notification rows |
| `l` | Expand current group node | No effect on leaf notification rows |
| `za` | Toggle fold for current group | Two-key sequence |
| `zz` | Clear pending `z` prefix | Internal sequence behavior (no action) |

## Search input mode

Search input mode starts with `/` and ends with `Esc`.

| Shortcut | Action | Notes |
|---|---|---|
| Any printable character | Append to search query | Includes keys like `q`, `g`, `G`, `:` and others |
| `Backspace` | Delete previous character | |
| `Enter` | Jump to selected target | Keeps search mode active |
| `Esc` | Exit search input mode | Clears search query |
| `Ctrl+j` / `Ctrl+k` | Move selection down/up | Navigation while staying in search input |
| `Ctrl+h` / `Ctrl+l` | No-op | Explicitly handled without action |

### Tabs navigation

In search input mode, the following shortcuts are handled as tabs navigation:
- `Alt+1`: Switch to first tab (recents)
- `Alt+2`: Switch to second tab (all)

Note: Ctrl+number keys don't produce a distinct key in most terminals, so Alt+number is used instead.

### Search-context Ctrl fallback

In search contexts (search input mode and search view mode), `Ctrl+<letter>` falls back to the corresponding single-letter binding for implemented one-letter shortcuts.

Examples:
- `Ctrl+d` behaves like `d` (dismiss).
- `Ctrl+r` / `Ctrl+u` mark read/unread.

## Search view mode

When view mode is `search` but search input is not active, normal keybindings still work, including:
- `j` / `k`, `gg`, `G`
- `d`, `R`, `u`, `Enter`, `q`, `?`, `/`
- `v` (cycle view mode)

## Confirmation dialog mode

Confirmation mode is used for destructive grouped actions (for example, `D` on a group).

| Shortcut | Action |
|---|---|
| `y` / `Y` | Confirm action |
| `Enter` | Confirm action |
| `n` / `N` | Cancel action |
| `Esc` | Cancel action |
| `Ctrl+c` | Cancel and quit TUI |

## Not shortcuts

The following keys are currently handled as no-op and are intentionally not navigation shortcuts:
- `Up` / `Down` arrow keys
