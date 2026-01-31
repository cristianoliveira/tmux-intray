# Problem Statement

Tmux provides signals (bells, activity markers, and messages) but not
notifications.

These signals are ephemeral, lose context, and cannot be reviewed or
navigated later. In workflows with many panes and long-running processes,
users often miss important events or forget where they happened.

There is no tmux-native way to:
- persist notifications
- associate them with a pane or window
- review them later
- jump back to their origin

As a result, users rely on memory, constant scanning, or OS-level
notifications, breaking the tmux mental model.
