# SQLite Storage Migration Guide (Gradual Opt-in)

> [!WARNING]
> SQLite storage is a beta opt-in path. TSV remains the default backend during this rollout phase.

This guide explains how to migrate safely from TSV to SQLite, how to roll back, and how to report feedback.

## Why opt in

- Better consistency through transactional writes.
- Improved behavior on larger datasets.
- SQL query layer generated with sqlc for typed, maintainable SQLite access.

## Rollout phases

1. **Default baseline (0%)**: Stay on `tsv`.
2. **Early opt-in (10-20%)**: Use `dual` mode and monitor behavior.
3. **Validated opt-in**: Move to `sqlite` after a stable period.
4. **Rollback ready**: Move back to `tsv` immediately if issues appear.

## Before you start

- Ensure tmux-intray is up to date.
- Locate your state directory:
  - Default: `~/.local/state/tmux-intray`
  - Override: `$TMUX_INTRAY_STATE_DIR`
- Optional backup of current TSV file:

```bash
cp "$TMUX_INTRAY_STATE_DIR/notifications.tsv" "$TMUX_INTRAY_STATE_DIR/notifications.tsv.bak"
```

## Step 1: Opt in with dual mode (recommended)

Set in your shell session:

```bash
export TMUX_INTRAY_STORAGE_BACKEND=dual
export TMUX_INTRAY_DUAL_READ_BACKEND=sqlite
export TMUX_INTRAY_DUAL_VERIFY_ONLY=0
```

Or persist in `~/.config/tmux-intray/config.sh`:

```bash
TMUX_INTRAY_STORAGE_BACKEND="dual"
TMUX_INTRAY_DUAL_READ_BACKEND="sqlite"
TMUX_INTRAY_DUAL_VERIFY_ONLY=0
```

Dual mode safeguards:

- TSV writes happen first and remain authoritative.
- SQLite write failures log warnings and do not block normal operation.

## Step 2: Promote to SQLite mode

After a stable period in dual mode, switch to:

```bash
export TMUX_INTRAY_STORAGE_BACKEND=sqlite
```

Or in `config.sh`:

```bash
TMUX_INTRAY_STORAGE_BACKEND="sqlite"
```

## Step 3: Roll back if needed

If anything looks wrong, revert instantly:

```bash
export TMUX_INTRAY_STORAGE_BACKEND=tsv
```

Or in `config.sh`:

```bash
TMUX_INTRAY_STORAGE_BACKEND="tsv"
```

Roll back scenarios:

- Unexpected query behavior or missing rows.
- Environment-specific SQLite file/permission issues.
- Regressions discovered during early adoption.

## sqlc-backed SQLite query layer

The SQLite implementation is built on sqlc-generated code.

- Query definitions: `internal/storage/sqlite/queries.sql`
- Schema: `internal/storage/sqlite/schema.sql`
- Generated code: `internal/storage/sqlite/sqlcgen/`

If you modify schema or queries:

```bash
make sqlc-generate
make sqlc-check
```

Keep generated files in sync with committed schema/query changes.

## Validation checklist for adopters

- Add/list/dismiss notifications in normal workflow.
- Confirm status panel counts still match expectations.
- Watch for warnings related to storage backend fallback.
- Run `tmux-intray list --all` and verify historical visibility.

## Feedback and bug reporting

Please open an issue and include:

- Backend mode (`tsv`, `dual`, `sqlite`)
- tmux-intray version
- OS + tmux version
- Reproduction steps
- Relevant debug output (`TMUX_INTRAY_DEBUG=1`)

Use the SQLite feedback issue template in `.github/ISSUE_TEMPLATE/sqlite-opt-in-feedback.yml`.
