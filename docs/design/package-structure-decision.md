# Package Structure Decision: Simple Flat cmd/ Structure

**Date**: 2026-02-03  
**Status**: Accepted

## Decision

Adopt a simple flat package structure where Cobra command definitions and business logic are combined in `cmd/*.go` files, with core functionality in internal packages.

## Rationale

- **Simplicity and reduced complexity**: Combining CLI definitions and business logic in `cmd/*.go` files reduces package nesting and simplifies the project structure.
- **Maintain clear separation of concerns**: Business logic still delegates to internal packages (`core`, `storage`, `config`, etc.), keeping core functionality independent.
- **Align with Go conventions**: The `cmd` directory is for application entry points; placing command implementations directly in `cmd/` follows common Go project layouts.
- **Refactor from previous design**: The original hybrid approach with `internal/commands/` was simplified after implementation experience showed unnecessary complexity.
- **Easier maintenance**: Fewer packages and imports reduce cognitive overhead and make the codebase easier to navigate.

## Implications

1. **Combine command logic**: Move business logic from `internal/commands/` packages into the corresponding `cmd/*.go` files.
2. **Update Cobra command files**: Cobra command definitions already exist in `cmd/`; they now contain both CLI parsing and business logic.
3. **Simplify imports**: Remove imports of `internal/commands/...`; commands directly use internal packages (`core`, `storage`, etc.).
4. **Preserve command interface**: Each command file exports a public function (e.g., `Add()`) for testing and reuse.
5. **Maintain separation**: Cobra handles flag parsing and help text; business logic delegates to internal packages.

## Example Structure After Decision

```
github.com/cristianoliveira/tmux-intray/
├── cmd/                       # CLI command implementations
│   ├── add.go                 # Add command (Cobra definition + business logic)
│   ├── list.go                # List command
│   ├── dismiss.go             # Dismiss command
│   ├── clear.go               # Clear command
│   ├── toggle.go              # Toggle command
│   ├── jump.go                # Jump command
│   ├── status.go              # Status command
│   ├── status-panel.go        # Status-panel command
│   ├── follow.go              # Follow command
│   ├── help.go                # Help command
│   ├── version.go             # Version command
│   ├── cleanup.go             # Cleanup command
│   ├── root.go                # Root command and CLI entry point
│   └── wrapper/               # Wrapper for Bash migration
│       └── main.go
├── internal/                  # Private application code
│   ├── core/                  # Core tmux interaction & tray management
│   ├── storage/               # File-based TSV storage with locking
│   ├── colors/                # Color output utilities
│   ├── config/                # Configuration loading
│   ├── hooks/                 # Hooks subsystem
│   └── tmuxintray/            # Tmux integration utilities
└── go.mod                     # Go module definition
```

## References

- `docs/design/go-package-structure.md` (original design document)
- `verification-report.md` (notes on package structure deviation)
- Cobra documentation: https://github.com/spf13/cobra