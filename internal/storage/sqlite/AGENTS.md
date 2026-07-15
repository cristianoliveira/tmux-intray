# SQLite Implementation

Implement storage operations, schema integration, and sqlc query use for notification persistence.

- Keep SQL behavior and record mapping inside storage boundaries.
- Preserve transaction, cleanup, and concurrency semantics.
- Treat `sqlcgen/` as generated output; change source SQL/schema and run `make sqlc-generate`.
- Test with isolated temporary databases and verify behavior after reopening.

Run: `go test ./internal/storage/sqlite`
