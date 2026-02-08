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

## Development Guide

Detailed development guidelines in [DEVELOPMENT.md](./DEVELOPMENT.md).

### Essential Documentation

- **Package Structure**: See [Go Package Structure](./docs/design/go-package-structure.md)
- **Testing Strategy**: See [Testing Strategy](./docs/testing/testing-strategy.md)
- **Configuration**: See [Configuration Guide](./docs/configuration.md)
- **CLI Reference**: See [CLI Reference](./docs/cli/CLI_REFERENCE.md)
- **Hooks System**: See [Hooks Documentation](./docs/hooks.md)
- **Troubleshooting**: See [Troubleshooting Guide](./docs/troubleshooting.md)

## Build/Lint/Test Commands

### Build
- `make install` – Install the CLI (go install)
- `make install-go` – Build Go binary locally (`./tmux-intray`)
- `make go-build` – Build Go binary to `./tmux-intray`

### Test
- `make tests` – Run all tests (Go + Bats)
- `go test ./...` – Run all Go tests
- `go test -v ./internal/core -run TestAddTrayItem` – Run single Go test
- `go test ./... -cover` – Run Go tests with coverage
- `make go-cover` – Run tests and generate coverage report
- `bats tests/` – Run all Bats tests
- `bats tests/basic.bats` – Run specific Bats test file

### Lint
- `make lint` – Run full linting (Go + shell)
- `make go-lint` – Run Go linting (gofmt + go vet)
- `./scripts/lint.sh` – Run ShellCheck linter
- `make security-check` – Run security-focused ShellCheck
- `gofmt -d .` – Check Go formatting
- `go vet ./...` – Run Go vet

### Format
- `make fmt` – Format all code (shfmt + gofmt)
- `make go-fmt` – Format Go code
- `shfmt -ln bash -i 4 -w <file>` – Format shell script
- `shfmt -ln bats -i 4 -w <file>` – Format Bats test file

### Quality Gates
- Pre‑commit hooks run automatically on `git commit`
- Manual run: `pre-commit run --all-files`
- `make all` – Run tests + lint (recommended before committing)
- `nix develop` – Enter dev environment (Go, bats, shellcheck, shfmt)

## Code Style Guidelines

### Go Code Style
- **Package Structure**: cmd/ (CLI entry points), internal/ (private packages), tests/ (Bats integration tests), scripts/ (helpers). See [Go Package Structure](./docs/design/go-package-structure.md)
- **Imports**: Group into stdlib, external, internal with blank lines. Sort alphabetically. Use `goimports` to automatically format imports and remove unused imports.
- **Formatting**: Tabs for indentation (enforced by gofmt), aim for 100 char lines, no trailing whitespace, always final newline.
- **Naming**: Packages `lowercase`, files `lowercase_with_underscores.go`, functions `PascalCase` (exported) / `camelCase` (private), constants `PascalCase` (exported) / `camelCase` (private), interfaces `PascalCase` ending in `er`, errors `Err` prefix.
- **Types**: Use concrete types unless interface needed. Use `string` for TSV field values. Use exported constants for field indices. Use `error` return values; never panic on expected errors.
- **Error Handling**: Always check and return errors. Wrap with `fmt.Errorf("context: %w", err)`. Define errors as `var ErrName = errors.New("...")`. Use `colors.Error()` for user-facing errors, `colors.Success()` for success. In tests, use `require.NoError(t, err)` and `assert.Error(t, err)`.
- **Testing**: Use `t.TempDir()` and `t.Setenv()`. Reset package state in `setupTest()`. Use table-driven tests. Follow `Test<FunctionName>` naming. Use subtests with `t.Run()`. Mock external dependencies. Use `github.com/stretchr/testify`. See [Testing Strategy](./docs/testing/testing-strategy.md)
- **Documentation**: Exported functions must have doc comments: `// FunctionName does X and returns Y.` Package comments at top: `// Package pkgname provides X.`
- **Cobra CLI**: Commands in `cmd/tmux-intray/` subdirectories. Each command has own package. Use `RunE` for error-prone commands. Hide completion command.

