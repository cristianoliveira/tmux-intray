## Project Philosophy

This project follows minimalist, Unix‑style principles focused on quiet notifications, persistent storage, and composable design. See [Project Philosophy](docs/philosophy.md) for the guiding principles that inform all design decisions.

## Development Guide

Detailed development guidelines in [DEVELOPMENT.md](./DEVELOPMENT.md).

### Code Placement Rules

- `cmd/tmux-intray/` must stay pure entrypoints: command wiring, flags, validation, dependency injection, and calls into internal packages only.
- Do not add business logic or helper modules under `cmd/tmux-intray/`; move reusable behavior to `internal/` and test it there.

### Essential Documentation

- **Package Structure**: See [Go Package Structure](./docs/design/go-package-structure.md)
- **Configuration**: See [Configuration Guide](./docs/configuration.md)
- **CLI Reference**: See [CLI Reference](./docs/cli/CLI_REFERENCE.md)
- **Hooks System**: See [Hooks Documentation](./docs/hooks.md)
- **Troubleshooting**: See [Troubleshooting Guide](./docs/troubleshooting.md)
- **Code design**: See [design](./docs/design/)


## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
4. **Clean up** - Clear stashes, prune remote branches
5. **Verify** - All changes committed AND pushed
6. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
