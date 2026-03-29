# Project Philosophy

Quiet notifications. Persistent storage. Composable tools. Simple design.

---

## Quiet by Default

Notifications should never interrupt your flow. They wait silently until you're ready.

> A quiet inbox for things that happen while you're not looking.

---

## Persistence Matters

Important information survives session restarts and system reboots. Data is not ephemeral.

> SQLite database with transactional storage. Never lose what matters.

---

## Composable Design

Integrate with existing tools. Output is designed for pipes, scripts, and automation.

> JSON output for `jq`, text output for `awk`, table format for `fzf`.

---

## Simplicity Over Features

Every feature earns its place through demonstrated need. One obvious way to do things.

> Minimal configuration with sensible defaults. No complexity without purpose.

---

## Context is King

Notifications are tied to their origin. Pane, session, window—preserve and use this context.

> Jump back to where the notification originated. Navigate with one command.

---

## Explicit is Better Than Implicit

Behavior is clear and obvious. No magic, no surprises, no hidden side effects.

> Commands do exactly what they say. Errors explain what went wrong.

---

## Extensible Without Complexity

Extend through documented interfaces, not by modifying internals.

> Hooks, templates, plugins. Customize without understanding the codebase.

---

## One and Only One Way

For any task, there is one obvious way to do it. Reduces cognitive load.

> One configuration format (TOML). One storage backend (SQLite). One CLI.

---

## Design Tradeoffs

### Why SQLite?

**Complexity** vs. **Reliability**.

SQLite wins. ACID guarantees, efficient queries, transaction support for hooks.

### Why Go?

**Complexity** vs. **Distribution**.

Go wins. Single binary, cross-platform, type safety, better performance.

### Why Custom TUI?

**Maintenance** vs. **Consistency**.

Custom wins. Rich features, consistent UX, integrated with data model.

---

## Anti-Patterns

- ❌ **Noisy**: Sounds, alerts, forced attention → Silent, on-demand review
- ❌ **Multiple Configs**: "JSON, YAML, or TOML" → TOML only
- ❌ **Implicit Defaults**: "It just works... somehow" → Document everything
- ❌ **Golden Hammer**: "Use this for everything" → Do notifications well
- ❌ **Breakage**: "We changed everything" → Migration paths, backward compatibility

---

## Decision Framework

When facing a design question:

1. **Quiet?** Does it interrupt users unnecessarily?
2. **Persistent?** Will data survive restarts and crashes?
3. **Composable?** Does it work with other tools?
4. **Simple?** Is there a simpler way?
5. **Context-aware?** Does it preserve origin?
6. **Explicit?** Is behavior clear?
7. **Extensible?** Can users customize it?
8. **One way?** Or are we adding another?

If no—rethink the design.

---

## In Practice

```bash
# Quiet: Add notification, no output
tmux-intray add "Task completed"

# Persistent: Survives tmux restart
tmux kill-server && tmux new-session
tmux-intray list  # Still shows all notifications

# Composable: Pipe to jq, fzf, awk
tmux-intray status --format=json | jq '.active'
tmux-intray list | fzf | awk '{print $1}'

# Simple: One obvious command
tmux-intray add "message"
tmux-intray list
tmux-intray jump <id>

# Context: Jump to origin pane
tmux-intray jump 42  # Goes to exact pane

# Explicit: Clear what happens
tmux-intray jump --no-mark-read 42  # Don't mark as read

# Extensible: Hooks, templates, plugins
~/.config/tmux-intray/hooks/post-add/slack.sh
tmux-intray status --format='{{unread-count}} pending'

# One Way: Not "add", "create", "new", "append"
tmux-intray add "message"  # Just one way
```

---

> These principles guide the project. They're not absolute rules, but a compass for consistent, thoughtful decisions.
