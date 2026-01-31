# tmux-notify – Full Design Notes

This document set captures the full problem statement, user stories,
functional requirements, storage design, and implementation reasoning
for a tmux-native notification system.

Goal:
Provide persistent, actionable notifications inside tmux, allowing users
to jump back to the pane/window that emitted them — without leaving tmux.
