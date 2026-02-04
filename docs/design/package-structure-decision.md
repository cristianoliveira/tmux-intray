# Package Structure Decision: Hybrid Cobra + internal/commands Approach

**Date**: 2026-02-03  
**Status**: Accepted

## Decision

Adopt a hybrid package structure where Cobra command definitions remain in `cmd/` as thin wrappers, and business logic moves to `internal/commands/<command>/` packages.

## Rationale

- **Leverage Cobra's CLI capabilities**: Cobra provides a robust CLI framework with built-in help, flag parsing, and subcommand dispatch. Keeping command definitions in `cmd/` aligns with Cobra's conventions and makes the CLI layer explicit.
- **Maintain modular, testable code separation**: Moving business logic to `internal/commands/` separates concerns—CLI parsing from command implementation. This enables unit testing of command logic without CLI dependencies and keeps each package focused.
- **Align with Go conventions**: The `cmd` directory is for application entry points, while `internal` holds private implementation code. This hybrid approach respects both conventions.
- **Preserve design intent**: The original design (`docs/design/go-package-structure.md`) placed commands in `internal/commands/`. This decision retains that modular layout while adding a thin Cobra wrapper in `cmd/` for CLI orchestration.
- **Facilitate gradual migration**: The scaffold already exists in `.gwt/gocli/internal/commands/`. Moving it to the root `internal/commands/` and wiring it to Cobra commands provides a clear migration path.

## Implications

1. **Move scaffold**: Relocate the command scaffold from `.gwt/gocli/internal/commands/` to root `internal/commands/`.
2. **Update Cobra command files**: Cobra command definitions (currently flat files in `cmd/`) must delegate to the corresponding `internal/commands/<command>` package's `Run` function.
3. **Adjust imports**: Update import paths in `cmd/` files to point to `github.com/cristianoliveira/tmux-intray/internal/commands/...`.
4. **Preserve command interface**: Each command package continues to export a `Run(args []string) error` function (or similar signature) for easy integration.
5. **Maintain separation**: Cobra handles flag parsing, validation, and help text; `internal/commands` handles business logic, storage interaction, and core functionality.

## Example Structure After Decision

```
github.com/cristianoliveira/tmux-intray/
├── cmd/
│   └── tmux-intray/
│       ├── main.go                 # CLI entry point, initializes Cobra
│       ├── add.go                  # Cobra command: delegates to internal/commands/add
│       ├── list.go                 # Cobra command: delegates to internal/commands/list
│       └── ... (other commands)
├── internal/
│   └── commands/
│       ├── add/
│       │   └── add.go              # Command logic, exports Run()
│       ├── list/
│       │   └── list.go
│       └── ...
└── go.mod
```

## References

- `docs/design/go-package-structure.md` (original design document)
- `verification-report.md` (notes on package structure deviation)
- Cobra documentation: https://github.com/spf13/cobra