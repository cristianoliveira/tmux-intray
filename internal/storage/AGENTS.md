# Storage

## Responsibility

Implement persistent notification storage and adapt SQLite records to domain repository contracts.

## Boundaries

- Keep storage-specific types behind adapters.
- Preserve transactional behavior and explicit locking where required.
- Put SQLite implementation in `sqlite/`; callers should depend on storage or domain interfaces.
- Never hand-edit `sqlite/sqlcgen/`; regenerate it from SQL/schema sources.
- Map database values to domain states and levels in one place.

## Tests

Use isolated temporary databases. Cover persistence, reopening, invalid data, cleanup, and transaction failure paths.

Run: `go test ./internal/storage/...`
