# Hooks

## Responsibility

Discover and execute documented pre/post hooks while managing asynchronous work and shutdown flushing.

## Boundaries

- Keep hook points and payloads explicit and stable.
- Isolate process execution and runtime configuration from core notification rules.
- Never silently discard hook failures; report or log according to calling contract.
- Ensure shutdown waits for pending work without leaking goroutines.

## Tests

Use temporary executable fixtures and bounded timeouts. Cover success, failure, missing hooks, cancellation, and shutdown.

Run: `go test ./internal/hooks`
