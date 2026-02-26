# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Project Philosophy

This project follows a minimalist, Unix‑style philosophy. See [Project Philosophy](./docs/philosophy.md) for design principles and rationale.

## Development Guide

Detailed development guidelines in [DEVELOPMENT.md](./DEVELOPMENT.md).

### Essential Documentation

- **Package Structure**: See [Go Package Structure](./docs/design/go-package-structure.md)
- **Configuration**: See [Configuration Guide](./docs/configuration.md)
- **CLI Reference**: See [CLI Reference](./docs/cli/CLI_REFERENCE.md)
- **Hooks System**: See [Hooks Documentation](./docs/hooks.md)
- **Troubleshooting**: See [Troubleshooting Guide](./docs/troubleshooting.md)
- **Code design**: See [design](./docs/design/)
