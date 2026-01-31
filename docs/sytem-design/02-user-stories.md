# User Stories

## Core User Stories

### Receive persistent notifications
As a tmux user,
I want panes to be able to emit notifications,
so that I can be informed about important events even when I am not
watching that pane.

### Know notification origin
As a tmux user,
I want each notification to be associated with the pane and window
that emitted it,
so that I do not have to guess where it came from.

### Passive awareness
As a tmux user,
I want a subtle indicator that notifications exist,
so that I am aware of them without being interrupted.

### Review notifications
As a tmux user,
I want to list active notifications inside tmux,
so that I can review what happened at my own pace.

### Jump to source
As a tmux user,
I want to jump directly to the pane that emitted a notification,
so that I can immediately act on it.

### Dismiss notifications
As a tmux user,
I want to dismiss individual notifications or clear all notifications,
so that old alerts do not clutter my workflow.

## Lifecycle Stories

### Persistence
As a tmux user,
I want notifications to persist while tmux is running,
so that detach/attach or window switching does not lose them.

### Script integration
As a tmux user,
I want to emit notifications from shell scripts and commands,
so that long-running tasks can notify me automatically.

## Non-goals
- OS notification replacement
- Auto-jumping without user action
- Heavy TUI applications