### Shell Script Style
- **Formatting**: 4 spaces indentation (no tabs), 80-100 char lines. Use `shfmt -ln bash -i 4 -w <file>`.
- **Naming**: Variables `lowercase_with_underscores`, constants `UPPERCASE_WITH_UNDERSCORES`, functions `snake_case`.
- **Error Handling**: Use `set -euo pipefail` at top. Validate early. Use `# shellcheck disable=SC1091` directives when needed.

## Project‑Specific Patterns

- **Storage (internal/storage)**: TSV format with file locking. State: "active"/"dismissed". Level: "info"/"warning"/"error". Field indices as package constants.
- **Tmux (internal/core)**: Mock `tmuxRunner` for testing. Use `GetCurrentTmuxContext()` for auto context detection. Escape/unescape messages.
- **Colors (internal/colors)**: `Error(msg)` (stderr, red), `Success(msg)` (stdout, green), `Warning(msg)` (stdout, yellow), `Info(msg)` (stdout, blue), `Debug(msg)` (stderr, cyan, only when TMUX_INTRAY_DEBUG). See [Configuration](./docs/configuration.md)
- **Hooks (internal/hooks)**: Async background operations. `Init()` to start, `Shutdown()` to stop, `WaitForPendingHooks()` to flush on exit. See [Hooks Documentation](./docs/hooks.md)

## Landing the Plane (Session Completion)

**MANDATORY WORKFLOW** (work NOT complete until `git push` succeeds):

1. File issues for remaining work
2. Run quality gates (if code changed): `make all`
3. Update issue status
4. PUSH TO REMOTE: `git pull --rebase && bd sync && git push` (MUST show "up to date")
5. Clean up: clear stashes, prune remote branches
6. Verify: All changes committed AND pushed
7. Hand off: Provide context for next session

**CRITICAL**: NEVER stop before pushing. NEVER say "ready to push when you are". If push fails, resolve and retry until it succeeds.

## Context Template for Subagents

```markdown
# Task: ${The task you are trying to accomplish}
## Problem
${The problem you are trying to solve}

## Solution / Research
${The solution you are proposing OR research topic}

## Acceptance Criteria
${Conditions that must be met for acceptance}

## Feedback to Leader (IMPORTANT)
${Stop/Start/Continue: Was it clear? What was unclear? Where did you lose time? How to improve?}

## Report
${Instructions for subagents to report results}
```

<!-- bv-agent-instructions-v1 -->

---

## Beads Workflow Integration

This project uses [beads_viewer](https://github.com/Dicklesworthstone/beads_viewer) for issue tracking. Issues are stored in `.beads/` and tracked in git.

### Essential Commands

```bash
# View issues (launches TUI - avoid in automated sessions)
bv

# CLI commands for agents (use these instead)
bd ready              # Show issues ready to work (no blockers)
bd list --status=open # All open issues
bd show <id>          # Full issue details with dependencies
bd create --title="..." --type=task --priority=2
bd update <id> --status=in_progress
bd close <id> --reason="Completed"
bd close <id1> <id2>  # Close multiple issues at once
bd sync               # Commit and push changes
```

### Workflow Pattern

1. **Start**: Run `bd ready` to find actionable work
2. **Claim**: Use `bd update <id> --status=in_progress`
3. **Work**: Implement the task
4. **Complete**: Use `bd close <id>`
5. **Sync**: Always run `bd sync` at session end

### Key Concepts

- **Dependencies**: Issues can block other issues. `bd ready` shows only unblocked work.
- **Priority**: P0=critical, P1=high, P2=medium, P3=low, P4=backlog (use numbers, not words)
- **Types**: task, bug, feature, epic, question, docs
- **Blocking**: `bd dep add <issue> <depends-on>` to add dependencies

### Session Protocol

**Before ending any session, run this checklist:**

```bash
git status              # Check what changed
git add <files>         # Stage code changes
bd sync                 # Commit beads changes
git commit -m "..."     # Commit code
bd sync                 # Commit any new beads changes
git push                # Push to remote
```

### Best Practices

- Check `bd ready` at session start to find available work
- Update status as you work (in_progress → closed)
- Create new issues with `bd create` when you discover tasks
- Use descriptive titles and set appropriate priority/type
- Always `bd sync` before ending session

<!-- end-bv-agent-instructions -->
