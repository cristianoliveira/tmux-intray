# Repository Guidance

## Product compass

Build a quiet, persistent, tmux-aware notification inbox. Prefer composable output, explicit behavior, minimal configuration, and one obvious path. See [docs/philosophy.md](docs/philosophy.md).

## Global engineering rules

- Use test-driven development; cover happy and unhappy paths.
- Keep Cobra commands as composition and transport code. Put reusable behavior under `internal/`.
- Preserve dependency direction defined in [docs/design/import-layering-map.md](docs/design/import-layering-map.md).
- Inject dependencies through interfaces or factories; composition roots own concrete wiring.
- Use lowercase errors without trailing punctuation. Wrap causes with `%w`.
- Use `snake_case` for TOML keys and sections.
- Do not hand-edit generated code under `internal/storage/sqlite/sqlcgen/`.

## Local guidance index

Read nearest nested `AGENTS.md` before changing code:

- [`cmd/tmux-intray/`](cmd/tmux-intray/AGENTS.md) — CLI composition and commands
- [`internal/app/`](internal/app/AGENTS.md) — use-case orchestration
- [`internal/core/`](internal/core/AGENTS.md) — notification and tmux business behavior
- [`internal/domain/`](internal/domain/AGENTS.md) — pure domain model
- [`internal/storage/`](internal/storage/AGENTS.md) — persistence boundary
- [`internal/tmux/`](internal/tmux/AGENTS.md) — tmux process adapter
- [`internal/config/`](internal/config/AGENTS.md) — TOML configuration
- [`internal/hooks/`](internal/hooks/AGENTS.md) — hook execution
- [`internal/settings/`](internal/settings/AGENTS.md) — persisted user settings
- [`internal/tui/`](internal/tui/AGENTS.md) — interactive presentation
- [`internal/format/`](internal/format/AGENTS.md) — CLI output
- [`internal/search/`](internal/search/AGENTS.md) — search strategies

Development commands and full package map live in [DEVELOPMENT.md](DEVELOPMENT.md).


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
